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
	"os"
	"strings"
	"testing"

	"encoding/json"

	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"github.com/spf13/cast"
)

func TestBrokers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Broker API Tests Suite")
}

var _ = Describe("Service Manager Broker API", func() {

	var (
		ctx          *common.TestContext
		brokerServer *ghttp.Server

		testBroker     common.Object
		expectedBroker common.Object

		catalogResponse []byte
		code            int
	)

	BeforeSuite(func() {
		os.Chdir("../..")

		ctx = common.NewTestContextFromAPIs(nil)
	})

	AfterSuite(func() {
		ctx.Cleanup()
	})

	BeforeEach(func() {
		code = http.StatusOK
		catalogResponse = []byte(common.Catalog)
		brokerServer = common.FakeBrokerServer(&code, &catalogResponse)

		testBroker = common.Object{
			"name":        "name",
			"broker_url":  brokerServer.URL(),
			"description": "description",
			"credentials": common.Object{
				"basic": common.Object{
					"username": "buser",
					"password": "bpass",
				},
			},
		}

		expectedBroker = common.Object{
			"name":        "name",
			"broker_url":  brokerServer.URL(),
			"description": "description",
		}
		common.RemoveAllBrokers(ctx.SMWithOAuth)
	})

	Describe("GET", func() {
		var id string

		AfterEach(func() {
			common.VerifyBrokerCatalogEndpointInvoked(brokerServer, 0)
		})

		Context("when the broker does not exist", func() {
			BeforeEach(func() {
				id = "12345"
			})

			It("returns 404", func() {
				ctx.SMWithOAuth.GET("/v1/service_brokers/"+id).
					Expect().
					Status(http.StatusNotFound).
					JSON().Object().
					Keys().Contains("error", "description")
			})
		})

		Context("when the broker exists", func() {
			BeforeEach(func() {
				reply := ctx.SMWithOAuth.POST("/v1/service_brokers").WithJSON(testBroker).
					Expect().
					Status(http.StatusCreated).
					JSON().Object().
					ContainsMap(expectedBroker)

				id = reply.Value("id").String().Raw()

				common.VerifyBrokerCatalogEndpointInvoked(brokerServer, 1)
				common.ClearReceivedRequests(&code, &catalogResponse, brokerServer)
			})

			It("returns the broker with given id", func() {
				ctx.SMWithOAuth.GET("/v1/service_brokers/"+id).
					Expect().
					Status(http.StatusOK).
					JSON().Object().
					ContainsMap(expectedBroker).
					Keys().NotContains("credentials", "catalog")
			})
		})
	})

	Describe("GET All", func() {
		AfterEach(func() {
			common.VerifyBrokerCatalogEndpointInvoked(brokerServer, 0)
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
				ctx.SMWithOAuth.POST("/v1/service_brokers").WithJSON(testBroker).
					Expect().
					Status(http.StatusCreated).
					JSON().Object().
					ContainsMap(expectedBroker).
					Keys().
					NotContains("credentials", "catalog")

				common.VerifyBrokerCatalogEndpointInvoked(brokerServer, 1)
				common.ClearReceivedRequests(&code, &catalogResponse, brokerServer)
			})

			It("returns all without catalog if no query parameter is provided", func() {
				ctx.SMWithOAuth.GET("/v1/service_brokers").
					Expect().
					Status(http.StatusOK).
					JSON().Object().Value("brokers").Array().First().Object().
					ContainsMap(expectedBroker).
					Keys().
					NotContains("credentials", "catalog")
			})

			It("returns all with catalog if query parameter is provided", func() {
				ctx.SMWithOAuth.GET("/v1/service_brokers").WithQuery("catalog", true).
					Expect().
					Status(http.StatusOK).
					JSON().Object().Value("brokers").Array().First().Object().
					ContainsMap(expectedBroker).
					ContainsKey("catalog").
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

				common.VerifyBrokerCatalogEndpointInvoked(brokerServer, 0)
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

				common.VerifyBrokerCatalogEndpointInvoked(brokerServer, 0)
			})
		})

		Context("when a request body field is missing", func() {
			assertPOSTReturns400WhenFieldIsMissing := func(field string) {
				BeforeEach(func() {
					delete(testBroker, field)
					delete(expectedBroker, field)
				})

				It("returns 400", func() {
					ctx.SMWithOAuth.POST("/v1/service_brokers").WithJSON(testBroker).
						Expect().
						Status(http.StatusBadRequest).
						JSON().Object().
						Keys().Contains("error", "description")

					common.VerifyBrokerCatalogEndpointInvoked(brokerServer, 0)
				})
			}

			assertPOSTReturns201WhenFieldIsMissing := func(field string) {
				BeforeEach(func() {
					delete(testBroker, field)
					delete(expectedBroker, field)
				})

				It("returns 201", func() {
					ctx.SMWithOAuth.POST("/v1/service_brokers").WithJSON(testBroker).
						Expect().
						Status(http.StatusCreated).
						JSON().Object().
						ContainsMap(expectedBroker).
						Keys().NotContains("catalog", "credentials")

					common.VerifyBrokerCatalogEndpointInvoked(brokerServer, 1)
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

		Context("when fetching catalog fails", func() {
			BeforeEach(func() {
				code = http.StatusInternalServerError
			})

			It("returns an error", func() {
				ctx.SMWithOAuth.POST("/v1/service_brokers").
					WithJSON(testBroker).
					Expect().Status(http.StatusInternalServerError).
					JSON().Object().
					Keys().Contains("error", "description")

				common.VerifyBrokerCatalogEndpointInvoked(brokerServer, 1)
			})
		})

		Context("when request is successful", func() {
			assertPOSTReturns201 := func() {
				It("returns 201", func() {
					ctx.SMWithOAuth.POST("/v1/service_brokers").WithJSON(testBroker).
						Expect().
						Status(http.StatusCreated).
						JSON().Object().
						ContainsMap(expectedBroker).
						Keys().NotContains("catalog", "credentials")

					common.VerifyBrokerCatalogEndpointInvoked(brokerServer, 1)
				})
			}

			Context("when broker URL does not end with trailing slash", func() {
				BeforeEach(func() {
					testBroker["broker_url"] = strings.TrimRight(cast.ToString(testBroker["broker_url"]), "/")
					expectedBroker["broker_url"] = strings.TrimRight(cast.ToString(expectedBroker["broker_url"]), "/")
				})

				assertPOSTReturns201()
			})

			Context("when broker URL ends with trailing slash", func() {
				BeforeEach(func() {
					testBroker["broker_url"] = cast.ToString(testBroker["broker_url"]) + "/"
					expectedBroker["broker_url"] = cast.ToString(expectedBroker["broker_url"]) + "/"
				})

				assertPOSTReturns201()
			})
		})

		Context("When broker with name already exists", func() {
			It("returns 409", func() {
				ctx.SMWithOAuth.POST("/v1/service_brokers").WithJSON(testBroker).
					Expect().
					Status(http.StatusCreated)

				ctx.SMWithOAuth.POST("/v1/service_brokers").WithJSON(testBroker).
					Expect().
					Status(http.StatusConflict).
					JSON().Object().
					Keys().Contains("error", "description")

				common.VerifyBrokerCatalogEndpointInvoked(brokerServer, 2)
			})
		})
	})

	Describe("PATCH", func() {
		var id string

		BeforeEach(func() {
			reply := ctx.SMWithOAuth.POST("/v1/service_brokers").WithJSON(testBroker).
				Expect().
				Status(http.StatusCreated).
				JSON().Object().
				ContainsMap(expectedBroker)

			id = reply.Value("id").String().Raw()

			common.VerifyBrokerCatalogEndpointInvoked(brokerServer, 1)
			common.ClearReceivedRequests(&code, &catalogResponse, brokerServer)
		})

		Context("when content type is not JSON", func() {
			It("returns 415", func() {
				ctx.SMWithOAuth.PATCH("/v1/service_brokers/"+id).
					WithText("text").
					Expect().Status(http.StatusUnsupportedMediaType).
					JSON().Object().
					Keys().Contains("error", "description")

				common.VerifyBrokerCatalogEndpointInvoked(brokerServer, 0)
			})
		})

		Context("when broker is missing", func() {
			BeforeEach(func() {
				id = "invalid_id"
			})

			It("returns 404", func() {
				ctx.SMWithOAuth.PATCH("/v1/service_brokers/"+id).
					WithJSON(testBroker).
					Expect().Status(http.StatusNotFound).
					JSON().Object().
					Keys().Contains("error", "description")
			})
		})

		Context("when request body is not valid JSON", func() {
			It("returns 400", func() {
				ctx.SMWithOAuth.PATCH("/v1/service_brokers/"+id).
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
				ctx.SMWithOAuth.PATCH("/v1/service_brokers/"+id).
					WithJSON(common.Object{"credentials": "123"}).
					Expect().
					Status(http.StatusBadRequest).
					JSON().Object().
					Keys().Contains("error", "description")
			})
		})

		Context("when request body contains incomplete credentials", func() {
			It("returns 400", func() {
				ctx.SMWithOAuth.PATCH("/v1/service_brokers/"+id).
					WithJSON(common.Object{"credentials": common.Object{"basic": common.Object{"password": ""}}}).
					Expect().
					Status(http.StatusBadRequest).
					JSON().Object().
					Keys().Contains("error", "description")
			})
		})

		Context("when broker with the name already exists", func() {
			var anotherTestBroker common.Object
			var anotherBrokerServer *ghttp.Server

			BeforeEach(func() {
				anotherBrokerServer = common.FakeBrokerServer(&code, &catalogResponse)

				anotherTestBroker = common.Object{
					"name":        "another_name",
					"broker_url":  anotherBrokerServer.URL(),
					"description": "another_description",
					"credentials": common.Object{
						"basic": common.Object{
							"username": "buser",
							"password": "bpass",
						},
					},
				}
			})

			It("returns 409", func() {
				ctx.SMWithOAuth.POST("/v1/service_brokers").
					WithJSON(anotherTestBroker).
					Expect().
					Status(http.StatusCreated)

				common.VerifyBrokerCatalogEndpointInvoked(anotherBrokerServer, 1)

				ctx.SMWithOAuth.PATCH("/v1/service_brokers/"+id).
					WithJSON(anotherTestBroker).
					Expect().Status(http.StatusConflict).
					JSON().Object().
					Keys().Contains("error", "description")

				common.VerifyBrokerCatalogEndpointInvoked(brokerServer, 0)
			})
		})

		Context("when updatable fields are being updated", func() {
			var (
				updatedBrokerServer   *ghttp.Server
				updatedBroker         common.Object
				expectedUpdatedBroker common.Object
			)

			BeforeEach(func() {
				updatedBrokerServer = common.FakeBrokerServer(&code, &catalogResponse)

				updatedBroker = common.Object{
					"name":        "updated_name",
					"description": "updated_description",
					"broker_url":  updatedBrokerServer.URL(),
					"credentials": common.Object{
						"basic": common.Object{
							"username": "updated_user",
							"password": "updated_password",
						},
					},
				}

				expectedUpdatedBroker = common.Object{
					"name":        updatedBroker["name"],
					"description": updatedBroker["description"],
					"broker_url":  updatedBroker["broker_url"],
				}
			})

			Context("when created_at provided in body", func() {
				It("should not change created_at", func() {
					createdAt := "2015-01-01T00:00:00Z"

					ctx.SMWithOAuth.PATCH("/v1/service_brokers/"+id).
						WithJSON(common.Object{"created_at": createdAt}).
						Expect().
						Status(http.StatusOK).JSON().Object().
						ContainsKey("created_at").
						ValueNotEqual("created_at", createdAt)

					ctx.SMWithOAuth.GET("/v1/service_brokers/"+id).
						Expect().
						Status(http.StatusOK).JSON().Object().
						ContainsKey("created_at").
						ValueNotEqual("created_at", createdAt)
				})
			})

			Context("when all updatable fields are updated at once", func() {
				It("returns 200", func() {
					ctx.SMWithOAuth.PATCH("/v1/service_brokers/"+id).
						WithJSON(updatedBroker).
						Expect().
						Status(http.StatusOK).
						JSON().Object().
						ContainsMap(expectedUpdatedBroker).
						Keys().NotContains("catalog", "credentials")

					ctx.SMWithOAuth.GET("/v1/service_brokers/"+id).
						Expect().
						Status(http.StatusOK).
						JSON().Object().
						ContainsMap(expectedUpdatedBroker).
						Keys().NotContains("catalog", "credentials")

					common.VerifyBrokerCatalogEndpointInvoked(updatedBrokerServer, 1)
				})
			})

			Context("when updatable fields are separately updated", func() {
				It("returns 200", func() {
					for prop, val := range updatedBroker {
						update := common.Object{}
						update[prop] = val

						reply := ctx.SMWithOAuth.PATCH("/v1/service_brokers/" + id).
							WithJSON(update).
							Expect().
							Status(http.StatusOK).
							JSON().Object()
						if strings.ToLower(prop) != "credentials" {
							reply.ContainsMap(update)
						}

						reply = ctx.SMWithOAuth.GET("/v1/service_brokers/" + id).
							Expect().
							Status(http.StatusOK).
							JSON().Object()
						if strings.ToLower(prop) != "credentials" {
							reply.ContainsMap(update)
						}
					}

					Expect(len(brokerServer.ReceivedRequests()) + len(updatedBrokerServer.ReceivedRequests())).To(Equal(len(updatedBroker)))

				})
			})
		})

		Context("when not updatable fields are provided in the request body", func() {
			Context("when broker id is provided in request body", func() {
				It("should not create the broker", func() {
					testBroker = common.Object{"id": "123"}
					ctx.SMWithOAuth.PATCH("/v1/service_brokers/" + id).
						WithJSON(testBroker).
						Expect().
						Status(http.StatusOK).
						JSON().Object().
						NotContainsMap(testBroker)

					ctx.SMWithOAuth.GET("/v1/service_brokers/123").
						Expect().
						Status(http.StatusNotFound)

					common.VerifyBrokerCatalogEndpointInvoked(brokerServer, 1)
				})
			})

			Context("when unmodifiable fields are provided request body", func() {
				var (
					err                error
					unmarshaledCatalog common.Object
				)

				BeforeEach(func() {
					testBroker = common.Object{
						"created_at": "2016-06-08T16:41:26Z",
						"updated_at": "2016-06-08T16:41:26Z",
						"credentials": common.Object{
							"basic": common.Object{
								"username": "updated_user",
								"password": "updated_password",
							},
						},
						"catalog": common.Object{},
					}

					unmarshaledCatalog = common.Object{}
					err = json.Unmarshal([]byte(common.Catalog), &unmarshaledCatalog)
				})

				It("should not change them", func() {
					ctx.SMWithOAuth.PATCH("/v1/service_brokers/" + id).
						WithJSON(testBroker).
						Expect().
						Status(http.StatusOK).
						JSON().Object().
						NotContainsMap(testBroker)

					ctx.SMWithOAuth.GET("/v1/service_brokers").
						WithQuery("catalog", true).
						Expect().
						Status(http.StatusOK).
						JSON().Object().Value("brokers").Array().First().Object().
						ContainsMap(expectedBroker).
						Value("catalog").Equal(unmarshaledCatalog)

					common.VerifyBrokerCatalogEndpointInvoked(brokerServer, 1)
				})
			})
		})

		Context("when underlying broker catalog is modified", func() {
			var (
				err            error
				updatedCatalog common.Object
			)

			BeforeEach(func() {
				updatedCatalog = common.Object{
					"services": []interface{}{},
				}

				catalogResponse, err = json.Marshal(updatedCatalog)
				Expect(err).ShouldNot(HaveOccurred())
			})

			It("updates the catalog for the broker", func() {
				ctx.SMWithOAuth.PATCH("/v1/service_brokers/" + id).
					WithJSON(common.Object{}).
					Expect().
					Status(http.StatusOK)

				ctx.SMWithOAuth.GET("/v1/service_brokers").
					WithQuery("catalog", true).
					Expect().
					Status(http.StatusOK).
					JSON().Object().Value("brokers").Array().First().Object().
					ContainsMap(expectedBroker).
					Value("catalog").Equal(updatedCatalog)

				common.VerifyBrokerCatalogEndpointInvoked(brokerServer, 1)
			})
		})
	})

	Describe("DELETE", func() {
		AfterEach(func() {
			common.VerifyBrokerCatalogEndpointInvoked(brokerServer, 0)
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
				reply := ctx.SMWithOAuth.POST("/v1/service_brokers").WithJSON(testBroker).
					Expect().
					Status(http.StatusCreated).
					JSON().Object().
					ContainsMap(expectedBroker)

				id = reply.Value("id").String().Raw()

				common.VerifyBrokerCatalogEndpointInvoked(brokerServer, 1)
				common.ClearReceivedRequests(&code, &catalogResponse, brokerServer)
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
		})
	})
})
