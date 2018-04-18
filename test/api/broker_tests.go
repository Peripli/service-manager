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
package api

import (
	"net/http"

	. "github.com/onsi/ginkgo"
)

func makeBroker(name string, url string, description string) Object {
	return Object{
		"name":        name,
		"broker_url":  url,
		"description": description,
		"credentials": Object{
			"basic": Object{
				"username": "buser",
				"password": "bpass",
			},
		},
	}
}

func testBrokers() {
	BeforeEach(func() {
		By("remove all service brokers")
		resp := SM.GET("/v1/service_brokers").
			Expect().Status(http.StatusOK).JSON().Object()
		for _, val := range resp.Value("brokers").Array().Iter() {
			id := val.Object().Value("id").String().Raw()
			SM.DELETE("/v1/service_brokers/" + id).
				Expect().Status(http.StatusOK)
		}
	})

	Describe("GET", func() {
		It("returns 404 if broker does not exist", func() {
			SM.GET("/v1/service_brokers/999").
				Expect().
				Status(http.StatusNotFound).
				JSON().Object().Keys().Contains("error", "description")
		})

		It("returns the broker with given id", func() {
			broker := makeBroker("broker1", "http://domain.com/broker", "")
			reply := SM.POST("/v1/service_brokers").WithJSON(broker).
				Expect().Status(http.StatusCreated).JSON().Object()
			id := reply.Value("id").String().Raw()

			reply = SM.GET("/v1/service_brokers/" + id).
				Expect().
				Status(http.StatusOK).
				JSON().Object()

			broker["id"] = id
			delete(broker, "credentials")
			MapContains(reply.Raw(), broker)
		})
	})

	Describe("GET All", func() {
		It("returns empty array if no brokers exist", func() {
			SM.GET("/v1/service_brokers").
				Expect().
				Status(http.StatusOK).
				JSON().Object().Value("brokers").Array().Empty()
		})

		It("returns all the brokers", func() {
			brokers := Array{}

			addBroker := func(name string, url string, description string) {
				broker := makeBroker(name, url, description)
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

	Describe("POST", func() {
		It("returns 415 if input is not valid JSON", func() {
			SM.POST("/v1/service_brokers").
				WithText("text").
				Expect().Status(http.StatusUnsupportedMediaType)
		})

		It("returns 400 if input is not valid JSON", func() {
			SM.POST("/v1/service_brokers").
				WithText("invalid json").
				WithHeader("content-type", "application/json").
				Expect().Status(http.StatusBadRequest)
		})

		It("returns 400 if mandatory field is missing", func() {
			newBroker := func() Object {
				return makeBroker("broker1", "http://domain.com/broker", "")
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

		It("succeeds if optional fields are skipped", func() {
			broker := makeBroker("broker1", "http://domain.com/broker", "")
			// delete optional fields
			delete(broker, "description")

			reply := SM.POST("/v1/service_brokers").
				WithJSON(broker).
				Expect().Status(http.StatusCreated).JSON().Object()

			delete(broker, "credentials")
			broker["id"] = reply.Value("id").String().Raw()
			// optional fields returned with default values
			broker["description"] = ""

			MapContains(reply.Raw(), broker)
		})

		It("returns the new broker", func() {
			broker := makeBroker("broker1", "http://domain.com/broker", "broker one")

			By("POST returns the new broker")

			reply := SM.POST("/v1/service_brokers").
				WithJSON(broker).
				Expect().Status(http.StatusCreated).JSON().Object()
			delete(broker, "credentials")

			id := reply.Value("id").String().NotEmpty().Raw()
			broker["id"] = id
			MapContains(reply.Raw(), broker)

			By("GET returns the same broker")

			reply = SM.GET("/v1/service_brokers/" + id).
				Expect().Status(http.StatusOK).JSON().Object()

			MapContains(reply.Raw(), broker)
		})

		It("returns 409 if duplicate name is provided", func() {
			broker := makeBroker("broker1", "http://domain.com/broker", "")

			SM.POST("/v1/service_brokers").
				WithJSON(broker).
				Expect().Status(http.StatusCreated)
			SM.POST("/v1/service_brokers").
				WithJSON(broker).
				Expect().Status(http.StatusConflict)
		})
	})

	Describe("PATCH", func() {
		var broker Object
		var id string

		BeforeEach(func() {
			By("Create new broker")
			broker = makeBroker("broker1", "http://domain.com/broker", "desc1")
			reply := SM.POST("/v1/service_brokers").
				WithJSON(broker).
				Expect().Status(http.StatusCreated).JSON().Object()
			delete(broker, "credentials")
			id = reply.Value("id").String().Raw()
			broker["id"] = id
		})

		It("returns 415 if input is not valid JSON", func() {
			SM.PATCH("/v1/service_brokers/" + id).
				WithText("text").
				Expect().Status(http.StatusUnsupportedMediaType)
		})

		It("returns 404 if broker is missing", func() {
			SM.PATCH("/v1/service_brokers/invalid_id").
				WithJSON(broker).
				Expect().Status(http.StatusNotFound)
		})

		It("returns 400 if input is not valid JSON", func() {
			SM.PATCH("/v1/service_brokers/"+id).
				WithText("invalid json").
				WithHeader("content-type", "application/json").
				Expect().Status(http.StatusBadRequest)
		})

		It("returns 409 if duplicate name is provided", func() {
			broker2 := makeBroker("broker2", "http://domain.com/broker2", "")

			SM.POST("/v1/service_brokers").
				WithJSON(broker2).
				Expect().Status(http.StatusCreated)
			SM.PATCH("/v1/service_brokers/" + id).
				WithJSON(broker2).
				Expect().Status(http.StatusConflict)
		})

		It("returns 200 if all properties are updated", func() {
			By("Update broker")

			updatedBroker := makeBroker("broker2", "http://domain.com/broker2", "desc2")
			updatedBroker["credentials"] = Object{
				"basic": Object{
					"username": "auser",
					"password": "apass",
				},
			}

			reply := SM.PATCH("/v1/service_brokers/" + id).
				WithJSON(updatedBroker).
				Expect().
				Status(http.StatusOK).JSON().Object()
			delete(updatedBroker, "credentials")

			MapContains(reply.Raw(), updatedBroker)

			By("Update is persisted")

			reply = SM.GET("/v1/service_brokers/" + id).
				Expect().
				Status(http.StatusOK).JSON().Object()

			MapContains(reply.Raw(), updatedBroker)
		})

		It("can update each property separately", func() {
			newBroker := makeBroker("bb8", "http://lucas.arts", "a robot")
			newBroker["credentials"] = Object{
				"basic": Object{
					"username": "auser",
					"password": "apass",
				},
			}

			for prop, val := range newBroker {
				update := Object{}
				update[prop] = val

				reply := SM.PATCH("/v1/service_brokers/" + id).
					WithJSON(update).
					Expect().
					Status(http.StatusOK).JSON().Object()

				if prop != "credentials" { // credentials are not returned
					broker[prop] = val
				}
				MapContains(reply.Raw(), broker)

				reply = SM.GET("/v1/service_brokers/" + id).
					Expect().
					Status(http.StatusOK).JSON().Object()

				MapContains(reply.Raw(), broker)
			}
		})

		It("should not update broker id if provided", func() {
			SM.PATCH("/v1/service_brokers/" + id).
				WithJSON(Object{"id": "123"}).
				Expect().
				Status(http.StatusOK)

			SM.GET("/v1/service_brokers/123").
				Expect().
				Status(http.StatusNotFound)
		})
	})

	Describe("DELETE", func() {
		It("returns 404 when trying to delete non-existing broker", func() {
			SM.DELETE("/v1/service_brokers/999").
				Expect().
				Status(http.StatusNotFound)
		})

		It("deletes broker", func() {
			broker := makeBroker("broker1", "http://domain.com/broker", "desc1")
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
}
