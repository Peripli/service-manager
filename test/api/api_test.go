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
	"context"

	"net/http/httptest"
	"os"
	"testing"

	"github.com/Peripli/service-manager/env"
	"github.com/Peripli/service-manager/sm"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/gavv/httpexpect"
	. "github.com/onsi/ginkgo"
)

func TestAPI(t *testing.T) {
	os.Chdir("../..")
	RunSpecs(t, "API Tests Suite")
}

func getServerRouter() *mux.Router {
	serverEnv := env.New(&env.ConfigFile{
		Path:   "./test/api",
		Name:   "application",
		Format: "yml",
	}, "SM")
	srv, err := sm.NewServer(context.Background(), serverEnv)
	if err != nil {
		logrus.Fatal("Error creating server: ", err)
	}
	return srv.Router
}

var _ = Describe("Service Manager API", func() {
	var testServer *httptest.Server

	BeforeSuite(func() {
		testServer = httptest.NewServer(getServerRouter())
		SM = httpexpect.New(GinkgoT(), testServer.URL)
	})

	AfterSuite(func() {
		if testServer != nil {
			testServer.Close()
		}
	})

	Describe("Service Brokers", testBrokers)

	Describe("Platforms", testPlatforms)
})
