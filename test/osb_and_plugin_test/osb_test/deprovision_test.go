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
	"net/http"

	"github.com/gavv/httpexpect"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web"
)

var _ = Describe("Deprovision", func() {
	Context("when instance is unknown to SM", func() {
		It("does not fail", func() {
			brokerServer.ServiceInstanceHandler = parameterizedHandler(http.StatusOK, `{}`)

			ctx.SMWithOAuth.GET(web.ServiceInstancesURL + "/" + SID).
				Expect().Status(http.StatusNotFound)

			ctx.SMWithBasic.DELETE(smBrokerURL+"/v2/service_instances/"+SID).
				WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				Expect().Status(http.StatusOK)
		})
	})

	Context("when call to slow service broker", func() {
		It("should timeout for service requests", func() {
			brokerServer.ShouldRecordRequests(true)
			done := make(chan struct{}, 1)
			brokerServer.ServiceInstanceHandler = slowResponseHandler(3, done)

			ctx.SMWithBasic.DELETE(smBrokerURL+"/v2/service_instances/123").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				Expect().Status(http.StatusServiceUnavailable).Body().Raw()
			done <- struct{}{}
		})
	})

	DescribeTable("call to broker with invalid request",
		func(query func() string, expectedStatusCode, expectedGetInstanceStatusCode int) {
			brokerServer.ServiceInstanceHandler = parameterizedHandler(http.StatusOK, `{}`)

			ctx.SMWithBasic.DELETE(smBrokerURL+"/v2/service_instances/"+SID).WithQueryString(query()).
				WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				Expect().Status(expectedStatusCode)

			ctx.SMWithOAuth.GET(web.ServiceInstancesURL + "/" + SID).
				Expect().Status(expectedGetInstanceStatusCode)

			verifyOperationExists(operationExpectations{
				Type:         types.DELETE,
				State:        types.SUCCEEDED,
				ResourceID:   SID,
				ResourceType: "/v1/service_instances",
				ExternalID:   "",
			})
		},
		Entry("when service_id is unknown to SM",
			func() string {
				return fmt.Sprintf("service_id=123&plan_id=%s", plan1CatalogID)
			},
			http.StatusOK,
			http.StatusNotFound,
		),
		Entry("when plan_id is unknown to SM",
			func() string {
				return fmt.Sprintf("service_id=%s&plan_id=123", service1CatalogID)
			},
			http.StatusOK,
			http.StatusNotFound,
		),
		Entry("when service_id is missing",
			func() string {
				return fmt.Sprintf("plan_id=%s", plan1CatalogID)
			},
			http.StatusOK,
			http.StatusNotFound,
		),
		Entry("when plan_id is missing",
			func() string {
				return fmt.Sprintf("service_id=%s", service1CatalogID)
			},
			http.StatusOK,
			http.StatusNotFound,
		),
	)

	DescribeTable("call to broker with invalid response",
		func(brokerHandler func(http.ResponseWriter, *http.Request), expectedStatusCode int, expectedDescriptionPattern string) {
			brokerServer.ServiceInstanceHandler = brokerHandler
			expectedDescription := fmt.Sprintf(expectedDescriptionPattern, brokerName)
			ctx.SMWithBasic.DELETE(smBrokerURL+"/v2/service_instances/"+SID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				Expect().Status(expectedStatusCode).
				JSON().Object().Value("description").String().Contains(expectedDescription)
		},
		Entry("should return an OSB compliant error when broker response is not a valid json",
			parameterizedHandler(http.StatusBadRequest, "[not a json]"),
			http.StatusBadRequest,
			"Service broker %s responded with invalid JSON: [not a json]",
		),
		Entry("should return the broker's response when broker response is valid json which is not an object",
			parameterizedHandler(http.StatusBadRequest, "3"),
			http.StatusBadRequest,
			"Service broker %s failed with: 3",
		),
		Entry("should assign broker's response body as description when broker response is error without description",
			parameterizedHandler(http.StatusBadRequest, `{"error": "ErrorType"}`),
			http.StatusBadRequest,
			`Service broker %s failed with: {"error": "ErrorType"}`,
		),
		Entry("should return it in description when broker response is JSON array",
			parameterizedHandler(http.StatusInternalServerError, `[1,2,3]`),
			http.StatusInternalServerError,
			`Service broker %s failed with: [1,2,3]`,
		),
	)

	DescribeTable("call to broker with valid request", func(brokerResponseStatusCode int, brokerResponseBody string, instanceExists bool, getInstanceStatusCode int, expectations operationExpectations) {
		brokerServer.ServiceInstanceHandler = parameterizedHandler(http.StatusOK, `{}`)
		if instanceExists {
			ctx.SMWithBasic.PUT(smBrokerURL+"/v2/service_instances/"+SID).
				WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				WithJSON(provisionRequestBodyMap()()).Expect().StatusRange(httpexpect.Status2xx)
		}
		brokerServer.ServiceInstanceHandler = parameterizedHandler(brokerResponseStatusCode, brokerResponseBody)
		ctx.SMWithBasic.DELETE(smBrokerURL+"/v2/service_instances/"+SID).
			WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
			Expect().Status(brokerResponseStatusCode)

		ctx.SMWithOAuth.GET(web.ServiceInstancesURL + "/" + SID).
			Expect().Status(getInstanceStatusCode)

		verifyOperationExists(expectations)
	},
		Entry("deletes the instance and creates operation delete succeeded when instance exists and broker returns 200 OK with operation",
			http.StatusOK,
			`{"operation":"abc123"}`,
			true,
			http.StatusNotFound,
			operationExpectations{
				Type:         types.DELETE,
				State:        types.SUCCEEDED,
				ResourceID:   SID,
				ResourceType: "/v1/service_instances",
				ExternalID:   "abc123",
			},
		),
		Entry("deletes the instance and creates operation delete succeeded when instance does not exist and broker returns 200 OK with operation",
			http.StatusOK,
			`{"operation":"abc123"}`,
			false,
			http.StatusNotFound,
			operationExpectations{
				Type:         types.DELETE,
				State:        types.SUCCEEDED,
				ResourceID:   SID,
				ResourceType: "/v1/service_instances",
				ExternalID:   "abc123",
			},
		),
		Entry("deletes the instance and creates operation delete succeeded when instance exists and broker returns 200 OK with no operation",
			http.StatusOK,
			`{}`,
			true,
			http.StatusNotFound,
			operationExpectations{
				Type:         types.DELETE,
				State:        types.SUCCEEDED,
				ResourceID:   SID,
				ResourceType: "/v1/service_instances",
				ExternalID:   "",
			},
		),
		Entry("deletes the instance and creates operation delete succeeded when instance does not exist and broker returns 200 OK with no operation",
			http.StatusOK,
			`{}`,
			false,
			http.StatusNotFound,
			operationExpectations{
				Type:         types.DELETE,
				State:        types.SUCCEEDED,
				ResourceID:   SID,
				ResourceType: "/v1/service_instances",
				ExternalID:   "",
			},
		),

		Entry("deletes the instance and creates operation delete succeeded when instance exists and broker returns 410 GONE with operation",
			http.StatusGone,
			`{"operation":"abc123"}`,
			true,
			http.StatusNotFound,
			operationExpectations{
				Type:         types.DELETE,
				State:        types.SUCCEEDED,
				ResourceID:   SID,
				ResourceType: "/v1/service_instances",
				ExternalID:   "abc123",
			},
		),
		Entry("deletes the instance and creates operation delete succeeded when instance does not exist and broker returns 410 GONE with operation",
			http.StatusGone,
			`{"operation":"abc123"}`,
			false,
			http.StatusNotFound,
			operationExpectations{
				Type:         types.DELETE,
				State:        types.SUCCEEDED,
				ResourceID:   SID,
				ResourceType: "/v1/service_instances",
				ExternalID:   "abc123",
			},
		),
		Entry("deletes the instance and creates operation delete succeeded when instance exists and broker returns 410 GONE with no operation",
			http.StatusGone,
			`{}`,
			true,
			http.StatusNotFound,
			operationExpectations{
				Type:         types.DELETE,
				State:        types.SUCCEEDED,
				ResourceID:   SID,
				ResourceType: "/v1/service_instances",
				ExternalID:   "",
			},
		),
		Entry("creates operation delete succeeded when instance does not exist and broker returns 410 GONE with no operation",
			http.StatusGone,
			`{}`,
			false,
			http.StatusNotFound,
			operationExpectations{
				Type:         types.DELETE,
				State:        types.SUCCEEDED,
				ResourceID:   SID,
				ResourceType: "/v1/service_instances",
				ExternalID:   "",
			},
		),

		Entry("does not delete the instance and creates operation delete in progress when instance exists and broker returns 202 ACCEPTED with operation",
			http.StatusAccepted,
			`{"operation":"abc123"}`,
			true,
			http.StatusOK,
			operationExpectations{
				Type:         types.DELETE,
				State:        types.IN_PROGRESS,
				ResourceID:   SID,
				ResourceType: "/v1/service_instances",
				ExternalID:   "abc123",
			},
		),
		Entry("creates operation delete in progress when instance does not exist and broker returns 202 ACCEPTED with operation",
			http.StatusAccepted,
			`{"operation":"abc123"}`,
			false,
			http.StatusNotFound,
			operationExpectations{
				Type:         types.DELETE,
				State:        types.IN_PROGRESS,
				ResourceID:   SID,
				ResourceType: "/v1/service_instances",
				ExternalID:   "abc123",
			},
		),
		Entry("does not delete the instance and creates operation delete in progress when instance exists and broker returns 202 ACCEPTED with no operation",
			http.StatusAccepted,
			`{}`,
			true,
			http.StatusOK,
			operationExpectations{
				Type:         types.DELETE,
				State:        types.IN_PROGRESS,
				ResourceID:   SID,
				ResourceType: "/v1/service_instances",
				ExternalID:   "",
			},
		),
		Entry("creates operation delete in progress when instance does not exist and broker returns 202 ACCEPETED with no operation",
			http.StatusAccepted,
			`{}`,
			false,
			http.StatusNotFound,
			operationExpectations{
				Type:         types.DELETE,
				State:        types.IN_PROGRESS,
				ResourceID:   SID,
				ResourceType: "/v1/service_instances",
				ExternalID:   "",
			},
		))

	Context("when broker responds with an error", func() {
		It("should fail", func() {
			brokerServer.ServiceInstanceHandler = parameterizedHandler(http.StatusInternalServerError, `internal server error`)
			assertFailingBrokerError(ctx.SMWithBasic.DELETE(smBrokerURL+"/v2/service_instances/"+SID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				Expect(), http.StatusInternalServerError, "internal server error")

			ctx.SMWithOAuth.List(web.ServiceInstancesURL).Path("$[*].id").Array().NotContains(SID)

			verifyOperationDoesNotExist(SID, "delete")
		})
	})

	Context("when call contains query params", func() {
		It("propagates them to the service broker", func() {
			headerKey, headerValue := generateRandomQueryParam()
			brokerServer.ServiceInstanceHandler = queryParameterVerificationHandler(headerKey, headerValue)
			ctx.SMWithBasic.DELETE(smBrokerURL+"/v2/service_instances/"+SID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				WithQuery(headerKey, headerValue).Expect().Status(http.StatusOK)
		})
	})

	Context("when broker times out", func() {
		It("should fail with 502", func(done chan<- interface{}) {
			brokerServer.ServiceInstanceHandler = delayingHandler(done)
			assertUnresponsiveBrokerError(ctx.SMWithBasic.DELETE(smBrokerURL+"/v2/service_instances/"+SID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				Expect())

			ctx.SMWithOAuth.List(web.ServiceInstancesURL).Path("$[*].id").Array().NotContains(SID)

			verifyOperationDoesNotExist(SID, "delete")
		}, testTimeout)
	})

	Context("when broker does not exist", func() {
		It("should fail with 401", func() {
			ctx.SMWithBasic.DELETE("http://localhost:32123/v1/osb/"+SID+"/v2/service_instances/"+SID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				Expect().Status(http.StatusUnauthorized)

			ctx.SMWithOAuth.List(web.ServiceInstancesURL).Path("$[*].id").Array().NotContains(SID)

			verifyOperationDoesNotExist(SID, "delete")
		})
	})

	Context("when broker is stopped", func() {
		It("should fail with 502", func() {
			credentials := brokerPlatformCredentialsIDMap[stoppedBrokerID]
			ctx.SMWithBasic.SetBasicCredentials(ctx, credentials.username, credentials.password)

			assertUnresponsiveBrokerError(ctx.SMWithBasic.DELETE(smUrlToStoppedBroker+"/v2/service_instances/"+SID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				Expect())

			ctx.SMWithOAuth.List(web.ServiceInstancesURL).Path("$[*].id").Array().NotContains(SID)

			verifyOperationDoesNotExist(SID, "delete")
		})
	})

	Context("broker platform credentials check", func() {
		BeforeEach(func() {
			brokerServer.ServiceInstanceHandler = parameterizedHandler(http.StatusCreated, `{}`)

			ctx.SMWithBasic.PUT(smBrokerURL+"/v2/service_instances/"+SID).
				WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				WithJSON(provisionRequestBodyMap()()).Expect().Status(http.StatusCreated)
		})

		Context("deprovision with invalid credentials", func() {
			BeforeEach(func() {
				ctx.SMWithBasic.SetBasicCredentials(ctx, "test", "test")
			})

			It("should return 401", func() {
				brokerServer.ServiceInstanceHandler = parameterizedHandler(http.StatusOK, `{}`)
				ctx.SMWithBasic.DELETE(smBrokerURL+"/v2/service_instances/"+SID).
					WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
					Expect().Status(http.StatusUnauthorized)
			})
		})
	})

	Context("deprovision operation is handled by external plugin", func() {
		BeforeEach(func() {
			brokerServer.ServiceInstanceHandler = parameterizedHandler(http.StatusOK, `{}`)
			ctx.SMWithBasic.PUT(smBrokerURL+"/v2/service_instances/"+SID).
				WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				WithJSON(provisionRequestBodyMap()()).Expect().StatusRange(httpexpect.Status2xx)

			shouldSaveOperationInContext = true
			brokerServer.ServiceInstanceHandler = parameterizedHandler(http.StatusAccepted, `{"operation":"abc123"}`)
		})

		It("osb store plugin should not create delete operation", func() {
			ctx.SMWithBasic.DELETE(smBrokerURL+"/v2/service_instances/"+SID).
				WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				Expect().Status(http.StatusAccepted)

			verifyOperationDoesNotExist(SID, string(types.DELETE))
		})
	})
})
