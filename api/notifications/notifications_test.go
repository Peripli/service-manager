package notifications

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"github.com/gorilla/websocket"

	"github.com/Peripli/service-manager/pkg/web"

	"github.com/Peripli/service-manager/pkg/ws"

	"github.com/Peripli/service-manager/storage/storagefakes"

	"github.com/Peripli/service-manager/pkg/types"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestNotificationController(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Notification controller Suite")
}

var _ = Describe("Notification controller", func() {
	var serveMux *http.ServeMux
	var server *httptest.Server
	var notificator *storagefakes.FakeNotificator
	var notificationQueue *storagefakes.FakeNotificationQueue
	var repository *storagefakes.FakeStorage
	var wsServer *ws.Server

	BeforeSuite(func() {
		notificator = &storagefakes.FakeNotificator{}
		notificationQueue = &storagefakes.FakeNotificationQueue{}
		repository = &storagefakes.FakeStorage{}
		repository.ListReturns(&types.Notifications{}, nil)

		wsServer = ws.NewServer(&ws.Settings{
			PingTimeout:  time.Second * 5,
			WriteTimeout: time.Second * 5,
		})

		c := NewController(repository, wsServer, notificator)

		serveMux = http.NewServeMux()
		serveMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			webReq := &web.Request{
				Request: r,
			}
			webReq.SetResponseWriter(w)

			c.handleWS(webReq)
		})
		server = httptest.NewServer(serveMux)
	})

	AfterSuite(func() {
		if server != nil {
			server.Close()
		}
	})

	Context("when notificator returns queue", func() {
		var notificationsChannel chan *types.Notification
		var wsconn *websocket.Conn

		JustBeforeEach(func() {
			notificationsChannel = make(chan *types.Notification, 10)
			notificator.RegisterConsumerReturns(notificationQueue, 0, nil)
			notificationQueue.ChannelReturns(notificationsChannel)

			var err error
			wsconn, _, err = wsconnect(server.URL)
			Expect(err).ShouldNot(HaveOccurred())
		})

		AfterEach(func() {
			if wsconn != nil {
				wsconn.Close()
			}
		})

		Context("when notificator returns queue", func() {
			It("consumer should receive the data from the queue", func() {
				nextNotification := generateNotification()
				notificationsChannel <- nextNotification

				assertNotificationMessage(wsconn, notificationToMap(nextNotification))
			})
		})

		Context("when there are not received notifications", func() {
			var firstNotification *types.Notification
			BeforeEach(func() {
				firstNotification = generateNotification()
				repository.ListReturns(&types.Notifications{
					Notifications: []*types.Notification{firstNotification},
				}, nil)
			})

			It("consumer should receive them prior to receiving new notifications", func() {
				nextNotification := generateNotification()
				notificationsChannel <- nextNotification

				assertNotificationMessage(wsconn, notificationToMap(firstNotification))
				assertNotificationMessage(wsconn, notificationToMap(nextNotification))
			})
		})
	})
})

func assertNotificationMessage(conn *websocket.Conn, expected map[string]interface{}) {
	var r map[string]interface{}
	err := conn.ReadJSON(&r)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(r["platform_id"]).To(BeEquivalentTo(expected["platform_id"]))
}

func wsconnect(host string) (*websocket.Conn, *http.Response, error) {
	endpoint, _ := url.Parse(host)

	return websocket.DefaultDialer.Dial("ws://"+endpoint.Host, nil)
}

func generateNotification() *types.Notification {
	uid, err := uuid.NewV4()
	Expect(err).ShouldNot(HaveOccurred())
	uid2, err := uuid.NewV4()
	Expect(err).ShouldNot(HaveOccurred())

	return &types.Notification{
		Base: types.Base{
			ID: uid.String(),
		},
		Revision:   1,
		PlatformID: uid2.String(),
		Resource:   "notification",
		Type:       "CREATED",
	}
}

func notificationToMap(not *types.Notification) map[string]interface{} {
	return map[string]interface{}{
		"id":          not.ID,
		"revision":    not.Revision,
		"platform_id": not.PlatformID,
		"resource":    not.Resource,
		"type":        not.Type,
	}
}
