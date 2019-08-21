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
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/spf13/pflag"

	"github.com/gofrs/uuid"

	"github.com/Peripli/service-manager/test/common"
	"github.com/gavv/httpexpect"

	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const simpleCatalog = `
{
  "services": [{
		"name": "no-tags-no-metadata",
		"id": "acb56d7c-XXXX-XXXX-XXXX-feb140a59a67",
		"description": "A fake service.",
		"dashboard_client": {
			"id": "id",
			"secret": "secret",
			"redirect_uri": "redirect_uri"		
		},    
		"plans": [{
			"random_extension": "random_extension",
			"name": "fake-plan-1",
			"id": "d3031751-XXXX-XXXX-XXXX-a42377d33202",
			"description": "Shared fake Server, 5tb persistent disk, 40 max concurrent connections.",
			"free": false
		}]
	}]
}
`

// TestOSB tests for OSB API
func TestOSB(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "OSB API Tests Suite")
}

func assertFailingBrokerError(req *httpexpect.Response) {
	req.Status(http.StatusNotAcceptable).JSON().Object().
		Value("description").String().Match("Service broker .* failed with: .*Failing service broker error")
}

func assertMissingBrokerError(req *httpexpect.Response) {
	req.Status(http.StatusNotFound).JSON().Object().
		Value("description").String().Contains("could not find such broker")
}

func assertUnresponsiveBrokerError(req *httpexpect.Response) {
	req.Status(http.StatusBadGateway).JSON().Object().
		Value("description").String().Contains("could not reach service broker")
}

func assertWorkingBrokerResponse(req *httpexpect.Response, expectedStatusCode int, expectedKeys ...string) {
	if expectedKeys == nil {
		expectedKeys = make([]string, 0, 0)
	}
	keys := req.Status(expectedStatusCode).JSON().Object().Keys()
	for _, key := range expectedKeys {
		keys.Contains(key)
	}
}

func getDummyService(idsToRemove ...string) *common.Object {
	result := &common.Object{
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
	common.SetResponse(rw, http.StatusNotAcceptable, common.Object{"description": "Failing service broker error", "error": "error"})
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
		common.SetResponse(writer, status, common.Object{})
		defer GinkgoRecover()
	}
}

