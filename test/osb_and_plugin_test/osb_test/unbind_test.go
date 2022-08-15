/*
 * Copyright 2018 The Service Manager Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package osb_test

import (
	. "github.com/onsi/ginkgo"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web"
	"net/http"
)

var _ = Describe("Unbind", func() {
	var IID = "11011"
	var BID = "01011"

	BeforeEach(func() {
		ctx.SMWithBasic.PUT(smBrokerURL+"/v2/service_instances/"+IID).
			WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
			WithJSON(provisionRequestBodyMapWith("plan_id", plan1CatalogID)()).
			Expect().Status(http.StatusCreated)
		ctx.SMWithBasic.PUT(smBrokerURL+"/v2/service_instances/"+IID+"/service_bindings/"+BID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
			WithJSON(provisionRequestBodyMap()()).Expect().Status(http.StatusCreated)
	})

	Context("when trying to delete binding", func() {
		It("should be successful", func() {
			brokerServer.BindingHandler = parameterizedHandler(http.StatusOK, `{}`)
			ctx.SMWithBasic.DELETE(smBrokerURL+"/v2/service_instances/"+IID+"/service_bindings/"+BID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				WithQueryObject(provisionRequestBodyMap()()).
				Expect().Status(http.StatusOK).JSON().Object()

			ctx.SMWithOAuth.GET(web.ServiceBindingsURL + "/" + BID).
				Expect().Status(http.StatusNotFound)

			verifyOperationExists(operationExpectations{
				Type:         types.DELETE,
				State:        types.SUCCEEDED,
				ResourceID:   BID,
				ResourceType: "/v1/service_bindings",
				ExternalID:   "",
			})

		})

		It("unbind using smaap api should be successful", func() {
			brokerServer.BindingHandler = parameterizedHandler(http.StatusOK, `{}`)
			ctx.SMWithOAuthForTenant.DELETE(web.ServiceBindingsURL+"/"+BID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				WithQueryObject(provisionRequestBodyMap()()).
				WithQuery("async", false).
				Expect().Status(http.StatusOK).JSON().Object()

			ctx.SMWithOAuth.GET(web.ServiceBindingsURL + "/" + BID).
				Expect().Status(http.StatusNotFound)

			verifyOperationExists(operationExpectations{
				Type:         types.DELETE,
				State:        types.SUCCEEDED,
				ResourceID:   BID,
				ResourceType: "/v1/service_bindings",
				ExternalID:   "",
			})

		})
	})

	Context("when call to failing service broker", func() {

		It("should return error", func() {
			brokerServer.BindingHandler = parameterizedHandler(http.StatusInternalServerError, `internal server error`)
			assertFailingBrokerError(
				ctx.SMWithBasic.DELETE(smBrokerURL+"/v2/service_instances/"+IID+"/service_bindings/"+BID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
					WithJSON(provisionRequestBodyMap()()).Expect(), http.StatusInternalServerError, `internal server error`)
			ctx.SMWithOAuth.GET(web.ServiceBindingsURL + "/" + BID).
				Expect().Status(http.StatusOK)
			verifyOperationDoesNotExist(BID, "delete")
		})
	})

	Context("when call to missing broker", func() {
		It("unbind fails with 401", func() {
			ctx.SMWithBasic.DELETE("http://localhost:3456/v1/osb/123"+"/v2/service_instances/"+IID+"/service_bindings/"+BID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				WithQueryObject(provisionRequestBodyMap()()).Expect().Status(http.StatusUnauthorized)
			ctx.SMWithOAuth.GET(web.ServiceBindingsURL + "/" + BID).
				Expect().Status(http.StatusOK)
			verifyOperationDoesNotExist(BID, "delete")
		})
	})

	Context("when call to stopped service broker", func() {
		It("should fail", func() {
			credentials := brokerPlatformCredentialsIDMap[stoppedBrokerID]
			ctx.SMWithBasic.SetBasicCredentials(ctx, credentials.username, credentials.password)

			assertUnresponsiveBrokerError(
				ctx.SMWithBasic.DELETE(smUrlToStoppedBroker+"/v2/service_instances/"+IID+"/service_bindings/"+BID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
					WithQueryObject(provisionRequestBodyMap()()).Expect())
			ctx.SMWithOAuth.GET(web.ServiceBindingsURL + "/" + BID).
				Expect().Status(http.StatusOK)
			verifyOperationDoesNotExist(BID, "delete")

		})
	})

	Context("when call contains query params", func() {
		It("propagates them to the service broker", func() {
			headerKey, headerValue := generateRandomQueryParam()
			brokerServer.BindingHandler = queryParameterVerificationHandler(headerKey, headerValue)
			ctx.SMWithBasic.DELETE(smBrokerURL+"/v2/service_instances/"+IID+"/service_bindings/"+BID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				WithJSON(provisionRequestBodyMap()()).WithQuery(headerKey, headerValue).Expect().Status(http.StatusOK)
			ctx.SMWithOAuth.GET(web.ServiceBindingsURL + "/" + BID).
				Expect().Status(http.StatusNotFound)
			verifyOperationExists(operationExpectations{
				Type:         types.DELETE,
				State:        types.SUCCEEDED,
				ResourceID:   BID,
				ResourceType: "/v1/service_bindings",
				ExternalID:   "",
			})
		})
	})

	Context("when broker doesn't respond in a timely manner", func() {
		It("should fail with 502", func(done chan<- interface{}) {
			brokerServer.BindingHandler = delayingHandler(done)
			assertUnresponsiveBrokerError(ctx.SMWithBasic.DELETE(smBrokerURL+"/v2/service_instances/"+IID+"/service_bindings/bid"+BID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				WithQueryObject(provisionRequestBodyMap()()).
				Expect())
		}, testTimeout)
	})

	Context("broker platform credentials check", func() {

		Context("unbind with invalid credentials", func() {
			BeforeEach(func() {
				ctx.SMWithBasic.SetBasicCredentials(ctx, "test", "test")
			})

			It("should return 401", func() {
				brokerServer.BindingHandler = parameterizedHandler(http.StatusOK, `{}`)
				ctx.SMWithBasic.DELETE(smBrokerURL+"/v2/service_instances/"+IID+"/service_bindings/"+BID).
					WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
					Expect().Status(http.StatusUnauthorized)

				verifyOperationDoesNotExist(BID, "delete")
				ctx.SMWithOAuth.GET(web.ServiceBindingsURL + "/" + BID).
					Expect().Status(http.StatusOK)
			})
		})
	})
})
