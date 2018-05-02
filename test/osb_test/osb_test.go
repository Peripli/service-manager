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
package osb_test

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"github.com/Peripli/service-manager/test/common"
	"github.com/gavv/httpexpect"
	"github.com/gorilla/mux"

	. "github.com/onsi/ginkgo"
)

type object = common.Object
type array = common.Array

func setResponse(rw http.ResponseWriter, status int, message string) {
	rw.WriteHeader(status)
	rw.Write([]byte(message))
}

func createBrokerRouter() *mux.Router {
	router := mux.NewRouter()

	router.HandleFunc("/good/v2/catalog", func(rw http.ResponseWriter, req *http.Request) {
		setResponse(rw, http.StatusOK, `{"services": [{ "name":"sv1" }]}`)
	})

	router.HandleFunc("/good/v2/service_instances/{instance_id}", func(rw http.ResponseWriter, req *http.Request) {
		setResponse(rw, http.StatusCreated, "{}")
	}).Methods("PUT")

	router.HandleFunc("/good/v2/service_instances/{instance_id}", func(rw http.ResponseWriter, req *http.Request) {
		setResponse(rw, http.StatusOK, "{}")
	}).Methods("DELETE")

	router.HandleFunc("/good/v2/service_instances/{instance_id}/service_bindings/{binding_id}", func(rw http.ResponseWriter, req *http.Request) {
		response := fmt.Sprintf(`{"credentials": {"instance_id": "%s" , "binding_id": "%s"}}`, mux.Vars(req)["instance_id"], mux.Vars(req)["binding_id"])
		setResponse(rw, http.StatusCreated, response)
	}).Methods("PUT")

	router.HandleFunc("/good/v2/service_instances/{instance_id}/service_bindings/{binding_id}", func(rw http.ResponseWriter, req *http.Request) {
		setResponse(rw, http.StatusOK, "{}")
	}).Methods("DELETE")

	router.PathPrefix("/bad/").HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		setResponse(rw, http.StatusNotAcceptable, `{"description": "expected error"}`)
	})

	return router
}

func registerBroker(brokerJSON object, SM *httpexpect.Expect) string {
	reply := SM.POST("/v1/service_brokers").
		WithJSON(brokerJSON).
		Expect().Status(http.StatusCreated).JSON().Object()
	return reply.Value("id").String().Raw()
}

func assertBadBrokerError(req *httpexpect.Request) {
	body := req.Expect().Status(http.StatusNotAcceptable).JSON().Object()
	body.ContainsKey("description")
	body.Value("description").Equal("expected error")
}

func assertMissingBrokerError(req *httpexpect.Request) {
	body := req.Expect().Status(http.StatusNotFound).JSON().Object()
	body.ContainsKey("description").
		Value("description").String().
		Equal("Could not find broker with id: missing_broker_id")
}

func getDummyService(idsToRemove ...string) *object {
	result := &object{
		"service_id":        "dummyId",
		"plan_id":           "dummyplanId",
		"organization_guid": "orgguid",
		"space_guid":        "spaceguid",
	}
	for _, id := range idsToRemove {
		delete(*result, id)
	}
	return result
}

type smreq func(path string, pathargs ...interface{}) *httpexpect.Request

func assertRequiredIDError(resp *httpexpect.Object, expectedIDName string) {
	resp.ContainsKey("description").
		Value("description").String().
		Equal(expectedIDName + " is required")
}

func requestWithMissingIDsInBody(req smreq, url string, idName string, expectedIDName string) {
	resp := req(url).WithHeader("X-Broker-API-Version", "2.13").
		WithJSON(getDummyService(idName)).
		Expect().Status(http.StatusBadRequest).JSON().Object()

	assertRequiredIDError(resp, expectedIDName)
}

func requestWithMissingIDsInQuery(req smreq, url string, idName string, expectedIDName string) {
	resp := req(url).WithHeader("X-Broker-API-Version", "2.13").
		WithQueryObject(getDummyService(idName)).
		Expect().Status(http.StatusBadRequest).JSON().Object()

	assertRequiredIDError(resp, expectedIDName)
}

