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

type object common.Object
type array common.Array

func createBrokerRouter() *mux.Router {
	router := mux.NewRouter()

	router.HandleFunc("/good/v2/catalog", func(rw http.ResponseWriter, req *http.Request) {
		rw.Write([]byte(`{"services": [{ "name":"sv1" }]}`))
	})
	router.HandleFunc("/good/v2/service_instances/{instance_id}", func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusCreated)
		rw.Write([]byte(`{}`))
	}).Methods("PUT")
	router.HandleFunc("/good/v2/service_instances/{instance_id}", func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(`{}`))
	}).Methods("DELETE")
	router.HandleFunc("/good/v2/service_instances/{instance_id}/service_bindings/{binding_id}", func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusCreated)
		rw.Write([]byte(`{}`))
	}).Methods("PUT")
	router.HandleFunc("/good/v2/service_instances/{instance_id}/service_bindings/{binding_id}", func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(`{}`))
	}).Methods("DELETE")

	router.HandleFunc("/bad/v2/catalog", func(rw http.ResponseWriter, req *http.Request) {
		rw.Write([]byte(`{"services": [{ "name":"sv1" }]}`))
	})

	return router
}

func registerBroker(brokerJSON object, SM *httpexpect.Expect) string {
	reply := SM.POST("/v1/service_brokers").
		WithJSON(brokerJSON).
		Expect().Status(http.StatusCreated).JSON().Object()
	return reply.Value("id").String().Raw()
}

// TestOSB tests for OSB API
func TestOSB(t *testing.T) {
	os.Chdir("../..")
	RunSpecs(t, "OSB API Tests Suite")
}

var _ = Describe("Service Manager OSB API", func() {
	dummyService := object{
		"service_id":        "dummyId",
		"plan_id":           "dummyplanId",
		"organization_guid": "orgguid",
		"space_guid":        "spaceguid",
	}
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
		It("should get catalog", func() {
			resp := SM.GET("/v1/osb/"+goodBrokerID+"/v2/catalog").WithHeader("X-Broker-API-Version", "2.13").
				Expect().Status(http.StatusOK).JSON().Object()

			resp.ContainsKey("services")
		})
	})

	Describe("Provision", func() {
		Context("when call to working broker", func() {
			It("provisions successfully", func() {
				resp := SM.PUT("/v1/osb/"+goodBrokerID+"/v2/service_instances/12345").WithHeader("X-Broker-API-Version", "2.13").
					WithJSON(dummyService).
					Expect().Status(http.StatusCreated).JSON().Object()

				resp.NotEmpty()
			})
		})

		Context("when call to missing broker", func() {
			It("provision fails", func() {
				resp := SM.PUT("/v1/osb/missing_broker_id/v2/service_instances/12345").WithHeader("X-Broker-API-Version", "2.13").
					WithJSON(dummyService).
					Expect().Status(http.StatusNotFound).JSON().Object()

				resp.ContainsKey("description").
					Value("description").String().
					Equal("Could not find broker with id: missing_broker_id")
			})
		})
	})

	Describe("Deprovision", func() {
		Context("when trying to deprovision existing service", func() {
			It("should be successfull", func() {
				SM.DELETE("/v1/osb/"+goodBrokerID+"/v2/service_instances/12345").WithHeader("X-Broker-API-Version", "2.13").
					WithQueryObject(dummyService).
					Expect().Status(http.StatusOK).JSON().Object()
			})
		})

		Context("when call to missing broker", func() {
			It("deprovision fails", func() {
				resp := SM.DELETE("/v1/osb/missing_broker_id/v2/service_instances/12345").WithHeader("X-Broker-API-Version", "2.13").
					WithQueryObject(dummyService).
					Expect().Status(http.StatusNotFound).JSON().Object()

				resp.ContainsKey("description").
					Value("description").String().
					Equal("Could not find broker with id: missing_broker_id")
			})
		})
	})
})
