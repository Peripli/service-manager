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
	"os"
	"testing"

	"fmt"

	"github.com/Peripli/service-manager/test/common"
	"github.com/gavv/httpexpect"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

func TestCatalog(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Aggregated Catalog API Tests Suite")
}

var _ = Describe("Service Manager Aggregated Catalog API", func() {
	const brokersLen = 3

	var (
		ctx          *common.TestContext
		brokerServer *ghttp.Server

		testBroker  map[string]interface{}
		testBrokers [brokersLen]map[string]interface{}

		catalogResponse []byte
		code            int
	)

	BeforeSuite(func() {
		os.Chdir("../..")

		ctx = common.NewTestContextFromAPIs()
	})

	AfterSuite(func() {
		ctx.Cleanup()
	})

	BeforeEach(func() {
		code = http.StatusOK
		catalogResponse = []byte(common.Catalog)
		brokerServer = common.FakeBrokerServer(&code, &catalogResponse)
		common.RemoveAllBrokers(ctx.SMWithOAuth)
	})

	Describe("GET", func() {
		Context("when no brokers exist", func() {
			It("returns 200 and empty brokers array", func() {
				obj := ctx.SMWithOAuth.GET("/v1/sm_catalog").
					Expect().
					Status(http.StatusOK).
					JSON().Object()

				obj.Keys().ContainsOnly("brokers")
				obj.Value("brokers").Array().Empty()
			})

		})

		Context("when a single broker exists", func() {
			BeforeEach(func() {
				testBroker = common.MakeBroker("name", brokerServer.URL(), "description")
			})

			It("returns 200 with a single element broker array and one corresponding service", func() {
				reply := ctx.SMWithOAuth.POST("/v1/service_brokers").WithJSON(testBroker).
					Expect().
					JSON().Object()

				id := reply.Value("id").String().Raw()

				obj := ctx.SMWithOAuth.GET("/v1/sm_catalog").
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
			var replies [brokersLen]*httpexpect.Object
			var ids [brokersLen]string

			BeforeEach(func() {
				for i := 0; i < brokersLen; i++ {
					testBrokers[i] = common.MakeBroker(fmt.Sprintf("name%d", i), brokerServer.URL(), "description")
				}
			})

			It("returns 200 with multiple element broker array with a single service each", func() {
				for i := 0; i < brokersLen; i++ {
					replies[i] = ctx.SMWithOAuth.POST("/v1/service_brokers").WithJSON(testBrokers[i]).
						Expect().
						JSON().Object()
					ids[i] = replies[i].Value("id").String().Raw()
				}

				obj := ctx.SMWithOAuth.GET("/v1/sm_catalog").
					Expect().
					Status(http.StatusOK).
					JSON().Object()

				obj.Keys().ContainsOnly("brokers")

				catalogBrokers := obj.Value("brokers").Array()
				catalogBrokers.Length().Equal(brokersLen)

				brokers := catalogBrokers.Iter()
				for i := 0; i < brokersLen; i++ {
					brokerObj := brokers[i].Object()
					Expect(ids).To(ContainElement(brokerObj.Value("id").String().Raw()))

					brokerCatalog := brokerObj.Value("catalog").Object()
					brokerCatalog.Value("services").Array().Length().Equal(1)

				}
			})

			It("returns 200 with single broker array when broker_id query param is provided", func() {
				for i := 0; i < brokersLen; i++ {
					replies[i] = ctx.SMWithOAuth.POST("/v1/service_brokers").WithJSON(testBrokers[i]).
						Expect().
						JSON().Object()
					ids[i] = replies[i].Value("id").String().Raw()
				}

				obj := ctx.SMWithOAuth.GET("/v1/sm_catalog").
					WithQuery("broker_id", ids[0]).
					Expect().
					Status(http.StatusOK).
					JSON().Object()

				obj.Keys().ContainsOnly("brokers")

				catalogBrokers := obj.Value("brokers").Array()
				catalogBrokers.Length().Equal(1)

				broker := catalogBrokers.First().Object()
				broker.Value("id").Equal(ids[0])

				brokerCatalog := broker.Value("catalog").Object()
				brokerCatalog.Value("services").Array().Length().Equal(1)
			})
		})

	})
})
