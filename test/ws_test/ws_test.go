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

package ws_test

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/url"
	"strconv"
	"testing"

	"github.com/Peripli/service-manager/pkg/util"

	"github.com/Peripli/service-manager/pkg/types"

	"github.com/Peripli/service-manager/pkg/web"

	"github.com/Peripli/service-manager/storage"

	"github.com/gorilla/websocket"

	"github.com/Peripli/service-manager/pkg/env"
	"github.com/Peripli/service-manager/pkg/sm"
	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestWsConn(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Websocket test suite")
}

var _ = Describe("WS", func() {
	var ctx *common.TestContext
	var wsconn *websocket.Conn
	queryParams := map[string]string{}
	var resp *http.Response
	var repository storage.Repository
	var platform *types.Platform

	BeforeEach(func() {
		queryParams = map[string]string{}

		ctx = common.NewTestContextBuilder().WithSMExtensions(func(ctx context.Context, smb *sm.ServiceManagerBuilder, e env.Environment) error {
			repository = smb.Storage
			return nil
		}).Build()
		Expect(repository).ToNot(BeNil())

		platform = common.RegisterPlatformInSM(common.GenerateRandomPlatform(), ctx.SMWithOAuth)
	})

	JustBeforeEach(func() {
		var err error
		wsconn, resp, err = wsconnect(ctx, platform, web.NotificationsURL, queryParams)
		Expect(err).ShouldNot(HaveOccurred())
	})

	AfterEach(func() {
		if repository != nil {
			_, err := repository.Delete(context.Background(), types.NotificationType)
			if err != nil {
				// TODO: Storage returns *util.HTTPError???
				httpErr := err.(*util.HTTPError)
				Expect(httpErr.StatusCode).Should(Equal(http.StatusNotFound))
			}
		}
		ctx.Cleanup()
	})

	JustAfterEach(func() {
		if wsconn != nil {
			wsconn.Close()
		}
	})

	Describe("establish websocket connection", func() {
		Context("when no notifications are present", func() {
			It("should receive last known revision response header 0", func() {
				Expect(resp.Header.Get("last_known_revision")).To(Equal("0"))
			})
		})

		Context("when notifications are created prior to connection", func() {
			var notification *types.Notification
			var notificationRevision int
			BeforeEach(func() {
				notification = common.GenerateRandomNotification()
				notification.PlatformID = platform.ID
				id, err := repository.Create(context.Background(), notification)
				Expect(err).ShouldNot(HaveOccurred())

				createdNotification, err := repository.Get(context.Background(), types.NotificationType, id)
				Expect(err).ShouldNot(HaveOccurred())
				notificationRevision = int((createdNotification.(*types.Notification)).Revision)
			})

			It("should receive last known revision response header greater than 0", func() {
				lastKnownRevision, err := strconv.Atoi(resp.Header.Get("last_known_revision"))
				Expect(err).ShouldNot(HaveOccurred())
				Expect(lastKnownRevision).To(BeNumerically(">", 1))
			})

			It("should receive them", func() {
				var r map[string]interface{}
				err := wsconn.ReadJSON(&r)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(r["id"]).To(Equal(notification.ID))
				Expect(r["platform_id"]).To(Equal(notification.PlatformID))
			})

			Context("and proxy knowns some notification revision", func() {
				var notification2 *types.Notification
				BeforeEach(func() {
					notification2 = common.GenerateRandomNotification()
					notification2.PlatformID = platform.ID
					_, err := repository.Create(context.Background(), notification2)
					Expect(err).ShouldNot(HaveOccurred())
					queryParams["last_known_revision"] = strconv.Itoa(notificationRevision)
				})

				It("should receive only these after the revision that it knowns", func() {
					var r map[string]interface{}
					err := wsconn.ReadJSON(&r)
					Expect(err).ShouldNot(HaveOccurred())
					Expect(r["id"]).To(Equal(notification2.ID))
					Expect(r["platform_id"]).To(Equal(notification2.PlatformID))
				})
			})

			Context("and revision known to proxy is not known to sm anymore", func() {
				It("should receive 410 Gone", func() {
					queryParams["last_known_revision"] = strconv.Itoa(notificationRevision - 1)
					_, resp, err := wsconnect(ctx, platform, web.NotificationsURL, queryParams)
					Expect(resp.StatusCode).To(Equal(http.StatusGone))
					Expect(err).Should(HaveOccurred())
				})
			})

			Context("when multiple connections are opened", func() {
				It("all other should not receive prior notifications, but only newly created", func() {
					wsconns := make([]*websocket.Conn, 0)
					pls := make([]*types.Platform, 0)
					for i := 0; i < 5; i++ {
						pl := common.RegisterPlatformInSM(common.GenerateRandomPlatform(), ctx.SMWithOAuth)
						pls = append(pls, pl)
						conn, _, err := wsconnect(ctx, pl, web.NotificationsURL, nil)
						Expect(err).ShouldNot(HaveOccurred())
						wsconns = append(wsconns, conn)

						notification := common.GenerateRandomNotification()
						notification.PlatformID = pl.ID
						_, err = repository.Create(context.Background(), notification)
						Expect(err).ShouldNot(HaveOccurred())
					}

					for i, conn := range wsconns {
						var r map[string]interface{}
						err := conn.ReadJSON(&r)
						Expect(err).ShouldNot(HaveOccurred())
						Expect(r["platform_id"]).To(Equal(pls[i].ID))
					}
				})
			})
		})

		Context("when revision known to proxy is invalid number", func() {
			It("should fail", func() {
				queryParams["last_known_revision"] = "not_a_number"
				_, resp, err := wsconnect(ctx, platform, web.NotificationsURL, queryParams)
				Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
				Expect(err).Should(HaveOccurred())
			})
		})

		Context("when notification are created after ws conn is created", func() {
			It("should receive new notifications", func() {
				notification := common.GenerateRandomNotification()
				notification.PlatformID = platform.ID
				_, err := repository.Create(context.Background(), notification)
				Expect(err).ShouldNot(HaveOccurred())

				var r map[string]interface{}
				err = wsconn.ReadJSON(&r)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(r["id"]).To(Equal(notification.ID))
				Expect(r["platform_id"]).To(Equal(notification.PlatformID))
			})
		})
	})
})

func wsconnect(ctx *common.TestContext, platform *types.Platform, path string, queryParams map[string]string) (*websocket.Conn, *http.Response, error) {
	smURL := ctx.Servers[common.SMServer].URL()
	smEndpoint, _ := url.Parse(smURL)
	smEndpoint.Scheme = "ws"
	smEndpoint.Path = path
	q := smEndpoint.Query()
	for k, v := range queryParams {
		q.Add(k, v)
	}
	smEndpoint.RawQuery = q.Encode()

	headers := http.Header{}
	encodedPlatform := base64.StdEncoding.EncodeToString([]byte(platform.Credentials.Basic.Username + ":" + platform.Credentials.Basic.Password))
	headers.Add("Authorization", "Basic "+encodedPlatform)

	wsEndpoint := smEndpoint.String()
	return websocket.DefaultDialer.Dial(wsEndpoint, headers)
}
