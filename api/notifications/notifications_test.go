package notifications

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

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
	var notificator *notificationsfakes.FakeNotificator
	var notificationQueue *notificationsfakes.FakeNotificationQueue
	var repository *storagefakes.FakeStorage
	var wsUpgrader *ws.SmUpgrader

	BeforeSuite(func() {
		notificator = &notificationsfakes.FakeNotificator{}
		notificationQueue = &notificationsfakes.FakeNotificationQueue{}
		repository = &storagefakes.FakeStorage{}
		repository.ListReturns(&types.Notifications{}, nil)

		wsUpgrader = ws.NewUpgrader(&ws.UpgraderOptions{
			PingTimeout: time.Second * 5,
		})

		c := NewController(repository, wsUpgrader, notificator)

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
		It("consumer should receive the data from the queue", func() {
			notificationsChannel := make(chan *types.Notification, 10)
			notificator.RegisterConsumerReturns(notificationQueue, 0, nil)
			notificationQueue.ChannelReturns(notificationsChannel, nil)
			notificationsChannel <- &types.Notification{
				PlatformID: "1234",
			}

			wsconn, _, err := wsconnect(server.URL)
			// Expect(resp.StatusCode).To(Equal(http.StatusOK))
			Expect(err).ShouldNot(HaveOccurred())
			var r map[string]interface{}
			err = wsconn.ReadJSON(&r)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(r["platform_id"]).To(Equal("1234"))
		})
	})
})

func wsconnect(host string) (*websocket.Conn, *http.Response, error) {
	endpoint, _ := url.Parse(host)

	return websocket.DefaultDialer.Dial("ws://"+endpoint.Host, nil)
}
