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
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/web"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("Bind", func() {
	var IID = "10101"
	var BID = "01010"
	BeforeEach(func() {
		brokerServer.BindingHandler = parameterizedHandler(http.StatusCreated, `{}`)
		json := provisionRequestBodyMapWith("plan_id", plan1CatalogID)()
		ctx.SMWithBasic.PUT(smBrokerURL+"/v2/service_instances/"+IID).
			WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
			WithJSON(json).
			Expect().Status(http.StatusCreated)
	})

	Context("call to working service broker", func() {

		It("should succeed", func() {
			brokerServer.BindingHandler = parameterizedHandler(http.StatusCreated, `{}`)
			ctx.SMWithBasic.PUT(smBrokerURL+"/v2/service_instances/"+IID+"/service_bindings/"+BID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				WithJSON(provisionRequestBodyMap()()).Expect().Status(http.StatusCreated)

			ctx.SMWithOAuth.GET(web.ServiceBindingsURL + "/" + BID).
				Expect().
				Status(http.StatusOK).
				JSON().
				Object().
				ContainsMap(map[string]interface{}{
					"id":                  BID,
					"service_instance_id": IID,
				})

			verifyOperationExists(operationExpectations{
				Type:         types.CREATE,
				State:        types.SUCCEEDED,
				ResourceID:   BID,
				ResourceType: "/v1/service_bindings",
				ExternalID:   "",
			})
		})

		It("same binding should return conflict", func() {
			brokerServer.BindingHandler = parameterizedHandler(http.StatusCreated, `{}`)
			ctx.SMWithBasic.PUT(smBrokerURL+"/v2/service_instances/"+IID+"/service_bindings/"+BID).
				WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				WithJSON(provisionRequestBodyMap()()).
				Expect().
				Status(http.StatusCreated)

			bindingJSON := ctx.SMWithOAuth.GET(web.ServiceBindingsURL + "/" + BID).
				Expect().
				Status(http.StatusOK).
				JSON().
				Object()

			brokerServer.BindingHandler = parameterizedHandler(http.StatusOK, `{}`)
			ctx.
				SMWithBasic.PUT(smBrokerURL+"/v2/service_instances/"+IID+"/service_bindings/"+BID).
				WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				WithJSON(provisionRequestBodyMap()()).
				Expect().
				Status(http.StatusConflict)

			ctx.
				SMWithOAuth.GET(web.ServiceBindingsURL + "/" + BID).
				Expect().
				Status(http.StatusOK).
				JSON().
				Object().
				ContainsMap(bindingJSON)

			verifyOperationExists(operationExpectations{
				Type:         types.CREATE,
				State:        types.SUCCEEDED,
				ResourceID:   BID,
				ResourceType: "/v1/service_bindings",
				ExternalID:   "",
			})
		})

		It("Binding to a server that supports gzip encoded responses", func() {
			brokerServer.BindingHandler = gzipHandler(http.StatusCreated, `{}`)
			ctx.SMWithBasic.PUT(smBrokerURL+"/v2/service_instances/"+IID+"/service_bindings/"+BID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				WithJSON(provisionRequestBodyMap()()).Expect().Status(http.StatusCreated)

			ctx.SMWithOAuth.GET(web.ServiceBindingsURL + "/" + BID).
				Expect().Status(http.StatusOK)

			verifyOperationExists(operationExpectations{
				Type:         types.CREATE,
				State:        types.SUCCEEDED,
				ResourceID:   BID,
				ResourceType: "/v1/service_bindings",
				ExternalID:   "",
			})
		})

		It("when ctx has value false should not store bindings", func() {
			shouldStoreBinding = false
			brokerServer.BindingHandler = parameterizedHandler(http.StatusCreated, `{}`)
			ctx.SMWithBasic.PUT(smBrokerURL+"/v2/service_instances/"+IID+"/service_bindings/"+BID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				WithJSON(provisionRequestBodyMap()()).Expect().Status(http.StatusCreated)

			ctx.SMWithOAuth.GET(web.ServiceBindingsURL + "/" + BID).
				Expect().Status(http.StatusNotFound)

			verifyOperationDoesNotExist(BID)
		})

	})

	Context("when call to failing service broker", func() {
		It("should fail", func() {
			brokerServer.BindingHandler = parameterizedHandler(http.StatusInternalServerError, `internal server error`)
			assertFailingBrokerError(
				ctx.SMWithBasic.PUT(smBrokerURL+"/v2/service_instances/"+IID+"/service_bindings/"+BID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
					WithJSON(provisionRequestBodyMap()()).Expect(), http.StatusInternalServerError, `internal server error`)

			ctx.SMWithOAuth.GET(web.ServiceBindingsURL + "/" + BID).
				Expect().Status(http.StatusNotFound)
			verifyOperationDoesNotExist(BID, "create")
		})
	})

	Context("when call to missing service broker", func() {
		It("should fail with 401", func() {
			ctx.SMWithBasic.PUT("http://localhost:3456/v1/osb/123"+"/v2/service_instances/"+IID+"/service_bindings/"+BID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				WithJSON(provisionRequestBodyMap()()).Expect().Status(http.StatusUnauthorized)

			ctx.SMWithOAuth.GET(web.ServiceBindingsURL + "/" + BID).
				Expect().Status(http.StatusNotFound)
			verifyOperationDoesNotExist(BID, "create")
		})
	})

	Context("when call to stopped service broker", func() {
		It("should fail", func() {
			credentials := brokerPlatformCredentialsIDMap[stoppedBrokerID]
			ctx.SMWithBasic.SetBasicCredentials(ctx, credentials.username, credentials.password)

			assertUnresponsiveBrokerError(ctx.SMWithBasic.PUT(smUrlToStoppedBroker+"/v2/service_instances/"+IID+"/service_bindings/"+BID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				WithJSON(provisionRequestBodyMap()()).Expect())

			ctx.SMWithOAuth.GET(web.ServiceBindingsURL + "/" + BID).
				Expect().Status(http.StatusNotFound)
			verifyOperationDoesNotExist(BID, "create")
		})
	})

	Context("when call contains query params", func() {
		It("propagates them to the service broker", func() {
			headerKey, headerValue := generateRandomQueryParam()
			brokerServer.BindingHandler = queryParameterVerificationHandler(headerKey, headerValue)
			ctx.SMWithBasic.PUT(smBrokerURL+"/v2/service_instances/"+IID+"/service_bindings/"+BID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				WithJSON(provisionRequestBodyMap()()).WithQuery(headerKey, headerValue).Expect().Status(http.StatusCreated)

			ctx.SMWithOAuth.GET(web.ServiceBindingsURL + "/" + BID).
				Expect().Status(http.StatusOK)
			verifyOperationExists(operationExpectations{
				Type:         types.CREATE,
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
			assertUnresponsiveBrokerError(ctx.SMWithBasic.PUT(smBrokerURL+"/v2/service_instances/"+IID+"/service_bindings/"+BID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				WithJSON(provisionRequestBodyMap()()).Expect())

			ctx.SMWithOAuth.GET(web.ServiceBindingsURL + "/" + BID).
				Expect().Status(http.StatusNotFound)
			verifyOperationDoesNotExist(BID, "create")
		}, testTimeout)
	})

	Context("bind request", func() {
		Context("multitenant check", func() {
			Context("from not an instance owner", func() {
				It("should return 404", func() {
					brokerServer.BindingHandler = parameterizedHandler(http.StatusCreated, `{}`)
					ctx.SMWithBasic.PUT(smBrokerURL+"/v2/service_instances/"+IID+"/service_bindings/"+BID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
						WithJSON(provisionRequestBodyMapWith("context."+TenantIdentifier, "other_tenant")()).Expect().Status(http.StatusNotFound)
					ctx.SMWithOAuth.GET(web.ServiceBindingsURL + "/" + BID).
						Expect().Status(http.StatusNotFound)
					verifyOperationDoesNotExist(BID, "create")
				})
			})
			Context("from an instance owner", func() {
				It("should return 201", func() {
					brokerServer.BindingHandler = parameterizedHandler(http.StatusCreated, `{}`)
					ctx.SMWithBasic.PUT(smBrokerURL+"/v2/service_instances/"+IID+"/service_bindings/"+BID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
						WithJSON(provisionRequestBodyMap()()).Expect().Status(http.StatusCreated)

					ctx.SMWithOAuth.GET(web.ServiceBindingsURL + "/" + BID).
						Expect().Status(http.StatusOK)
					verifyOperationExists(operationExpectations{
						Type:         types.CREATE,
						State:        types.SUCCEEDED,
						ResourceID:   BID,
						ResourceType: "/v1/service_bindings",
						ExternalID:   "",
					})
				})
			})
		})
		Context("broker platform credentials check", func() {
			Context("bind with invalid credentials", func() {
				BeforeEach(func() {
					ctx.SMWithBasic.SetBasicCredentials(ctx, "test", "test")
				})

				It("should return 401", func() {
					brokerServer.BindingHandler = parameterizedHandler(http.StatusCreated, `{}`)
					ctx.SMWithBasic.PUT(smBrokerURL+"/v2/service_instances/"+IID+"/service_bindings/"+BID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
						WithJSON(provisionRequestBodyMap()()).Expect().Status(http.StatusUnauthorized)
					ctx.SMWithOAuth.GET(web.ServiceBindingsURL + "/" + BID).
						Expect().Status(http.StatusNotFound)
					verifyOperationDoesNotExist(BID, "create")
				})
			})
		})
	})
})
