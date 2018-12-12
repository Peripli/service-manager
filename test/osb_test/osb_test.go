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
	"strings"

	"github.com/gofrs/uuid"

	"github.com/Peripli/service-manager/test/common"
	"github.com/gavv/httpexpect"

	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type object = common.Object
type array = common.Array

// TestOSB tests for OSB API
func TestOSB(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "OSB API Tests Suite")
}

func assertBrokenBrokerError(req *httpexpect.Request) {
	req.Expect().Status(http.StatusNotAcceptable).JSON().Object().
		Value("description").String().Contains("broken service broker error")
}

func assertMissingBrokerError(req *httpexpect.Request) {
	req.Expect().Status(http.StatusNotFound).JSON().Object().
		Value("description").String().Contains("could not find broker")
}

func assertStoppedBrokerError(req *httpexpect.Request) {
	req.Expect().Status(http.StatusBadGateway).JSON().Object().
		Value("description").String().Contains("could not reach service broker")
}

func assertWorkingBrokerResponse(req *httpexpect.Request, expectedStatusCode int, expectedKeys ...string) {
	if expectedKeys == nil {
		expectedKeys = make([]string, 0, 0)
	}
	keys := req.Expect().Status(expectedStatusCode).JSON().Object().Keys()
	for _, key := range expectedKeys {
		keys.Contains(key)
	}
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

func generateRandomQueryParam() (string, string) {
	key, err := uuid.NewV4()
	Expect(err).ToNot(HaveOccurred())
	value, err := uuid.NewV4()
	Expect(err).ToNot(HaveOccurred())
	return key.String(), value.String()
}

func failingHandler(rw http.ResponseWriter, _ *http.Request) {
	common.SetResponse(rw, http.StatusNotAcceptable, object{"description": "broken service broker error", "error": "error"})
}

func queryParameterVerificationHandler(key, value string) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		var status int
		query := request.URL.Query()
		actualValue := query.Get(key)
		Expect(actualValue).To(Equal(value))
		if request.Method == http.MethodPut {
			status = http.StatusCreated
		} else {
			status = http.StatusOK
		}
		common.SetResponse(writer, status, object{})
		defer GinkgoRecover()
	}
}

