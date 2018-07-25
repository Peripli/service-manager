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
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/Peripli/service-manager/test/common"
	"github.com/gavv/httpexpect"

	. "github.com/onsi/ginkgo"
)

type object = common.Object

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
	resp := req(url).WithHeader("X-Broker-API-Version", "oidc_authn.13").
		WithJSON(getDummyService(idName)).
		Expect().Status(http.StatusBadRequest).JSON().Object()

	assertRequiredIDError(resp, expectedIDName)
}

func requestWithMissingIDsInQuery(req smreq, url string, idName string, expectedIDName string) {
	resp := req(url).WithHeader("X-Broker-API-Version", "oidc_authn.13").
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
	var (
		ctx                        *common.TestContext
		validBroker, failingBroker *common.Broker
	)

	BeforeSuite(func() {
		ctx = common.NewTestContext()
		validBrokerServer := httptest.NewServer(common.NewValidBrokerRouter())
		failingBrokerServer := httptest.NewServer(common.NewFailingBrokerRouter())
		validBroker = ctx.RegisterBroker("broker1", validBrokerServer)
		failingBroker = ctx.RegisterBroker("broker2", failingBrokerServer)
	})

	AfterSuite(func() {
		ctx.Cleanup()
	})

	Describe("Catalog", func() {
		assertGETCatalogReturns200 := func() {
			It("should get catalog", func() {
				resp := ctx.SMWithBasic.GET(validBroker.OSBURL+"/v2/catalog").WithHeader("X-Broker-API-Version", "oidc_authn.13").
					Expect().Status(http.StatusOK).JSON().Object()

				resp.ContainsKey("services")
			})
		}

		Context("when call to working broker", func() {
			assertGETCatalogReturns200()
		})

		Context("when call to broken broker", func() {
			assertGETCatalogReturns200()
		})

		Context("when call to missing broker", func() {
			It("should get error", func() {
				assertMissingBrokerError(
					ctx.SMWithBasic.GET("/v1/osb/missing_broker_id/v2/catalog").WithHeader("X-Broker-API-Version", "oidc_authn.13"))
			})
		})
	})

	Describe("Provision", func() {
		Context("when call to working broker", func() {
			It("provisions successfully", func() {
				resp := ctx.SMWithBasic.PUT(validBroker.OSBURL+"/v2/service_instances/12345").WithHeader("X-Broker-API-Version", "oidc_authn.13").
					WithJSON(getDummyService()).
					Expect().Status(http.StatusCreated).JSON().Object()
				resp.Empty()
			})
		})

		Context("when call to broken broker", func() {
			It("should get error", func() {
				assertBadBrokerError(
					ctx.SMWithBasic.PUT(failingBroker.OSBURL+"/v2/service_instances/12345").WithHeader("X-Broker-API-Version", "oidc_authn.13").
						WithJSON(getDummyService()))
			})
		})

		Context("when call to missing broker", func() {
			It("provision fails", func() {
				assertMissingBrokerError(
					ctx.SMWithBasic.PUT("/v1/osb/missing_broker_id/v2/service_instances/12345").WithHeader("X-Broker-API-Version", "oidc_authn.13").
						WithJSON(getDummyService()))
			})
		})
	})

	Describe("Deprovision", func() {
		Context("when trying to deprovision existing service", func() {
			It("should be successfull", func() {
				ctx.SMWithBasic.DELETE(validBroker.OSBURL+"/v2/service_instances/12345").WithHeader("X-Broker-API-Version", "oidc_authn.13").
					WithQueryObject(getDummyService()).
					Expect().Status(http.StatusOK).JSON().Object()
			})
		})

		Context("when call to broken broker", func() {
			It("should get error", func() {
				assertBadBrokerError(
					ctx.SMWithBasic.DELETE(failingBroker.OSBURL+"/v2/service_instances/12345").WithHeader("X-Broker-API-Version", "oidc_authn.13").
						WithQueryObject(getDummyService()))
			})
		})

		Context("when call to missing broker", func() {
			It("deprovision fails", func() {
				assertMissingBrokerError(
					ctx.SMWithBasic.DELETE("/v1/osb/missing_broker_id/v2/service_instances/12345").WithHeader("X-Broker-API-Version", "oidc_authn.13").
						WithQueryObject(getDummyService()))
			})
		})
	})

	Describe("Bind", func() {
		Context("when broker is working properly", func() {
			It("should be successfull", func() {
				resp := ctx.SMWithBasic.PUT(validBroker.OSBURL+"/v2/service_instances/iid/service_bindings/bid").WithHeader("X-Broker-API-Version", "oidc_authn.13").
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
					ctx.SMWithBasic.PUT(failingBroker.OSBURL+"/v2/service_instances/iid/service_bindings/bid").WithHeader("X-Broker-API-Version", "oidc_authn.13").
						WithJSON(getDummyService()))
			})
		})

		Context("when call to missing broker", func() {
			It("bind fails", func() {
				assertMissingBrokerError(ctx.SMWithBasic.PUT("/v1/osb/missing_broker_id/v2/service_instances/iid/service_bindings/bid").WithHeader("X-Broker-API-Version", "oidc_authn.13").
					WithJSON(getDummyService()))
			})
		})
	})

	Describe("Unbind", func() {
		Context("when trying to delete binding", func() {
			It("should be successfull", func() {
				ctx.SMWithBasic.DELETE(validBroker.OSBURL+"/v2/service_instances/iid/service_bindings/bid").WithHeader("X-Broker-API-Version", "oidc_authn.13").
					WithQueryObject(getDummyService()).
					Expect().Status(http.StatusOK).JSON().Object()

			})
		})

		Context("for brokern broker", func() {
			It("should return error", func() {
				assertBadBrokerError(
					ctx.SMWithBasic.DELETE(failingBroker.OSBURL+"/v2/service_instances/iid/service_bindings/bid").WithHeader("X-Broker-API-Version", "oidc_authn.13").
						WithQueryObject(getDummyService()))
			})
		})

		Context("when call to missing broker", func() {
			It("unbind fails", func() {
				assertMissingBrokerError(
					ctx.SMWithBasic.DELETE("/v1/osb/missing_broker_id/v2/service_instances/iid/service_bindings/bid").WithHeader("X-Broker-API-Version", "oidc_authn.13").
						WithQueryObject(getDummyService()))
			})
		})
	})
})
