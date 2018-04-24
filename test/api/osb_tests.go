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
	"net"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	. "github.com/onsi/ginkgo"
)

func brokerRouter() *mux.Router {
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

func registerBroker(brokerJSON Object) string {
	reply := SM.POST("/v1/service_brokers").
		WithJSON(brokerJSON).
		Expect().Status(http.StatusCreated).JSON().Object()
	return reply.Value("id").String().Raw()
}

// func smRequest(url string, method string) {

// }

func testOsb() {
	var listener net.Listener
	var goodBrokerID string
	var badBrokerID string

	BeforeEach(func() {
		listener, _ = net.Listen("tcp", ":0")
		broker := brokerRouter()
		go http.Serve(listener, broker)

		port := strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)
		goodBrokerJSON := makeBroker("broker1", "http://localhost:"+port+"/good", "")
		badBrokerJSON := makeBroker("broker2", "http://localhost:"+port+"/bad", "")
		goodBrokerID = registerBroker(goodBrokerJSON)
		badBrokerID = registerBroker(badBrokerJSON)
	})

	AfterEach(func() {
		removeAllBrokers()
		listener.Close()
	})

	Describe("Catalog", func() {
		It("should get catalog", func() {
			resp := SM.GET("/v1/osb/"+goodBrokerID+"/v2/catalog").WithHeader("X-Broker-API-Version", "2.13").
				Expect().Status(http.StatusOK).JSON().Object()

			resp.ContainsKey("services")
		})
	})

	Describe("Provision", func() {
		Context("call working broker", func() {
			It("provisions successfully", func() {
				instanceID := "12345"
				service := Object{
					"service_id":        "dummyId",
					"plan_id":           "dummyplanId",
					"organization_guid": "orgguid",
					"space_guid":        "spaceguid",
				}
				resp := SM.PUT("/v1/osb/"+goodBrokerID+"/v2/service_instances/"+instanceID).WithHeader("X-Broker-API-Version", "2.13").
					WithJSON(service).
					Expect().Status(http.StatusCreated).JSON().Object()

				resp.NotEmpty().ContainsKey("async")
			})
		})
	})
}
