package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Peripli/service-manager/api"
	"github.com/Peripli/service-manager/server"
	_ "github.com/Peripli/service-manager/storage/postgres"

	"github.com/gavv/httpexpect"
	. "github.com/onsi/ginkgo"
)

type Object map[string]interface{}
type Array []interface{}

func TestAPI(t *testing.T) {
	RunSpecs(t, "API Tests Suite")
}

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

var _ = Describe("Service Manager API", func() {
	var testServer *httptest.Server
	var sm *httpexpect.Expect

	BeforeSuite(func() {
		srv, err := server.New(api.Default(), server.DefaultConfiguration())
		if err != nil {
			panic(err)
		}
		testServer = httptest.NewServer(srv.Router)
		sm = httpexpect.New(GinkgoT(), testServer.URL)
	})

	AfterSuite(func() {
		if testServer != nil {
			testServer.Close()
		}
	})

	Describe("Service Brokers", func() {
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
				reply.ValueEqual("id", id)
				reply.ValueEqual("name", "broker1")
				reply.ValueEqual("broker_url", "http://domain.com/broker")
				reply.NotContainsKey("credentials")
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

				addPlatform := func(name string, url string, description string) {
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

				addPlatform("broker1", "http://host1.com", "broker one")
				addPlatform("broker2", "http://host2.com", "broker two")
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

				delete(reply.Raw(), "created_at")
				delete(reply.Raw(), "updated_at")

				reply.Equal(broker)
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
				delete(reply.Raw(), "created_at")
				delete(reply.Raw(), "updated_at")
				reply.Equal(broker)

				By("GET returns the same broker")

				reply = sm.GET("/v1/service_brokers/" + id).
					Expect().Status(http.StatusOK).JSON().Object()
				delete(reply.Raw(), "created_at")
				delete(reply.Raw(), "updated_at")
				reply.Equal(broker)
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

	})
})
