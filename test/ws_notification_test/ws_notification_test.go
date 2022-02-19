/*
 * Copyright 2018 The Service Manager Authors
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */

package ws_notification_test

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/Peripli/service-manager/pkg/query"

	"github.com/Peripli/service-manager/api/notifications"

	"github.com/spf13/pflag"

	"github.com/Peripli/service-manager/pkg/util"

	"github.com/Peripli/service-manager/pkg/types"

	"github.com/Peripli/service-manager/pkg/web"

	"github.com/Peripli/service-manager/storage"

	"github.com/gorilla/websocket"

	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestWsConn(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Notification test suite")
}

var pingTimeout = 1 * time.Second

var _ = Describe("WS", func() {
	var ctx *common.TestContext
	var wsconn *websocket.Conn
	queryParams := map[string]string{}
	var resp *http.Response
	var repository storage.Repository
	var platform *types.Platform
	const version = "2.0.1"
	wsconnectWithPlatform := func(queryParams map[string]string) (*types.Platform, *websocket.Conn, *http.Response, error) {
		platform := common.RegisterPlatformInSM(common.GenerateRandomPlatform(), ctx.SMWithOAuth, map[string]string{})
		conn, resp, err := ctx.ConnectWebSocket(platform, queryParams, nil)
		return platform, conn, resp, err
	}

	BeforeEach(func() {
		queryParams = map[string]string{}

		ctx = common.NewTestContextBuilderWithSecurity().
			WithEnvPreExtensions(func(set *pflag.FlagSet) {
				Expect(set.Set("websocket.ping_timeout", pingTimeout.String())).ShouldNot(HaveOccurred())
			}).Build()
		repository = ctx.SMRepository
		Expect(repository).ToNot(BeNil())

		platform = common.RegisterPlatformInSM(common.GenerateRandomPlatform(), ctx.SMWithOAuth, map[string]string{})
	})

	JustBeforeEach(func() {
		var err error
		wsconn, resp, err = ctx.ConnectWebSocket(platform, queryParams, nil)
		Expect(err).ShouldNot(HaveOccurred())
	})

	AfterEach(func() {
		if repository != nil {
			err := repository.Delete(context.Background(), types.NotificationType)
			if err != nil {
				Expect(err).To(Equal(util.ErrNotFoundInStorage))
			}
		}
		ctx.Cleanup()
	})

	Context("when non-websocket request is received", func() {
		It("should reject it", func() {
			ctx.SMWithBasic.GET(web.NotificationsURL).WithHeader(notifications.AgentVersionHeader, version).Expect().
				Status(http.StatusBadRequest).
				JSON().Object().Value("error").Equal("WebsocketUpgradeError")
			idCriteria := query.Criterion{
				LeftOp:   "id",
				Operator: query.EqualsOperator,
				RightOp:  []string{"basic-auth-default-test-platform"},
				Type:     query.FieldQuery,
			}
			obj, err := repository.Get(context.TODO(), types.PlatformType, idCriteria)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(obj.(*types.Platform).Version).To(Equal(version))
		})
	})

	Context("when ping is received", func() {
		var pongCh chan struct{}

		JustBeforeEach(func() {
			pongCh = make(chan struct{})
			Expect(wsconn.SetReadDeadline(time.Time{})).ShouldNot(HaveOccurred())
			wsconn.SetPongHandler(func(data string) error {
				Expect(data).To(Equal("pingping"))
				close(pongCh)
				return nil
			})
			go func() {
				_, _, err := wsconn.ReadMessage()
				Expect(err).Should(HaveOccurred())
			}()
		})

		It("should respond with pong", func(done Done) {
			err := wsconn.WriteMessage(websocket.PingMessage, []byte("pingping"))
			Expect(err).ShouldNot(HaveOccurred())
			Eventually(pongCh).Should(BeClosed())
			close(done)
		})
	})

	Context("when ping is not sent on time", func() {
		It("should close the connection", func(done Done) {
			wsconn.SetCloseHandler(func(code int, msg string) error {
				close(done)
				return nil
			})
			go func() {
				wsconn.ReadMessage()
			}()
		}, pingTimeout.Seconds()+1)
	})

	Context("when no notifications are present", func() {
		It("should not receive last known revision response header", func() {
			Expect(resp.Header.Get(notifications.LastKnownRevisionHeader)).To(Equal(""))
		})

		It("should receive max ping timeout response header", func() {
			Expect(resp.Header.Get(notifications.MaxPingPeriodHeader)).ToNot(BeEmpty())
		})

		Context("and revision known to proxy is 0", func() {
			It("should receive 410 Gone", func() {
				queryParams[notifications.LastKnownRevisionQueryParam] = "0"
				_, resp, _ := ctx.ConnectWebSocket(platform, queryParams, nil)
				Expect(resp.StatusCode).To(Equal(http.StatusGone))
			})
		})
	})

	Context("when notifications are created prior to connection", func() {
		var notification *types.Notification
		BeforeEach(func() {
			notification = createNotification(repository, platform.ID)
		})

		It("should receive last known revision response header", func() {
			lastKnownRevision, err := strconv.Atoi(resp.Header.Get(notifications.LastKnownRevisionHeader))
			Expect(err).ShouldNot(HaveOccurred())
			Expect(lastKnownRevision).To(BeNumerically(">", types.InvalidRevision))
		})

		Context("and proxy connects without last_notification_revision query parameter", func() {
			It("should send only new notifications without those already in the db", func() {
				newNotification := createNotification(repository, platform.ID)
				expectNotification(wsconn, newNotification.ID, newNotification.PlatformID)
			})
		})

		Context("and revision known to proxy is 0", func() {
			It("should receive 410 Gone", func() {
				queryParams[notifications.LastKnownRevisionQueryParam] = "0"
				_, resp, _ := ctx.ConnectWebSocket(platform, queryParams, nil)
				Expect(resp.StatusCode).To(Equal(http.StatusGone))
			})
		})

		Context("and proxy knows some notification revision", func() {
			var notification2 *types.Notification
			BeforeEach(func() {
				notification2 = createNotification(repository, platform.ID)
				queryParams[notifications.LastKnownRevisionQueryParam] = strconv.FormatInt(notification.Revision, 10)
			})

			It("should receive only these after the revision that it knowns", func() {
				expectNotification(wsconn, notification2.ID, notification2.PlatformID)
			})
		})

		Context("and revision known to proxy is not known to sm anymore", func() {
			It("should receive 410 Gone", func() {
				queryParams[notifications.LastKnownRevisionQueryParam] = strconv.FormatInt(notification.Revision-1, 10)
				_, resp, err := ctx.ConnectWebSocket(platform, queryParams, nil)
				Expect(resp.StatusCode).To(Equal(http.StatusGone))
				Expect(err).Should(HaveOccurred())
			})
		})

		Context("and proxy known revision is greater than sm known revision", func() {
			It("should receive 410 Gone", func() {
				queryParams[notifications.LastKnownRevisionQueryParam] = strconv.FormatInt(notification.Revision+1, 10)
				_, resp, err := ctx.ConnectWebSocket(platform, queryParams, nil)
				Expect(resp.StatusCode).To(Equal(http.StatusGone))
				Expect(err).Should(HaveOccurred())
			})
		})

		Context("when multiple connections are opened", func() {
			It("all other should not receive prior notifications, but only newly created", func() {
				wsconns := make([]*websocket.Conn, 0)
				createdNotifications := make([]*types.Notification, 0)
				for i := 0; i < 5; i++ {
					pl, conn, _, err := wsconnectWithPlatform(nil)

					Expect(err).ShouldNot(HaveOccurred())
					wsconns = append(wsconns, conn)

					n := createNotification(repository, pl.ID)
					createdNotifications = append(createdNotifications, n)
				}

				for i, conn := range wsconns {
					expectNotification(conn, createdNotifications[i].ID, createdNotifications[i].PlatformID)
				}
			})
		})
	})

	Context("platform health", func() {
		var pongCh chan struct{}
		var conn *websocket.Conn
		var newPlatform *types.Platform
		var idCriteria query.Criterion
		var headers map[string]string

		assertPlatformIsActive := func() {
			obj, err := repository.Get(context.TODO(), types.PlatformType, idCriteria)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(obj.(*types.Platform).Active).To(BeTrue())
		}

		BeforeEach(func() {
			pongCh = make(chan struct{})
			newPlatform = common.RegisterPlatformInSM(common.GenerateRandomPlatform(), ctx.SMWithOAuth, map[string]string{})
			idCriteria = query.Criterion{
				LeftOp:   "id",
				Operator: query.EqualsOperator,
				RightOp:  []string{newPlatform.ID},
				Type:     query.FieldQuery,
			}
			Expect(newPlatform.Active).To(BeFalse())

			var err error
			conn, _, err = ctx.ConnectWebSocket(newPlatform, queryParams, headers)
			Expect(err).ShouldNot(HaveOccurred())

			conn.SetPongHandler(func(data string) error {
				Expect(data).To(Equal("pingping"))
				close(pongCh)
				return nil
			})

			go func() {
				_, _, err := conn.ReadMessage()
				Expect(err).Should(HaveOccurred())
			}()
		})

		Context("when ping is received", func() {

			It("should switch platform's active status to true", func() {
				Expect(conn.WriteControl(websocket.PingMessage, []byte("pingping"), time.Now().Add(pingTimeout))).ShouldNot(HaveOccurred())
				Eventually(pongCh).Should(BeClosed()) // wait for a pong message
				assertPlatformIsActive()
			})

		})

		Context("when ping is not received", func() {
			It("should switch platform's active status to false", func() {
				Expect(conn.WriteControl(websocket.PingMessage, []byte("pingping"), time.Now().Add(pingTimeout))).ShouldNot(HaveOccurred())
				Eventually(pongCh).Should(BeClosed()) // wait for a pong message
				assertPlatformIsActive()

				By(fmt.Sprintf("Ping received and active status is true, then when %v ping timeout passes, active status should be set to false", pingTimeout))

				ctx, _ := context.WithTimeout(context.TODO(), pingTimeout+time.Second)
				ticker := time.NewTicker(pingTimeout / 3)
				for {
					select {
					case <-ticker.C:
						obj, err := repository.Get(context.TODO(), types.PlatformType, idCriteria)
						Expect(err).ShouldNot(HaveOccurred())
						p := obj.(*types.Platform)
						if p.Active == false && !p.LastActive.IsZero() {
							return
						}
					case <-ctx.Done():
						Fail("Timeout: platform active status not set to false")
					}
				}
			})
		})
	})

	Context("when same platform is connected twice", func() {
		It("should send same notifications to both", func() {
			conn, _, err := ctx.ConnectWebSocket(platform, queryParams, nil)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(conn).ShouldNot(BeNil())

			notification := createNotification(repository, platform.ID)
			expectNotification(conn, notification.ID, platform.ID)
			expectNotification(wsconn, notification.ID, platform.ID)
		})
	})

	Context("when revision known to proxy is invalid number", func() {
		It("should return status 400", func() {
			queryParams[notifications.LastKnownRevisionQueryParam] = "not_a_number"
			_, resp, err := ctx.ConnectWebSocket(platform, queryParams, nil)
			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
			Expect(err).Should(HaveOccurred())
		})
	})

	Context("when notification are created after ws conn is created", func() {
		It("should receive new notifications", func() {
			notification := createNotification(repository, platform.ID)
			expectNotification(wsconn, notification.ID, notification.PlatformID)
		})
	})

	Context("when one notification with empty platform and one notification with platform are created", func() {
		var notificationEmptyPlatform, notification *types.Notification
		BeforeEach(func() {
			initialNotification := createNotification(repository, "")
			queryParams[notifications.LastKnownRevisionQueryParam] = strconv.FormatInt(initialNotification.Revision, 10)

			notification = createNotification(repository, platform.ID)
			notificationEmptyPlatform = createNotification(repository, "")
		})

		It("one connection should receive both, but other only the one with empty platform", func() {
			expectNotification(wsconn, notification.ID, notification.PlatformID)
			expectNotification(wsconn, notificationEmptyPlatform.ID, "")

			By("creating new connection")
			_, newWsConn, _, err := wsconnectWithPlatform(queryParams)
			Expect(err).ShouldNot(HaveOccurred())
			expectNotification(newWsConn, notificationEmptyPlatform.ID, "")
		})
	})
})

func createNotification(repository storage.Repository, platformID string) *types.Notification {
	notification := common.GenerateRandomNotification()
	notification.PlatformID = platformID
	result, err := repository.Create(context.Background(), notification)
	Expect(err).ShouldNot(HaveOccurred())

	byID := query.ByField(query.EqualsOperator, "id", result.GetID())
	createdNotification, err := repository.Get(context.Background(), types.NotificationType, byID)
	Expect(err).ShouldNot(HaveOccurred())
	return createdNotification.(*types.Notification)
}

func expectNotification(wsconn *websocket.Conn, notificationID, platformID string) {
	notification := readNotification(wsconn)
	Expect(notification["id"]).To(Equal(notificationID))
	if platformID == "" {
		Expect(notification["platform_id"]).To(BeNil())
	} else {
		Expect(notification["platform_id"]).To(Equal(platformID))
	}
}

func readNotification(wsconn *websocket.Conn) map[string]interface{} {
	var r map[string]interface{}
	err := wsconn.ReadJSON(&r)
	Expect(err).ShouldNot(HaveOccurred())
	return r
}
