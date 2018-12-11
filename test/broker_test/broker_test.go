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
package broker_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/gavv/httpexpect"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/spf13/cast"
)

func TestBrokers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Broker API Tests Suite")
}

var _ = Describe("Service Manager Broker API", func() {

	var (
		ctx          *common.TestContext
		brokerServer *common.BrokerServer

		brokerServerJSON       common.Object
		expectedBrokerResponse common.Object
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
		brokerServer = common.NewBrokerServer()
		ctx = common.NewTestContext(nil)
		brokerServer.Reset()
		brokerName := "brokerName"
		brokerDescription := "description"

		brokerServerJSON = common.Object{
			"name":        brokerName,
			"broker_url":  brokerServer.URL,
			"description": brokerDescription,
			"credentials": common.Object{
				"basic": common.Object{
					"username": brokerServer.Username,
					"password": brokerServer.Password,
				},
			},
		}
		expectedBrokerResponse = common.Object{
			"name":        brokerName,
			"broker_url":  brokerServer.URL,
			"description": brokerDescription,
		}
		common.RemoveAllBrokers(ctx.SMWithOAuth)
	})

	Describe("GET", func() {
		var id string

		AfterEach(func() {
			assertInvocationCount(brokerServer.CatalogEndpointRequests, 0)
		})

		Context("when the broker does not exist", func() {
			It("returns 404", func() {
				ctx.SMWithOAuth.GET("/v1/service_brokers/12345").
					Expect().
					Status(http.StatusNotFound).
					JSON().Object().
					Keys().Contains("error", "description")
			})
		})

		Context("when the broker exists", func() {
			BeforeEach(func() {
				reply := ctx.SMWithOAuth.POST("/v1/service_brokers").WithJSON(brokerServerJSON).
					Expect().
					Status(http.StatusCreated).
					JSON().Object().
					ContainsMap(expectedBrokerResponse)

				id = reply.Value("id").String().Raw()

				assertInvocationCount(brokerServer.CatalogEndpointRequests, 1)
				brokerServer.ResetCallHistory()
			})

			It("returns the broker with given id", func() {
				ctx.SMWithOAuth.GET("/v1/service_brokers/"+id).
					Expect().
					Status(http.StatusOK).
					JSON().Object().
					ContainsMap(expectedBrokerResponse).
					Keys().NotContains("credentials", "services")
			})
		})
	})

	Describe("List", func() {
		AfterEach(func() {
			assertInvocationCount(brokerServer.CatalogEndpointRequests, 0)
		})

		Context("when no brokers exist", func() {
			It("returns empty array", func() {
				ctx.SMWithOAuth.GET("/v1/service_brokers").
					Expect().
					Status(http.StatusOK).
					JSON().Object().Value("brokers").Array().
					Empty()
			})
		})

		Context("when brokers exist", func() {
			BeforeEach(func() {
				ctx.SMWithOAuth.POST("/v1/service_brokers").WithJSON(brokerServerJSON).
					Expect().
					Status(http.StatusCreated).
					JSON().Object().
					ContainsMap(expectedBrokerResponse).
					Keys().
					NotContains("credentials", "services")

				assertInvocationCount(brokerServer.CatalogEndpointRequests, 1)
				brokerServer.ResetCallHistory()
			})

			It("returns all without catalog if no query parameter is provided", func() {
				ctx.SMWithOAuth.GET("/v1/service_brokers").
					Expect().
					Status(http.StatusOK).
					JSON().Object().Value("brokers").Array().First().Object().
					ContainsMap(expectedBrokerResponse).
					Keys().
					NotContains("credentials", "services")
			})

			It("returns all with catalog if query parameter is provided", func() {
				ctx.SMWithOAuth.GET("/v1/service_brokers").WithQuery("catalog", true).
					Expect().
					Status(http.StatusOK).
					JSON().Object().Value("brokers").Array().First().Object().
					ContainsMap(expectedBrokerResponse).
					ContainsKey("services").
					NotContainsKey("credentials")
			})

			It("is accessible with basic authentication", func() {
				ctx.SMWithBasic.GET("/v1/service_brokers").WithQuery("catalog", true).
					Expect().
					Status(http.StatusOK).
					JSON().Object().Value("brokers").Array().First().Object().
					ContainsMap(expectedBrokerResponse).
					ContainsKey("services").
					NotContainsKey("credentials")
			})
		})
	})

	Describe("POST", func() {
		Context("when content type is not JSON", func() {
			It("returns 415", func() {
				ctx.SMWithOAuth.POST("/v1/service_brokers").WithText("text").
					Expect().
					Status(http.StatusUnsupportedMediaType).
					JSON().Object().
					Keys().Contains("error", "description")

				assertInvocationCount(brokerServer.CatalogEndpointRequests, 0)
			})
		})

		Context("when request body is not a valid JSON", func() {
			It("returns 400", func() {
				ctx.SMWithOAuth.POST("/v1/service_brokers").
					WithText("invalid json").
					WithHeader("content-type", "application/json").
					Expect().
					Status(http.StatusBadRequest).
					JSON().Object().
					Keys().Contains("error", "description")

				assertInvocationCount(brokerServer.CatalogEndpointRequests, 0)
			})
		})

		Context("when a request body field is missing", func() {
			assertPOSTReturns400WhenFieldIsMissing := func(field string) {
				BeforeEach(func() {
					delete(brokerServerJSON, field)
					delete(expectedBrokerResponse, field)
				})

				It("returns 400", func() {
					ctx.SMWithOAuth.POST("/v1/service_brokers").WithJSON(brokerServerJSON).
						Expect().
						Status(http.StatusBadRequest).
						JSON().Object().
						Keys().Contains("error", "description")

					assertInvocationCount(brokerServer.CatalogEndpointRequests, 0)
				})
			}

			assertPOSTReturns201WhenFieldIsMissing := func(field string) {
				BeforeEach(func() {
					delete(brokerServerJSON, field)
					delete(expectedBrokerResponse, field)
				})

				It("returns 201", func() {
					ctx.SMWithOAuth.POST("/v1/service_brokers").WithJSON(brokerServerJSON).
						Expect().
						Status(http.StatusCreated).
						JSON().Object().
						ContainsMap(expectedBrokerResponse).
						Keys().NotContains("services", "credentials")

					assertInvocationCount(brokerServer.CatalogEndpointRequests, 1)
				})
			}

			Context("when name field is missing", func() {
				assertPOSTReturns400WhenFieldIsMissing("name")
			})

			Context("when broker_url field is missing", func() {
				assertPOSTReturns400WhenFieldIsMissing("broker_url")
			})

			Context("when credentials field is missing", func() {
				assertPOSTReturns400WhenFieldIsMissing("credentials")
			})

			Context("when description field is missing", func() {
				assertPOSTReturns201WhenFieldIsMissing("description")
			})
		})

		Context("when obtaining the broker catalog fails because the broker is not reachable", func() {
			BeforeEach(func() {
				brokerServerJSON["broker_url"] = "http://localhost:12345"
			})

			It("returns 400", func() {
				ctx.SMWithOAuth.POST("/v1/service_brokers").WithJSON(brokerServerJSON).
					Expect().
					Status(http.StatusBadRequest).JSON().Object().Keys().Contains("error", "description")
			})
		})

		Context("when the broker catalog is incomplete", func() {
			verifyPOSTWhenCatalogFieldIsMissing := func(responseVerifier func(r *httpexpect.Response), fieldPath string) {
				BeforeEach(func() {
					catalog, err := sjson.Delete(common.Catalog, fieldPath)
					Expect(err).ToNot(HaveOccurred())

					brokerServer.Catalog = common.JSONToMap(catalog)
				})

				It("returns correct response", func() {
					responseVerifier(ctx.SMWithOAuth.POST("/v1/service_brokers").WithJSON(brokerServerJSON).Expect())

					assertInvocationCount(brokerServer.CatalogEndpointRequests, 1)

				})
			}

			verifyPOSTWhenCatalogFieldHasValue := func(responseVerifier func(r *httpexpect.Response), fieldPath string, fieldValue interface{}) {
				BeforeEach(func() {
					catalog, err := sjson.Set(common.Catalog, fieldPath, fieldValue)
					Expect(err).ToNot(HaveOccurred())

					brokerServer.Catalog = common.JSONToMap(catalog)
				})

				It("returns correct response", func() {
					responseVerifier(ctx.SMWithOAuth.POST("/v1/service_brokers").WithJSON(brokerServerJSON).Expect())

					assertInvocationCount(brokerServer.CatalogEndpointRequests, 1)

				})
			}

			Context("when the broker catalog contains an incomplete service", func() {
				Context("that has an empty catalog id", func() {
					verifyPOSTWhenCatalogFieldIsMissing(func(r *httpexpect.Response) {
						r.Status(http.StatusBadRequest).JSON().Object().Keys().Contains("error", "description")
					}, "services.0.id")
				})

				Context("that has an empty catalog name", func() {
					verifyPOSTWhenCatalogFieldIsMissing(func(r *httpexpect.Response) {
						r.Status(http.StatusBadRequest).JSON().Object().Keys().Contains("error", "description")
					}, "services.0.name")
				})

				Context("that has an empty description", func() {
					verifyPOSTWhenCatalogFieldIsMissing(func(r *httpexpect.Response) {
						r.Status(http.StatusCreated).JSON().Object().Keys().NotContains("services", "credentials")
					}, "services.0.description")
				})

				Context("that has invalid tags", func() {
					verifyPOSTWhenCatalogFieldHasValue(func(r *httpexpect.Response) {
						r.Status(http.StatusBadRequest).JSON().Object().Keys().NotContains("services", "credentials")
					}, "services.0.tags", "{invalid")
				})

				Context("that has invalid requires", func() {
					verifyPOSTWhenCatalogFieldHasValue(func(r *httpexpect.Response) {
						r.Status(http.StatusBadRequest).JSON().Object().Keys().NotContains("services", "credentials")
					}, "services.0.requires", "{invalid")
				})

				Context("that has invalid metadata", func() {
					verifyPOSTWhenCatalogFieldHasValue(func(r *httpexpect.Response) {
						r.Status(http.StatusBadRequest).JSON().Object().Keys().NotContains("services", "credentials")
					}, "services.0.metadata", "{invalid")
				})
			})

			Context("when broker catalog contains an incomplete plan", func() {
				Context("that has an empty catalog id", func() {
					verifyPOSTWhenCatalogFieldIsMissing(func(r *httpexpect.Response) {
						r.Status(http.StatusBadRequest).JSON().Object().Keys().Contains("error", "description")
					}, "services.0.plans.0.id")
				})

				Context("that has an empty catalog name", func() {
					verifyPOSTWhenCatalogFieldIsMissing(func(r *httpexpect.Response) {
						r.Status(http.StatusBadRequest).JSON().Object().Keys().Contains("error", "description")
					}, "services.0.plans.0.name")
				})

				Context("that has an empty description", func() {
					verifyPOSTWhenCatalogFieldIsMissing(func(r *httpexpect.Response) {
						r.Status(http.StatusCreated).JSON().Object().Keys().NotContains("services", "credentials")
					}, "services.0.plans.0.description")
				})

				Context("that has invalid metadata", func() {
					verifyPOSTWhenCatalogFieldHasValue(func(r *httpexpect.Response) {
						r.Status(http.StatusBadRequest).JSON().Object().Keys().NotContains("services", "credentials")
					}, "services.0.plans.0.metadata", "{invalid")
				})

				Context("that has invalid schemas", func() {
					verifyPOSTWhenCatalogFieldHasValue(func(r *httpexpect.Response) {
						r.Status(http.StatusBadRequest).JSON().Object().Keys().NotContains("services", "credentials")
					}, "services.0.plans.0.schemas", "{invalid")
				})
			})
		})

		Context("when fetching catalog fails", func() {
			BeforeEach(func() {
				brokerServer.CatalogHandler = func(w http.ResponseWriter, req *http.Request) {
					common.SetResponse(w, http.StatusInternalServerError, common.Object{})
				}
			})

			It("returns 400", func() {
				ctx.SMWithOAuth.POST("/v1/service_brokers").
					WithJSON(brokerServerJSON).
					Expect().Status(http.StatusBadRequest).
					JSON().Object().
					Keys().Contains("error", "description")

				assertInvocationCount(brokerServer.CatalogEndpointRequests, 1)
			})
		})

		Context("when request is successful", func() {
			assertPOSTReturns201 := func() {
				It("returns 201", func() {
					ctx.SMWithOAuth.POST("/v1/service_brokers").WithJSON(brokerServerJSON).
						Expect().
						Status(http.StatusCreated).
						JSON().Object().
						ContainsMap(expectedBrokerResponse).
						Keys().NotContains("services", "credentials")

					assertInvocationCount(brokerServer.CatalogEndpointRequests, 1)
				})
			}

			Context("when broker URL does not end with trailing slash", func() {
				BeforeEach(func() {
					brokerServerJSON["broker_url"] = strings.TrimRight(cast.ToString(brokerServerJSON["broker_url"]), "/")
					expectedBrokerResponse["broker_url"] = strings.TrimRight(cast.ToString(expectedBrokerResponse["broker_url"]), "/")
				})

				assertPOSTReturns201()
			})

			Context("when broker URL ends with trailing slash", func() {
				BeforeEach(func() {
					brokerServerJSON["broker_url"] = cast.ToString(brokerServerJSON["broker_url"]) + "/"
					expectedBrokerResponse["broker_url"] = cast.ToString(expectedBrokerResponse["broker_url"]) + "/"
				})

				assertPOSTReturns201()
			})
		})

		Context("when broker with name already exists", func() {
			It("returns 409", func() {
				ctx.SMWithOAuth.POST("/v1/service_brokers").WithJSON(brokerServerJSON).
					Expect().
					Status(http.StatusCreated)

				ctx.SMWithOAuth.POST("/v1/service_brokers").WithJSON(brokerServerJSON).
					Expect().
					Status(http.StatusConflict).
					JSON().Object().
					Keys().Contains("error", "description")

				assertInvocationCount(brokerServer.CatalogEndpointRequests, 2)
			})
		})
	})

	Describe("PATCH", func() {
		var brokerID string

		BeforeEach(func() {
			reply := ctx.SMWithOAuth.POST("/v1/service_brokers").WithJSON(brokerServerJSON).
				Expect().
				Status(http.StatusCreated).
				JSON().Object().
				ContainsMap(expectedBrokerResponse)

			brokerID = reply.Value("id").String().Raw()

			assertInvocationCount(brokerServer.CatalogEndpointRequests, 1)
			brokerServer.ResetCallHistory()
		})

		Context("when content type is not JSON", func() {
			It("returns 415", func() {
				ctx.SMWithOAuth.PATCH("/v1/service_brokers/"+brokerID).
					WithText("text").
					Expect().Status(http.StatusUnsupportedMediaType).
					JSON().Object().
					Keys().Contains("error", "description")

				assertInvocationCount(brokerServer.CatalogEndpointRequests, 0)
			})
		})

		Context("when broker is missing", func() {
			It("returns 404", func() {
				ctx.SMWithOAuth.PATCH("/v1/service_brokers/no_such_id").
					WithJSON(brokerServerJSON).
					Expect().Status(http.StatusNotFound).
					JSON().Object().
					Keys().Contains("error", "description")
			})
		})

		Context("when request body is not valid JSON", func() {
			It("returns 400", func() {
				ctx.SMWithOAuth.PATCH("/v1/service_brokers/"+brokerID).
					WithText("invalid json").
					WithHeader("content-type", "application/json").
					Expect().
					Status(http.StatusBadRequest).
					JSON().Object().
					Keys().Contains("error", "description")
			})
		})

		Context("when request body contains invalid credentials", func() {
			It("returns 400", func() {
				ctx.SMWithOAuth.PATCH("/v1/service_brokers/"+brokerID).
					WithJSON(common.Object{"credentials": "123"}).
					Expect().
					Status(http.StatusBadRequest).
					JSON().Object().
					Keys().Contains("error", "description")
			})
		})

		Context("when request body contains incomplete credentials", func() {
			It("returns 400", func() {
				ctx.SMWithOAuth.PATCH("/v1/service_brokers/"+brokerID).
					WithJSON(common.Object{"credentials": common.Object{"basic": common.Object{"password": ""}}}).
					Expect().
					Status(http.StatusBadRequest).
					JSON().Object().
					Keys().Contains("error", "description")
			})
		})

		Context("when broker with the name already exists", func() {
			var anotherTestBroker common.Object
			var anotherBrokerServer *common.BrokerServer

			BeforeEach(func() {
				anotherBrokerServer = common.NewBrokerServer()
				anotherBrokerServer.Username = "username"
				anotherBrokerServer.Password = "password"
				anotherTestBroker = common.Object{
					"name":        "another_name",
					"broker_url":  anotherBrokerServer.URL,
					"description": "another_description",
					"credentials": common.Object{
						"basic": common.Object{
							"username": anotherBrokerServer.Username,
							"password": anotherBrokerServer.Password,
						},
					},
				}
			})

			AfterEach(func() {
				if anotherBrokerServer != nil {
					anotherBrokerServer.Close()
				}
			})

			It("returns 409", func() {
				ctx.SMWithOAuth.POST("/v1/service_brokers").
					WithJSON(anotherTestBroker).
					Expect().
					Status(http.StatusCreated)

				assertInvocationCount(anotherBrokerServer.CatalogEndpointRequests, 1)

				ctx.SMWithOAuth.PATCH("/v1/service_brokers/"+brokerID).
					WithJSON(anotherTestBroker).
					Expect().Status(http.StatusConflict).
					JSON().Object().
					Keys().Contains("error", "description")

				assertInvocationCount(brokerServer.CatalogEndpointRequests, 0)
			})
		})

		Context("when credentials are updated", func() {
			It("returns 200", func() {
				brokerServer.Username = "updatedUsername"
				brokerServer.Password = "updatedPassword"
				updatedCredentials := common.Object{
					"credentials": common.Object{
						"basic": common.Object{
							"username": brokerServer.Username,
							"password": brokerServer.Password,
						},
					},
				}
				reply := ctx.SMWithOAuth.PATCH("/v1/service_brokers/" + brokerID).
					WithJSON(updatedCredentials).
					Expect().
					Status(http.StatusOK).
					JSON().Object()

				assertInvocationCount(brokerServer.CatalogEndpointRequests, 1)

				reply = ctx.SMWithOAuth.GET("/v1/service_brokers/" + brokerID).
					Expect().
					Status(http.StatusOK).
					JSON().Object()
				reply.ContainsMap(expectedBrokerResponse)
			})
		})

		Context("when created_at provided in body", func() {
			It("should not change created_at", func() {
				createdAt := "2015-01-01T00:00:00Z"

				ctx.SMWithOAuth.PATCH("/v1/service_brokers/"+brokerID).
					WithJSON(common.Object{"created_at": createdAt}).
					Expect().
					Status(http.StatusOK).JSON().Object().
					ContainsKey("created_at").
					ValueNotEqual("created_at", createdAt)

				assertInvocationCount(brokerServer.CatalogEndpointRequests, 1)

				ctx.SMWithOAuth.GET("/v1/service_brokers/"+brokerID).
					Expect().
					Status(http.StatusOK).JSON().Object().
					ContainsKey("created_at").
					ValueNotEqual("created_at", createdAt)
			})
		})

		Context("when new broker server is available", func() {
			var (
				updatedBrokerServer           *common.BrokerServer
				updatedBrokerJSON             common.Object
				expectedUpdatedBrokerResponse common.Object
			)

			BeforeEach(func() {
				updatedBrokerServer = common.NewBrokerServer()
				updatedBrokerServer.Username = "updated_user"
				updatedBrokerServer.Password = "updated_password"
				updatedBrokerJSON = common.Object{
					"name":        "updated_name",
					"description": "updated_description",
					"broker_url":  updatedBrokerServer.URL,
					"credentials": common.Object{
						"basic": common.Object{
							"username": updatedBrokerServer.Username,
							"password": updatedBrokerServer.Password,
						},
					},
				}

				expectedUpdatedBrokerResponse = common.Object{
					"name":        updatedBrokerJSON["name"],
					"description": updatedBrokerJSON["description"],
					"broker_url":  updatedBrokerJSON["broker_url"],
				}
			})

			AfterEach(func() {
				if updatedBrokerServer != nil {
					updatedBrokerServer.Close()
				}
			})

			Context("when all updatable fields are updated at once", func() {
				It("returns 200", func() {
					ctx.SMWithOAuth.PATCH("/v1/service_brokers/"+brokerID).
						WithJSON(updatedBrokerJSON).
						Expect().
						Status(http.StatusOK).
						JSON().Object().
						ContainsMap(expectedUpdatedBrokerResponse).
						Keys().NotContains("services", "credentials")

					assertInvocationCount(updatedBrokerServer.CatalogEndpointRequests, 1)

					ctx.SMWithOAuth.GET("/v1/service_brokers/"+brokerID).
						Expect().
						Status(http.StatusOK).
						JSON().Object().
						ContainsMap(expectedUpdatedBrokerResponse).
						Keys().NotContains("services", "credentials")
				})
			})

			Context("when broker_url is changed and the credentials are correct", func() {
				It("returns 200", func() {
					updatedBrokerJSON := common.Object{
						"broker_url": updatedBrokerServer.URL,
					}
					updatedBrokerServer.Username = brokerServer.Username
					updatedBrokerServer.Password = brokerServer.Password

					ctx.SMWithOAuth.PATCH("/v1/service_brokers/"+brokerID).
						WithJSON(updatedBrokerJSON).
						Expect().
						Status(http.StatusOK).
						JSON().Object().
						ContainsMap(updatedBrokerJSON).
						Keys().NotContains("services", "credentials")

					assertInvocationCount(brokerServer.CatalogEndpointRequests, 0)
					assertInvocationCount(updatedBrokerServer.CatalogEndpointRequests, 1)

					ctx.SMWithOAuth.GET("/v1/service_brokers/"+brokerID).
						Expect().
						Status(http.StatusOK).
						JSON().Object().
						ContainsMap(updatedBrokerJSON).
						Keys().NotContains("services", "credentials")
				})
			})

			Context("when broker_url is changed but the credentials are wrong", func() {
				It("returns 400", func() {
					updatedBrokerJSON := common.Object{
						"broker_url": updatedBrokerServer.URL,
					}
					ctx.SMWithOAuth.PATCH("/v1/service_brokers/"+brokerID).
						WithJSON(updatedBrokerJSON).
						Expect().
						Status(http.StatusBadRequest).JSON().Object().Keys().Contains("error", "description")

					assertInvocationCount(brokerServer.CatalogEndpointRequests, 0)

					ctx.SMWithOAuth.GET("/v1/service_brokers/"+brokerID).
						Expect().
						Status(http.StatusOK).
						JSON().Object().
						ContainsMap(expectedBrokerResponse).
						Keys().NotContains("services", "credentials")
				})
			})

		})

		Context("when fields are updated one by one", func() {
			It("returns 200", func() {
				for _, prop := range []string{"name", "description"} {
					updatedBrokerJSON := common.Object{}
					updatedBrokerJSON[prop] = "updated"
					ctx.SMWithOAuth.PATCH("/v1/service_brokers/"+brokerID).
						WithJSON(updatedBrokerJSON).
						Expect().
						Status(http.StatusOK).
						JSON().Object().
						ContainsMap(updatedBrokerJSON).
						Keys().NotContains("services", "credentials")

					ctx.SMWithOAuth.GET("/v1/service_brokers/"+brokerID).
						Expect().
						Status(http.StatusOK).
						JSON().Object().
						ContainsMap(updatedBrokerJSON).
						Keys().NotContains("services", "credentials")

				}
				assertInvocationCount(brokerServer.CatalogEndpointRequests, 2)
			})
		})

		Context("when not updatable fields are provided in the request body", func() {
			Context("when broker id is provided in request body", func() {
				It("should not create the broker", func() {
					brokerServerJSON = common.Object{"id": "123"}
					ctx.SMWithOAuth.PATCH("/v1/service_brokers/" + brokerID).
						WithJSON(brokerServerJSON).
						Expect().
						Status(http.StatusOK).
						JSON().Object().
						NotContainsMap(brokerServerJSON)

					ctx.SMWithOAuth.GET("/v1/service_brokers/123").
						Expect().
						Status(http.StatusNotFound)

					assertInvocationCount(brokerServer.CatalogEndpointRequests, 1)
				})
			})

			Context("when unmodifiable fields are provided in the request body", func() {
				BeforeEach(func() {
					brokerServerJSON = common.Object{
						"created_at": "2016-06-08T16:41:26Z",
						"updated_at": "2016-06-08T16:41:26Z",
						"services":   common.Array{common.Object{"name": "serviceName"}},
					}
				})

				It("should not change them", func() {
					ctx.SMWithOAuth.PATCH("/v1/service_brokers/" + brokerID).
						WithJSON(brokerServerJSON).
						Expect().
						Status(http.StatusOK).
						JSON().Object().
						NotContainsMap(brokerServerJSON)

					ctx.SMWithOAuth.GET("/v1/service_brokers").
						Expect().
						Status(http.StatusOK).
						JSON().Object().Value("brokers").Array().First().Object().
						ContainsMap(expectedBrokerResponse)

					assertInvocationCount(brokerServer.CatalogEndpointRequests, 1)
				})
			})
		})

		Context("when obtaining the broker catalog fails because the broker is not reachable", func() {
			BeforeEach(func() {
				brokerServerJSON["broker_url"] = "http://localhost:12345"
			})

			It("returns 400", func() {
				ctx.SMWithOAuth.PATCH("/v1/service_brokers/"+brokerID).WithJSON(brokerServerJSON).
					Expect().
					Status(http.StatusBadRequest).JSON().Object().Keys().Contains("error", "description")
			})
		})

		Context("when the broker catalog is modified", func() {
			Context("when a new service offering with new plans is added", func() {
				var anotherServiceID string
				var anotherPlanID string

				BeforeEach(func() {
					anotherPlan := common.JSONToMap(common.AnotherPlan)
					anotherPlanID = anotherPlan["id"].(string)
					anotherServiceWithAnotherPlan, err := sjson.Set(common.AnotherService, "plans.-1", anotherPlan)
					Expect(err).ShouldNot(HaveOccurred())

					anotherService := common.JSONToMap(anotherServiceWithAnotherPlan)
					anotherServiceID = anotherService["id"].(string)
					Expect(anotherServiceID).ToNot(BeEmpty())

					catalog, err := sjson.Set(common.Catalog, "services.-1", anotherService)

					Expect(err).ShouldNot(HaveOccurred())
					brokerServer.Catalog = common.JSONToMap(catalog)
				})

				It("is returned from the Services API associated with the correct broker", func() {
					ctx.SMWithOAuth.GET("/v1/service_offerings").
						Expect().
						Status(http.StatusOK).
						JSON().
						Path("$.service_offerings[*].catalog_id").Array().NotContains(anotherServiceID)
					ctx.SMWithOAuth.PATCH("/v1/service_brokers/" + brokerID).
						WithJSON(common.Object{}).
						Expect().
						Status(http.StatusOK)
					servicesJsonResp := ctx.SMWithOAuth.GET("/v1/service_offerings").
						Expect().
						Status(http.StatusOK).
						JSON()
					servicesJsonResp.Path("$.service_offerings[*].catalog_id").Array().Contains(anotherServiceID)
					servicesJsonResp.Path("$.service_offerings[*].broker_id").Array().Contains(brokerID)

					var soID string
					for _, so := range servicesJsonResp.Object().Value("service_offerings").Array().Iter() {
						sbID := so.Object().Value("broker_id").String().Raw()
						Expect(sbID).ToNot(BeEmpty())

						catalogID := so.Object().Value("catalog_id").String().Raw()
						Expect(catalogID).ToNot(BeEmpty())

						if catalogID == anotherServiceID && sbID == brokerID {
							soID = so.Object().Value("id").String().Raw()
							Expect(soID).ToNot(BeEmpty())
							break
						}
					}

					plansJsonResp := ctx.SMWithOAuth.GET("/v1/service_plans").
						Expect().
						Status(http.StatusOK).
						JSON()
					plansJsonResp.Path("$.service_plans[*].catalog_id").Array().Contains(anotherPlanID)
					plansJsonResp.Path("$.service_plans[*].service_offering_id").Array().Contains(soID)

					assertInvocationCount(brokerServer.CatalogEndpointRequests, 1)
				})
			})

			verifyPATCHWhenCatalogFieldIsMissing := func(responseVerifier func(r *httpexpect.Response), fieldPath string) {
				BeforeEach(func() {
					catalog, err := sjson.Delete(common.Catalog, fieldPath)
					Expect(err).ToNot(HaveOccurred())

					brokerServer.Catalog = common.JSONToMap(catalog)
				})

				It("returns correct response", func() {
					responseVerifier(ctx.SMWithOAuth.PATCH("/v1/service_brokers/" + brokerID).WithJSON(brokerServerJSON).Expect())

					assertInvocationCount(brokerServer.CatalogEndpointRequests, 1)

				})
			}

			verifyPATCHWhenCatalogFieldHasValue := func(responseVerifier func(r *httpexpect.Response), fieldPath string, fieldValue interface{}) {
				BeforeEach(func() {
					catalog, err := sjson.Set(common.Catalog, fieldPath, fieldValue)
					Expect(err).ToNot(HaveOccurred())

					brokerServer.Catalog = common.JSONToMap(catalog)
				})

				It("returns correct response", func() {
					responseVerifier(ctx.SMWithOAuth.PATCH("/v1/service_brokers/" + brokerID).WithJSON(brokerServerJSON).Expect())

					assertInvocationCount(brokerServer.CatalogEndpointRequests, 1)

				})
			}

			Context("when a new service offering is added", func() {
				var anotherServiceID string

				BeforeEach(func() {
					anotherService := common.JSONToMap(common.AnotherService)
					anotherServiceID = anotherService["id"].(string)
					Expect(anotherServiceID).ToNot(BeEmpty())

					currServices := common.JSONToMap(common.Catalog)["services"].([]interface{})
					currServices = append(currServices, anotherService)

					brokerServer.Catalog = map[string]interface{}{"services": currServices}
				})

				It("is returned from the Services API associated with the correct broker", func() {
					ctx.SMWithOAuth.GET("/v1/service_offerings").
						Expect().
						Status(http.StatusOK).
						JSON().
						Path("$.service_offerings[*].catalog_id").Array().NotContains(anotherServiceID)

					ctx.SMWithOAuth.PATCH("/v1/service_brokers/" + brokerID).
						WithJSON(common.Object{}).
						Expect().
						Status(http.StatusOK)

					jsonResp := ctx.SMWithOAuth.GET("/v1/service_offerings").
						Expect().
						Status(http.StatusOK).
						JSON()
					jsonResp.Path("$.service_offerings[*].catalog_id").Array().Contains(anotherServiceID)
					jsonResp.Path("$.service_offerings[*].broker_id").Array().Contains(brokerID)

					assertInvocationCount(brokerServer.CatalogEndpointRequests, 1)
				})
			})
			Context("when an existing service offering is removed", func() {
				var serviceOfferingID string

				BeforeEach(func() {
					catalogServiceID := gjson.Get(common.Catalog, "services.0.id").Str
					Expect(catalogServiceID).ToNot(BeEmpty())

					serviceOfferings := ctx.SMWithOAuth.GET("/v1/service_offerings").
						Expect().
						Status(http.StatusOK).
						JSON().Object().Value("service_offerings").Array().Iter()

					for _, so := range serviceOfferings {
						sbID := so.Object().Value("broker_id").String().Raw()
						Expect(catalogServiceID).ToNot(BeEmpty())

						catalogID := so.Object().Value("catalog_id").String().Raw()
						Expect(catalogServiceID).ToNot(BeEmpty())

						if catalogID == catalogServiceID && sbID == brokerID {
							serviceOfferingID = so.Object().Value("id").String().Raw()
							Expect(catalogServiceID).ToNot(BeEmpty())
							break
						}
					}
					s, err := sjson.Delete(common.Catalog, "services.0")
					Expect(err).ShouldNot(HaveOccurred())
					brokerServer.Catalog = common.JSONToMap(s)
				})

				It("is no longer returned by the Services and Plans API", func() {
					plans := ctx.SMWithOAuth.GET("/v1/service_plans").
						Expect().
						Status(http.StatusOK).
						JSON().Object().Value("service_plans").Array().Iter()

					var planIDsForService []interface{}
					for _, plan := range plans {
						soID := plan.Object().Value("service_offering_id").String().Raw()
						Expect(soID).ToNot(BeEmpty())
						if soID == serviceOfferingID {
							planID := plan.Object().Value("id").String().Raw()
							Expect(soID).ToNot(BeEmpty())

							planIDsForService = append(planIDsForService, planID)
						}
					}
					ctx.SMWithOAuth.PATCH("/v1/service_brokers/" + brokerID).
						WithJSON(common.Object{}).
						Expect().
						Status(http.StatusOK)

					ctx.SMWithOAuth.GET("/v1/service_offerings").
						Expect().
						Status(http.StatusOK).
						JSON().Path("$.service_offerings[*].id").Array().NotContains(serviceOfferingID)

					ctx.SMWithOAuth.GET("/v1/service_plans").
						Expect().
						Status(http.StatusOK).
						JSON().Path("$.service_plans[*].id").Array().NotContains(planIDsForService)

					assertInvocationCount(brokerServer.CatalogEndpointRequests, 1)
				})
			})

			Context("when an existing service offering is modified", func() {
				Context("when catalog service id is removed", func() {
					verifyPATCHWhenCatalogFieldIsMissing(func(r *httpexpect.Response) {
						r.Status(http.StatusBadRequest).JSON().Object().Keys().Contains("error", "description")
					}, "services.0.id")
				})

				Context("when catalog service name is removed", func() {
					verifyPATCHWhenCatalogFieldIsMissing(func(r *httpexpect.Response) {
						r.Status(http.StatusBadRequest).JSON().Object().Keys().Contains("error", "description")
					}, "services.0.name")
				})

				Context("when catalog service description is removed", func() {
					verifyPATCHWhenCatalogFieldIsMissing(func(r *httpexpect.Response) {
						r.Status(http.StatusOK)
					}, "services.0.description")
				})

				Context("when tags are invalid json", func() {
					verifyPATCHWhenCatalogFieldHasValue(func(r *httpexpect.Response) {
						r.Status(http.StatusBadRequest).JSON().Object().Keys().Contains("error", "description")
					}, "services.0.tags", "{invalid")
				})

				Context("when requires is invalid json", func() {
					verifyPATCHWhenCatalogFieldHasValue(func(r *httpexpect.Response) {
						r.Status(http.StatusBadRequest).JSON().Object().Keys().Contains("error", "description")
					}, "services.0.requires", "{invalid")
				})

				Context("when metadata is invalid json", func() {
					verifyPATCHWhenCatalogFieldHasValue(func(r *httpexpect.Response) {
						r.Status(http.StatusBadRequest).JSON().Object().Keys().Contains("error", "description")
					}, "services.0.metadata", "{invalid")
				})
			})

			Context("when a new service plan is added", func() {
				var anotherPlanID string
				var serviceOfferingID string

				BeforeEach(func() {
					anotherPlan := common.JSONToMap(common.AnotherPlan)
					anotherPlanID = anotherPlan["id"].(string)
					Expect(anotherPlan).ToNot(BeEmpty())
					catalogServiceID := gjson.Get(common.Catalog, "services.0.id").Str
					Expect(catalogServiceID).ToNot(BeEmpty())

					serviceOfferings := ctx.SMWithOAuth.GET("/v1/service_offerings").
						Expect().
						Status(http.StatusOK).
						JSON().Object().Value("service_offerings").Array().Iter()

					for _, so := range serviceOfferings {
						sbID := so.Object().Value("broker_id").String().Raw()
						Expect(sbID).ToNot(BeEmpty())

						catalogID := so.Object().Value("catalog_id").String().Raw()
						Expect(catalogID).ToNot(BeEmpty())

						if catalogID == catalogServiceID && sbID == brokerID {
							serviceOfferingID = so.Object().Value("id").String().Raw()
							Expect(catalogServiceID).ToNot(BeEmpty())
							break
						}
					}
					s, err := sjson.Set(common.Catalog, "services.0.plans.2", anotherPlan)
					Expect(err).ShouldNot(HaveOccurred())
					brokerServer.Catalog = common.JSONToMap(s)
				})

				It("is returned from the Plans API associated with the correct service offering", func() {
					ctx.SMWithOAuth.GET("/v1/service_plans").
						Expect().
						Status(http.StatusOK).
						JSON().
						Path("$.service_plans[*].catalog_id").Array().NotContains(anotherPlanID)

					ctx.SMWithOAuth.PATCH("/v1/service_brokers/" + brokerID).
						WithJSON(common.Object{}).
						Expect().
						Status(http.StatusOK)

					jsonResp := ctx.SMWithOAuth.GET("/v1/service_plans").
						Expect().
						Status(http.StatusOK).
						JSON()
					jsonResp.Path("$.service_plans[*].catalog_id").Array().Contains(anotherPlanID)
					jsonResp.Path("$.service_plans[*].service_offering_id").Array().Contains(serviceOfferingID)

					assertInvocationCount(brokerServer.CatalogEndpointRequests, 1)
				})
			})

			Context("when an existing service plan is removed", func() {
				var removedPlanCatalogID string

				BeforeEach(func() {
					removedPlanCatalogID = gjson.Get(common.Catalog, "services.0.plans.0.id").Str
					Expect(removedPlanCatalogID).ToNot(BeEmpty())
					s, err := sjson.Delete(common.Catalog, "services.0.plans.0")
					Expect(err).ShouldNot(HaveOccurred())
					brokerServer.Catalog = common.JSONToMap(s)
				})

				It("is no longer returned by the Plans API", func() {
					ctx.SMWithOAuth.GET("/v1/service_plans").
						Expect().
						Status(http.StatusOK).
						JSON().Path("$.service_plans[*].catalog_id").Array().Contains(removedPlanCatalogID)

					ctx.SMWithOAuth.PATCH("/v1/service_brokers/" + brokerID).
						WithJSON(common.Object{}).
						Expect().
						Status(http.StatusOK)

					ctx.SMWithOAuth.GET("/v1/service_plans").
						Expect().
						Status(http.StatusOK).
						JSON().Path("$.service_plans[*].catalog_id").Array().NotContains(removedPlanCatalogID)

					assertInvocationCount(brokerServer.CatalogEndpointRequests, 1)
				})
			})

			Context("when an existing service plan is modified", func() {
				Context("when catalog plan id is removed", func() {
					verifyPATCHWhenCatalogFieldIsMissing(func(r *httpexpect.Response) {
						r.Status(http.StatusBadRequest).JSON().Object().Keys().Contains("error", "description")
					}, "services.0.plans.0.id")
				})

				Context("when catalog plan name is removed", func() {
					verifyPATCHWhenCatalogFieldIsMissing(func(r *httpexpect.Response) {
						r.Status(http.StatusBadRequest).JSON().Object().Keys().Contains("error", "description")
					}, "services.0.plans.0.name")
				})

				Context("when catalog plan description is removed", func() {
					verifyPATCHWhenCatalogFieldIsMissing(func(r *httpexpect.Response) {
						r.Status(http.StatusOK)
					}, "services.0.plans.0.description")
				})

				Context("when schemas is invalid json", func() {
					verifyPATCHWhenCatalogFieldHasValue(func(r *httpexpect.Response) {
						r.Status(http.StatusBadRequest).JSON().Object().Keys().Contains("error", "description")
					}, "services.0.plans.0.schemas", "{invalid")
				})

				Context("when metadata is invalid json", func() {
					verifyPATCHWhenCatalogFieldHasValue(func(r *httpexpect.Response) {
						r.Status(http.StatusBadRequest).JSON().Object().Keys().Contains("error", "description")
					}, "services.0.plans.0.metadata", "{invalid")
				})
			})
		})
	})

	Describe("DELETE", func() {
		AfterEach(func() {
			assertInvocationCount(brokerServer.CatalogEndpointRequests, 0)
		})

		Context("when broker does not exist", func() {
			It("returns 404", func() {
				ctx.SMWithOAuth.DELETE("/v1/service_brokers/999").
					Expect().
					Status(http.StatusNotFound).
					JSON().Object().
					Keys().Contains("error", "description")
			})
		})

		Context("when broker exists", func() {
			var id string

			BeforeEach(func() {
				reply := ctx.SMWithOAuth.POST("/v1/service_brokers").WithJSON(brokerServerJSON).
					Expect().
					Status(http.StatusCreated).
					JSON().Object().
					ContainsMap(expectedBrokerResponse)

				id = reply.Value("id").String().Raw()

				assertInvocationCount(brokerServer.CatalogEndpointRequests, 1)
				brokerServer.ResetCallHistory()
			})

			It("returns 200", func() {
				ctx.SMWithOAuth.GET("/v1/service_brokers/" + id).
					Expect().
					Status(http.StatusOK)

				ctx.SMWithOAuth.DELETE("/v1/service_brokers/" + id).
					Expect().
					Status(http.StatusOK).JSON().Object().Empty()

				ctx.SMWithOAuth.GET("/v1/service_brokers/" + id).
					Expect().
					Status(http.StatusNotFound)
			})

			It("deletes the related service offerings", func() {
				ctx.SMWithOAuth.GET("/v1/service_brokers/" + id).
					Expect().
					Status(http.StatusOK)

				ctx.SMWithOAuth.DELETE("/v1/service_brokers/" + id).
					Expect().
					Status(http.StatusOK).JSON().Object().Empty()

				ctx.SMWithOAuth.GET("/v1/service_offerings").
					Expect().
					Status(http.StatusOK).
					JSON().
					Path("$.service_offerings[*].broker_id").Array().NotContains(id)
			})

			It("deletes the related service plans", func() {
				ctx.SMWithOAuth.GET("/v1/service_brokers/" + id).
					Expect().
					Status(http.StatusOK)

				serviceOfferings := ctx.SMWithOAuth.GET("/v1/service_offerings").
					Expect().
					Status(http.StatusOK).
					JSON().Object().Value("service_offerings").Array().Iter()

				serviceIDsForBroker := make([]interface{}, 0)
				for _, so := range serviceOfferings {
					brokerID := so.Object().Value("broker_id").String().Raw()
					Expect(brokerID).ToNot(BeEmpty())
					if brokerID == id {
						id := so.Object().Value("id").Raw()
						Expect(id).ToNot(BeEmpty())
						serviceIDsForBroker = append(serviceIDsForBroker, id)
					}
				}

				plans := ctx.SMWithOAuth.GET("/v1/service_plans").
					Expect().
					Status(http.StatusOK).
					JSON().Object().Value("service_plans").Array().Iter()

				planIDsForBroker := make([]interface{}, 0)
				for _, plan := range plans {
					serviceIDForPlan := plan.Object().Value("service_offering_id").String().Raw()
					Expect(serviceIDForPlan).ToNot(BeEmpty())
					for _, serviceIDForBroker := range serviceIDsForBroker {
						if serviceIDForPlan == serviceIDForBroker {
							planID := plan.Object().Value("id").String().Raw()
							Expect(planID).ToNot(BeEmpty())
							planIDsForBroker = append(planIDsForBroker, planID)
						}
					}
				}

				ctx.SMWithOAuth.DELETE("/v1/service_brokers/" + id).
					Expect().
					Status(http.StatusOK).JSON().Object().Empty()

				ctx.SMWithOAuth.GET("/v1/service_plans").
					Expect().
					Status(http.StatusOK).
					JSON().Path("$.service_plans[*].id").Array().NotContains(planIDsForBroker)
			})
		})
	})
})

func assertInvocationCount(requests []*http.Request, invocationCount int) {
	Expect(len(requests)).To(Equal(invocationCount))
}
