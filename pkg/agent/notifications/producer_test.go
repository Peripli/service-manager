package notifications_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	smnotifications "github.com/Peripli/service-manager/api/notifications"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/sirupsen/logrus"

	"github.com/Peripli/service-manager/pkg/agent/notifications"
	"github.com/Peripli/service-manager/pkg/agent/sm"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/gorilla/websocket"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var invalidRevisionStr = strconv.FormatInt(types.InvalidRevision, 10)

func TestNotifications(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Notifications Suite")
}

var _ = Describe("Notifications", func() {
	Describe("Producer Settings", func() {
		var settings *notifications.ProducerSettings
		BeforeEach(func() {
			settings = notifications.DefaultProducerSettings()
		})
		It("Default settings are valid", func() {
			err := settings.Validate()
			Expect(err).ToNot(HaveOccurred())
		})

		Context("When MinPingPeriod is invalid", func() {
			It("Validate returns error", func() {
				settings.MinPingPeriod = 0
				err := settings.Validate()
				Expect(err).To(HaveOccurred())
			})
		})

		Context("When ReconnectDelay is invalid", func() {
			It("Validate returns error", func() {
				settings.ReconnectDelay = -1 * time.Second
				err := settings.Validate()
				Expect(err).To(HaveOccurred())
			})
		})
		Context("When PongTimeout is invalid", func() {
			It("Validate returns error", func() {
				settings.PongTimeout = 0
				err := settings.Validate()
				Expect(err).To(HaveOccurred())
			})
		})
		Context("When PingPeriodPercentage is invalid", func() {
			It("Validate returns error", func() {
				settings.PingPeriodPercentage = 0
				err := settings.Validate()
				Expect(err).To(HaveOccurred())

				settings.PingPeriodPercentage = 100
				err = settings.Validate()
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when resync period is missing", func() {
			It("returns an error", func() {
				settings.ResyncPeriod = 0
				err := settings.Validate()
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("Producer", func() {
		var ctx context.Context
		var cancelFunc func()
		var logInterceptor *logWriter

		var producerSettings *notifications.ProducerSettings
		var smSettings *sm.Settings
		var producer *notifications.Producer
		var producerCtx context.Context

		var server *wsServer
		var group *sync.WaitGroup

		BeforeEach(func() {
			group = &sync.WaitGroup{}
			ctx, cancelFunc = context.WithCancel(context.Background())
			logInterceptor = &logWriter{}
			logInterceptor.Reset()
			var err error
			producerCtx, err = log.Configure(ctx, &log.Settings{
				Level:  "debug",
				Format: "text",
				Output: "ginkgowriter",
			})
			Expect(err).ToNot(HaveOccurred())
			log.AddHook(logInterceptor)
			server = newWSServer()
			server.Start()
			smSettings = &sm.Settings{
				URL:                  server.url,
				User:                 "admin",
				Password:             "admin",
				RequestTimeout:       2 * time.Second,
				NotificationsAPIPath: "/v1/notifications",
			}
			producerSettings = &notifications.ProducerSettings{
				MinPingPeriod:        100 * time.Millisecond,
				ReconnectDelay:       100 * time.Millisecond,
				PongTimeout:          20 * time.Millisecond,
				ResyncPeriod:         300 * time.Millisecond,
				PingPeriodPercentage: 60,
				MessagesQueueSize:    10,
			}
		})

		JustBeforeEach(func() {
			var err error
			producer, err = notifications.NewProducer(producerSettings, smSettings)
			Expect(err).ToNot(HaveOccurred())
		})

		notification := &types.Notification{
			Revision: 123,
			Payload:  json.RawMessage("{}"),
		}
		message := &notifications.Message{Notification: notification}

		AfterEach(func() {
			cancelFunc()
			server.Close()
		})

		Context("During websocket connect", func() {
			It("Sends correct basic credentials", func(done Done) {
				server.onRequest = func(r *http.Request) {
					defer GinkgoRecover()
					username, password, ok := r.BasicAuth()
					Expect(ok).To(BeTrue())
					Expect(username).To(Equal(smSettings.User))
					Expect(password).To(Equal(smSettings.Password))
					close(done)
				}
				producer.Start(producerCtx, group)
			})
		})

		Context("When last notification revision is not found", func() {
			BeforeEach(func() {
				requestCnt := 0
				server.onRequest = func(r *http.Request) {
					requestCnt++
					if requestCnt == 1 {
						server.statusCode = http.StatusGone
					} else {
						server.statusCode = 0
					}
				}
			})

			It("Sends restart message", func() {
				Eventually(producer.Start(producerCtx, group)).Should(Receive(Equal(&notifications.Message{Resync: true})))
			})
		})

		Context("When notifications is sent by the server", func() {
			BeforeEach(func() {
				server.onClientConnected = func(conn *websocket.Conn) {
					defer GinkgoRecover()
					err := conn.WriteJSON(notification)
					Expect(err).ToNot(HaveOccurred())
				}
			})

			It("forwards the notification on the channel", func() {
				Eventually(producer.Start(producerCtx, group)).Should(Receive(Equal(message)))
			})
		})

		Context("When connection is closed", func() {
			It("reconnects with last known notification revision", func(done Done) {
				requestCount := 0
				server.onRequest = func(r *http.Request) {
					defer GinkgoRecover()
					requestCount++
					if requestCount > 1 {
						rev := r.URL.Query().Get(smnotifications.LastKnownRevisionQueryParam)
						Expect(rev).To(Equal(strconv.FormatInt(notification.Revision, 10)))
						close(done)
					}
				}
				once := &sync.Once{}
				server.onClientConnected = func(conn *websocket.Conn) {
					once.Do(func() {
						defer GinkgoRecover()
						err := conn.WriteJSON(notification)
						Expect(err).ToNot(HaveOccurred())
						conn.Close()
					})
				}
				messages := producer.Start(producerCtx, group)
				Eventually(messages).Should(Receive(Equal(message)))
			})
		})

		Context("When websocket is connected", func() {
			It("Pings the servers within max_ping_period", func(done Done) {
				times := make(chan time.Time, 10)
				server.onClientConnected = func(conn *websocket.Conn) {
					times <- time.Now()
				}
				server.pingHandler = func(string) error {
					defer GinkgoRecover()
					now := time.Now()
					times <- now
					if len(times) == 3 {
						start := <-times
						for i := 0; i < 2; i++ {
							t := <-times
							pingPeriod, err := time.ParseDuration(server.maxPingPeriod)
							Expect(err).ToNot(HaveOccurred())
							Expect(t.Sub(start)).To(BeNumerically("<", pingPeriod))
							start = t
						}
						close(done)
					}
					server.conn.WriteControl(websocket.PongMessage, []byte{}, now.Add(1*time.Second))
					return nil
				}
				producer.Start(producerCtx, group)
			})
		})

		Context("When invalid last notification revision is sent", func() {
			BeforeEach(func() {
				server.lastNotificationRevision = "-5"
			})

			It("returns error", func() {
				producer.Start(producerCtx, group)
				Eventually(logInterceptor.String).Should(ContainSubstring("invalid last notification revision"))
			})
		})

		Context("When server does not return pong within the timeout", func() {
			It("Reconnects", func(done Done) {
				var initialPingTime time.Time
				var timeBetweenReconnection time.Duration
				server.pingHandler = func(s string) error {
					if initialPingTime.IsZero() {
						initialPingTime = time.Now()
					}
					return nil
				}
				server.onClientConnected = func(conn *websocket.Conn) {
					if !initialPingTime.IsZero() {
						defer GinkgoRecover()
						timeBetweenReconnection = time.Now().Sub(initialPingTime)
						pingPeriod, err := time.ParseDuration(server.maxPingPeriod)
						Expect(err).ToNot(HaveOccurred())
						Expect(timeBetweenReconnection).To(BeNumerically("<", producerSettings.ReconnectDelay+pingPeriod))
						close(done)
					}
				}
				producer.Start(producerCtx, group)
			})
		})

		Context("When notification is not a valid JSON", func() {
			It("Logs error and reconnects", func(done Done) {
				connectionAttempts := 0
				server.onClientConnected = func(conn *websocket.Conn) {
					defer GinkgoRecover()
					err := conn.WriteMessage(websocket.TextMessage, []byte("not-json"))
					Expect(err).ToNot(HaveOccurred())
					connectionAttempts++
					if connectionAttempts == 2 {
						Eventually(logInterceptor.String).Should(ContainSubstring("unmarshal"))
						close(done)
					}
				}
				producer.Start(producerCtx, group)
			})
		})

		assertLogContainsMessageOnReconnect := func(message string, done Done) {
			connectionAttempts := 0
			server.onClientConnected = func(conn *websocket.Conn) {
				defer GinkgoRecover()
				connectionAttempts++
				if connectionAttempts == 2 {
					Eventually(logInterceptor.String).Should(ContainSubstring(message))
					close(done)
				}
			}
			producer.Start(producerCtx, group)
		}

		Context("When last_notification_revision is not a number", func() {
			BeforeEach(func() {
				server.lastNotificationRevision = "not a number"
			})
			It("Logs error and reconnects", func(done Done) {
				assertLogContainsMessageOnReconnect(server.lastNotificationRevision, done)
			})
		})

		Context("When max_ping_period is not a number", func() {
			BeforeEach(func() {
				server.maxPingPeriod = "not a number"
			})
			It("Logs error and reconnects", func(done Done) {
				assertLogContainsMessageOnReconnect(server.maxPingPeriod, done)
			})
		})

		Context("When max_ping_period is less than the configured min ping period", func() {
			BeforeEach(func() {
				server.maxPingPeriod = (producerSettings.MinPingPeriod - 20*time.Millisecond).String()
			})
			It("Logs error and reconnects", func(done Done) {
				assertLogContainsMessageOnReconnect(server.maxPingPeriod, done)
			})
		})

		Context("When SM URL is not valid", func() {
			It("Returns error", func() {
				smSettings.URL = "::invalid-url"
				newProducer, err := notifications.NewProducer(producerSettings, smSettings)
				Expect(newProducer).To(BeNil())
				Expect(err).To(HaveOccurred())
			})
		})

		Context("When SM returns error status", func() {
			BeforeEach(func() {
				server.statusCode = http.StatusInternalServerError
			})
			It("Logs and reconnects", func(done Done) {
				connectionAttempts := 0
				server.onRequest = func(r *http.Request) {
					connectionAttempts++
					if connectionAttempts == 2 {
						server.statusCode = 0
					}
				}
				server.onClientConnected = func(conn *websocket.Conn) {
					Eventually(logInterceptor.String).Should(ContainSubstring("bad handshake"))
					close(done)
				}
				producer.Start(producerCtx, group)
			})
		})

		Context("When cannot connect to given address", func() {
			BeforeEach(func() {
				smSettings.URL = "http://bad-host"
			})
			It("Logs the error and tries to reconnect", func() {
				var err error
				producer, err = notifications.NewProducer(producerSettings, smSettings)
				Expect(err).ToNot(HaveOccurred())
				producer.Start(producerCtx, group)
				Eventually(logInterceptor.String).Should(ContainSubstring("no such host"))
				Eventually(logInterceptor.String).Should(ContainSubstring("Reattempting to establish websocket connection"))
			})
		})

		Context("When context is canceled", func() {
			It("Releases the group", func() {
				testCtx, cancel := context.WithCancel(context.Background())
				waitGroup := &sync.WaitGroup{}
				producer.Start(testCtx, waitGroup)
				cancel()
				waitGroup.Wait()
			})
		})

		Context("When messages queue is full", func() {
			BeforeEach(func() {
				producerSettings.MessagesQueueSize = 2
				server.onClientConnected = func(conn *websocket.Conn) {
					defer GinkgoRecover()
					err := conn.WriteJSON(notification)
					Expect(err).ToNot(HaveOccurred())
					err = conn.WriteJSON(notification)
					Expect(err).ToNot(HaveOccurred())
				}
			})
			It("Canceling context stops the goroutines", func() {
				producer.Start(producerCtx, group)
				Eventually(logInterceptor.String).Should(ContainSubstring("Received notification "))
				cancelFunc()
				Eventually(logInterceptor.String).Should(ContainSubstring("Exiting notification reader"))
			})
		})

		Context("When resync time elapses", func() {
			BeforeEach(func() {
				producerSettings.ResyncPeriod = 10 * time.Millisecond
			})
			It("Sends a resync message", func(done Done) {
				messages := producer.Start(producerCtx, group)
				Expect(<-messages).To(Equal(&notifications.Message{Resync: true})) // on initial connect
				Expect(<-messages).To(Equal(&notifications.Message{Resync: true})) // first time the timer ticks
				Expect(<-messages).To(Equal(&notifications.Message{Resync: true})) // second time the timer ticks
				close(done)
			}, (500 * time.Millisecond).Seconds())
		})

		Context("When a force resync (410 GONE) is triggered", func() {
			BeforeEach(func() {
				producerSettings.ResyncPeriod = 200 * time.Millisecond
				producerSettings.ReconnectDelay = 150 * time.Millisecond

				requestCnt := 0
				server.onRequest = func(r *http.Request) {
					requestCnt++
					if requestCnt == 1 {
						server.statusCode = 500
					} else {
						server.statusCode = 0
					}
				}
			})
			It("Resets the resync timer period", func() {
				messages := producer.Start(producerCtx, group)
				start := time.Now()
				m := <-messages
				t := time.Now().Sub(start)
				Expect(m).To(Equal(&notifications.Message{Resync: true})) // on initial connect
				Expect(t).To(BeNumerically(">=", producerSettings.ReconnectDelay))

				m = <-messages
				t = time.Now().Sub(start)
				Expect(m).To(Equal(&notifications.Message{Resync: true})) // first time the timer ticks
				Expect(t).To(BeNumerically(">=", producerSettings.ResyncPeriod+producerSettings.ReconnectDelay))
			})
		})
	})
})

type wsServer struct {
	url                      string
	server                   *httptest.Server
	mux                      *http.ServeMux
	conn                     *websocket.Conn
	connMutex                sync.Mutex
	lastNotificationRevision string
	maxPingPeriod            string
	statusCode               int
	onClientConnected        func(*websocket.Conn)
	onRequest                func(r *http.Request)
	pingHandler              func(string) error
}

func newWSServer() *wsServer {
	s := &wsServer{
		maxPingPeriod:            (100 * time.Millisecond).String(),
		lastNotificationRevision: strconv.FormatInt(types.InvalidRevision, 10),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/notifications", s.handler)
	s.mux = mux
	return s
}

func (s *wsServer) Start() {
	s.server = httptest.NewServer(s.mux)
	s.url = s.server.URL
}

func (s *wsServer) Close() {
	if s == nil {
		return
	}
	if s.server != nil {
		s.server.Close()
	}

	s.connMutex.Lock()
	defer s.connMutex.Unlock()
	if s.conn != nil {
		s.conn.Close()
	}
}

func (s *wsServer) handler(w http.ResponseWriter, r *http.Request) {
	if s.onRequest != nil {
		s.onRequest(r)
	}

	var err error
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	header := http.Header{}
	if s.lastNotificationRevision != invalidRevisionStr {
		header.Set(smnotifications.LastKnownRevisionHeader, s.lastNotificationRevision)
	}
	header.Set(smnotifications.MaxPingPeriodHeader, s.maxPingPeriod)
	if s.statusCode != 0 {
		w.WriteHeader(s.statusCode)
		w.Write([]byte{})
		return
	}

	s.connMutex.Lock()
	defer s.connMutex.Unlock()
	s.conn, err = upgrader.Upgrade(w, r, header)
	if err != nil {
		log.C(r.Context()).WithError(err).Error("Could not upgrade websocket connection")
		return
	}
	if s.pingHandler != nil {
		s.conn.SetPingHandler(s.pingHandler)
	}
	if s.onClientConnected != nil {
		s.onClientConnected(s.conn)
	}
	go reader(s.conn)
}

func reader(conn *websocket.Conn) {
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			return
		}
	}
}

type logWriter struct {
	strings.Builder
	bufferMutex sync.Mutex
}

func (w *logWriter) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (w *logWriter) Fire(entry *logrus.Entry) error {
	str, err := entry.String()
	if err != nil {
		return err
	}
	w.bufferMutex.Lock()
	defer w.bufferMutex.Unlock()
	_, err = w.WriteString(str)
	return err
}

func (w *logWriter) String() string {
	w.bufferMutex.Lock()
	defer w.bufferMutex.Unlock()
	return w.Builder.String()
}
