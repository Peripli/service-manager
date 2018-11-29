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

package service_test

import (
	"net/http"
	"testing"

	"github.com/Peripli/service-manager/test/common"

	. "github.com/onsi/ginkgo"

	. "github.com/onsi/gomega"
)

func TestServiceOfferings(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Offerings Tests Suite")
}

var _ = Describe("Service Manager Service Offerings API", func() {
	var (
		ctx              *common.TestContext
		brokerServer     *common.BrokerServer
		brokerServerJSON common.Object
	)

	BeforeSuite(func() {
		brokerServer = common.NewBrokerServer()
		ctx = common.NewTestContext(nil)
	})

	AfterSuite(func() {
		ctx.Cleanup()
		if brokerServer != nil {
			brokerServer.Close()
		}
	})

	BeforeEach(func() {
		brokerServer.Reset()
		brokerServerJSON = common.Object{
			"name":        "brokerName",
			"broker_url":  brokerServer.URL,
			"description": "description",
			"credentials": common.Object{
				"basic": common.Object{
					"username": brokerServer.Username,
					"password": brokerServer.Password,
				},
			},
		}
		common.RemoveAllBrokers(ctx.SMWithOAuth)
	})

	Describe("GET", func() {
		Context("when the service offering does not exist", func() {
			It("returns 404", func() {
				ctx.SMWithOAuth.GET("/v1/service_offerings/12345").
					Expect().
					Status(http.StatusNotFound).
					JSON().Object().
					Keys().Contains("error", "description")
			})
		})

		Context("when the service offering exists", func() {
			BeforeEach(func() {
				ctx.SMWithOAuth.POST("/v1/service_brokers").WithJSON(brokerServerJSON).
					Expect().
					Status(http.StatusCreated)

				brokerServer.ResetCallHistory()
			})

			It("returns the service offering with the given id", func() {
				id := ctx.SMWithOAuth.GET("/v1/service_offerings").
					Expect().
					Status(http.StatusOK).
					JSON().Object().Value("service_offerings").Array().First().Object().
					Value("id").String().Raw()

				Expect(id).ToNot(BeEmpty())

				ctx.SMWithOAuth.GET("/v1/service_offerings/"+id).
					Expect().
					Status(http.StatusOK).
					JSON().Object().
					Keys().Contains("id", "description")
			})
		})
	})

	Describe("List", func() {
		Context("when no service offerings exist", func() {
			It("returns an empty array", func() {
				ctx.SMWithOAuth.GET("/v1/service_offerings").
					Expect().
					Status(http.StatusOK).
					JSON().Object().Value("service_offerings").Array().
					Empty()
			})
		})

		Context("when a broker is registered", func() {
			BeforeEach(func() {
				ctx.SMWithOAuth.POST("/v1/service_brokers").WithJSON(brokerServerJSON).
					Expect().
					Status(http.StatusCreated)

				brokerServer.ResetCallHistory()
			})

			Context("when catalog_name parameter is not provided", func() {
				It("returns the broker's service offerings as part of the response", func() {
					ctx.SMWithOAuth.GET("/v1/service_offerings").
						Expect().
						Status(http.StatusOK).
						JSON().Object().Value("service_offerings").Array().Length().Equal(1)
				})
			})

			Context("when catalog_name query parameter is provided", func() {
				It("returns the service offering with the specified catalog_name", func() {
					serviceCatalogName := common.DefaultCatalog()["services"].([]interface{})[0].(map[string]interface{})["name"]
					Expect(serviceCatalogName).ToNot(BeEmpty())

					ctx.SMWithOAuth.GET("/v1/service_offerings").WithQuery("catalog_name", serviceCatalogName).
						Expect().
						Status(http.StatusOK).
						JSON().Object().Value("service_offerings").Array().Length().Equal(1)
				})
			})
		})
	})
})
