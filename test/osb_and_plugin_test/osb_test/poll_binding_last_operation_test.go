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
	"fmt"
	"github.com/Peripli/service-manager/pkg/types"
	. "github.com/onsi/ginkgo/v2"
	"net/http"
)

type testCase struct {
	expectGetBindingOperationStatus int
	expectOperationType             types.OperationCategory
	expectOperationState            types.OperationState
	expectGetBindingStatus          int
	expectBindingReady              bool
	bindingResponseState            string
}

var _ = Describe("Get Binding Last Operation", func() {
	Context("when call to working service broker", func() {
		It("should succeed", func() {
			brokerServer.BindingLastOpHandler = parameterizedHandler(http.StatusOK, `{}`)
			ctx.SMWithBasic.GET(smBrokerURL+"/v2/service_instances/"+SID+"/service_bindings/bid/last_operation").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				Expect().Status(http.StatusOK)
		})
	})

	Context("when call to failing service broker", func() {
		It("should fail", func() {
			brokerServer.BindingLastOpHandler = parameterizedHandler(http.StatusInternalServerError, `internal server error`)
			assertFailingBrokerError(
				ctx.SMWithBasic.GET(smBrokerURL+"/v2/service_instances/iid/service_bindings/bid/last_operation").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
					Expect(), http.StatusInternalServerError, "internal server error")

		})
	})

	Context("when call to missing service broker", func() {
		It("should fail with 401", func() {
			ctx.SMWithBasic.GET("http://localhost:3456/v1/osb/123"+"/v2/service_instances/iid/service_bindings/bid/last_operation").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				Expect().Status(http.StatusUnauthorized)
		})
	})

	Context("when call to stopped service broker", func() {
		It("should fail", func() {
			credentials := brokerPlatformCredentialsIDMap[stoppedBrokerID]
			ctx.SMWithBasic.SetBasicCredentials(ctx, credentials.username, credentials.password)

			assertUnresponsiveBrokerError(
				ctx.SMWithBasic.GET(smUrlToStoppedBroker+"/v2/service_instances/iid/service_bindings/bid/last_operation").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).Expect())
		})
	})

	Context("when call contains query params", func() {
		It("propagates them to the service broker", func() {
			headerKey, headerValue := generateRandomQueryParam()
			brokerServer.BindingHandler = queryParameterVerificationHandler(headerKey, headerValue)
			ctx.SMWithBasic.GET(smBrokerURL+"/v2/service_instances/iid/service_bindings/bid/last_operation").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				WithJSON(provisionRequestBodyMap()()).WithQuery(headerKey, headerValue).Expect().Status(http.StatusOK)
		})
	})

	Context("when broker doesn't respond in a timely manner", func() {
		It("should fail with 502", func(done chan<- interface{}) {
			brokerServer.BindingLastOpHandler = delayingHandler(done)
			assertUnresponsiveBrokerError(ctx.SMWithBasic.GET(smBrokerURL+"/v2/service_instances/iid/service_bindings/bid/last_operation").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				Expect())
		}, testTimeout)
	})

	Context("broker platform credentials check", func() {
		BeforeEach(func() {
			brokerServer.ServiceInstanceHandler = parameterizedHandler(http.StatusCreated, `{}`)
			ctx.SMWithBasic.PUT(smBrokerURL+"/v2/service_instances/"+SID).
				WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				WithJSON(provisionRequestBodyMap()()).Expect().Status(http.StatusCreated)
		})

		Context("get binding last operation from invalid credentials", func() {
			BeforeEach(func() {
				ctx.SMWithBasic.SetBasicCredentials(ctx, "test", "test")
			})

			It("should return 401", func() {
				brokerServer.BindingLastOpHandler = parameterizedHandler(http.StatusOK, `{}`)
				ctx.SMWithBasic.GET(smBrokerURL+"/v2/service_instances/"+SID+"/service_bindings/bid/last_operation").
					WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
					Expect().Status(http.StatusUnauthorized)
			})
		})
	})

	Context("get binding last operation store in SM", func() {
		var BID = "1111223"
		BeforeEach(func() {
			brokerServer.ServiceInstanceHandler = parameterizedHandler(http.StatusCreated, `{}`)
			ctx.SMWithBasic.PUT(smBrokerURL+"/v2/service_instances/"+SID).
				WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				WithJSON(provisionRequestBodyMap()()).Expect().Status(http.StatusCreated)

		})

		Context("Bind", func() {
			BeforeEach(func() {
				brokerServer.BindingHandler = parameterizedHandler(http.StatusAccepted, `{}`)
				ctx.SMWithBasic.PUT(smBrokerURL+"/v2/service_instances/"+SID+"/service_bindings/"+BID).
					WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
					WithJSON(provisionRequestBodyMap()()).Expect().Status(http.StatusAccepted)
			})

			DescribeTable("", func(t testCase) {
				bindingFlowTests(t, BID)
			}, []TableEntry{
				Entry("last op in progress", testCase{
					bindingResponseState:            "in progress",
					expectGetBindingOperationStatus: http.StatusOK,
					expectOperationType:             types.CREATE,
					expectOperationState:            types.IN_PROGRESS,
					expectGetBindingStatus:          http.StatusOK,
					expectBindingReady:              false,
				}),
				Entry("last op succeeded", testCase{
					bindingResponseState:            "succeeded",
					expectGetBindingOperationStatus: http.StatusOK,
					expectOperationType:             types.CREATE,
					expectOperationState:            types.SUCCEEDED,
					expectGetBindingStatus:          http.StatusOK,
					expectBindingReady:              true,
				}),
				Entry("last op failed", testCase{
					bindingResponseState:            "failed",
					expectGetBindingOperationStatus: http.StatusOK,
					expectOperationType:             types.CREATE,
					expectOperationState:            types.FAILED,
					expectGetBindingStatus:          http.StatusNotFound,
				}),
				Entry("last op error", testCase{
					bindingResponseState:            "",
					expectGetBindingOperationStatus: http.StatusInternalServerError,
					expectOperationType:             types.CREATE,
					expectOperationState:            types.IN_PROGRESS,
					expectGetBindingStatus:          http.StatusOK,
					expectBindingReady:              false,
				}),
				Entry("unknown state from broker", testCase{
					bindingResponseState:            "blabla",
					expectGetBindingOperationStatus: http.StatusInternalServerError,
					expectOperationType:             types.CREATE,
					expectOperationState:            types.IN_PROGRESS,
					expectGetBindingStatus:          http.StatusOK,
					expectBindingReady:              false,
				}),
			})
		})

		Context("Unbind", func() {
			BeforeEach(func() {
				brokerServer.BindingHandler = parameterizedHandler(http.StatusCreated, `{}`)
				ctx.SMWithBasic.PUT(smBrokerURL+"/v2/service_instances/"+SID+"/service_bindings/"+BID).
					WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
					WithJSON(provisionRequestBodyMap()()).Expect().Status(http.StatusCreated)

				brokerServer.BindingHandler = parameterizedHandler(http.StatusAccepted, `{}`)
				ctx.SMWithBasic.DELETE(smBrokerURL+"/v2/service_instances/"+SID+"/service_bindings/"+BID).
					WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
					WithJSON(provisionRequestBodyMap()()).Expect().Status(http.StatusAccepted)
			})

			DescribeTable("", func(t testCase) {
				bindingFlowTests(t, BID)
			}, []TableEntry{
				Entry("last op in progress", testCase{
					bindingResponseState:            "in progress",
					expectGetBindingOperationStatus: http.StatusOK,
					expectOperationType:             types.DELETE,
					expectOperationState:            types.IN_PROGRESS,
					expectGetBindingStatus:          http.StatusOK,
					expectBindingReady:              true,
				}),
				Entry("last op succeeded", testCase{
					bindingResponseState:            "succeeded",
					expectGetBindingOperationStatus: http.StatusOK,
					expectOperationType:             types.DELETE,
					expectOperationState:            types.SUCCEEDED,
					expectGetBindingStatus:          http.StatusNotFound,
				}),
				Entry("last op succeeded status gone", testCase{
					bindingResponseState:            "",
					expectGetBindingOperationStatus: http.StatusGone,
					expectOperationType:             types.DELETE,
					expectOperationState:            types.SUCCEEDED,
					expectGetBindingStatus:          http.StatusNotFound,
				}),
				Entry("last op failed", testCase{
					bindingResponseState:            "failed",
					expectGetBindingOperationStatus: http.StatusOK,
					expectOperationType:             types.DELETE,
					expectOperationState:            types.FAILED,
					expectGetBindingStatus:          http.StatusOK,
					expectBindingReady:              true,
				}),
				Entry("last op error", testCase{
					bindingResponseState:            "",
					expectGetBindingOperationStatus: http.StatusInternalServerError,
					expectOperationType:             types.DELETE,
					expectOperationState:            types.IN_PROGRESS,
					expectGetBindingStatus:          http.StatusOK,
					expectBindingReady:              true,
				}),
			})
		})

	})
})

func bindingFlowTests(t testCase, BID string) {
	brokerServer.BindingLastOpHandler = parameterizedHandler(t.expectGetBindingOperationStatus, fmt.Sprintf(`{"state": "%s"}`, t.bindingResponseState))
	ctx.
		SMWithBasic.GET(smBrokerURL+"/v2/service_instances/"+SID+"/service_bindings/"+BID+"/last_operation").
		WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
		Expect().
		Status(t.expectGetBindingOperationStatus)

	verifyOperationExists(operationExpectations{
		Type:         t.expectOperationType,
		State:        t.expectOperationState,
		ResourceID:   BID,
		ResourceType: "/v1/service_bindings",
		ExternalID:   "",
	})
	bindingResponse := ctx.SMWithOAuth.GET("/v1/service_bindings/"+BID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
		Expect().Status(t.expectGetBindingStatus)

	if t.expectGetBindingStatus == http.StatusOK {
		bindingResponse.JSON().Object().Value("ready").Boolean().Equal(t.expectBindingReady)
	}
}
