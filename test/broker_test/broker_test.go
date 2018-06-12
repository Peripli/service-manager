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
	"net/http/httptest"
	"os"
	"testing"

	"github.com/Peripli/service-manager/test/common"
	"github.com/gavv/httpexpect"
	. "github.com/onsi/ginkgo"
)

type object = common.Object
type array = common.Array

// TestBrokers tests for broker API
func TestBrokers(t *testing.T) {
	os.Chdir("../..")
	RunSpecs(t, "Broker API Tests Suite")
}

var _ = Describe("Service Manager Broker API", func() {
	var SM *httpexpect.Expect
	var testServer *httptest.Server

	BeforeSuite(func() {
		testServer = httptest.NewServer(common.GetServerRouter())
		SM = httpexpect.New(GinkgoT(), testServer.URL)
	})

	AfterSuite(func() {
		if testServer != nil {
			testServer.Close()
		}
	})

	BeforeEach(func() {
		common.RemoveAllBrokers(SM)
	})

	Describe("GET", func() {
		Context("When broker does not exist", func() {
			It("returns 404", func() {
				SM.GET("/v1/service_brokers/999").
					Expect().
					Status(http.StatusNotFound).
					JSON().Object().Keys().Contains("error", "description")

			})
		})
		Context("When broker exists", func() {
			It("returns the broker with given id", func() {
				broker := common.MakeBroker("broker1", "http://domain.com/broker", "")
				reply := SM.POST("/v1/service_brokers").WithJSON(broker).
					Expect().Status(http.StatusCreated).JSON().Object()
				id := reply.Value("id").String().Raw()

				reply = SM.GET("/v1/service_brokers/" + id).
					Expect().
					Status(http.StatusOK).
					JSON().Object()

				reply.NotContainsKey("credentials")
				broker["id"] = id
				delete(broker, "credentials")
				common.MapContains(reply.Raw(), broker)
			})
		})
	})

	Describe("GET All", func() {
		Context("When no broker exists", func() {
			It("returns empty array", func() {
				SM.GET("/v1/service_brokers").
					Expect().
					Status(http.StatusOK).
					JSON().Object().Value("brokers").Array().Empty()
			})
		})
		Context("When there are brokers", func() {
			FIt("returns all", func() {
				brokers := array{}

				addBroker := func(name string, url string, description string) {
					broker := common.MakeBroker(name, url, description)
					reply := SM.POST("/v1/service_brokers").WithJSON(broker).
						Expect().Status(http.StatusCreated).JSON().Object()
					id := reply.Value("id").String().Raw()
					broker["id"] = id
					delete(broker, "credentials")
					brokers = append(brokers, broker)

					replyArray := SM.GET("/v1/service_brokers").
						Expect().
						Status(http.StatusOK).
						JSON().Object().Value("brokers").Array()
					for _, v := range replyArray.Iter() {
						obj := v.Object().Raw()
						delete(obj, "created_at")
						delete(obj, "updated_at")
					}
					replyArray.ContainsOnly(brokers...)
				}

				addBroker("broker1", "http://host1.com", "broker one")
				addBroker("broker2", "http://host2.com", "broker two")
			})
		})
	})

	Describe("POST", func() {
		Context("When content type is not JSON", func() {
			It("returns 415", func() {
				SM.POST("/v1/service_brokers").
					WithText("text").
					Expect().Status(http.StatusUnsupportedMediaType)
			})
		})

		Context("When user input is not valid JSON", func() {
			It("returns 400", func() {
				SM.POST("/v1/service_brokers").
					WithText("invalid json").
					WithHeader("content-type", "application/json").
					Expect().Status(http.StatusBadRequest)
			})
		})

		Context("When mandatory field is missing", func() {
			It("returns 400", func() {
				newBroker := func() object {
					return common.MakeBroker("broker1", "http://domain.com/broker", "")
				}
				SM.POST("/v1/service_brokers").
					WithJSON(newBroker()).
					Expect().Status(http.StatusCreated)

				for _, prop := range []string{"name", "broker_url", "credentials"} {
					broker := newBroker()
					delete(broker, prop)

					SM.POST("/v1/service_brokers").
						WithJSON(broker).
						Expect().Status(http.StatusBadRequest)
				}
			})
		})

		Context("When optional fields are missing", func() {
			It("succeeds", func() {
				broker := common.MakeBroker("broker1", "http://domain.com/broker", "")
				// delete optional fields
				delete(broker, "description")

				reply := SM.POST("/v1/service_brokers").
					WithJSON(broker).
					Expect().Status(http.StatusCreated).JSON().Object()

				delete(broker, "credentials")
				broker["id"] = reply.Value("id").String().Raw()
				// optional fields returned with default values
				broker["description"] = ""

				common.MapContains(reply.Raw(), broker)
			})
		})

		Context("When no fields are missing", func() {
			It("returns the new broker", func() {
				broker := common.MakeBroker("broker1", "http://domain.com/broker", "broker one")

				By("POST returns the new broker")

				reply := SM.POST("/v1/service_brokers").
					WithJSON(broker).
					Expect().Status(http.StatusCreated).JSON().Object()
				delete(broker, "credentials")

				id := reply.Value("id").String().NotEmpty().Raw()
				broker["id"] = id
				common.MapContains(reply.Raw(), broker)

				By("GET returns the same broker")

				reply = SM.GET("/v1/service_brokers/" + id).
					Expect().Status(http.StatusOK).JSON().Object()

				common.MapContains(reply.Raw(), broker)
			})
		})

		Context("When duplicate name is provided", func() {
			It("returns 409", func() {
				broker := common.MakeBroker("broker1", "http://domain.com/broker", "")

				SM.POST("/v1/service_brokers").
					WithJSON(broker).
					Expect().Status(http.StatusCreated)
				SM.POST("/v1/service_brokers").
					WithJSON(broker).
					Expect().Status(http.StatusConflict)
			})
		})
	})

	Describe("PATCH", func() {
		var broker object
		var id string

		BeforeEach(func() {
			By("Create new broker")
			broker = common.MakeBroker("broker1", "http://domain.com/broker", "desc1")
			reply := SM.POST("/v1/service_brokers").
				WithJSON(broker).
				Expect().Status(http.StatusCreated).JSON().Object()
			delete(broker, "credentials")
			id = reply.Value("id").String().Raw()
			broker["id"] = id
		})

		Context("When content type is not JSON", func() {
			It("returns 415", func() {
				SM.PATCH("/v1/service_brokers/" + id).
					WithText("text").
					Expect().Status(http.StatusUnsupportedMediaType)
			})
		})

		Context("When broker is missing", func() {
			It("returns 404", func() {
				SM.PATCH("/v1/service_brokers/invalid_id").
					WithJSON(broker).
					Expect().Status(http.StatusNotFound)
			})
		})

		Context("When input is not valid JSON", func() {
			It("returns 400 if input is not valid JSON", func() {
				SM.PATCH("/v1/service_brokers/"+id).
					WithText("invalid json").
					WithHeader("content-type", "application/json").
					Expect().Status(http.StatusBadRequest)
			})
		})

		Context("When duplicate name is provided", func() {
			It("returns 409", func() {
				broker2 := common.MakeBroker("broker2", "http://domain.com/broker2", "")

				SM.POST("/v1/service_brokers").
					WithJSON(broker2).
					Expect().Status(http.StatusCreated)
				SM.PATCH("/v1/service_brokers/" + id).
					WithJSON(broker2).
					Expect().Status(http.StatusConflict)
			})
		})

		Context("When all properties are updated", func() {
			It("returns 200", func() {
				By("Update broker")

				updatedBroker := common.MakeBroker("broker2", "http://domain.com/broker2", "desc2")
				updatedBroker["credentials"] = object{
					"basic": object{
						"username": "auser",
						"password": "apass",
					},
				}

				reply := SM.PATCH("/v1/service_brokers/" + id).
					WithJSON(updatedBroker).
					Expect().
					Status(http.StatusOK).JSON().Object()
				delete(updatedBroker, "credentials")

				common.MapContains(reply.Raw(), updatedBroker)

				By("Update is persisted")

				reply = SM.GET("/v1/service_brokers/" + id).
					Expect().
					Status(http.StatusOK).JSON().Object()
			})
		})

		Context("When properties are separately updated", func() {
			It("can update each one", func() {
				newBroker := common.MakeBroker("bb8", "http://lucas.arts", "a robot")
				newBroker["credentials"] = object{
					"basic": object{
						"username": "auser",
						"password": "apass",
					},
				}

				for prop, val := range newBroker {
					update := object{}
					update[prop] = val

					reply := SM.PATCH("/v1/service_brokers/" + id).
						WithJSON(update).
						Expect().
						Status(http.StatusOK).JSON().Object()

					if prop != "credentials" { // credentials are not returned
						broker[prop] = val
					}
					common.MapContains(reply.Raw(), broker)

					reply = SM.GET("/v1/service_brokers/" + id).
						Expect().
						Status(http.StatusOK).JSON().Object()

					common.MapContains(reply.Raw(), broker)
				}
			})
		})

		Context("When broker id is provided", func() {
			It("should not change id", func() {
				SM.PATCH("/v1/service_brokers/" + id).
					WithJSON(object{"id": "123"}).
					Expect().
					Status(http.StatusOK)

				SM.GET("/v1/service_brokers/123").
					Expect().
					Status(http.StatusNotFound)
			})
		})

		Context("When malformed credentials are provided", func() {
			It("returns 400", func() {
				SM.PATCH("/v1/service_brokers/" + id).
					WithJSON(object{"credentials": "123"}).
					Expect().
					Status(http.StatusBadRequest)
			})
		})

		Context("When incomplete credentials are provided", func() {
			It("returns 400", func() {
				SM.PATCH("/v1/service_brokers/" + id).
					WithJSON(object{"credentials": object{"basic": object{}}}).
					Expect().
					Status(http.StatusBadRequest)
			})
		})
	})

	Describe("DELETE", func() {
		Context("Non-existing broker", func() {
			It("returns 404", func() {
				SM.DELETE("/v1/service_brokers/999").
					Expect().
					Status(http.StatusNotFound)
			})
		})

		Context("Existing broker", func() {
			It("succeeds", func() {
				broker := common.MakeBroker("broker1", "http://domain.com/broker", "desc1")
				reply := SM.POST("/v1/service_brokers").
					WithJSON(broker).
					Expect().Status(http.StatusCreated).JSON().Object()
				id := reply.Value("id").String().Raw()

				SM.GET("/v1/service_brokers/" + id).
					Expect().
					Status(http.StatusOK)

				SM.DELETE("/v1/service_brokers/" + id).
					Expect().
					Status(http.StatusOK).JSON().Object().Empty()

				SM.GET("/v1/service_brokers/" + id).
					Expect().
					Status(http.StatusNotFound)
			})
		})
	})
})
