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
	"context"
	"fmt"
	"net/http"

	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/test"
	"github.com/Peripli/service-manager/test/common"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/gavv/httpexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Provision", func() {
	DescribeTable("call to broker with invalid request",
		func(serviceInstanceFunc func() map[string]interface{}, expectedStatusCode, expectedGetInstanceStatusCode int) {
			brokerServer.ServiceInstanceHandler = parameterizedHandler(http.StatusCreated, `{}`)
			serviceInstance := serviceInstanceFunc()

			ctx.SMWithBasic.PUT(smBrokerURL+"/v2/service_instances/"+SID).
				WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				WithJSON(serviceInstance).Expect().Status(expectedStatusCode)

			ctx.SMWithOAuth.GET(web.ServiceInstancesURL + "/" + SID).
				Expect().Status(expectedGetInstanceStatusCode)

			verifyOperationDoesNotExist(SID, "create")
		},
		Entry("when service_id is unknown to SM",
			provisionRequestBodyMapWith("service_id", "abcd1234"),
			http.StatusNotFound,
			http.StatusNotFound),
		Entry("when plan_id is unknown to SM",
			provisionRequestBodyMapWith("plan_id", "abcd1234"),
			http.StatusNotFound,
			http.StatusNotFound),
		Entry("when service_id is missing",
			provisionRequestBodyMap("service_id"),
			http.StatusBadRequest,
			http.StatusNotFound),
		Entry("when plan_id is missing",
			provisionRequestBodyMap("plan_id"),
			http.StatusBadRequest,
			http.StatusNotFound),
		Entry("when plan is not visible",
			provisionRequestBodyMapWith("plan_id", plan3CatalogID),
			http.StatusNotFound,
			http.StatusNotFound),
	)

	DescribeTable("call to broker with invalid response",
		func(brokerHandler func(http.ResponseWriter, *http.Request), expectedStatusCode int, expectedDescriptionPattern string) {
			brokerServer.ServiceInstanceHandler = brokerHandler
			expectedDescription := fmt.Sprintf(expectedDescriptionPattern, brokerName)
			response := ctx.SMWithBasic.PUT(smBrokerURL+"/v2/service_instances/"+SID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				WithJSON(provisionRequestBodyMap()()).Expect().Status(expectedStatusCode)
			if expectedStatusCode > 399 {
				response.JSON().Object().Value("description").String().Contains(expectedDescription)
			}
		},
		Entry("should return an OSB compliant error when broker response is not a valid json",
			parameterizedHandler(http.StatusCreated, "[not a json]"),
			http.StatusCreated,
			"Service broker %s responded with invalid JSON: [not a json]",
		),
		Entry("should return the broker's response when broker response is valid json which is not an object",
			parameterizedHandler(http.StatusOK, "3"),
			http.StatusOK,
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

	DescribeTable("call to broker with valid request",
		func(serviceInstanceFunc func() map[string]interface{}, brokerResponseCode int, brokerResponseBody string, instanceExpectations map[string]interface{}, operationExpectations operationExpectations) {
			brokerServer.ServiceInstanceHandler = parameterizedHandler(brokerResponseCode, brokerResponseBody)
			serviceInstance := serviceInstanceFunc()

			ctx.SMWithBasic.PUT(smBrokerURL+"/v2/service_instances/"+SID).
				WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				WithJSON(serviceInstance).Expect().StatusRange(httpexpect.Status2xx)

			ctx.SMWithOAuth.GET(web.ServiceInstancesURL + "/" + SID).
				Expect().Status(http.StatusOK).JSON().Object().ContainsMap(instanceExpectations)

			verifyOperationExists(operationExpectations)
		},
		Entry("succeeds when request body is complete",
			provisionRequestBodyMap(),
			http.StatusCreated,
			`{}`,
			map[string]interface{}{
				"id":   SID,
				"name": "my-db",
			},
			operationExpectations{
				Type:         types.CREATE,
				State:        types.SUCCEEDED,
				ResourceID:   SID,
				ResourceType: "/v1/service_instances",
				ExternalID:   "",
			}),
		Entry("succeeds when context is missing",
			provisionRequestBodyMap("context"),
			http.StatusCreated,
			`{}`,
			map[string]interface{}{
				"id":   SID,
				"name": SID,
			},
			operationExpectations{
				Type:         types.CREATE,
				State:        types.SUCCEEDED,
				ResourceID:   SID,
				ResourceType: "/v1/service_instances",
				ExternalID:   "",
			}),
		Entry("succeeds when organization_guid is missing",
			provisionRequestBodyMap("organization_guid"),
			http.StatusCreated,
			`{}`,
			map[string]interface{}{
				"id":   SID,
				"name": "my-db",
			},
			operationExpectations{
				Type:         types.CREATE,
				State:        types.SUCCEEDED,
				ResourceID:   SID,
				ResourceType: "/v1/service_instances",
				ExternalID:   "",
			}),
		Entry("succeeds when space_guid is missing",
			provisionRequestBodyMap("space_guid"),
			http.StatusCreated,
			`{}`,
			map[string]interface{}{
				"id":   SID,
				"name": "my-db",
			},
			operationExpectations{
				Type:         types.CREATE,
				State:        types.SUCCEEDED,
				ResourceID:   SID,
				ResourceType: "/v1/service_instances",
				ExternalID:   "",
			}),
		Entry("succeeds when parameters are missing",
			provisionRequestBodyMap("parameters"),
			http.StatusCreated,
			`{}`,
			map[string]interface{}{
				"id":   SID,
				"name": "my-db",
			},
			operationExpectations{
				Type:         types.CREATE,
				State:        types.SUCCEEDED,
				ResourceID:   SID,
				ResourceType: "/v1/service_instances",
				ExternalID:   "",
			}),
		Entry("succeeds when maintenance_info is missing",
			provisionRequestBodyMap("maintenance_info"),
			http.StatusCreated,
			`{}`,
			map[string]interface{}{
				"id":   SID,
				"name": "my-db",
			},
			operationExpectations{
				Type:         types.CREATE,
				State:        types.SUCCEEDED,
				ResourceID:   SID,
				ResourceType: "/v1/service_instances",
				ExternalID:   "",
			}),
		Entry("succeeds when broker responds synchronously with 201 CREATED",
			provisionRequestBodyMap(),
			http.StatusCreated,
			`{}`,
			map[string]interface{}{
				"id":   SID,
				"name": "my-db",
			},
			operationExpectations{
				Type:         types.CREATE,
				State:        types.SUCCEEDED,
				ResourceID:   SID,
				ResourceType: "/v1/service_instances",
				ExternalID:   "",
			}),
		Entry("succeeds when broker responds asynchronously with 202 ACCEPTED with no operation",
			provisionRequestBodyMap(),
			http.StatusAccepted,
			`{}`,
			map[string]interface{}{
				"id":   SID,
				"name": "my-db",
			},
			operationExpectations{
				Type:         types.CREATE,
				State:        types.IN_PROGRESS,
				ResourceID:   SID,
				ResourceType: "/v1/service_instances",
				ExternalID:   "",
			}),
		Entry("succeeds when broker responds asynchronously with 202 ACCEPTED with operation",
			provisionRequestBodyMap(),
			http.StatusAccepted,
			`{"operation":"abc123"}`,
			map[string]interface{}{
				"id":   SID,
				"name": "my-db",
			},
			operationExpectations{
				Type:         types.CREATE,
				State:        types.IN_PROGRESS,
				ResourceID:   SID,
				ResourceType: "/v1/service_instances",
				ExternalID:   "abc123",
			}),
		Entry("succeeds when broker responds synchronously with 200 OK",
			provisionRequestBodyMap(),
			http.StatusOK,
			`{}`,
			map[string]interface{}{
				"id":   SID,
				"name": "my-db",
			},
			operationExpectations{
				Type:         types.CREATE,
				State:        types.SUCCEEDED,
				ResourceID:   SID,
				ResourceType: "/v1/service_instances",
				ExternalID:   "",
			},
		))

	Context("when instance already exists", func() {
		It("returns 409", func() {
			brokerServer.ServiceInstanceHandler = parameterizedHandler(http.StatusOK, `{}`)
			serviceInstance := provisionRequestBodyMap()()

			ctx.SMWithBasic.PUT(smBrokerURL+"/v2/service_instances/"+SID).
				WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				WithJSON(serviceInstance).Expect().Status(http.StatusOK)

			ctx.SMWithOAuth.GET(web.ServiceInstancesURL + "/" + SID).
				Expect().Status(http.StatusOK)

			ctx.SMWithBasic.PUT(smBrokerURL+"/v2/service_instances/"+SID).
				WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				WithJSON(serviceInstance).Expect().Status(http.StatusConflict)
		})
	})

	Context("when call contains query params", func() {
		It("propagates them to the service broker", func() {
			headerKey, headerValue := generateRandomQueryParam()
			brokerServer.ServiceInstanceHandler = queryParameterVerificationHandler(headerKey, headerValue)
			ctx.SMWithBasic.PUT(smBrokerURL+"/v2/service_instances/"+SID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				WithJSON(provisionRequestBodyMap()()).WithQuery(headerKey, headerValue).Expect().Status(http.StatusCreated)
		})
	})

	Context("when broker times out", func() {
		It("should fail with 502", func(done chan<- interface{}) {
			brokerServer.ServiceInstanceHandler = delayingHandler(done)
			assertUnresponsiveBrokerError(ctx.SMWithBasic.PUT(smBrokerURL+"/v2/service_instances/"+SID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				WithJSON(provisionRequestBodyMap()()).Expect())

			ctx.SMWithOAuth.List(web.ServiceInstancesURL).Path("$[*].id").Array().NotContains(SID)

			verifyOperationDoesNotExist(SID, "create")
		}, testTimeout)
	})

	Context("when broker does not exist", func() {
		It("should fail with 401", func() {
			ctx.SMWithBasic.PUT("http://localhost:32123/v1/osb/"+SID+"/v2/service_instances/"+SID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				WithJSON(provisionRequestBodyMap()()).Expect().Status(http.StatusUnauthorized)

			ctx.SMWithOAuth.List(web.ServiceInstancesURL).Path("$[*].id").Array().NotContains(SID)

			verifyOperationDoesNotExist(SID, "create")
		})
	})

	Context("when broker is stopped", func() {
		It("should fail with 502", func() {
			credentials := brokerPlatformCredentialsIDMap[stoppedBrokerID]
			ctx.SMWithBasic.SetBasicCredentials(ctx, credentials.username, credentials.password)

			assertUnresponsiveBrokerError(ctx.SMWithBasic.PUT(smUrlToStoppedBroker+"/v2/service_instances/"+SID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				WithJSON(common.JSONToMap(buildRequestBody(service0CatalogID, plan0CatalogID, "cloudfoundry", "my-db"))).Expect())

			ctx.SMWithOAuth.List(web.ServiceInstancesURL).Path("$[*].id").Array().NotContains(SID)

			verifyOperationDoesNotExist(SID, "create")
		})
	})

	Context("provision instance of plan", func() {
		var platform *types.Platform
		var platformJSON common.Object

		JustBeforeEach(func() {
			brokerServer.ServiceInstanceHandler = parameterizedHandler(http.StatusCreated, `{}`)
			utils.BrokerWithTLS.BrokerServer.ServiceInstanceHandler = parameterizedHandler(http.StatusCreated, `{}`)
			platform = common.RegisterPlatformInSM(platformJSON, ctx.SMWithOAuth, map[string]string{})

			utils.SetAuthContext(ctx.SMWithOAuth).AddPlanVisibilityForPlatform(utils.SelectBroker(&utils.BrokerWithTLS).GetPlanCatalogId(0, 0), platform.ID, organizationGUID)
			utils.SetAuthContext(ctx.SMWithOAuth).AddPlanVisibilityForPlatform(plan1CatalogID, platform.ID, organizationGUID)

			SMWithBasic := &common.SMExpect{Expect: ctx.SM.Builder(func(req *httpexpect.Request) {
				username, password := platform.Credentials.Basic.Username, platform.Credentials.Basic.Password
				req.WithBasicAuth(username, password).WithClient(ctx.HttpClient)
			})}

			username, password := test.RegisterBrokerPlatformCredentials(SMWithBasic, brokerID)
			utils.SetAuthContext(SMWithBasic).RegisterPlatformToBroker(username, password, utils.BrokerWithTLS.ID)
			ctx.SMWithBasic.SetBasicCredentials(ctx, username, password)
		})

		AfterEach(func() {
			err := ctx.SMRepository.Delete(context.TODO(), types.BrokerPlatformCredentialType,
				query.ByField(query.EqualsOperator, "platform_id", platform.ID))
			Expect(err).ToNot(HaveOccurred())

			ctx.SMWithOAuth.DELETE(web.VisibilitiesURL + "?fieldQuery=" + fmt.Sprintf("platform_id eq '%s'", platform.ID))
			ctx.SMWithOAuth.DELETE(web.PlatformsURL + "/" + platform.ID).Expect().Status(http.StatusOK)
		})

		Context("in CF platform", func() {
			BeforeEach(func() {
				platformJSON = common.MakePlatform("cf-platform", "cf-platform", "cloudfoundry", "test-platform-cf")
			})
			Context("when creating a the broker over tls", func() {
				It("creating broker with valid tls settings", func() {
					provisionRequestBody = buildRequestBody(utils.GetServiceCatalogId(0), utils.
						SelectBroker(&utils.BrokerWithTLS).GetPlanCatalogId(0, 0), "cloudfoundry", "my-db")
					ctx.SMWithBasic.PUT(utils.GetBrokerOSBURL(utils.BrokerWithTLS.ID)+"/v2/service_instances/test").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
						WithJSON(common.JSONToMap(provisionRequestBody)).Expect().Status(http.StatusCreated)
				})
			})

			It("should return 404 if plan is not visible in the org", func() {
				ctx.SMWithBasic.PUT(smBrokerURL+"/v2/service_instances/"+SID).
					WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
					WithJSON(provisionRequestBodyMapWith("plan_id", plan2CatalogID)()).
					Expect().Status(http.StatusNotFound)
			})

			It("should return 400 if no context is available", func() {
				ctx.SMWithBasic.PUT(smBrokerURL+"/v2/service_instances/"+SID).
					WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
					WithJSON(provisionRequestBodyMapWith("plan_id", plan1CatalogID, "context")()).
					Expect().Status(http.StatusBadRequest)
			})

			It("should return 400 if organization_id is not in the context", func() {
				body := provisionRequestBodyMapWith("plan_id", plan1CatalogID)()
				body["context"] = "{}"
				ctx.SMWithBasic.PUT(smBrokerURL+"/v2/service_instances/"+SID).
					WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
					WithJSON(body).
					Expect().Status(http.StatusBadRequest)
			})

			It("should return 201 if plan is visible in the org", func() {
				ctx.SMWithBasic.PUT(smBrokerURL+"/v2/service_instances/"+SID).
					WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
					WithJSON(provisionRequestBodyMapWith("plan_id", plan1CatalogID)()).
					Expect().Status(http.StatusCreated)
			})
		})

		Context("in K8S platform", func() {
			BeforeEach(func() {
				platformJSON = common.MakePlatform("k8s-platform", "k8s-platform", "kubernetes", "test-platform-k8s")
			})

			It("should return 404 if plan is not visible in the platform", func() {
				ctx.SMWithBasic.PUT(smBrokerURL+"/v2/service_instances/"+SID).
					WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
					WithJSON(provisionRequestBodyMapWith("plan_id", plan2CatalogID)()).
					Expect().Status(http.StatusNotFound)
			})

			It("should return 201 if plan is visible in the platform", func() {
				ctx.SMWithBasic.PUT(smBrokerURL+"/v2/service_instances/"+SID).
					WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
					WithJSON(provisionRequestBodyMapWith("plan_id", plan1CatalogID)()).
					Expect().Status(http.StatusCreated)
			})
		})
	})
})
