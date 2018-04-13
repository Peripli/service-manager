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
		resp := sm.GET("/v1/service_brokers").
			Expect().Status(http.StatusOK).JSON().Object()
		for _, val := range resp.Value("brokers").Array().Iter() {
			id := val.Object().Value("id").String().Raw()
			sm.DELETE("/v1/service_brokers/" + id).
				Expect().Status(http.StatusOK)
		}
	})

	Describe("GET", func() {
		It("returns 404 if broker does not exist", func() {
			sm.GET("/v1/service_brokers/999").
				Expect().
				Status(http.StatusNotFound).
				JSON().Object().Keys().Contains("error", "description")
		})

		It("returns the broker with given id", func() {
			broker := makeBroker("broker1", "http://domain.com/broker", "")
			reply := sm.POST("/v1/service_brokers").WithJSON(broker).
				Expect().Status(http.StatusCreated).JSON().Object()
			id := reply.Value("id").String().Raw()

			reply = sm.GET("/v1/service_brokers/" + id).
				Expect().
				Status(http.StatusOK).
				JSON().Object()

			broker["id"] = id
			delete(broker, "credentials")
			mapContains(reply.Raw(), broker)
		})
	})

	Describe("GET All", func() {
		It("returns empty array if no brokers exist", func() {
			sm.GET("/v1/service_brokers").
				Expect().
				Status(http.StatusOK).
				JSON().Object().Value("brokers").Array().Empty()
		})

		It("returns all the brokers", func() {
			brokers := Array{}

			addBroker := func(name string, url string, description string) {
				broker := makeBroker(name, url, description)
				reply := sm.POST("/v1/service_brokers").WithJSON(broker).
					Expect().Status(http.StatusCreated).JSON().Object()
				id := reply.Value("id").String().Raw()
				broker["id"] = id
				delete(broker, "credentials")
				brokers = append(brokers, broker)

				replyArray := sm.GET("/v1/service_brokers").
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
			sm.POST("/v1/service_brokers").
				WithText("text").
				Expect().Status(http.StatusUnsupportedMediaType)
		})

		It("returns 400 if input is not valid JSON", func() {
			sm.POST("/v1/service_brokers").
				WithText("invalid json").
				WithHeader("content-type", "application/json").
				Expect().Status(http.StatusBadRequest)
		})

		It("returns 400 if mandatory field is missing", func() {
			newBroker := func() Object {
				return makeBroker("broker1", "http://domain.com/broker", "")
			}
			sm.POST("/v1/service_brokers").
				WithJSON(newBroker()).
				Expect().Status(http.StatusCreated)

			for _, prop := range []string{"name", "broker_url", "credentials"} {
				broker := newBroker()
				delete(broker, prop)

				sm.POST("/v1/service_brokers").
					WithJSON(broker).
					Expect().Status(http.StatusBadRequest)
			}
		})

		It("succeeds if optional fields are skipped", func() {
			broker := makeBroker("broker1", "http://domain.com/broker", "")
			// delete optional fields
			delete(broker, "description")

			reply := sm.POST("/v1/service_brokers").
				WithJSON(broker).
				Expect().Status(http.StatusCreated).JSON().Object()

			delete(broker, "credentials")
			broker["id"] = reply.Value("id").String().Raw()
			// optional fields returned with default values
			broker["description"] = ""

			mapContains(reply.Raw(), broker)
		})

		It("returns the new broker", func() {
			broker := makeBroker("broker1", "http://domain.com/broker", "broker one")

			By("POST returns the new broker")

			reply := sm.POST("/v1/service_brokers").
				WithJSON(broker).
				Expect().Status(http.StatusCreated).JSON().Object()
			delete(broker, "credentials")

			id := reply.Value("id").String().NotEmpty().Raw()
			broker["id"] = id
			mapContains(reply.Raw(), broker)

			By("GET returns the same broker")

			reply = sm.GET("/v1/service_brokers/" + id).
				Expect().Status(http.StatusOK).JSON().Object()

			mapContains(reply.Raw(), broker)
		})

		It("returns 409 if duplicate name is provided", func() {
			broker := makeBroker("broker1", "http://domain.com/broker", "")

			sm.POST("/v1/service_brokers").
				WithJSON(broker).
				Expect().Status(http.StatusCreated)
			sm.POST("/v1/service_brokers").
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
			reply := sm.POST("/v1/service_brokers").
				WithJSON(broker).
				Expect().Status(http.StatusCreated).JSON().Object()
			delete(broker, "credentials")
			id = reply.Value("id").String().Raw()
			broker["id"] = id
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

			reply := sm.PATCH("/v1/service_brokers/" + id).
				WithJSON(updatedBroker).
				Expect().
				Status(http.StatusOK).JSON().Object()
			delete(updatedBroker, "credentials")

			mapContains(reply.Raw(), updatedBroker)

			By("Update is persisted")

			reply = sm.GET("/v1/service_brokers/" + id).
				Expect().
				Status(http.StatusOK).JSON().Object()

			mapContains(reply.Raw(), updatedBroker)
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

				reply := sm.PATCH("/v1/service_brokers/" + id).
					WithJSON(update).
					Expect().
					Status(http.StatusOK).JSON().Object()

				if prop != "credentials" { // credentials are not returned
					broker[prop] = val
				}
				mapContains(reply.Raw(), broker)

				reply = sm.GET("/v1/service_brokers/" + id).
					Expect().
					Status(http.StatusOK).JSON().Object()

				mapContains(reply.Raw(), broker)
			}
		})

		It("should not update broker id if provided", func() {
			sm.PATCH("/v1/service_brokers/" + id).
				WithJSON(Object{"id": "123"}).
				Expect().
				Status(http.StatusOK)

			sm.GET("/v1/service_brokers/123").
				Expect().
				Status(http.StatusNotFound)
		})
	})

	Describe("DELETE", func() {
		It("returns 404 when trying to delete non-existing broker", func() {
			sm.DELETE("/v1/service_brokers/999").
				Expect().
				Status(http.StatusNotFound)
		})

		It("deletes broker", func() {
			broker := makeBroker("broker1", "http://domain.com/broker", "desc1")
			reply := sm.POST("/v1/service_brokers").
				WithJSON(broker).
				Expect().Status(http.StatusCreated).JSON().Object()
			id := reply.Value("id").String().Raw()

			sm.GET("/v1/service_brokers/" + id).
				Expect().
				Status(http.StatusOK)

			sm.DELETE("/v1/service_brokers/" + id).
				Expect().
				Status(http.StatusOK).JSON().Object().Empty()

			sm.GET("/v1/service_brokers/" + id).
				Expect().
				Status(http.StatusNotFound)
		})
	})
}