var _ = Describe("Service Manager OSB API", func() {
	const (
		timeoutDuration             = time.Millisecond * 500
		additionalDelayAfterTimeout = time.Second
	)

	var (
		ctx *common.TestContext

		validBrokerServer    *common.BrokerServer
		validBrokerID        string
		smUrlToWorkingBroker string

		brokerServerWithEmptyCatalog *common.BrokerServer
		emptyCatalogBrokerID         string
		smUrlToEmptyCatalogBroker    string

		smUrlToMissingBroker             string
		smUrlToSimpleBrokerCatalogBroker string

		stoppedBrokerServer  *common.BrokerServer
		stoppedBrokerID      string
		smUrlToStoppedBroker string

		failingBrokerServer  *common.BrokerServer
		failingBrokerID      string
		smUrlToFailingBroker string

		queryParameterVerificationServer   *common.BrokerServer
		queryParameterVerificationServerID string
		smUrlToQueryVerificationBroker     string
		headerKey                          string
		headerValue                        string

		timeoutBrokerServer  *common.BrokerServer
		timeoutBrokerID      string
		smUrlToTimeoutBroker string
	)

	delayingHandler := func(done chan<- interface{}) func(rw http.ResponseWriter, req *http.Request) {
		return func(rw http.ResponseWriter, req *http.Request) {
			brokerDelay := timeoutDuration + additionalDelayAfterTimeout
			timeoutContext, _ := context.WithTimeout(req.Context(), brokerDelay)
			<-timeoutContext.Done()
			common.SetResponse(rw, http.StatusTeapot, common.Object{})
			close(done)
		}
	}

	resetBrokersHandlers := func() {
		validBrokerServer.ResetHandlers()
		brokerServerWithEmptyCatalog.ResetHandlers()
		stoppedBrokerServer.ResetHandlers()
		failingBrokerServer.ResetHandlers()
		queryParameterVerificationServer.ResetHandlers()
		timeoutBrokerServer.ResetHandlers()

		failingBrokerServer.ServiceInstanceHandler = failingHandler
		failingBrokerServer.BindingHandler = failingHandler
		failingBrokerServer.CatalogHandler = failingHandler
		failingBrokerServer.ServiceInstanceLastOpHandler = failingHandler
		failingBrokerServer.BindingLastOpHandler = failingHandler
		failingBrokerServer.BindingAdaptCredentialsHandler = failingHandler

		queryParameterVerificationServer.ServiceInstanceHandler = queryParameterVerificationHandler(headerKey, headerValue)
		queryParameterVerificationServer.BindingHandler = queryParameterVerificationHandler(headerKey, headerValue)
		queryParameterVerificationServer.CatalogHandler = queryParameterVerificationHandler(headerKey, headerValue)
		queryParameterVerificationServer.ServiceInstanceLastOpHandler = queryParameterVerificationHandler(headerKey, headerValue)
		queryParameterVerificationServer.BindingLastOpHandler = queryParameterVerificationHandler(headerKey, headerValue)

	}

	resetBrokersCallHistory := func() {
		validBrokerServer.ResetCallHistory()
		brokerServerWithEmptyCatalog.ResetCallHistory()
		stoppedBrokerServer.ResetCallHistory()
		failingBrokerServer.ResetCallHistory()
		queryParameterVerificationServer.ResetCallHistory()
		timeoutBrokerServer.ResetCallHistory()
	}

	BeforeSuite(func() {
		ctx = common.NewTestContextBuilder().WithEnvPreExtensions(func(set *pflag.FlagSet) {
			Expect(set.Set("httpclient.response_header_timeout", timeoutDuration.String())).ToNot(HaveOccurred())
		}).Build()
		validBrokerID, _, validBrokerServer = ctx.RegisterBroker()
		smUrlToWorkingBroker = validBrokerServer.URL() + "/v1/osb/" + validBrokerID

		emptyCatalogBrokerID, _, brokerServerWithEmptyCatalog = ctx.RegisterBrokerWithCatalog(common.NewEmptySBCatalog())
		smUrlToEmptyCatalogBroker = brokerServerWithEmptyCatalog.URL() + "/v1/osb/" + emptyCatalogBrokerID

		simpleBrokerCatalogID, _, brokerServerWithSimpleCatalog := ctx.RegisterBrokerWithCatalog(simpleCatalog)
		smUrlToSimpleBrokerCatalogBroker = brokerServerWithSimpleCatalog.URL() + "/v1/osb/" + simpleBrokerCatalogID

		failingBrokerID, _, failingBrokerServer = ctx.RegisterBroker()
		smUrlToFailingBroker = failingBrokerServer.URL() + "/v1/osb/" + failingBrokerID

		UUID, err := uuid.NewV4()
		if err != nil {
			Expect(err).ToNot(HaveOccurred())
		}
		smUrlToMissingBroker = "http://localhost:32123/v1/osb/" + UUID.String()

		stoppedBrokerID, _, stoppedBrokerServer = ctx.RegisterBroker()
		stoppedBrokerServer.Close()

		smUrlToStoppedBroker = stoppedBrokerServer.URL() + "/v1/osb/" + stoppedBrokerID

		headerKey, headerValue = generateRandomQueryParam()
		queryParameterVerificationServerID, _, queryParameterVerificationServer = ctx.RegisterBroker()
		smUrlToQueryVerificationBroker = queryParameterVerificationServer.URL() + "/v1/osb/" + queryParameterVerificationServerID

		timeoutBrokerID, _, timeoutBrokerServer = ctx.RegisterBroker()
		smUrlToTimeoutBroker = timeoutBrokerServer.URL() + "/v1/osb/" + timeoutBrokerID
	})

	BeforeEach(func() {
		resetBrokersHandlers()
		resetBrokersCallHistory()
	})

	AfterSuite(func() {
		ctx.Cleanup()
	})

	Describe("Catalog", func() {
		Context("when call to working service broker", func() {
			It("should succeed", func() {
				assertWorkingBrokerResponse(
					ctx.SMWithBasic.GET(smUrlToWorkingBroker+"/v2/catalog").WithHeader("X-Broker-API-Version", "oidc_authn.13").Expect(),
					http.StatusOK, "services")

			})

			It("should return valid catalog if it's missing some properties", func() {
				req := ctx.SMWithBasic.GET(smUrlToSimpleBrokerCatalogBroker+"/v2/catalog").WithHeader("X-Broker-API-Version", "oidc_authn.13").Expect()
				req.Status(http.StatusOK)

				service := req.JSON().Object().Value("services").Array().First().Object()
				service.Keys().NotContains("tags", "metadata", "requires")

				plan := service.Value("plans").Array().First().Object()
				plan.Keys().NotContains("metadata", "schemas")
			})

			It("should return valid catalog with all catalog extensions if catalog extensions are present", func() {
				resp := ctx.SMWithBasic.GET(smUrlToSimpleBrokerCatalogBroker+"/v2/catalog").WithHeader("X-Broker-API-Version", "oidc_authn.13").
					Expect().
					Status(http.StatusOK).JSON()

				resp.Path("$.services[*].dashboard_client[*]").Array().Contains("id", "secret", "redirect_uri")
				resp.Path("$.services[*].plans[*].random_extension").Array().Contains("random_extension")
			})

			It("should not reach service broker", func() {
				assertWorkingBrokerResponse(
					ctx.SMWithBasic.GET(smUrlToWorkingBroker+"/v2/catalog").WithHeader("X-Broker-API-Version", "oidc_authn.13").Expect(),
					http.StatusOK, "services")

				Expect(len(validBrokerServer.CatalogEndpointRequests)).To(Equal(0))
			})

			Context("when call to empty catalog broker", func() {
				It("should succeed and return empty services", func() {
					call := ctx.SMWithBasic.GET(smUrlToEmptyCatalogBroker+"/v2/catalog").WithHeader("X-Broker-API-Version", "oidc_authn.13").Expect()

					assertWorkingBrokerResponse(
						call,
						http.StatusOK, "services")

					call.JSON().Object().Value("services").Array().Empty()
					Expect(len(validBrokerServer.CatalogEndpointRequests)).To(Equal(0))
				})
			})
		})

		Context("when call to failing service broker", func() {
			It("should succeed because broker is not actually invoked", func() {
				assertWorkingBrokerResponse(
					ctx.SMWithBasic.GET(smUrlToFailingBroker+"/v2/catalog").WithHeader("X-Broker-API-Version", "oidc_authn.13").Expect(),
					http.StatusOK, "services")

				Expect(len(failingBrokerServer.CatalogEndpointRequests)).To(Equal(0))
			})
		})

		Context("when call to missing service broker", func() {
			It("should fail", func() {
				assertMissingBrokerError(
					ctx.SMWithBasic.GET(smUrlToMissingBroker+"/v2/catalog").WithHeader("X-Broker-API-Version", "oidc_authn.13").Expect())
			})
		})

		Context("when call to stopped service broker", func() {
			It("should succeed because broker is not actually invoked", func() {
				assertWorkingBrokerResponse(
					ctx.SMWithBasic.GET(smUrlToStoppedBroker+"/v2/catalog").WithHeader("X-Broker-API-Version", "oidc_authn.13").Expect(),
					http.StatusOK, "services")

				Expect(len(stoppedBrokerServer.CatalogEndpointRequests)).To(Equal(0))
			})
		})
	})

	Describe("Provision", func() {

		Context("call to working service broker", func() {
			It("should succeed", func() {
				assertWorkingBrokerResponse(
					ctx.SMWithBasic.PUT(smUrlToWorkingBroker+"/v2/service_instances/12345").WithHeader("X-Broker-API-Version", "oidc_authn.13").
						WithJSON(getDummyService()).Expect(), http.StatusCreated)
			})
		})

		Context("when call to failing service broker", func() {
			It("should fail", func() {
				assertFailingBrokerError(
					ctx.SMWithBasic.PUT(smUrlToFailingBroker+"/v2/service_instances/12345").WithHeader("X-Broker-API-Version", "oidc_authn.13").
						WithJSON(getDummyService()).Expect())
			})
		})

		Context("when call to missing broker", func() {
			It("should fail", func() {
				assertMissingBrokerError(
					ctx.SMWithBasic.PUT(smUrlToMissingBroker+"/v2/service_instances/12345").WithHeader("X-Broker-API-Version", "oidc_authn.13").
						WithJSON(getDummyService()).Expect())
			})
		})

		Context("when call to stopped service broker", func() {
			It("should fail", func() {
				assertUnresponsiveBrokerError(
					ctx.SMWithBasic.PUT(smUrlToStoppedBroker+"/v2/service_instances/12345").WithHeader("X-Broker-API-Version", "oidc_authn.13").
						WithJSON(getDummyService()).Expect())
			})
		})

		Context("when call contains query params", func() {
			It("propagates them to the service broker", func() {
				assertWorkingBrokerResponse(
					ctx.SMWithBasic.PUT(smUrlToQueryVerificationBroker+"/v2/service_instances/12345").WithHeader("X-Broker-API-Version", "oidc_authn.13").
						WithJSON(getDummyService()).WithQuery(headerKey, headerValue).Expect(), http.StatusCreated)
			})
		})

		Context("when broker doesn't respond in a timely manner", func() {
			It("should fail with 502", func(done chan<- interface{}) {
				timeoutBrokerServer.ServiceInstanceHandler = delayingHandler(done)
				assertUnresponsiveBrokerError(ctx.SMWithBasic.PUT(smUrlToTimeoutBroker+"/v2/service_instances/12345").WithHeader("X-Broker-API-Version", "oidc_authn.13").
					WithJSON(getDummyService()).Expect())
			})
		})

	})
	Describe("Deprovision", func() {

		Context("when trying to deprovision existing service", func() {
			It("should be successfull", func() {
				ctx.SMWithBasic.DELETE(smUrlToWorkingBroker+"/v2/service_instances/12345").WithHeader("X-Broker-API-Version", "oidc_authn.13").
					WithQueryObject(getDummyService()).
					Expect().Status(http.StatusOK).JSON().Object()
			})
		})

		Context("when call to failing broker", func() {
			It("should fail", func() {
				assertFailingBrokerError(
					ctx.SMWithBasic.DELETE(smUrlToFailingBroker+"/v2/service_instances/12345").WithHeader("X-Broker-API-Version", "oidc_authn.13").
						WithQueryObject(getDummyService()).Expect())
			})
		})

		Context("when call to missing service broker", func() {
			It("should fail", func() {
				assertMissingBrokerError(
					ctx.SMWithBasic.DELETE(smUrlToMissingBroker+"/v2/service_instances/12345").WithHeader("X-Broker-API-Version", "oidc_authn.13").
						WithQueryObject(getDummyService()).Expect())
			})
		})

		Context("when call to stopped service broker", func() {
			It("should fail", func() {
				assertUnresponsiveBrokerError(ctx.SMWithBasic.DELETE(smUrlToStoppedBroker+"/v2/service_instances/12345").WithHeader("X-Broker-API-Version", "oidc_authn.13").
					WithQueryObject(getDummyService()).Expect())
			})
		})

		Context("when call contains query params", func() {
			It("propagates them to the service broker", func() {
				assertWorkingBrokerResponse(
					ctx.SMWithBasic.DELETE(smUrlToQueryVerificationBroker+"/v2/service_instances/12345").WithHeader("X-Broker-API-Version", "oidc_authn.13").
						WithJSON(getDummyService()).WithQuery(headerKey, headerValue).Expect(), http.StatusOK)
			})
		})

		Context("when broker doesn't respond in a timely manner", func() {
			It("should fail with 502", func(done chan<- interface{}) {
				timeoutBrokerServer.ServiceInstanceHandler = delayingHandler(done)
				assertUnresponsiveBrokerError(ctx.SMWithBasic.DELETE(smUrlToTimeoutBroker+"/v2/service_instances/12345").WithHeader("X-Broker-API-Version", "oidc_authn.13").
					WithJSON(getDummyService()).Expect())
			})
		})
	})

	Describe("Bind", func() {

		Context("call to working service broker", func() {
			It("should succeed", func() {
				assertWorkingBrokerResponse(
					ctx.SMWithBasic.PUT(smUrlToWorkingBroker+"/v2/service_instances/iid/service_bindings/bid").WithHeader("X-Broker-API-Version", "oidc_authn.13").
						WithJSON(getDummyService()).Expect(), http.StatusCreated, "credentials")
			})
		})

		Context("when call to broker service broker", func() {
			It("should fail", func() {
				assertFailingBrokerError(
					ctx.SMWithBasic.PUT(smUrlToFailingBroker+"/v2/service_instances/iid/service_bindings/bid").WithHeader("X-Broker-API-Version", "oidc_authn.13").
						WithJSON(getDummyService()).Expect())
			})
		})

		Context("when call to missing service broker", func() {
			It("should fail", func() {
				assertMissingBrokerError(ctx.SMWithBasic.PUT(smUrlToMissingBroker+"/v2/service_instances/iid/service_bindings/bid").WithHeader("X-Broker-API-Version", "oidc_authn.13").
					WithJSON(getDummyService()).Expect())
			})
		})

		Context("when call to stopped service broker", func() {
			It("should fail", func() {
				assertUnresponsiveBrokerError(ctx.SMWithBasic.PUT(smUrlToStoppedBroker+"/v2/service_instances/iid/service_bindings/bid").WithHeader("X-Broker-API-Version", "oidc_authn.13").
					WithJSON(getDummyService()).Expect())
			})
		})

		Context("when call contains query params", func() {
			It("propagates them to the service broker", func() {
				assertWorkingBrokerResponse(
					ctx.SMWithBasic.PUT(smUrlToQueryVerificationBroker+"/v2/service_instances/iid/service_bindings/bid").WithHeader("X-Broker-API-Version", "oidc_authn.13").
						WithJSON(getDummyService()).WithQuery(headerKey, headerValue).Expect(), http.StatusCreated)
			})
		})

		Context("when broker doesn't respond in a timely manner", func() {
			It("should fail with 502", func(done chan<- interface{}) {
				timeoutBrokerServer.BindingHandler = delayingHandler(done)
				assertUnresponsiveBrokerError(ctx.SMWithBasic.PUT(smUrlToTimeoutBroker+"/v2/service_instances/iid/service_bindings/bid").WithHeader("X-Broker-API-Version", "oidc_authn.13").
					WithJSON(getDummyService()).Expect())
			})
		})
	})

	Describe("Unbind", func() {

		Context("when trying to delete binding", func() {
			It("should be successful", func() {
				ctx.SMWithBasic.DELETE(smUrlToWorkingBroker+"/v2/service_instances/iid/service_bindings/bid").WithHeader("X-Broker-API-Version", "oidc_authn.13").
					WithQueryObject(getDummyService()).
					Expect().Status(http.StatusOK).JSON().Object()

			})
		})

		Context("when call to failing service broker", func() {
			It("should return error", func() {
				assertFailingBrokerError(
					ctx.SMWithBasic.DELETE(smUrlToFailingBroker+"/v2/service_instances/iid/service_bindings/bid").WithHeader("X-Broker-API-Version", "oidc_authn.13").
						WithQueryObject(getDummyService()).Expect())
			})
		})

		Context("when call to missing broker", func() {
			It("unbind fails", func() {
				assertMissingBrokerError(
					ctx.SMWithBasic.DELETE(smUrlToMissingBroker+"/v2/service_instances/iid/service_bindings/bid").WithHeader("X-Broker-API-Version", "oidc_authn.13").
						WithQueryObject(getDummyService()).Expect())
			})
		})

		Context("when call to stopped service broker", func() {
			It("should fail", func() {
				assertUnresponsiveBrokerError(
					ctx.SMWithBasic.DELETE(smUrlToStoppedBroker+"/v2/service_instances/iid/service_bindings/bid").WithHeader("X-Broker-API-Version", "oidc_authn.13").
						WithQueryObject(getDummyService()).Expect())

			})
		})

		Context("when call contains query params", func() {
			It("propagates them to the service broker", func() {
				assertWorkingBrokerResponse(
					ctx.SMWithBasic.DELETE(smUrlToQueryVerificationBroker+"/v2/service_instances/iid/service_bindings/bid").WithHeader("X-Broker-API-Version", "oidc_authn.13").
						WithJSON(getDummyService()).WithQuery(headerKey, headerValue).Expect(), http.StatusOK)
			})
		})

		Context("when broker doesn't respond in a timely manner", func() {
			It("should fail with 502", func(done chan<- interface{}) {
				timeoutBrokerServer.BindingHandler = delayingHandler(done)
				assertUnresponsiveBrokerError(ctx.SMWithBasic.DELETE(smUrlToTimeoutBroker+"/v2/service_instances/iid/service_bindings/bid").WithHeader("X-Broker-API-Version", "oidc_authn.13").
					WithQueryObject(getDummyService()).
					Expect())
			})
		})
	})

	Describe("Get Service Instance Last Operation", func() {

		Context("when call to working service broker", func() {
			It("should succeed", func() {
				assertWorkingBrokerResponse(
					ctx.SMWithBasic.GET(smUrlToWorkingBroker+"/v2/service_instances/iid/last_operation").WithHeader("X-Broker-API-Version", "oidc_authn.13").Expect(),
					http.StatusOK, "state")
			})
		})

		Context("when call to failing service broker", func() {
			It("should fail", func() {
				assertFailingBrokerError(
					ctx.SMWithBasic.GET(smUrlToFailingBroker+"/v2/service_instances/iid/last_operation").WithHeader("X-Broker-API-Version", "oidc_authn.13").Expect())

			})
		})

		Context("when call to missing service broker", func() {
			It("should fail", func() {
				assertMissingBrokerError(
					ctx.SMWithBasic.GET(smUrlToMissingBroker+"/v2/service_instances/iid/last_operation").WithHeader("X-Broker-API-Version", "oidc_authn.13").Expect())
			})
		})

		Context("when call to stopped service broker", func() {
			It("should fail", func() {
				assertUnresponsiveBrokerError(
					ctx.SMWithBasic.GET(smUrlToStoppedBroker+"/v2/service_instances/iid/last_operation").WithHeader("X-Broker-API-Version", "oidc_authn.13").Expect())
			})
		})

		Context("when call contains query params", func() {
			It("propagates them to the service broker", func() {
				assertWorkingBrokerResponse(
					ctx.SMWithBasic.GET(smUrlToQueryVerificationBroker+"/v2/service_instances/iid/last_operation").WithHeader("X-Broker-API-Version", "oidc_authn.13").
						WithJSON(getDummyService()).WithQuery(headerKey, headerValue).Expect(), http.StatusOK)
			})
		})

		Context("when broker doesn't respond in a timely manner", func() {
			It("should fail with 502", func(done chan<- interface{}) {
				timeoutBrokerServer.ServiceInstanceLastOpHandler = delayingHandler(done)
				assertUnresponsiveBrokerError(ctx.SMWithBasic.GET(smUrlToTimeoutBroker+"/v2/service_instances/iid/last_operation").WithHeader("X-Broker-API-Version", "oidc_authn.13").
					Expect())
			})
		})
	})

	Describe("Get Binding Last Operation", func() {

		Context("when call to working service broker", func() {
			It("should succeed", func() {
				assertWorkingBrokerResponse(
					ctx.SMWithBasic.GET(smUrlToWorkingBroker+"/v2/service_instances/iid/service_bindings/bid/last_operation").WithHeader("X-Broker-API-Version", "oidc_authn.13").Expect(),
					http.StatusOK, "state")
			})
		})

		Context("when call to failing service broker", func() {
			It("should fail", func() {
				assertFailingBrokerError(
					ctx.SMWithBasic.GET(smUrlToFailingBroker+"/v2/service_instances/iid/service_bindings/bid/last_operation").WithHeader("X-Broker-API-Version", "oidc_authn.13").Expect())

			})
		})

		Context("when call to missing service broker", func() {
			It("should fail", func() {
				assertMissingBrokerError(
					ctx.SMWithBasic.GET(smUrlToMissingBroker+"/v2/service_instances/iid/service_bindings/bid/last_operation").WithHeader("X-Broker-API-Version", "oidc_authn.13").Expect())
			})
		})

		Context("when call to stopped service broker", func() {
			It("should fail", func() {
				assertUnresponsiveBrokerError(
					ctx.SMWithBasic.GET(smUrlToStoppedBroker+"/v2/service_instances/iid/service_bindings/bid/last_operation").WithHeader("X-Broker-API-Version", "oidc_authn.13").Expect())
			})
		})

		Context("when call contains query params", func() {
			It("propagates them to the service broker", func() {
				assertWorkingBrokerResponse(
					ctx.SMWithBasic.GET(smUrlToQueryVerificationBroker+"/v2/service_instances/iid/service_bindings/bid/last_operation").WithHeader("X-Broker-API-Version", "oidc_authn.13").
						WithJSON(getDummyService()).WithQuery(headerKey, headerValue).Expect(), http.StatusOK)
			})
		})

		Context("when broker doesn't respond in a timely manner", func() {
			It("should fail with 502", func(done chan<- interface{}) {
				timeoutBrokerServer.BindingLastOpHandler = delayingHandler(done)
				assertUnresponsiveBrokerError(ctx.SMWithBasic.GET(smUrlToTimeoutBroker+"/v2/service_instances/iid/service_bindings/bid/last_operation").WithHeader("X-Broker-API-Version", "oidc_authn.13").
					Expect())
			})
		})
	})

	Describe("Post Binding Adapt Credentials", func() {
		Context("when call to working service broker", func() {
			It("should succeed", func() {
				assertWorkingBrokerResponse(
					ctx.SMWithBasic.POST(smUrlToWorkingBroker+"/v2/service_instances/iid/service_bindings/bid/adapt_credentials").WithHeader("X-Broker-API-Version", "oidc_authn.13").WithJSON(&common.Object{}).Expect(),
					http.StatusOK, "credentials")
			})
		})

		Context("when call to broken service broker", func() {
			It("should fail", func() {
				assertFailingBrokerError(
					ctx.SMWithBasic.POST(smUrlToFailingBroker+"/v2/service_instances/iid/service_bindings/bid/adapt_credentials").WithHeader("X-Broker-API-Version", "oidc_authn.13").WithJSON(&common.Object{}).Expect())

			})
		})

		Context("when call to missing service broker", func() {
			It("should fail", func() {
				assertMissingBrokerError(
					ctx.SMWithBasic.POST(smUrlToMissingBroker+"/v2/service_instances/iid/service_bindings/bid/adapt_credentials").WithHeader("X-Broker-API-Version", "oidc_authn.13").WithJSON(&common.Object{}).Expect())

			})
		})

		Context("when call to stopped service broker", func() {
			It("should fail", func() {
				assertUnresponsiveBrokerError(
					ctx.SMWithBasic.POST(smUrlToStoppedBroker+"/v2/service_instances/iid/service_bindings/bid/adapt_credentials").WithHeader("X-Broker-API-Version", "oidc_authn.13").WithJSON(&common.Object{}).Expect())

			})
		})

		Context("when call contains query params", func() {
			It("propagates them to the service broker", func() {
				assertWorkingBrokerResponse(
					ctx.SMWithBasic.POST(smUrlToQueryVerificationBroker+"/v2/service_instances/iid/service_bindings/bid/adapt_credentials").WithHeader("X-Broker-API-Version", "oidc_authn.13").
						WithJSON(getDummyService()).WithQuery(headerKey, headerValue).Expect(), http.StatusOK)
			})
		})

		Context("when broker doesn't respond in a timely manner", func() {
			It("should fail with 502", func(done chan<- interface{}) {
				timeoutBrokerServer.BindingAdaptCredentialsHandler = delayingHandler(done)
				assertUnresponsiveBrokerError(ctx.SMWithBasic.POST(smUrlToTimeoutBroker+"/v2/service_instances/iid/service_bindings/bid/adapt_credentials").WithHeader("X-Broker-API-Version", "oidc_authn.13").
					WithJSON(getDummyService()).Expect())
			})
		})
	})

	Describe("Prefixed broker path", func() {
		Context("when call to working broker", func() {

			const brokerPathPrefix = "/broker_prefix"
			var (
				server           common.FakeServer
				osbURL           string
				prefixedBrokerID string
			)

			BeforeEach(func() {
				brokerHandler := &prefixedBrokerHandler{brokerPathPrefix}
				server = &prefixedBrokerServer{Server: httptest.NewServer(brokerHandler)}
				brokerURL := server.URL() + brokerPathPrefix

				brokerJSON := common.Object{
					"name":        "prefixed_broker",
					"broker_url":  brokerURL,
					"description": "",
					"credentials": common.Object{
						"basic": common.Object{
							"username": "buser",
							"password": "bpass",
						},
					},
				}

				prefixedBrokerID = common.RegisterBrokerInSM(brokerJSON, ctx.SMWithOAuth, map[string]string{})["id"].(string)
				ctx.Servers[common.BrokerServerPrefix+prefixedBrokerID] = server
				osbURL = "/v1/osb/" + prefixedBrokerID
			})

			AfterEach(func() {
				ctx.CleanupBroker(prefixedBrokerID)
			})

			It("should get catalog", func() {
				ctx.SMWithBasic.GET(osbURL + "/v2/catalog").
					Expect().Status(http.StatusOK).JSON().Object().ContainsKey("services")
			})
		})
	})

	Describe("Malfunctioning broker", func() {
		Context("when broker response is not a valid json", func() {
			It("should return an OSB compliant error", func() {
				failingBrokerServer.ServiceInstanceHandler = func(rw http.ResponseWriter, _ *http.Request) {
					rw.WriteHeader(http.StatusCreated)
					rw.Write([]byte("[not a json]"))
				}
				ctx.SMWithBasic.PUT(smUrlToFailingBroker+"/v2/service_instances/12345").WithHeader("X-Broker-API-Version", "oidc_authn.13").
					WithJSON(getDummyService()).Expect().Status(http.StatusCreated).JSON().Object().Value("description").String().Match("Service broker .* responded with invalid JSON: \\[not a json\\]")
			})
		})

		Context("when broker response error description is empty", func() {
			It("should assing default description", func() {
				failingBrokerServer.ServiceInstanceHandler = func(rw http.ResponseWriter, _ *http.Request) {
					rw.WriteHeader(http.StatusBadRequest)
					rw.Write([]byte(`{"error": "ErrorType"}`))
				}
				ctx.SMWithBasic.PUT(smUrlToFailingBroker+"/v2/service_instances/12345").WithHeader("X-Broker-API-Version", "oidc_authn.13").
					WithJSON(getDummyService()).Expect().Status(http.StatusBadRequest).JSON().Object().Value("description").String().Match("Service broker .* failed with: {}")
			})
		})
	})
})

type prefixedBrokerHandler struct {
	brokerPathPrefix string
}

func (pbh *prefixedBrokerHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if strings.HasPrefix(req.URL.Path, pbh.brokerPathPrefix) {
		common.SetResponse(w, http.StatusOK, common.Object{"services": common.Array{}})
	} else {
		common.SetResponse(w, http.StatusNotFound, common.Object{})
	}
}

type prefixedBrokerServer struct {
	*httptest.Server
}

func (pbs *prefixedBrokerServer) URL() string {
	return pbs.Server.URL
}
