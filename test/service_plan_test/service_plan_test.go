/*
 *    Copyright 2018 The Service Manager Authors
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

package service_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/Peripli/service-manager/test/common"

	. "github.com/onsi/ginkgo"

	. "github.com/onsi/gomega"
)

func TestServicePlans(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Plans Tests Suite")
}

var _ = Describe("Service Manager Service Plans API", func() {
	var (
		ctx              *common.TestContext
		brokerServer     *common.BrokerServer
		brokerServerJSON common.Object
	)

	BeforeSuite(func() {
		brokerServer = common.NewBrokerServer()
		ctx = common.NewTestContext(nil)
	})

	AfterSuite(func() {
		ctx.Cleanup()
		if brokerServer != nil {
			brokerServer.Close()
		}
	})

	BeforeEach(func() {
		brokerServer.Reset()
		brokerServerJSON = common.Object{
			"name":        "brokerName",
			"broker_url":  brokerServer.URL,
			"description": "description",
			"credentials": common.Object{
				"basic": common.Object{
					"username": brokerServer.Username,
					"password": brokerServer.Password,
				},
			},
		}
		common.RemoveAllBrokers(ctx.SMWithOAuth)
	})

	Describe("GET", func() {
		Context("when the service plan does not exist", func() {
			It("returns 404", func() {
				ctx.SMWithOAuth.GET("/v1/service_plans/12345").
					Expect().
					Status(http.StatusNotFound).
					JSON().Object().
					Keys().Contains("error", "description")
			})
		})

		Context("when the service plan exists", func() {
			BeforeEach(func() {
				ctx.SMWithOAuth.POST("/v1/service_brokers").WithJSON(brokerServerJSON).
					Expect().
					Status(http.StatusCreated)

				brokerServer.ResetCallHistory()
			})

			It("returns the service plan with the given id", func() {
				id := ctx.SMWithOAuth.GET("/v1/service_plans").
					Expect().
					Status(http.StatusOK).
					JSON().Object().Value("service_plans").Array().First().Object().
					Value("id").String().Raw()

				Expect(id).ToNot(BeEmpty())

				ctx.SMWithOAuth.GET("/v1/service_plans/"+id).
					Expect().
					Status(http.StatusOK).
					JSON().Object().
					Keys().Contains("id", "description")
			})
		})
	})

	Describe("List", func() {
		Context("when no service plans exist", func() {
			It("returns an empty array", func() {
				ctx.SMWithOAuth.GET("/v1/service_plans").
					Expect().
					Status(http.StatusOK).
					JSON().Object().Value("service_plans").Array().
					Empty()
			})
		})

		Context("when a broker is registered", func() {
			BeforeEach(func() {
				ctx.SMWithOAuth.POST("/v1/service_brokers").WithJSON(brokerServerJSON).
					Expect().
					Status(http.StatusCreated)

				brokerServer.ResetCallHistory()
			})

			It("is accessible with basic authentication", func() {
				ctx.SMWithBasic.GET("/v1/service_plans").
					Expect().
					Status(http.StatusOK).
					JSON().Object().Value("service_plans").Array().Length().Equal(2)
			})

			Context("when catalog_name parameter is not provided", func() {
				It("returns the broker's service plans as part of the response", func() {
					ctx.SMWithOAuth.GET("/v1/service_plans").
						Expect().
						Status(http.StatusOK).
						JSON().Object().Value("service_plans").Array().Length().Equal(2)
				})
			})

			//Context("when catalog_name query parameter is provided", func() {
			//	It("returns the service plan with the specified catalog_name", func() {
			//		serviceCatalogName := common.DefaultCatalog()["services"].([]interface{})[0].(map[string]interface{})["plans"].([]interface{})[0].(map[string]interface{})["name"]
			//		Expect(serviceCatalogName).ToNot(BeEmpty())
			//
			//		ctx.SMWithOAuth.GET("/v1/service_plans").WithQuery("catalog_name", serviceCatalogName).
			//			Expect().
			//			Status(http.StatusOK).
			//			JSON().Object().Value("service_plans").Array().Length().Equal(1)
			//	})
			//})
		})
	})

	Describe("List", func() {

		//platformWithNilDescription := common.MakeRandomizedPlatformWithNoDescription()
		//platformWithUnknownKeys := common.Object{
		//	"unknownkey": "unknownvalue",
		//}

		//nonExistingPlatform := common.GenerateRandomPlatform()

		var r map[string][]common.Object

		type testCase struct {
			createResourcesBeforeOp   map[string][]interface{}
			expectedResourcesBeforeOp map[string][]interface{}
			OpFieldQueryTemplate      string
			OpFieldQueryArgs          common.Object

			expectedResourcesAfterOp   map[string][]interface{}
			unexpectedResourcesAfterOp map[string][]interface{}
			expectedStatusCode         int
		}

		testCases := []testCase{
			{
				createResourcesBeforeOp: map[string][]interface{}{
					"service_brokers": {r["service_brokers"][0], r["service_brokers"][1]},
				},
				expectedResourcesBeforeOp: map[string][]interface{}{
					"service_brokers":   {r["service_brokers"][0], r["service_brokers"][1]},
					"service_offerings": {r["service_offerings"][0], r["service_offerings"][1]},
					"service_plans":     {r["service_plans"][0], r["service_plans"][1], r["service_plans"][2], r["service_plans"][3]},
				},
				OpFieldQueryTemplate: "%s+=+%s",
				OpFieldQueryArgs:     r["service_plans"][0],
				expectedResourcesAfterOp: map[string][]interface{}{
					"service_plans": {r["service_plans"][0]},
				},
				unexpectedResourcesAfterOp: map[string][]interface{}{
					"service_plans": {r["service_plans"][1], r["service_plans"][2], r["service_plans"][3]},
				},
				expectedStatusCode: http.StatusOK,
			},
		}

		verifyListOp := func(t *testCase, query string) func() {
			return func() {
				expectedAfterOpIDs := common.ExtractResourceIDs(t.expectedResourcesAfterOp)
				unexpectedAfterOpIDs := common.ExtractResourceIDs(t.unexpectedResourcesAfterOp)

				BeforeEach(func() {
					for key := range t.createResourcesBeforeOp {
						beforeOpIDs := common.ExtractResourceIDs(t.createResourcesBeforeOp)
						q := fmt.Sprintf("id+in+[%s]", strings.Join(beforeOpIDs[key], ","))

						By(fmt.Sprintf("[SETUP]: Cleaning up [%s] with fieldquery [%s]", key, q))

						ctx.SMWithOAuth.DELETE("/v1/"+key).WithQuery("fieldQuery", q).
							Expect()

						for _, entity := range t.createResourcesBeforeOp[key] {
							By(fmt.Sprintf("[SETUP]: Creating entity in [/v1/%s] with body [%s]", key, entity))

							ctx.SMWithOAuth.POST("/v1/" + key).WithJSON(entity).
								Expect().Status(http.StatusCreated)
						}

					}

					for key, entities := range t.expectedResourcesBeforeOp {
						By(fmt.Sprintf("[SETUP]: Verifying expected [%s] before operation after present", key))

						beforeOpArray := ctx.SMWithOAuth.GET("/v1/" + key).
							Expect().
							Status(http.StatusOK).JSON().Object().Value(key).Array()

						for _, v := range beforeOpArray.Iter() {
							obj := v.Object().Raw()
							delete(obj, "created_at")
							delete(obj, "updated_at")
							if _, ok := t.createResourcesBeforeOp[key]; !ok {
								delete(obj, "id")
							}
						}
						beforeOpArray.Contains(entities...)
					}
				})

				//TODO next level parameterization of api type
				It(fmt.Sprintf("returns status %d and service_plans with ids %s and NOT with ids %s", t.expectedStatusCode, expectedAfterOpIDs, unexpectedAfterOpIDs), func() {

					By("[TEST]: ======= Expectations Summary =======")

					By(fmt.Sprintf("[TEST]: Listing service_plans with fieldquery [%s]", query))
					By(fmt.Sprintf("[TEST]: Currently present service_plans ids: [%s]", t.expectedResourcesBeforeOp))
					By(fmt.Sprintf("[TEST]: Expected service_plans ids after operations: [%s]", expectedAfterOpIDs))
					By(fmt.Sprintf("[TEST]: Unexpected service_plans ids after operation: [%s]", unexpectedAfterOpIDs))

					By("[TEST]: ====================================")

					req := ctx.SMWithOAuth.GET("/v1/service_plans")
					if query != "" {
						req = req.WithQuery("fieldQuery", query)
					}
					resp := req.
						Expect().
						Status(t.expectedStatusCode)

					if t.expectedStatusCode != http.StatusOK {
						By(fmt.Sprintf("[TEST]: Verifying error and description fields are returned after list operation"))

						resp.JSON().Object().Keys().Contains("error", "description")
					} else {
						for key, entities := range t.expectedResourcesAfterOp {
							By(fmt.Sprintf("[TEST]: Verifying expected r are returned after list operation"))

							array := resp.JSON().Object().Value(key).Array()
							for _, v := range array.Iter() {
								obj := v.Object().Raw()
								delete(obj, "created_at")
								delete(obj, "updated_at")
							}
							array.Contains(entities...)

						}

						for key, entities := range t.unexpectedResourcesAfterOp {
							By(fmt.Sprintf("[TEST]: Verifying expected r are returned after list operation"))

							array := resp.JSON().Object().Value(key).Array()
							for _, v := range array.Iter() {
								obj := v.Object().Raw()
								delete(obj, "created_at")
								delete(obj, "updated_at")
							}
							array.NotContains(entities...)

						}
					}
				})
			}
		}

		for _, t := range testCases {
			t := t
			if len(t.OpFieldQueryArgs) == 0 && t.OpFieldQueryTemplate != "" {
				panic("Invalid test input")
			}

			var queries []string
			for key, value := range t.OpFieldQueryArgs {
				queries = append(queries, fmt.Sprintf(t.OpFieldQueryTemplate, key, value))
			}
			query := strings.Join(queries, ",")

			Context(fmt.Sprintf("with multi field query=%s", query), verifyListOp(&t, query))

			for _, query := range queries {
				query := query
				Context(fmt.Sprintf("with field query=%s", query), verifyListOp(&t, query))
			}
		}
	})
})
