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
package api_itest

import (
	"net/http/httptest"
	"testing"

	"github.com/Peripli/service-manager/api"
	"github.com/Peripli/service-manager/server"
	_ "github.com/Peripli/service-manager/storage/postgres"

	"github.com/gavv/httpexpect"
	. "github.com/onsi/ginkgo"
)

var sm *httpexpect.Expect

func TestAPI(t *testing.T) {
	RunSpecs(t, "API Tests Suite")
}

var _ = Describe("Service Manager API", func() {
	var testServer *httptest.Server

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

	Describe("Service Brokers", testBrokers)

	Describe("Platforms", testPlatforms)
})
