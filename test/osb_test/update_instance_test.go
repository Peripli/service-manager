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

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/gavv/httpexpect"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
)

var _ = Describe("Update", func() {
	Context("when instance is unknown to SM", func() {
		It("does not fail", func() {
			brokerServer.ServiceInstanceHandler = parameterizedHandler(http.StatusOK, `{}`)

			ctx.SMWithOAuth.GET(web.ServiceInstancesURL + "/" + SID).
				Expect().Status(http.StatusNotFound)

			ctx.SMWithBasic.PATCH(smBrokerURL+"/v2/service_instances/"+SID).
				WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				WithJSON(updateRequestBodyMap()()).Expect().Status(http.StatusOK)
		})
	})

	DescribeTable("call to broker with invalid request",
		func(requestBody func() map[string]interface{}, expectedStatusCode, expectedGetInstanceStatusCode int) {
			brokerServer.ServiceInstanceHandler = parameterizedHandler(http.StatusOK, `{}`)

			ctx.SMWithBasic.PUT(smBrokerURL+"/v2/service_instances/"+SID).
				WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				WithJSON(provisionRequestBodyMap()()).Expect().Status(http.StatusOK)

			ctx.SMWithBasic.PATCH(smBrokerURL+"/v2/service_instances/"+SID).
				WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				WithJSON(requestBody()).Expect().Status(expectedStatusCode)

			ctx.SMWithOAuth.GET(web.ServiceInstancesURL + "/" + SID).
				Expect().Status(expectedGetInstanceStatusCode)

			verifyOperationDoesNotExist(SID, "UPDATE")
		},
		Entry("when service_id is unknown to SM",
			updateRequestBodyMapWith("service_id", "abcd1234"),
			http.StatusNotFound,
			http.StatusOK),
		Entry("when plan_id is unknown to SM",
			updateRequestBodyMapWith("plan_id", "abcd1234"),
			http.StatusNotFound,
			http.StatusOK),
		Entry("when service_id is missing",
			updateRequestBodyMap("service_id"),
			http.StatusBadRequest,
			http.StatusOK),
	)

	DescribeTable("call to broker with invalid response",
		func(brokerHandler func(http.ResponseWriter, *http.Request), expectedStatusCode int, expectedDescriptionPattern string) {
			brokerServer.ServiceInstanceHandler = brokerHandler
			expectedDescription := fmt.Sprintf(expectedDescriptionPattern, brokerName)
			ctx.SMWithBasic.PATCH(smBrokerURL+"/v2/service_instances/"+SID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				WithJSON(updateRequestBodyMap()()).Expect().Status(expectedStatusCode).
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

	DescribeTable("call to broker with valid request", func(updateRequest func() map[string]interface{}, brokerResponseStatusCode int, brokerResponseBody string, instanceExpectations func() map[string]interface{}, operationExpectations operationExpectations) {
		brokerServer.ServiceInstanceHandler = parameterizedHandler(http.StatusOK, `{}`)
		ctx.SMWithBasic.PUT(smBrokerURL+"/v2/service_instances/"+SID).
			WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
			WithJSON(provisionRequestBodyMap()()).Expect().StatusRange(httpexpect.Status2xx)

		brokerServer.ServiceInstanceHandler = parameterizedHandler(brokerResponseStatusCode, brokerResponseBody)
		ctx.SMWithBasic.PATCH(smBrokerURL+"/v2/service_instances/"+SID).
			WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).WithJSON(updateRequest()).
			Expect().Status(brokerResponseStatusCode)

		ctx.SMWithOAuth.GET(web.ServiceInstancesURL + "/" + SID).
			Expect().Status(http.StatusOK).JSON().Object().ContainsMap(instanceExpectations())

		verifyOperationExists(operationExpectations)
	},
		Entry("updates the instance and creates operation update succeeded when update contains new plan and maintenance_info and broker responds with 200 OK and operation in body",
			updateRequestBodyMap(),
			http.StatusOK,
			`{"operation":"abc123"}`,
			func() map[string]interface{} {
				return map[string]interface{}{
					"service_plan_id": findSMPlanIDForCatalogPlanID(plan2CatalogID),
					"maintenance_info": map[string]interface{}{
						"version": "new",
					},
				}
			},
			operationExpectations{
				Type:         types.UPDATE,
				State:        types.SUCCEEDED,
				ResourceID:   SID,
				ResourceType: "/v1/service_instances",
				ExternalID:   "abc123",
			},
		),
		Entry("updates the instance and creates operation update succeeded when update contains new plan and maintenance_info and broker responds with 202 ACCEPTED and operation in body",
			updateRequestBodyMap(),
			http.StatusAccepted,
			`{"operation":"abc123"}`,
			func() map[string]interface{} {
				return map[string]interface{}{
					"service_plan_id": findSMPlanIDForCatalogPlanID(plan2CatalogID),
					"maintenance_info": map[string]interface{}{
						"version": "new",
					},
				}
			},
			operationExpectations{
				Type:         types.UPDATE,
				State:        types.IN_PROGRESS,
				ResourceID:   SID,
				ResourceType: "/v1/service_instances",
				ExternalID:   "abc123",
			},
		),
		Entry("updates the instance and creates operation update succeeded when update contains new plan and maintenance_info and broker responds with 200 OK and no operation in body",
			updateRequestBodyMap(),
			http.StatusOK,
			`{}`,
			func() map[string]interface{} {
				return map[string]interface{}{
					"service_plan_id": findSMPlanIDForCatalogPlanID(plan2CatalogID),
					"maintenance_info": map[string]interface{}{
						"version": "new",
					},
				}
			},
			operationExpectations{
				Type:         types.UPDATE,
				State:        types.SUCCEEDED,
				ResourceID:   SID,
				ResourceType: "/v1/service_instances",
				ExternalID:   "",
			},
		),
		Entry("updates the instance and creates operation update succeeded when update contains new plan and maintenance_info and broker responds with 202 ACCEPTED and no operation in body",
			updateRequestBodyMap(),
			http.StatusAccepted,
			`{}`,
			func() map[string]interface{} {
				return map[string]interface{}{
					"service_plan_id": findSMPlanIDForCatalogPlanID(plan2CatalogID),
					"maintenance_info": map[string]interface{}{
						"version": "new",
					},
				}
			},
			operationExpectations{
				Type:         types.UPDATE,
				State:        types.IN_PROGRESS,
				ResourceID:   SID,
				ResourceType: "/v1/service_instances",
				ExternalID:   "",
			},
		),

		Entry("updates the instance and creates operation update succeeded when update does not contain plan_id and broker responds with 200 OK and operation in body",
			updateRequestBodyMap("plan_id"),
			http.StatusOK,
			`{"operation":"abc123"}`,
			func() map[string]interface{} {
				return map[string]interface{}{
					"service_plan_id": findSMPlanIDForCatalogPlanID(plan1CatalogID),
					"maintenance_info": map[string]interface{}{
						"version": "new",
					},
				}
			},
			operationExpectations{
				Type:         types.UPDATE,
				State:        types.SUCCEEDED,
				ResourceID:   SID,
				ResourceType: "/v1/service_instances",
				ExternalID:   "abc123",
			},
		),
		Entry("updates the instance and creates operation update succeeded when update does not contain plan_id and broker responds with 200 OK and operation in body",
			updateRequestBodyMap("maintenance_info"),
			http.StatusOK,
			`{"operation":"abc123"}`,
			func() map[string]interface{} {
				return map[string]interface{}{
					"service_plan_id": findSMPlanIDForCatalogPlanID(plan2CatalogID),
					"maintenance_info": map[string]interface{}{
						"version": "old",
					},
				}
			},
			operationExpectations{
				Type:         types.UPDATE,
				State:        types.SUCCEEDED,
				ResourceID:   SID,
				ResourceType: "/v1/service_instances",
				ExternalID:   "abc123",
			},
		),
		Entry("updates the instance and creates operation update succeeded when update does not contain organization_id and broker responds with 200 OK and operation in body",
			updateRequestBodyMap("organization_id", "space_id", "context", "parameters", "previous_values"),
			http.StatusOK,
			`{"operation":"abc123"}`,
			func() map[string]interface{} {
				return map[string]interface{}{
					"service_plan_id": findSMPlanIDForCatalogPlanID(plan2CatalogID),
					"maintenance_info": map[string]interface{}{
						"version": "new",
					},
				}
			},
			operationExpectations{
				Type:         types.UPDATE,
				State:        types.SUCCEEDED,
				ResourceID:   SID,
				ResourceType: "/v1/service_instances",
				ExternalID:   "abc123",
			},
		),
	)
})