var _ = Describe("Service Manager OSB API", func() {
	var (
		ctx *common.TestContext

		workingBrokerURL                   string
		failingBrokerURL                   string
		missingBrokerURL                   string
		stoppedBrokerURL                   string
		queryParamVerificationBrokerOSBURL string
		headerKey                          string
		headerValue                        string
	)

	BeforeSuite(func() {
		ctx = common.NewTestContext(nil)
		validBrokerID, validBrokerServer := ctx.RegisterBroker()
		workingBrokerURL = validBrokerServer.URL + "/v1/osb/" + validBrokerID

		failingBrokerID, failingBrokerServer := ctx.RegisterBroker()
		failingBrokerURL = failingBrokerServer.URL + "/v1/osb/" + failingBrokerID
		failingBrokerServer.ServiceInstanceHandler = failingHandler
		failingBrokerServer.BindingHandler = failingHandler
		failingBrokerServer.CatalogHandler = failingHandler
		failingBrokerServer.ServiceInstanceLastOpHandler = failingHandler
		failingBrokerServer.BindingLastOpHandler = failingHandler
		failingBrokerServer.BindingAdaptCredentialsHandler = failingHandler

		UUID, err := uuid.NewV4()
		if err != nil {
			Expect(err).ToNot(HaveOccurred())
		}
		missingBrokerURL = "http://localhost:32123/v1/osb/" + UUID.String()

		stoppedBrokerID, stoppedBrokerServer := ctx.RegisterBroker()
		stoppedBrokerServer.Close()

		stoppedBrokerURL = stoppedBrokerServer.URL + "/v1/osb/" + stoppedBrokerID

		headerKey, headerValue = generateRandomQueryParam()
		queryParameterVerificationServerID, queryParameterVerificationServer := ctx.RegisterBroker()
		queryParameterVerificationServer.ServiceInstanceHandler = queryParameterVerificationHandler(headerKey, headerValue)
		queryParameterVerificationServer.BindingHandler = queryParameterVerificationHandler(headerKey, headerValue)
		queryParameterVerificationServer.CatalogHandler = queryParameterVerificationHandler(headerKey, headerValue)
		queryParameterVerificationServer.ServiceInstanceLastOpHandler = queryParameterVerificationHandler(headerKey, headerValue)
		queryParameterVerificationServer.BindingLastOpHandler = queryParameterVerificationHandler(headerKey, headerValue)
		queryParamVerificationBrokerOSBURL = queryParameterVerificationServer.URL + "/v1/osb/" + queryParameterVerificationServerID

	})

	AfterSuite(func() {
		ctx.Cleanup()
	})

	Describe("Catalog", func() {
		Context("when call to working service broker", func() {
			It("should succeed", func() {
				assertWorkingBrokerResponse(
					ctx.SMWithBasic.GET(workingBrokerURL+"/v2/catalog").WithHeader("X-Broker-API-Version", "oidc_authn.13"),
					http.StatusOK, "services")

			})
		})

		Context("when call to broken service broker", func() {
			It("should fail", func() {
				assertBrokenBrokerError(
					ctx.SMWithBasic.GET(failingBrokerURL+"/v2/catalog").WithHeader("X-Broker-API-Version", "oidc_authn.13"))

			})
		})

		Context("when call to missing service broker", func() {
			It("should fail", func() {
				assertMissingBrokerError(
					ctx.SMWithBasic.GET(missingBrokerURL+"/v2/catalog").WithHeader("X-Broker-API-Version", "oidc_authn.13"))
			})
		})

		Context("when call to stopped service broker", func() {
			It("should fail", func() {
				assertStoppedBrokerError(
					ctx.SMWithBasic.GET(stoppedBrokerURL+"/v2/catalog").WithHeader("X-Broker-API-Version", "oidc_authn.13"))
			})
		})

		Context("when call contains query params", func() {
			It("propagates them to the service broker", func() {
				assertWorkingBrokerResponse(
					ctx.SMWithBasic.GET(queryParamVerificationBrokerOSBURL+"/v2/catalog").WithHeader("X-Broker-API-Version", "oidc_authn.13").
						WithJSON(getDummyService()).WithQuery(headerKey, headerValue), http.StatusOK)
			})
		})
	})

	Describe("Provision", func() {
		Context("call to working service broker", func() {
			It("should succeed", func() {
				assertWorkingBrokerResponse(
					ctx.SMWithBasic.PUT(workingBrokerURL+"/v2/service_instances/12345").WithHeader("X-Broker-API-Version", "oidc_authn.13").
						WithJSON(getDummyService()), http.StatusCreated)
			})
		})

		Context("when call to broken service broker", func() {
			It("should fail", func() {
				assertBrokenBrokerError(
					ctx.SMWithBasic.PUT(failingBrokerURL+"/v2/service_instances/12345").WithHeader("X-Broker-API-Version", "oidc_authn.13").
						WithJSON(getDummyService()))
			})
		})

		Context("when call to missing broker", func() {
			It("should fail", func() {
				assertMissingBrokerError(
					ctx.SMWithBasic.PUT(missingBrokerURL+"/v2/service_instances/12345").WithHeader("X-Broker-API-Version", "oidc_authn.13").
						WithJSON(getDummyService()))
			})
		})

		Context("when call to stopped service broker", func() {
			It("should fail", func() {
				assertStoppedBrokerError(
					ctx.SMWithBasic.PUT(stoppedBrokerURL+"/v2/service_instances/12345").WithHeader("X-Broker-API-Version", "oidc_authn.13").
						WithJSON(getDummyService()))
			})
		})

		Context("when call contains query params", func() {
			It("propagates them to the service broker", func() {
				assertWorkingBrokerResponse(
					ctx.SMWithBasic.PUT(queryParamVerificationBrokerOSBURL+"/v2/service_instances/12345").WithHeader("X-Broker-API-Version", "oidc_authn.13").
						WithJSON(getDummyService()).WithQuery(headerKey, headerValue), http.StatusCreated)
			})
		})
	})
	Describe("Deprovision", func() {
		Context("when trying to deprovision existing service", func() {
			It("should be successfull", func() {
				ctx.SMWithBasic.DELETE(workingBrokerURL+"/v2/service_instances/12345").WithHeader("X-Broker-API-Version", "oidc_authn.13").
					WithQueryObject(getDummyService()).
					Expect().Status(http.StatusOK).JSON().Object()
			})
		})

		Context("when call to broken broker", func() {
			It("should fail", func() {
				assertBrokenBrokerError(
					ctx.SMWithBasic.DELETE(failingBrokerURL+"/v2/service_instances/12345").WithHeader("X-Broker-API-Version", "oidc_authn.13").
						WithQueryObject(getDummyService()))
			})
		})

		Context("when call to missing service broker", func() {
			It("should fail", func() {
				assertMissingBrokerError(
					ctx.SMWithBasic.DELETE(missingBrokerURL+"/v2/service_instances/12345").WithHeader("X-Broker-API-Version", "oidc_authn.13").
						WithQueryObject(getDummyService()))
			})
		})

		Context("when call to stopped service broker", func() {
			It("should fail", func() {
				assertStoppedBrokerError(ctx.SMWithBasic.DELETE(stoppedBrokerURL+"/v2/service_instances/12345").WithHeader("X-Broker-API-Version", "oidc_authn.13").
					WithQueryObject(getDummyService()))
			})
		})

		Context("when call contains query params", func() {
			It("propagates them to the service broker", func() {
				assertWorkingBrokerResponse(
					ctx.SMWithBasic.DELETE(queryParamVerificationBrokerOSBURL+"/v2/service_instances/12345").WithHeader("X-Broker-API-Version", "oidc_authn.13").
						WithJSON(getDummyService()).WithQuery(headerKey, headerValue), http.StatusOK)
			})
		})
	})

	Describe("Bind", func() {
		Context("call to working service broker", func() {
			It("should succeed", func() {
				assertWorkingBrokerResponse(
					ctx.SMWithBasic.PUT(workingBrokerURL+"/v2/service_instances/iid/service_bindings/bid").WithHeader("X-Broker-API-Version", "oidc_authn.13").
						WithJSON(getDummyService()), http.StatusCreated, "credentials")
			})
		})

		Context("when call to broker service broker", func() {
			It("should fail", func() {
				assertBrokenBrokerError(
					ctx.SMWithBasic.PUT(failingBrokerURL+"/v2/service_instances/iid/service_bindings/bid").WithHeader("X-Broker-API-Version", "oidc_authn.13").
						WithJSON(getDummyService()))
			})
		})

		Context("when call to missing service broker", func() {
			It("should fail", func() {
				assertMissingBrokerError(ctx.SMWithBasic.PUT(missingBrokerURL+"/v2/service_instances/iid/service_bindings/bid").WithHeader("X-Broker-API-Version", "oidc_authn.13").
					WithJSON(getDummyService()))
			})
		})

		Context("when call to stopped service broker", func() {
			It("should fail", func() {
				assertStoppedBrokerError(ctx.SMWithBasic.PUT(stoppedBrokerURL+"/v2/service_instances/iid/service_bindings/bid").WithHeader("X-Broker-API-Version", "oidc_authn.13").
					WithJSON(getDummyService()))
			})
		})

		Context("when call contains query params", func() {
			It("propagates them to the service broker", func() {
				assertWorkingBrokerResponse(
					ctx.SMWithBasic.PUT(queryParamVerificationBrokerOSBURL+"/v2/service_instances/iid/service_bindings/bid").WithHeader("X-Broker-API-Version", "oidc_authn.13").
						WithJSON(getDummyService()).WithQuery(headerKey, headerValue), http.StatusCreated)
			})
		})
	})

	Describe("Unbind", func() {
		Context("when trying to delete binding", func() {
			It("should be successfull", func() {
				ctx.SMWithBasic.DELETE(workingBrokerURL+"/v2/service_instances/iid/service_bindings/bid").WithHeader("X-Broker-API-Version", "oidc_authn.13").
					WithQueryObject(getDummyService()).
					Expect().Status(http.StatusOK).JSON().Object()

			})
		})

		Context("when call to broken service broker", func() {
			It("should return error", func() {
				assertBrokenBrokerError(
					ctx.SMWithBasic.DELETE(failingBrokerURL+"/v2/service_instances/iid/service_bindings/bid").WithHeader("X-Broker-API-Version", "oidc_authn.13").
						WithQueryObject(getDummyService()))
			})
		})

		Context("when call to missing broker", func() {
			It("unbind fails", func() {
				assertMissingBrokerError(
					ctx.SMWithBasic.DELETE(missingBrokerURL+"/v2/service_instances/iid/service_bindings/bid").WithHeader("X-Broker-API-Version", "oidc_authn.13").
						WithQueryObject(getDummyService()))
			})
		})

		Context("when call to stopped service broker", func() {
			It("should fail", func() {
				assertStoppedBrokerError(
					ctx.SMWithBasic.DELETE(stoppedBrokerURL+"/v2/service_instances/iid/service_bindings/bid").WithHeader("X-Broker-API-Version", "oidc_authn.13").
						WithQueryObject(getDummyService()))

			})
		})

		Context("when call contains query params", func() {
			It("propagates them to the service broker", func() {
				assertWorkingBrokerResponse(
					ctx.SMWithBasic.DELETE(queryParamVerificationBrokerOSBURL+"/v2/service_instances/iid/service_bindings/bid").WithHeader("X-Broker-API-Version", "oidc_authn.13").
						WithJSON(getDummyService()).WithQuery(headerKey, headerValue), http.StatusOK)
			})
		})
	})

	Describe("Get Service Instance Last Operation", func() {
		Context("when call to working service broker", func() {
			It("should succeed", func() {
				assertWorkingBrokerResponse(
					ctx.SMWithBasic.GET(workingBrokerURL+"/v2/service_instances/iid/last_operation").WithHeader("X-Broker-API-Version", "oidc_authn.13"),
					http.StatusOK, "state")
			})
		})

		Context("when call to broken service broker", func() {
			It("should fail", func() {
				assertBrokenBrokerError(
					ctx.SMWithBasic.GET(failingBrokerURL+"/v2/service_instances/iid/last_operation").WithHeader("X-Broker-API-Version", "oidc_authn.13"))

			})
		})

		Context("when call to missing service broker", func() {
			It("should fail", func() {
				assertMissingBrokerError(
					ctx.SMWithBasic.GET(missingBrokerURL+"/v2/service_instances/iid/last_operation").WithHeader("X-Broker-API-Version", "oidc_authn.13"))
			})
		})

		Context("when call to stopped service broker", func() {
			It("should fail", func() {
				assertStoppedBrokerError(
					ctx.SMWithBasic.GET(stoppedBrokerURL+"/v2/service_instances/iid/last_operation").WithHeader("X-Broker-API-Version", "oidc_authn.13"))
			})
		})

		Context("when call contains query params", func() {
			It("propagates them to the service broker", func() {
				assertWorkingBrokerResponse(
					ctx.SMWithBasic.GET(queryParamVerificationBrokerOSBURL+"/v2/service_instances/iid/last_operation").WithHeader("X-Broker-API-Version", "oidc_authn.13").
						WithJSON(getDummyService()).WithQuery(headerKey, headerValue), http.StatusOK)
			})
		})
	})

	Describe("Get Binding Last Operation", func() {
		Context("when call to working service broker", func() {
			It("should succeed", func() {
				assertWorkingBrokerResponse(
					ctx.SMWithBasic.GET(workingBrokerURL+"/v2/service_instances/iid/service_bindings/bid/last_operation").WithHeader("X-Broker-API-Version", "oidc_authn.13"),
					http.StatusOK, "state")
			})
		})

		Context("when call to broken service broker", func() {
			It("should fail", func() {
				assertBrokenBrokerError(
					ctx.SMWithBasic.GET(failingBrokerURL+"/v2/service_instances/iid/service_bindings/bid/last_operation").WithHeader("X-Broker-API-Version", "oidc_authn.13"))

			})
		})

		Context("when call to missing service broker", func() {
			It("should fail", func() {
				assertMissingBrokerError(
					ctx.SMWithBasic.GET(missingBrokerURL+"/v2/service_instances/iid/service_bindings/bid/last_operation").WithHeader("X-Broker-API-Version", "oidc_authn.13"))
			})
		})

		Context("when call to stopped service broker", func() {
			It("should fail", func() {
				assertStoppedBrokerError(
					ctx.SMWithBasic.GET(stoppedBrokerURL+"/v2/service_instances/iid/service_bindings/bid/last_operation").WithHeader("X-Broker-API-Version", "oidc_authn.13"))
			})
		})

		Context("when call contains query params", func() {
			It("propagates them to the service broker", func() {
				assertWorkingBrokerResponse(
					ctx.SMWithBasic.GET(queryParamVerificationBrokerOSBURL+"/v2/service_instances/iid/service_bindings/bid/last_operation").WithHeader("X-Broker-API-Version", "oidc_authn.13").
						WithJSON(getDummyService()).WithQuery(headerKey, headerValue), http.StatusOK)
			})
		})
	})

	Describe("Post Binding Adapt Credentials", func() {
		Context("when call to working service broker", func() {
			It("should succeed", func() {
				assertWorkingBrokerResponse(
					ctx.SMWithBasic.POST(workingBrokerURL+"/v2/service_instances/iid/service_bindings/bid/adapt_credentials").WithHeader("X-Broker-API-Version", "oidc_authn.13").WithJSON(&object{}),
					http.StatusOK, "credentials")
			})
		})

		Context("when call to broken service broker", func() {
			It("should fail", func() {
				assertBrokenBrokerError(
					ctx.SMWithBasic.POST(failingBrokerURL+"/v2/service_instances/iid/service_bindings/bid/adapt_credentials").WithHeader("X-Broker-API-Version", "oidc_authn.13").WithJSON(&object{}))

			})
		})

		Context("when call to missing service broker", func() {
			It("should fail", func() {
				assertMissingBrokerError(
					ctx.SMWithBasic.POST(missingBrokerURL+"/v2/service_instances/iid/service_bindings/bid/adapt_credentials").WithHeader("X-Broker-API-Version", "oidc_authn.13").WithJSON(&object{}))

			})
		})

		Context("when call to stopped service broker", func() {
			It("should fail", func() {
				assertStoppedBrokerError(
					ctx.SMWithBasic.POST(stoppedBrokerURL+"/v2/service_instances/iid/service_bindings/bid/adapt_credentials").WithHeader("X-Broker-API-Version", "oidc_authn.13").WithJSON(&object{}))

			})
		})

		Context("when call contains query params", func() {
			It("propagates them to the service broker", func() {
				assertWorkingBrokerResponse(
					ctx.SMWithBasic.GET(queryParamVerificationBrokerOSBURL+"/v2/service_instances/iid/service_bindings/bid/last_operation").WithHeader("X-Broker-API-Version", "oidc_authn.13").
						WithJSON(getDummyService()).WithQuery(headerKey, headerValue), http.StatusOK)
			})
		})

	})

	Describe("Prefixed broker path", func() {
		Context("when call to working broker", func() {

			const brokerPathPrefix = "/broker_prefix"
			var (
				prefixedBrokerServer *httptest.Server
				osbURL               string
				prefixedBrokerID     string
			)

			BeforeEach(func() {
				brokerHandler := &prefixedBrokerHandler{brokerPathPrefix}
				prefixedBrokerServer = httptest.NewServer(brokerHandler)
				brokerURL := prefixedBrokerServer.URL + brokerPathPrefix

				brokerJSON := object{
					"name":        "prefixed_broker",
					"broker_url":  brokerURL,
					"description": "",
					"credentials": object{
						"basic": object{
							"username": "buser",
							"password": "bpass",
						},
					},
				}
				prefixedBrokerID = common.RegisterBrokerInSM(brokerJSON, ctx.SMWithOAuth)
				osbURL = "/v1/osb/" + prefixedBrokerID
			})

			AfterEach(func() {
				ctx.CleanupBroker(prefixedBrokerID)
				if prefixedBrokerServer != nil {
					prefixedBrokerServer.Close()
				}
			})

			It("should get catalog", func() {
				ctx.SMWithBasic.GET(osbURL + "/v2/catalog").
					Expect().Status(http.StatusOK).JSON().Object().ContainsKey("services")
			})
		})
	})

})

type prefixedBrokerHandler struct {
	brokerPathPrefix string
}

func (pbh *prefixedBrokerHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if strings.HasPrefix(req.URL.Path, pbh.brokerPathPrefix) {
		common.SetResponse(w, http.StatusOK, object{"services": array{}})
	} else {
		common.SetResponse(w, http.StatusNotFound, object{})
	}
}