// TestOSB tests for OSB API
func TestOSB(t *testing.T) {
	os.Chdir("../..")
	RunSpecs(t, "OSB API Tests Suite")
}

var _ = Describe("Service Manager OSB API", func() {
	var SM *httpexpect.Expect
	var testServer *httptest.Server

	var listener net.Listener
	var goodBrokerID string
	var badBrokerID string

	BeforeSuite(func() {
		testServer = httptest.NewServer(common.GetServerRouter())
		SM = httpexpect.New(GinkgoT(), testServer.URL)
		common.RemoveAllBrokers(SM)
		listener, _ = net.Listen("tcp", ":0")
		brokerRouter := createBrokerRouter()
		go http.Serve(listener, brokerRouter)

		port := strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)
		goodBrokerJSON := common.MakeBroker("broker1", "http://localhost:"+port+"/good", "")
		badBrokerJSON := common.MakeBroker("broker2", "http://localhost:"+port+"/bad", "")
		goodBrokerID = registerBroker(goodBrokerJSON, SM)
		badBrokerID = registerBroker(badBrokerJSON, SM)
	})

	AfterSuite(func() {
		if testServer != nil {
			common.RemoveAllBrokers(SM)
			testServer.Close()
		}
		if listener != nil {
			listener.Close()
		}
	})

	Describe("Catalog", func() {
		Context("when call to working broker", func() {
			It("should get catalog", func() {
				resp := SM.GET("/v1/osb/"+goodBrokerID+"/v2/catalog").WithHeader("X-Broker-API-Version", "2.13").
					Expect().Status(http.StatusOK).JSON().Object()

				resp.ContainsKey("services")
			})
		})
		Context("when call to broken broker", func() {
			It("should get error", func() {
				assertBadBrokerError(
					SM.GET("/v1/osb/"+badBrokerID+"/v2/catalog").WithHeader("X-Broker-API-Version", "2.13"))
			})
		})
		Context("when call to missing broker", func() {
			It("should get error", func() {
				assertMissingBrokerError(
					SM.GET("/v1/osb/missing_broker_id/v2/catalog").WithHeader("X-Broker-API-Version", "2.13"))
			})
		})
	})

	Describe("Provision", func() {
		Context("when call to working broker", func() {
			It("provisions successfully", func() {
				resp := SM.PUT("/v1/osb/"+goodBrokerID+"/v2/service_instances/12345").WithHeader("X-Broker-API-Version", "2.13").
					WithJSON(getDummyService()).
					Expect().Status(http.StatusCreated).JSON().Object()

				resp.NotEmpty()
			})
		})
		Context("when call to broken broker", func() {
			It("should get error", func() {
				assertBadBrokerError(
					SM.PUT("/v1/osb/"+badBrokerID+"/v2/service_instances/12345").WithHeader("X-Broker-API-Version", "2.13").
						WithJSON(getDummyService()))
			})
		})
		Context("when call to missing broker", func() {
			It("provision fails", func() {
				assertMissingBrokerError(
					SM.PUT("/v1/osb/missing_broker_id/v2/service_instances/12345").WithHeader("X-Broker-API-Version", "2.13").
						WithJSON(getDummyService()))
			})
		})
		Context("when call to working broker with missing mandatory body fields", func() {
			It("provision fails", func() {
				url := "/v1/osb/" + goodBrokerID + "/v2/service_instances/12345"
				requestWithMissingIDsInBody(SM.PUT, url, "service_id", "serviceID")
				requestWithMissingIDsInBody(SM.PUT, url, "plan_id", "planID")
				requestWithMissingIDsInBody(SM.PUT, url, "organization_guid", "organizationGUID")
				requestWithMissingIDsInBody(SM.PUT, url, "space_guid", "spaceGUID")
			})
		})
	})

	Describe("Deprovision", func() {
		Context("when trying to deprovision existing service", func() {
			It("should be successfull", func() {
				SM.DELETE("/v1/osb/"+goodBrokerID+"/v2/service_instances/12345").WithHeader("X-Broker-API-Version", "2.13").
					WithQueryObject(getDummyService()).
					Expect().Status(http.StatusOK).JSON().Object()
			})
		})
		Context("when call to broken broker", func() {
			It("should get error", func() {
				assertBadBrokerError(
					SM.DELETE("/v1/osb/"+badBrokerID+"/v2/service_instances/12345").WithHeader("X-Broker-API-Version", "2.13").
						WithQueryObject(getDummyService()))
			})
		})
		Context("when call to missing broker", func() {
			It("deprovision fails", func() {
				assertMissingBrokerError(
					SM.DELETE("/v1/osb/missing_broker_id/v2/service_instances/12345").WithHeader("X-Broker-API-Version", "2.13").
						WithQueryObject(getDummyService()))
			})
		})
		Context("when call to working broker with missing mandatory body fields", func() {
			It("deprovision fails", func() {
				url := "/v1/osb/" + goodBrokerID + "/v2/service_instances/12345"
				requestWithMissingIDsInQuery(SM.DELETE, url, "service_id", "serviceID")
				requestWithMissingIDsInQuery(SM.DELETE, url, "plan_id", "planID")
			})
		})
	})

	Describe("Bind", func() {
		Context("when broker is working properly", func() {
			It("should be successfull", func() {
				resp := SM.PUT("/v1/osb/"+goodBrokerID+"/v2/service_instances/iid/service_bindings/bid").WithHeader("X-Broker-API-Version", "2.13").
					WithJSON(getDummyService()).
					Expect().Status(http.StatusCreated).JSON().Object()
				credentials := resp.Value("credentials").Object()
				credentials.Value("instance_id").String().Equal("iid")
				credentials.Value("binding_id").String().Equal("bid")
			})
		})
		Context("when broker is not working properly", func() {
			It("should fail", func() {
				assertBadBrokerError(
					SM.PUT("/v1/osb/"+badBrokerID+"/v2/service_instances/iid/service_bindings/bid").WithHeader("X-Broker-API-Version", "2.13").
						WithJSON(getDummyService()))
			})
		})
		Context("when call to missing broker", func() {
			It("bind fails", func() {
				assertMissingBrokerError(SM.PUT("/v1/osb/missing_broker_id/v2/service_instances/iid/service_bindings/bid").WithHeader("X-Broker-API-Version", "2.13").
					WithJSON(getDummyService()))
			})
		})
		Context("when call to working broker with missing mandatory body fields", func() {
			It("bind fails", func() {
				url := "/v1/osb/" + goodBrokerID + "/v2/service_instances/iid/service_bindings/bid"
				requestWithMissingIDsInBody(SM.PUT, url, "service_id", "serviceID")
				requestWithMissingIDsInBody(SM.PUT, url, "plan_id", "planID")
			})
		})
	})

	Describe("Unbind", func() {
		Context("when trying to delete binding", func() {
			It("should be successfull", func() {
				SM.DELETE("/v1/osb/"+goodBrokerID+"/v2/service_instances/iid/service_bindings/bid").WithHeader("X-Broker-API-Version", "2.13").
					WithQueryObject(getDummyService()).
					Expect().Status(http.StatusOK).JSON().Object()

			})
		})
		Context("for brokern broker", func() {
			It("should return error", func() {
				assertBadBrokerError(
					SM.DELETE("/v1/osb/"+badBrokerID+"/v2/service_instances/iid/service_bindings/bid").WithHeader("X-Broker-API-Version", "2.13").
						WithQueryObject(getDummyService()))
			})
		})
		Context("when call to missing broker", func() {
			It("unbind fails", func() {
				assertMissingBrokerError(
					SM.DELETE("/v1/osb/missing_broker_id/v2/service_instances/iid/service_bindings/bid").WithHeader("X-Broker-API-Version", "2.13").
						WithQueryObject(getDummyService()))
			})
		})
		Context("when call to working broker with missing mandatory body fields", func() {
			It("Unbind fails", func() {
				url := "/v1/osb/" + goodBrokerID + "/v2/service_instances/iid/service_bindings/bid"
				requestWithMissingIDsInQuery(SM.DELETE, url, "service_id", "serviceID")
				requestWithMissingIDsInQuery(SM.DELETE, url, "plan_id", "planID")
			})
		})
	})
})
