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
package catalog_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/Peripli/service-manager/test/common"
	"github.com/gavv/httpexpect"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

func TestBrokers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Aggregated Catalog API Tests Suite")
}

var _ = Describe("Service Manager Aggregated Catalog API", func() {

	const catalog = `{
  "services": [
    {
      "bindable": true,
      "description": "service",
      "id": "98418a7a-002e-4ff9-b66a-d03fc3d56b16",
      "metadata": {
        "displayName": "test",
        "longDescription": "test"
      },
      "name": "test",
      "plan_updateable": false,
      "plans": [
        {
          "description": "test",
          "free": true,
          "id": "9bb3b29e-bbf9-4900-b926-2f8e9c9a3347",
          "metadata": {
            "bullets": [
              "Plan with basic functionality and relaxed security, excellent for development and try-out purposes"
            ],
            "displayName": "lite"
          },
          "name": "lite"
        }
      ],
      "tags": [
        "test"
      ]
    }
  ]
}`

	var (
		SM *httpexpect.Expect

		serviceManagerServer *httptest.Server
		brokerServer         *ghttp.Server

		testBroker     map[string]interface{}

		catalogResponse []byte
		code            int
	)

	BeforeSuite(func() {
		os.Chdir("../..")

		serviceManagerServer = httptest.NewServer(common.GetServerRouter())
		SM = httpexpect.New(GinkgoT(), serviceManagerServer.URL)
	})

	AfterSuite(func() {
		if serviceManagerServer != nil {
			serviceManagerServer.Close()
		}
	})

	BeforeEach(func() {
		code = http.StatusOK
		catalogResponse = []byte(catalog)
		brokerServer = common.FakeBrokerServer(&code, &catalogResponse)

		testBroker = map[string]interface{}{
			"name":        "name",
			"broker_url":  brokerServer.URL(),
			"description": "description",
			"credentials": map[string]interface{}{
				"basic": map[string]interface{}{
					"username": "buser",
					"password": "bpass",
				},
			},
		}

		common.RemoveAllBrokers(SM)
	})

	Describe("GET", func() {

		Context("when no brokers exist", func() {

			It("returns 200 and empty brokers array", func() {
				obj := SM.GET("/v1/sm_catalog").
						Expect().
						Status(http.StatusOK).
						JSON().Object()

				obj.Keys().ContainsOnly("brokers")
				obj.Value("brokers").Array().Empty()
			})

		})

		Context("when a single broker exists", func() {
			It("returns 200 with a single element broker array and one corresponding service", func() {
				reply := SM.POST("/v1/service_brokers").WithJSON(testBroker).
					Expect().
					JSON().Object()

				id := reply.Value("id").String().Raw()

				obj := SM.GET("/v1/sm_catalog").
					Expect().
					Status(http.StatusOK).
					JSON().Object()

				obj.Keys().ContainsOnly("brokers")

				catalogBrokers := obj.Value("brokers").Array()
				catalogBrokers.Length().Equal(1)

				broker := catalogBrokers.First().Object()
				broker.Value("id").Equal(id)
				brokerCatalog := broker.Value("catalog").Object()
				brokerCatalog.Value("services").Array().Length().Equal(1)
			})
		})

		Context("when multiple brokers exist", func() {
			It("returns 200 with multiple element broker array with a single service each", func() {
				/*
				reply := SM.POST("/v1/service_brokers").WithJSON(testBroker).
					Expect().
					JSON().Object()

				id := reply.Value("id").String().Raw()

				obj := SM.GET("/v1/sm_catalog").
					Expect().
					Status(http.StatusOK).
					JSON().Object()

				obj.Keys().ContainsOnly("brokers")

				catalogBrokers := obj.Value("brokers").Array()
				catalogBrokers.Length().Equal(1)

				broker := catalogBrokers.First().Object()
				broker.Value("id").Equal(id)
				brokerCatalog := broker.Value("catalog").Object()
				brokerCatalog.Value("services").Array().Length().Equal(1)
				*/
			})
		})

	})
})
