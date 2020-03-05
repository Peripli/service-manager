/*
 * Copyright 2018 The Service Manager Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package osb_test

import (
	"context"
	"fmt"
	"github.com/Peripli/service-manager/test"
	"github.com/gavv/httpexpect"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"
)

const simpleCatalog = `
{
  "services": [{
		"name": "no-tags-no-metadata",
		"id": "acb56d7c-XXXX-XXXX-XXXX-feb140a59a67",
		"description": "A fake service.",
		"dashboard_client": {
			"id": "id",
			"secret": "secret",
			"redirect_uri": "redirect_uri"		
		},    
		"plans": [{
			"random_extension": "random_extension",
			"name": "fake-plan-1",
			"id": "d3031751-XXXX-XXXX-XXXX-a42377d33202",
			"description": "Shared fake Server, 5tb persistent disk, 40 max concurrent connections.",
			"free": false
		}]
	}]
}
`

var _ = Describe("Catalog", func() {
	Context("when call to working service broker", func() {
		It("should succeed", func() {
			ctx.SMWithBasic.GET(smBrokerURL+"/v2/catalog").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				Expect().Status(http.StatusOK)
		})

		Context("when call to simple broker catalog broker", func() {
			BeforeEach(func() {
				credentials := brokerPlatformCredentialsIDMap[simpleBrokerCatalogID]
				ctx.SMWithBasic.SetBasicCredentials(ctx, credentials.username, credentials.password)
			})

			It("should return valid catalog if it's missing some properties", func() {
				req := ctx.SMWithBasic.GET(smUrlToSimpleBrokerCatalogBroker+"/v2/catalog").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).Expect()
				req.Status(http.StatusOK)

				service := req.JSON().Object().Value("services").Array().First().Object()
				service.Keys().NotContains("tags", "metadata", "requires")

				plan := service.Value("plans").Array().First().Object()
				plan.Keys().NotContains("metadata", "schemas")
			})

			It("should return valid catalog with all catalog extensions if catalog extensions are present", func() {
				resp := ctx.SMWithBasic.GET(smUrlToSimpleBrokerCatalogBroker+"/v2/catalog").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
					Expect().
					Status(http.StatusOK).JSON()

				resp.Path("$.services[*].dashboard_client[*]").Array().Contains("id", "secret", "redirect_uri")
				resp.Path("$.services[*].plans[*].random_extension").Array().Contains("random_extension")
			})
		})

		It("should not reach service broker", func() {
			ctx.SMWithBasic.GET(smBrokerURL+"/v2/catalog").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				Expect().Status(http.StatusOK)

			Expect(len(brokerServer.CatalogEndpointRequests)).To(Equal(0))
		})

		Context("when call to empty catalog broker", func() {
			It("should succeed and return empty services", func() {
				credentials := brokerPlatformCredentialsIDMap[emptyCatalogBrokerID]
				ctx.SMWithBasic.SetBasicCredentials(ctx, credentials.username, credentials.password)

				ctx.SMWithBasic.GET(smUrlToEmptyCatalogBroker+"/v2/catalog").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
					Expect().Status(http.StatusOK).JSON().Object().Value("services").Array().Empty()
				Expect(len(brokerServerWithEmptyCatalog.CatalogEndpointRequests)).To(Equal(0))
			})
		})
	})

	Context("when call to failing service broker", func() {
		It("should succeed because broker is not actually invoked", func() {
			brokerServer.CatalogHandler = parameterizedHandler(http.StatusInternalServerError, `{}`)
			ctx.SMWithBasic.GET(smBrokerURL+"/v2/catalog").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				Expect().Status(http.StatusOK)

			Expect(len(brokerServer.CatalogEndpointRequests)).To(Equal(0))
		})
	})

	Context("when call to missing service broker", func() {
		It("should fail with 401", func() {
			ctx.SMWithBasic.GET("http://localhost:3456/v1/osb/123"+"/v2/catalog").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				Expect().Status(http.StatusUnauthorized)
		})
	})

	Context("when call to stopped service broker", func() {
		It("should succeed because broker is not actually invoked", func() {
			credentials := brokerPlatformCredentialsIDMap[stoppedBrokerID]
			ctx.SMWithBasic.SetBasicCredentials(ctx, credentials.username, credentials.password)

			ctx.SMWithBasic.GET(smUrlToStoppedBroker+"/v2/catalog").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				Expect().Status(http.StatusOK)

			Expect(len(stoppedBrokerServer.CatalogEndpointRequests)).To(Equal(0))
		})
	})

	Describe("filtering catalog for k8s platform based on visibilities", func() {
		var brokerID string
		var plan1, plan2, plan3 string
		var plan1ID, plan1CatalogID, plan2ID, plan2CatalogID, plan3ID, plan3CatalogID string
		var k8sPlatform *types.Platform
		var k8sAgent *common.SMExpect

		getSMPlanIDByCatalogID := func(planCatalogID string) string {
			plans, err := ctx.SMRepository.List(context.Background(), types.ServicePlanType, query.ByField(query.EqualsOperator, "catalog_id", planCatalogID))
			Expect(err).ShouldNot(HaveOccurred())
			Expect(plans.Len()).To(BeNumerically("==", 1))
			return plans.ItemAt(0).GetID()
		}

		assertBrokerPlansVisibleForPlatform := func(brokerID string, agent *common.SMExpect, plans ...interface{}) {
			result := agent.GET(fmt.Sprintf("%s/%s/v2/catalog", web.OSBURL, brokerID)).
				Expect().Status(http.StatusOK).JSON().Path("$.services[*].plans[*].id").Array()

			result.Length().Equal(len(plans))
			if len(plans) > 0 {
				result.ContainsOnly(plans...)
			}
		}

		BeforeEach(func() {
			catalog := common.NewEmptySBCatalog()
			plan1 = common.GenerateTestPlan()
			plan2 = common.GenerateTestPlan()
			plan3 = common.GenerateTestPlan()
			service1 := common.GenerateTestServiceWithPlans(plan1, plan2)
			catalog.AddService(service1)
			service2 := common.GenerateTestServiceWithPlans(plan3)
			catalog.AddService(service2)
			brokerID, _, _ = ctx.RegisterBrokerWithCatalog(catalog)

			plan1CatalogID = gjson.Get(plan1, "id").String()
			plan2CatalogID = gjson.Get(plan2, "id").String()
			plan3CatalogID = gjson.Get(plan3, "id").String()
			plan1ID = getSMPlanIDByCatalogID(plan1CatalogID)
			plan2ID = getSMPlanIDByCatalogID(plan2CatalogID)
			plan3ID = getSMPlanIDByCatalogID(plan3CatalogID)

			username, password := test.RegisterBrokerPlatformCredentials(SMWithBasicPlatform, brokerID)
			ctx.SMWithBasic.SetBasicCredentials(ctx, username, password)

			k8sPlatformJSON := common.MakePlatform("k8s-platform", "k8s-platform", "kubernetes", "test-platform-k8s")
			k8sPlatform = common.RegisterPlatformInSM(k8sPlatformJSON, ctx.SMWithOAuth, map[string]string{})

			SMWithBasicK8S := &common.SMExpect{Expect: ctx.SM.Builder(func(req *httpexpect.Request) {
				username, password := k8sPlatform.Credentials.Basic.Username, k8sPlatform.Credentials.Basic.Password
				req.WithBasicAuth(username, password).WithClient(ctx.HttpClient)
			})}

			username, password = test.RegisterBrokerPlatformCredentials(SMWithBasicK8S, brokerID)
			k8sAgent = &common.SMExpect{Expect: ctx.SM.Builder(func(req *httpexpect.Request) {
				req.WithBasicAuth(username, password)
			})}

		})

		AfterEach(func() {
			ctx.CleanupBroker(brokerID)
			ctx.SMWithOAuth.DELETE(web.PlatformsURL + "/" + k8sPlatform.ID).
				Expect().Status(http.StatusOK)
		})

		Context("for platform with no visibilities", func() {
			It("should return empty services catalog", func() {
				assertBrokerPlansVisibleForPlatform(brokerID, k8sAgent)
				assertBrokerPlansVisibleForPlatform(brokerID, ctx.SMWithBasic)
			})
		})

		Context("for platform with visibilities for 2 plans from 2 services", func() {
			It("should return 2 plans", func() {
				assertBrokerPlansVisibleForPlatform(brokerID, k8sAgent)
				assertBrokerPlansVisibleForPlatform(brokerID, ctx.SMWithBasic)

				ctx.SMWithOAuth.POST(web.VisibilitiesURL).WithJSON(common.Object{
					"service_plan_id": plan1ID,
					"platform_id":     k8sPlatform.ID,
				}).Expect().Status(http.StatusCreated)
				ctx.SMWithOAuth.POST(web.VisibilitiesURL).WithJSON(common.Object{
					"service_plan_id": plan3ID,
					"platform_id":     k8sPlatform.ID,
				}).Expect().Status(http.StatusCreated)

				assertBrokerPlansVisibleForPlatform(brokerID, k8sAgent, plan3CatalogID, plan1CatalogID)
				assertBrokerPlansVisibleForPlatform(brokerID, ctx.SMWithBasic)

				ctx.SMWithOAuth.POST(web.VisibilitiesURL).WithJSON(common.Object{
					"service_plan_id": plan1ID,
					"platform_id":     ctx.TestPlatform.ID,
				}).Expect().Status(http.StatusCreated)
				ctx.SMWithOAuth.POST(web.VisibilitiesURL).WithJSON(common.Object{
					"service_plan_id": plan3ID,
					"platform_id":     ctx.TestPlatform.ID,
				}).Expect().Status(http.StatusCreated)

				assertBrokerPlansVisibleForPlatform(brokerID, k8sAgent, plan3CatalogID, plan1CatalogID)
				assertBrokerPlansVisibleForPlatform(brokerID, ctx.SMWithBasic, plan3CatalogID, plan1CatalogID)
			})
		})

		Context("for platform with non-public visibility for one plan", func() {
			It("should return 1 plan", func() {
				assertBrokerPlansVisibleForPlatform(brokerID, k8sAgent)

				ctx.SMWithOAuth.POST(web.VisibilitiesURL).WithJSON(common.Object{
					"service_plan_id": plan2ID,
					"platform_id":     k8sPlatform.ID,
				}).Expect().Status(http.StatusCreated)

				assertBrokerPlansVisibleForPlatform(brokerID, k8sAgent, plan2CatalogID)
			})
		})

		Context("for platform with public visibility for one plan", func() {
			It("should return 1 plan", func() {
				assertBrokerPlansVisibleForPlatform(brokerID, k8sAgent)

				k8sAgent.GET(fmt.Sprintf("%s/%s/v2/catalog", web.OSBURL, brokerID)).
					Expect().Status(http.StatusOK).JSON().Object().Value("services").Array().Empty()

				ctx.SMWithOAuth.POST(web.VisibilitiesURL).WithJSON(common.Object{
					"service_plan_id": plan1ID,
				}).Expect().Status(http.StatusCreated)
				assertBrokerPlansVisibleForPlatform(brokerID, k8sAgent, plan1CatalogID)
			})
		})
	})

	Describe("Prefixed broker path", func() {
		Context("when call to working broker", func() {

			const brokerPathPrefix = "/broker_prefix"
			var (
				server           common.FakeServer
				oldSMWithBasic   *common.SMExpect
				osbURL           string
				prefixedBrokerID string
			)

			assertCredentialsNotChanged := func(oldSMExpect, newSMExpect *common.SMExpect) {
				By("new credentials should be invalid")
				newSMExpect.GET(osbURL + "/v2/catalog").
					Expect().Status(http.StatusUnauthorized)

				By("old ones should be valid")
				oldSMExpect.GET(osbURL + "/v2/catalog").
					Expect().Status(http.StatusOK).JSON().Object().ContainsKey("services")
			}

			BeforeEach(func() {
				brokerHandler := &prefixedBrokerHandler{brokerPathPrefix}
				server = &prefixedBrokerServer{Server: httptest.NewServer(brokerHandler)}
				brokerURL := server.URL() + brokerPathPrefix

				brokerJSON := common.Object{
					"name":        "prefixed_broker",
					"broker_url":  brokerURL,
					"description": "",
					"credentials": common.Object{
						"basic": common.Object{
							"username": "buser",
							"password": "bpass",
						},
					},
				}

				prefixedBrokerID = common.RegisterBrokerInSM(brokerJSON, ctx.SMWithOAuth, map[string]string{})["id"].(string)
				ctx.Servers[common.BrokerServerPrefix+prefixedBrokerID] = server
				osbURL = "/v1/osb/" + prefixedBrokerID

				username, password := test.RegisterBrokerPlatformCredentials(SMWithBasicPlatform, prefixedBrokerID)
				ctx.SMWithBasic.SetBasicCredentials(ctx, username, password)

				oldSMWithBasic = &common.SMExpect{Expect: ctx.SMWithBasic.Expect}

				ctx.SMWithBasic.GET(osbURL + "/v2/catalog").
					Expect().Status(http.StatusOK).JSON().Object().ContainsKey("services")
			})

			AfterEach(func() {
				ctx.CleanupBroker(prefixedBrokerID)
				ctx.CleanupPlatforms()
			})

			It("should get catalog", func() {
				By("broker platform credentials")
				ctx.SMWithBasic.GET(osbURL + "/v2/catalog").
					Expect().Status(http.StatusOK).JSON().Object().ContainsKey("services")

				By("platform credentials")
				SMWithBasicPlatform.GET(osbURL + "/v2/catalog").
					Expect().Status(http.StatusOK).JSON().Object().ContainsKey("services")
			})

			Context("when broker platform credentials change in context of a notification processing", func() {
				Context("and notification is found in SM DB", func() {
					JustBeforeEach(func() {
						ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + prefixedBrokerID).
							WithJSON(common.Object{}).
							Expect().Status(http.StatusOK)
					})

					Context("and notification properties match the ones provided in the set credentials request", func() {
						It("should still get catalog", func() {
							notifications, err := ctx.SMRepository.List(context.TODO(), types.NotificationType,
								query.OrderResultBy("created_at", query.DescOrder))
							Expect(err).ToNot(HaveOccurred())

							newUsername, newPassword := test.RegisterBrokerPlatformCredentialsWithNotificationID(SMWithBasicPlatform, prefixedBrokerID, notifications.ItemAt(0).GetID())
							ctx.SMWithBasic.SetBasicCredentials(ctx, newUsername, newPassword)

							By("new credentials not yet used - old ones should still be valid")
							oldSMWithBasic.GET(osbURL + "/v2/catalog").
								Expect().Status(http.StatusOK).JSON().Object().ContainsKey("services")

							By("new credentials used - old ones should be invalidated")
							ctx.SMWithBasic.GET(osbURL + "/v2/catalog").
								Expect().Status(http.StatusOK).JSON().Object().ContainsKey("services")

							oldSMWithBasic.GET(osbURL + "/v2/catalog").
								Expect().Status(http.StatusUnauthorized)

						})
					})

					Context("and notification properties does NOT match the ones provided in the set credentials request", func() {
						When("provided notification id is for a different platform", func() {
							var newPlatform *types.Platform

							BeforeEach(func() {
								newPlatform = ctx.RegisterPlatform()
							})

							It("should return 400", func() {
								notifications, err := ctx.SMRepository.List(context.TODO(), types.NotificationType,
									query.OrderResultBy("created_at", query.DescOrder),
									query.ByField(query.EqualsOperator, "platform_id", newPlatform.ID))
								Expect(err).ToNot(HaveOccurred())

								newUsername, newPassword := test.RegisterBrokerPlatformCredentialsWithNotificationIDExpect(SMWithBasicPlatform, prefixedBrokerID, notifications.ItemAt(0).GetID(), http.StatusBadRequest)
								ctx.SMWithBasic.SetBasicCredentials(ctx, newUsername, newPassword)

								assertCredentialsNotChanged(oldSMWithBasic, ctx.SMWithBasic)
							})
						})

						When("provided notification id is for a different broker", func() {
							It("should return 400", func() {
								notifications, err := ctx.SMRepository.List(context.TODO(), types.NotificationType,
									query.OrderResultBy("created_at", query.DescOrder))
								Expect(err).ToNot(HaveOccurred())

								newUsername, newPassword := test.RegisterBrokerPlatformCredentialsWithNotificationIDExpect(SMWithBasicPlatform, "non-existing-broker-id", notifications.ItemAt(0).GetID(), http.StatusBadRequest)
								ctx.SMWithBasic.SetBasicCredentials(ctx, newUsername, newPassword)

								assertCredentialsNotChanged(oldSMWithBasic, ctx.SMWithBasic)
							})
						})
					})

				})

				Context("and notification is not found in SM DB", func() {
					It("should return 400", func() {
						newUsername, newPassword := test.RegisterBrokerPlatformCredentialsWithNotificationIDExpect(SMWithBasicPlatform, prefixedBrokerID, "non-existing-id", http.StatusBadRequest)
						ctx.SMWithBasic.SetBasicCredentials(ctx, newUsername, newPassword)

						assertCredentialsNotChanged(oldSMWithBasic, ctx.SMWithBasic)
					})
				})
			})

			Context("when broker platform credentials change out of notification processing context when already exist", func() {
				It("should return 409", func() {
					newUsername, newPassword := test.RegisterBrokerPlatformCredentialsExpect(SMWithBasicPlatform, prefixedBrokerID, http.StatusConflict)
					ctx.SMWithBasic.SetBasicCredentials(ctx, newUsername, newPassword)

					assertCredentialsNotChanged(oldSMWithBasic, ctx.SMWithBasic)
				})
			})
		})
	})
})

type prefixedBrokerHandler struct {
	brokerPathPrefix string
}

func (pbh *prefixedBrokerHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if strings.HasPrefix(req.URL.Path, pbh.brokerPathPrefix) {
		common.SetResponse(w, http.StatusOK, common.Object{"services": common.Array{}})
	} else {
		common.SetResponse(w, http.StatusNotFound, common.Object{})
	}
}

type prefixedBrokerServer struct {
	*httptest.Server
}

func (pbs *prefixedBrokerServer) URL() string {
	return pbs.Server.URL
}
