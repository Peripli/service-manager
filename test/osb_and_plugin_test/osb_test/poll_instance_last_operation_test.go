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
	"github.com/Peripli/service-manager/pkg/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Get Service Instance Last Operation", func() {
	const brokerLastOperationStateFailed = `{"state":"failed", "description": "an error happened"}`
	BeforeEach(func() {
		brokerServer.ServiceInstanceLastOpHandler = parameterizedHandler(http.StatusOK, `{}`)
	})

	Context("when operation is unknown to SM", func() {
		It("does not fail", func() {
			byResourceID := query.ByField(query.EqualsOperator, "resource_id", SID)
			objectList, err := ctx.SMRepository.List(context.TODO(), types.OperationType, byResourceID)
			Expect(err).ToNot(HaveOccurred())
			Expect(objectList.Len()).To(Equal(0))

			ctx.SMWithBasic.GET(smBrokerURL+"/v2/service_instances/"+SID+"/last_operation").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				Expect().Status(http.StatusOK)
		})
	})

	DescribeTable("call to broker with invalid response",
		func(brokerHandler func(http.ResponseWriter, *http.Request), expectedStatusCode int, expectedDescriptionPattern string) {
			brokerServer.ServiceInstanceLastOpHandler = brokerHandler
			expectedDescription := fmt.Sprintf(expectedDescriptionPattern, brokerName)
			ctx.SMWithBasic.GET(smBrokerURL+"/v2/service_instances/"+SID+"/last_operation").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
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

	Context("when polling CREATE for which operation exists", func() {
		BeforeEach(func() {
			By(fmt.Sprintf("Creating service instance with id %s", SID))
			brokerServer.ServiceInstanceHandler = parameterizedHandler(http.StatusAccepted, `{}`)
			ctx.SMWithBasic.PUT(smBrokerURL+"/v2/service_instances/"+SID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				WithJSON(provisionRequestBodyMap()()).Expect().Status(http.StatusAccepted)

			By(fmt.Sprintf("Verifying service instance with id %s is created with ready=false", SID))
			ctx.SMWithOAuth.GET("/v1/service_instances/"+SID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				Expect().Status(http.StatusOK).JSON().Object().Value("ready").Boolean().Equal(false)

			By(fmt.Sprintf("Verifying operation create in progress for service instance with id %s was created", SID))
			verifyOperationExists(operationExpectations{
				Type:         types.CREATE,
				State:        types.IN_PROGRESS,
				ResourceID:   SID,
				ResourceType: "/v1/service_instances",
				ExternalID:   "",
			})
		})

		Context("that has succeeded", func() {
			BeforeEach(func() {
				brokerServer.ServiceInstanceLastOpHandler = parameterizedHandler(http.StatusOK, `{"state":"succeeded"}`)
			})

			It("updates the operation to succeeded and updates the instance to ready=true", func() {
				By(fmt.Sprintf("Getting last operation for service instance with id %s", SID))
				ctx.SMWithBasic.GET(smBrokerURL+"/v2/service_instances/"+SID+"/last_operation").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
					Expect().Status(http.StatusOK)

				By(fmt.Sprintf("Verifying service instance with id %s is updated to ready=true", SID))
				ctx.SMWithOAuth.GET("/v1/service_instances/"+SID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
					Expect().Status(http.StatusOK).JSON().Object().Value("ready").Boolean().Equal(true)

				By(fmt.Sprintf("Verifying create operation for service instance with id %s is updated to succeeded", SID))
				verifyOperationExists(operationExpectations{
					Type:         types.CREATE,
					State:        types.SUCCEEDED,
					ResourceID:   SID,
					ResourceType: "/v1/service_instances",
					ExternalID:   "",
				})
			})

			Context("when instance does not exist", func() {
				BeforeEach(func() {
					brokerServer.ServiceInstanceHandler = parameterizedHandler(http.StatusOK, `{}`)

					ctx.SMWithOAuthForTenant.DELETE("/v1/service_instances/"+SID).
						WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
						WithQuery("async", false).
						Expect().Status(http.StatusOK)

					ctx.SMWithOAuth.GET("/v1/service_instances/"+SID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
						Expect().Status(http.StatusNotFound)
				})

				It("returns 200", func() {
					By(fmt.Sprintf("Getting last operation for service instance with id %s", SID))
					ctx.SMWithBasic.GET(smBrokerURL+"/v2/service_instances/"+SID+"/last_operation").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
						Expect().Status(http.StatusOK).Body().Equal(`{"state":"succeeded"}`)
				})
			})
		})

		Context("that has failed", func() {
			BeforeEach(func() {
				brokerServer.ServiceInstanceLastOpHandler = parameterizedHandler(http.StatusOK, brokerLastOperationStateFailed)
			})

			It("updates the operation to failed and deletes the instance", func() {
				By(fmt.Sprintf("Getting last operation for service instance with id %s", SID))
				ctx.SMWithBasic.GET(smBrokerURL+"/v2/service_instances/"+SID+"/last_operation").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
					Expect().Status(http.StatusOK)

				By(fmt.Sprintf("Verifying service instance with id %s is deleted", SID))
				ctx.SMWithOAuth.GET("/v1/service_instances/"+SID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
					Expect().Status(http.StatusNotFound)

				By(fmt.Sprintf("Verifying create operation for service instance with id %s is updated to failed", SID))
				verifyOperationExists(operationExpectations{
					Type:         types.CREATE,
					State:        types.FAILED,
					ResourceID:   SID,
					ResourceType: "/v1/service_instances",
					ExternalID:   "",
					Errors:       []byte("an error happened"),
				})
			})
		})

		Context("that is in progress", func() {
			BeforeEach(func() {
				brokerServer.ServiceInstanceLastOpHandler = parameterizedHandler(http.StatusOK, `{"state":"in progress"}`)
			})

			It("does not update the operation and does not update the instance", func() {
				By(fmt.Sprintf("Getting last operation for service instance with id %s", SID))
				ctx.SMWithBasic.GET(smBrokerURL+"/v2/service_instances/"+SID+"/last_operation").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
					Expect().Status(http.StatusOK)

				By(fmt.Sprintf("Verifying service instance with id %s is not changed", SID))
				ctx.SMWithOAuth.GET("/v1/service_instances/"+SID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
					Expect().Status(http.StatusOK).JSON().Object().Value("ready").Boolean().Equal(false)

				By(fmt.Sprintf("Verifying create operation for service instance with id %s is in progress", SID))
				verifyOperationExists(operationExpectations{
					Type:         types.CREATE,
					State:        types.IN_PROGRESS,
					ResourceID:   SID,
					ResourceType: "/v1/service_instances",
					ExternalID:   "",
				})
			})
		})

		Context("that returns no state", func() {
			BeforeEach(func() {
				brokerServer.ServiceInstanceLastOpHandler = parameterizedHandler(http.StatusOK, `{"state":"in progress"}`)
			})

			It("does not update the operation and does not update the instance", func() {
				By(fmt.Sprintf("Getting last operation for service instance with id %s", SID))
				ctx.SMWithBasic.GET(smBrokerURL+"/v2/service_instances/"+SID+"/last_operation").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
					Expect().Status(http.StatusOK)

				By(fmt.Sprintf("Verifying service instance with id %s is not changed", SID))
				ctx.SMWithOAuth.GET("/v1/service_instances/"+SID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
					Expect().Status(http.StatusOK).JSON().Object().Value("ready").Boolean().Equal(false)

				By(fmt.Sprintf("Verifying create operation for service instance with id %s is in progress", SID))
				verifyOperationExists(operationExpectations{
					Type:         types.CREATE,
					State:        types.IN_PROGRESS,
					ResourceID:   SID,
					ResourceType: "/v1/service_instances",
					ExternalID:   "",
				})
			})
		})

		Context("that returns 410 gone", func() {
			BeforeEach(func() {
				brokerServer.ServiceInstanceLastOpHandler = parameterizedHandler(http.StatusGone, `{}`)
			})

			It("returns 410", func() {
				By(fmt.Sprintf("Getting last operation for service instance with id %s", SID))
				ctx.SMWithBasic.GET(smBrokerURL+"/v2/service_instances/"+SID+"/last_operation").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
					Expect().Status(http.StatusGone)

				By(fmt.Sprintf("Verifying service instance with id %s is not updated to ready=true", SID))
				ctx.SMWithOAuth.GET("/v1/service_instances/"+SID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
					Expect().Status(http.StatusOK).JSON().Object().Value("ready").Boolean().Equal(false)

				By(fmt.Sprintf("Verifying create operation for service instance with id %s is not updated to succeeded", SID))
				verifyOperationExists(operationExpectations{
					Type:         types.CREATE,
					State:        types.IN_PROGRESS,
					ResourceID:   SID,
					ResourceType: "/v1/service_instances",
					ExternalID:   "",
				})
			})
		})
	})

	Context("when polling UPDATE for which operation exists", func() {
		BeforeEach(func() {
			By(fmt.Sprintf("Creating service instance with id %s", SID))
			brokerServer.ServiceInstanceHandler = parameterizedHandler(http.StatusOK, `{}`)
			ctx.SMWithBasic.PUT(smBrokerURL+"/v2/service_instances/"+SID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				WithJSON(provisionRequestBodyMap()()).Expect().Status(http.StatusOK)

			By(fmt.Sprintf("Updating service instance with id %s", SID))
			brokerServer.ServiceInstanceHandler = parameterizedHandler(http.StatusAccepted, `{}`)
			ctx.SMWithBasic.PATCH(smBrokerURL+"/v2/service_instances/"+SID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				WithJSON(updateRequestBodyMap()()).Expect().Status(http.StatusAccepted)

			By(fmt.Sprintf("Verifying service instance with id %s is updated with new plan id", SID))
			ctx.SMWithOAuth.GET("/v1/service_instances/"+SID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				Expect().Status(http.StatusOK).JSON().Object().Value("service_plan_id").String().Equal(findSMPlanIDForCatalogPlanID(plan2CatalogID))

			By(fmt.Sprintf("Verifying operation update in progress for service instance with id %s was created", SID))
			verifyOperationExists(operationExpectations{
				Type:         types.UPDATE,
				State:        types.IN_PROGRESS,
				ResourceID:   SID,
				ResourceType: "/v1/service_instances",
				ExternalID:   "",
			})
		})

		Context("that has succeeded", func() {
			BeforeEach(func() {
				brokerServer.ServiceInstanceLastOpHandler = parameterizedHandler(http.StatusOK, `{"state":"succeeded"}`)
			})

			It("updates the operation to succeeded", func() {
				By(fmt.Sprintf("Getting last operation for service instance with id %s", SID))
				ctx.SMWithBasic.GET(smBrokerURL+"/v2/service_instances/"+SID+"/last_operation").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
					Expect().Status(http.StatusOK)

				By(fmt.Sprintf("Verifying service instance with id %s is updated with new plan id", SID))
				ctx.SMWithOAuth.GET("/v1/service_instances/"+SID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
					Expect().Status(http.StatusOK).JSON().Object().Value("service_plan_id").String().Equal(findSMPlanIDForCatalogPlanID(plan2CatalogID))

				By(fmt.Sprintf("Verifying update operation for service instance with id %s is updated to succeeded", SID))
				verifyOperationExists(operationExpectations{
					Type:         types.UPDATE,
					State:        types.SUCCEEDED,
					ResourceID:   SID,
					ResourceType: "/v1/service_instances",
					ExternalID:   "",
				})
			})
		})

		Context("that has failed", func() {
			BeforeEach(func() {
				brokerServer.ServiceInstanceLastOpHandler = parameterizedHandler(http.StatusOK, brokerLastOperationStateFailed)
			})

			It("updates the operation to failed and rollbacks the instance", func() {
				By(fmt.Sprintf("Getting last operation for service instance with id %s", SID))
				ctx.SMWithBasic.GET(smBrokerURL+"/v2/service_instances/"+SID+"/last_operation").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
					Expect().Status(http.StatusOK)

				By(fmt.Sprintf("Verifying service instance with id %s is rolledback to old plan id", SID))
				object := ctx.SMWithOAuth.GET("/v1/service_instances/"+SID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
					Expect().Status(http.StatusOK).JSON().Object()
				object.Value("service_plan_id").String().Equal(findSMPlanIDForCatalogPlanID(plan1CatalogID))
				object.Value("usable").Boolean().Equal(true)

				By(fmt.Sprintf("Verifying update operation for service instance with id %s is updated to failed", SID))
				verifyOperationExists(operationExpectations{
					Type:         types.UPDATE,
					State:        types.FAILED,
					ResourceID:   SID,
					ResourceType: types.ServiceInstanceType,
					ExternalID:   "",
					Errors:       []byte("an error happened"),
				})
			})

			Context("when broker responds with instance_usable=false", func() {
				BeforeEach(func() {
					brokerServer.ServiceInstanceLastOpHandler = parameterizedHandler(http.StatusOK, `{"state":"failed", "description": "an error happened", "instance_usable": false}`)
				})

				It("updates the instance to instance_usable=false", func() {
					By(fmt.Sprintf("Getting last operation for service instance with id %s", SID))
					ctx.SMWithBasic.GET(smBrokerURL+"/v2/service_instances/"+SID+"/last_operation").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
						Expect().Status(http.StatusOK)

					By(fmt.Sprintf("Verifying service instance with id %s is updated to to usable=false", SID))
					ctx.SMWithOAuth.GET("/v1/service_instances/"+SID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
						Expect().Status(http.StatusOK).JSON().Object().Value("usable").Boolean().Equal(false)
				})
			})

			Context("when broker responds with instance_usable=true", func() {
				BeforeEach(func() {
					brokerServer.ServiceInstanceLastOpHandler = parameterizedHandler(http.StatusOK, `{"state":"failed", "description": "an error happened", "instance_usable": true}`)
				})

				It("updates the instance to instance_usable=true", func() {
					By(fmt.Sprintf("Getting last operation for service instance with id %s", SID))
					ctx.SMWithBasic.GET(smBrokerURL+"/v2/service_instances/"+SID+"/last_operation").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
						Expect().Status(http.StatusOK)

					By(fmt.Sprintf("Verifying service instance with id %s is updated to to usable=true", SID))
					ctx.SMWithOAuth.GET("/v1/service_instances/"+SID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
						Expect().Status(http.StatusOK).JSON().Object().Value("usable").Boolean().Equal(true)
				})
			})

			Context("when instance does not exist", func() {
				BeforeEach(func() {
					brokerServer.ServiceInstanceHandler = parameterizedHandler(http.StatusOK, `{}`)

					ctx.SMWithOAuthForTenant.DELETE("/v1/service_instances/"+SID).
						WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
						WithQuery("async", false).
						Expect().Status(http.StatusOK)

					ctx.SMWithOAuth.GET("/v1/service_instances/"+SID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
						Expect().Status(http.StatusNotFound)
				})

				It("returns response from broker", func() {
					By(fmt.Sprintf("Getting last operation for service instance with id %s", SID))
					ctx.SMWithBasic.GET(smBrokerURL+"/v2/service_instances/"+SID+"/last_operation").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
						Expect().Status(http.StatusOK).Body().Equal(brokerLastOperationStateFailed)
				})
			})
		})

		Context("that is in progress", func() {
			BeforeEach(func() {
				brokerServer.ServiceInstanceLastOpHandler = parameterizedHandler(http.StatusOK, `{"state":"in progress"}`)
			})

			It("does not update the operation and does not update the instance", func() {
				By(fmt.Sprintf("Getting last operation for service instance with id %s", SID))
				ctx.SMWithBasic.GET(smBrokerURL+"/v2/service_instances/"+SID+"/last_operation").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
					Expect().Status(http.StatusOK)

				By(fmt.Sprintf("Verifying service instance with id %s is not changed", SID))
				object := ctx.SMWithOAuth.GET("/v1/service_instances/"+SID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
					Expect().Status(http.StatusOK).JSON().Object()
				object.Value("service_plan_id").String().Equal(findSMPlanIDForCatalogPlanID(plan2CatalogID))
				object.Value("usable").Boolean().Equal(true)

				By(fmt.Sprintf("Verifying update operation for service instance with id %s is in progress", SID))
				verifyOperationExists(operationExpectations{
					Type:         types.UPDATE,
					State:        types.IN_PROGRESS,
					ResourceID:   SID,
					ResourceType: "/v1/service_instances",
					ExternalID:   "",
				})
			})
		})

		Context("that returns no state", func() {
			BeforeEach(func() {
				brokerServer.ServiceInstanceLastOpHandler = parameterizedHandler(http.StatusOK, `{}`)
			})

			It("does not update the operation and does not update the instance", func() {
				By(fmt.Sprintf("Getting last operation for service instance with id %s", SID))
				ctx.SMWithBasic.GET(smBrokerURL+"/v2/service_instances/"+SID+"/last_operation").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
					Expect().Status(http.StatusOK)

				By(fmt.Sprintf("Verifying service instance with id %s is not changed", SID))
				object := ctx.SMWithOAuth.GET("/v1/service_instances/"+SID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
					Expect().Status(http.StatusOK).JSON().Object()
				object.Value("service_plan_id").String().Equal(findSMPlanIDForCatalogPlanID(plan2CatalogID))
				object.Value("usable").Boolean().Equal(true)

				By(fmt.Sprintf("Verifying updatw operation for service instance with id %s is in progress", SID))
				verifyOperationExists(operationExpectations{
					Type:         types.UPDATE,
					State:        types.IN_PROGRESS,
					ResourceID:   SID,
					ResourceType: "/v1/service_instances",
					ExternalID:   "",
				})
			})
		})

		Context("that returns 410 gone", func() {
			BeforeEach(func() {
				brokerServer.ServiceInstanceLastOpHandler = parameterizedHandler(http.StatusGone, `{}`)
			})

			It("returns 410", func() {
				By(fmt.Sprintf("Getting last operation for service instance with id %s", SID))
				ctx.SMWithBasic.GET(smBrokerURL+"/v2/service_instances/"+SID+"/last_operation").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
					Expect().Status(http.StatusGone)

				By(fmt.Sprintf("Verifying update operation for service instance with id %s is still in progress", SID))
				verifyOperationExists(operationExpectations{
					Type:         types.UPDATE,
					State:        types.IN_PROGRESS,
					ResourceID:   SID,
					ResourceType: "/v1/service_instances",
					ExternalID:   "",
				})
			})
		})
	})

	Context("when polling DELETE for which operation exists", func() {
		BeforeEach(func() {
			By(fmt.Sprintf("Creating service instance with id %s", SID))
			brokerServer.ServiceInstanceHandler = parameterizedHandler(http.StatusOK, `{}`)
			ctx.SMWithBasic.PUT(smBrokerURL+"/v2/service_instances/"+SID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				WithJSON(provisionRequestBodyMap()()).Expect().Status(http.StatusOK)
		})

		JustBeforeEach(func() {
			By(fmt.Sprintf("Deleting service instance with id %s", SID))
			brokerServer.ServiceInstanceHandler = parameterizedHandler(http.StatusAccepted, `{}`)
			ctx.SMWithBasic.DELETE(smBrokerURL+"/v2/service_instances/"+SID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				Expect().Status(http.StatusAccepted)

			By(fmt.Sprintf("Verifying operation delete in progress for service instance with id %s was created", SID))
			verifyOperationExists(operationExpectations{
				Type:         types.DELETE,
				State:        types.IN_PROGRESS,
				ResourceID:   SID,
				ResourceType: "/v1/service_instances",
				ExternalID:   "",
			})
		})

		Context("that has succeeded", func() {
			BeforeEach(func() {
				brokerServer.ServiceInstanceLastOpHandler = parameterizedHandler(http.StatusOK, `{"state":"succeeded"}`)
			})

			It("deletes the instance and updates the operation to delete succeeded", func() {
				By(fmt.Sprintf("Getting last operation for service instance with id %s", SID))
				ctx.SMWithBasic.GET(smBrokerURL+"/v2/service_instances/"+SID+"/last_operation").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
					Expect().Status(http.StatusOK)

				By(fmt.Sprintf("Verifying service instance with id %s is deleted", SID))
				ctx.SMWithOAuth.GET("/v1/service_instances/"+SID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
					Expect().Status(http.StatusNotFound)

				By(fmt.Sprintf("Verifying delete operation for service instance with id %s is updated to succeeded", SID))
				verifyOperationExists(operationExpectations{
					Type:         types.DELETE,
					State:        types.SUCCEEDED,
					ResourceID:   SID,
					ResourceType: "/v1/service_instances",
					ExternalID:   "",
				})
			})

			Context("when instance does not exist", func() {
				BeforeEach(func() {
					brokerServer.ServiceInstanceHandler = parameterizedHandler(http.StatusOK, `{}`)

					ctx.SMWithOAuthForTenant.DELETE("/v1/service_instances/"+SID).
						WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
						WithQuery("async", false).
						Expect().Status(http.StatusOK)

					ctx.SMWithOAuth.GET("/v1/service_instances/"+SID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
						Expect().Status(http.StatusNotFound)
				})

				It("does not fail", func() {
					By(fmt.Sprintf("Getting last operation for service instance with id %s", SID))
					ctx.SMWithBasic.GET(smBrokerURL+"/v2/service_instances/"+SID+"/last_operation").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
						Expect().Status(http.StatusOK)
				})
			})
		})

		Context("that has failed", func() {
			BeforeEach(func() {
				brokerServer.ServiceInstanceLastOpHandler = parameterizedHandler(http.StatusOK, brokerLastOperationStateFailed)
			})

			It("updates the operation to failed and does not delete the instance", func() {
				By(fmt.Sprintf("Getting last operation for service instance with id %s", SID))
				ctx.SMWithBasic.GET(smBrokerURL+"/v2/service_instances/"+SID+"/last_operation").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
					Expect().Status(http.StatusOK)

				By(fmt.Sprintf("Verifying service instance with id %s is not deleted", SID))
				object := ctx.SMWithOAuth.GET("/v1/service_instances/"+SID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
					Expect().Status(http.StatusOK).JSON().Object()
				object.Value("usable").Boolean().Equal(true)

				By(fmt.Sprintf("Verifying delete operation for service instance with id %s is updated to failed", SID))
				verifyOperationExists(operationExpectations{
					Type:         types.DELETE,
					State:        types.FAILED,
					ResourceID:   SID,
					ResourceType: "/v1/service_instances",
					ExternalID:   "",
					Errors:       []byte("an error happened"),
				})
			})

			Context("when broker responds with instance_usable=false", func() {
				BeforeEach(func() {
					brokerServer.ServiceInstanceLastOpHandler = parameterizedHandler(http.StatusOK, `{"state":"failed", "description": "an error happened", "instance_usable": false}`)
				})

				It("updates the instance to instance_usable=false", func() {
					By(fmt.Sprintf("Getting last operation for service instance with id %s", SID))
					ctx.SMWithBasic.GET(smBrokerURL+"/v2/service_instances/"+SID+"/last_operation").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
						Expect().Status(http.StatusOK)

					By(fmt.Sprintf("Verifying service instance with id %s is updated to to usable=false", SID))
					ctx.SMWithOAuth.GET("/v1/service_instances/"+SID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
						Expect().Status(http.StatusOK).JSON().Object().Value("usable").Boolean().Equal(false)
				})
			})

			Context("when broker responds with instance_usable=true", func() {
				BeforeEach(func() {
					brokerServer.ServiceInstanceLastOpHandler = parameterizedHandler(http.StatusOK, `{"state":"failed", "description": "an error happened", "instance_usable": true}`)
				})

				It("updates the operation to failed and does not delete the instance", func() {
					By(fmt.Sprintf("Getting last operation for service instance with id %s", SID))
					ctx.SMWithBasic.GET(smBrokerURL+"/v2/service_instances/"+SID+"/last_operation").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
						Expect().Status(http.StatusOK)

					By(fmt.Sprintf("Verifying service instance with id %s is not deleted", SID))
					object := ctx.SMWithOAuth.GET("/v1/service_instances/"+SID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
						Expect().Status(http.StatusOK).JSON().Object()
					object.Value("usable").Boolean().Equal(true)
				})
			})

			Context("when instance does not exist", func() {
				BeforeEach(func() {
					brokerServer.ServiceInstanceHandler = parameterizedHandler(http.StatusOK, `{}`)

					ctx.SMWithOAuthForTenant.DELETE("/v1/service_instances/"+SID).
						WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
						WithQuery("async", false).
						Expect().Status(http.StatusOK)

					ctx.SMWithOAuth.GET("/v1/service_instances/"+SID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
						Expect().Status(http.StatusNotFound)
				})

				It("returns response from broker", func() {
					By(fmt.Sprintf("Getting last operation for service instance with id %s", SID))
					ctx.SMWithBasic.GET(smBrokerURL+"/v2/service_instances/"+SID+"/last_operation").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
						Expect().Status(http.StatusOK).Body().Equal(brokerLastOperationStateFailed)
				})
			})
		})

		Context("that is in progress", func() {
			BeforeEach(func() {
				brokerServer.ServiceInstanceLastOpHandler = parameterizedHandler(http.StatusOK, `{"state":"in progress"}`)
			})

			It("does not update the operation and does not delete the instance", func() {
				By(fmt.Sprintf("Getting last operation for service instance with id %s", SID))
				ctx.SMWithBasic.GET(smBrokerURL+"/v2/service_instances/"+SID+"/last_operation").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
					Expect().Status(http.StatusOK)

				By(fmt.Sprintf("Verifying service instance with id %s is not deleted", SID))
				ctx.SMWithOAuth.GET("/v1/service_instances/"+SID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
					Expect().Status(http.StatusOK)

				By(fmt.Sprintf("Verifying delete operation for service instance with id %s is in progress", SID))
				verifyOperationExists(operationExpectations{
					Type:         types.DELETE,
					State:        types.IN_PROGRESS,
					ResourceID:   SID,
					ResourceType: "/v1/service_instances",
					ExternalID:   "",
				})
			})
		})

		Context("that returns no state", func() {
			BeforeEach(func() {
				brokerServer.ServiceInstanceLastOpHandler = parameterizedHandler(http.StatusOK, `{}`)
			})

			It("does not update the operation and does not delete the instance", func() {
				By(fmt.Sprintf("Getting last operation for service instance with id %s", SID))
				ctx.SMWithBasic.GET(smBrokerURL+"/v2/service_instances/"+SID+"/last_operation").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
					Expect().Status(http.StatusOK)

				By(fmt.Sprintf("Verifying service instance with id %s is not deleted", SID))
				ctx.SMWithOAuth.GET("/v1/service_instances/"+SID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
					Expect().Status(http.StatusOK)

				By(fmt.Sprintf("Verifying delete operation for service instance with id %s is in progress", SID))
				verifyOperationExists(operationExpectations{
					Type:         types.DELETE,
					State:        types.IN_PROGRESS,
					ResourceID:   SID,
					ResourceType: "/v1/service_instances",
					ExternalID:   "",
				})
			})
		})

		Context("that returns 410 gone", func() {
			BeforeEach(func() {
				brokerServer.ServiceInstanceLastOpHandler = parameterizedHandler(http.StatusGone, `{}`)
			})

			It("deletes the instance and updates the operation to delete succeeded", func() {
				By(fmt.Sprintf("Getting last operation for service instance with id %s", SID))
				ctx.SMWithBasic.GET(smBrokerURL+"/v2/service_instances/"+SID+"/last_operation").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
					Expect().Status(http.StatusGone)

				By(fmt.Sprintf("Verifying service instance with id %s is deleted", SID))
				ctx.SMWithOAuth.GET("/v1/service_instances/"+SID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
					Expect().Status(http.StatusNotFound)

				By(fmt.Sprintf("Verifying delete operation for service instance with id %s is updated to succeeded", SID))
				verifyOperationExists(operationExpectations{
					Type:         types.DELETE,
					State:        types.SUCCEEDED,
					ResourceID:   SID,
					ResourceType: "/v1/service_instances",
					ExternalID:   "",
				})
			})

			Context("when instance does not exist", func() {
				BeforeEach(func() {
					brokerServer.ServiceInstanceHandler = parameterizedHandler(http.StatusOK, `{}`)

					ctx.SMWithOAuthForTenant.DELETE("/v1/service_instances/"+SID).
						WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
						WithQuery("async", false).
						Expect().Status(http.StatusOK)

					ctx.SMWithOAuth.GET("/v1/service_instances/"+SID).WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
						Expect().Status(http.StatusNotFound)
				})

				It("does not fail", func() {
					By(fmt.Sprintf("Getting last operation for service instance with id %s", SID))
					ctx.SMWithBasic.GET(smBrokerURL+"/v2/service_instances/"+SID+"/last_operation").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
						Expect().Status(http.StatusGone)
				})
			})
		})
	})

	Context("when call to working service broker", func() {
		It("should succeed", func() {
			ctx.SMWithBasic.GET(smBrokerURL+"/v2/service_instances/"+SID+"/last_operation").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				Expect().Status(http.StatusOK)
		})
	})

	Context("when call to failing service broker", func() {
		It("should fail", func() {
			brokerServer.ServiceInstanceLastOpHandler = parameterizedHandler(http.StatusInternalServerError, `internal server error`)

			assertFailingBrokerError(
				ctx.SMWithBasic.GET(smBrokerURL+"/v2/service_instances/"+SID+"/last_operation").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
					Expect(), http.StatusInternalServerError, `internal server error`)

		})
	})

	Context("when call to missing service broker", func() {
		It("should fail with 401", func() {
			ctx.SMWithBasic.GET("http://localhost:3456/v1/osb/123"+"/v2/service_instances/"+SID+"/last_operation").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				Expect().Status(http.StatusUnauthorized)
		})
	})

	Context("when call to stopped service broker", func() {
		It("should fail", func() {
			credentials := brokerPlatformCredentialsIDMap[stoppedBrokerID]
			ctx.SMWithBasic.SetBasicCredentials(ctx, credentials.username, credentials.password)

			assertUnresponsiveBrokerError(
				ctx.SMWithBasic.GET(smUrlToStoppedBroker+"/v2/service_instances/"+SID+"/last_operation").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).Expect())
		})
	})

	Context("when call contains query params", func() {
		It("propagates them to the service broker", func() {
			headerKey, headerValue := generateRandomQueryParam()
			brokerServer.ServiceInstanceLastOpHandler = queryParameterVerificationHandler(headerKey, headerValue)
			ctx.SMWithBasic.GET(smBrokerURL+"/v2/service_instances/"+SID+"/last_operation").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				WithQuery(headerKey, headerValue).Expect().Status(http.StatusOK)
		})
	})

	Context("when broker doesn't respond in a timely manner", func() {
		It("should fail with 502", func(done chan<- interface{}) {
			brokerServer.ServiceInstanceLastOpHandler = delayingHandler(done)
			assertUnresponsiveBrokerError(ctx.SMWithBasic.GET(smBrokerURL+"/v2/service_instances/"+SID+"/last_operation").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				Expect())
		}, testTimeout)
	})

	Context("when broker headers are written, but response is slow", func() {
		It("should fail with 503", func() {
			done := make(chan struct{}, 1)
			brokerServer.ServiceInstanceLastOpHandler = slowResponseHandler(5, done)
			assertSMTimeoutError(ctx.SMWithBasic.GET(smBrokerURL+"/v2/service_instances/"+SID+"/last_operation").WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				Expect())
			done <- struct{}{}
		}, testTimeout)
	})

	Context("broker platform credentials check", func() {
		BeforeEach(func() {
			brokerServer.ServiceInstanceHandler = parameterizedHandler(http.StatusCreated, `{}`)
			ctx.SMWithBasic.PUT(smBrokerURL+"/v2/service_instances/"+SID).
				WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				WithJSON(provisionRequestBodyMap()()).Expect().Status(http.StatusCreated)
		})

		Context("get instance last operaion with invalid credentials", func() {
			BeforeEach(func() {
				ctx.SMWithBasic.SetBasicCredentials(ctx, "test", "test")
			})

			It("should return 401", func() {
				brokerServer.ServiceInstanceLastOpHandler = parameterizedHandler(http.StatusOK, `{}`)
				ctx.SMWithBasic.GET(smBrokerURL+"/v2/service_instances/"+SID+"/last_operation").
					WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
					Expect().Status(http.StatusUnauthorized)
			})
		})
	})

	Context("pollInstance operation is handled by external plugin", func() {
		BeforeEach(func() {
			brokerServer.ServiceInstanceHandler = parameterizedHandler(http.StatusCreated, `{}`)
			ctx.SMWithBasic.PUT(smBrokerURL+"/v2/service_instances/"+SID).
				WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				WithJSON(provisionRequestBodyMap()()).Expect().Status(http.StatusCreated)

			verifyOperationExists(operationExpectations{
				Type:         types.CREATE,
				State:        types.SUCCEEDED,
				ResourceID:   SID,
				ResourceType: "/v1/service_instances",
				ExternalID:   "",
			})
			shouldSaveOperationInContext = true
		})

		It("osb store plugin should return the response from the external plugin without updating DB", func() {
			ctx.SMWithBasic.GET(smBrokerURL+"/v2/service_instances/"+SID+"/last_operation").
				WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				Expect().Status(http.StatusOK).Body().Equal(string(fakeStateResponseBody))
			//verify state is not changed
			verifyOperationExists(operationExpectations{
				Type:         types.CREATE,
				State:        types.SUCCEEDED,
				ResourceID:   SID,
				ResourceType: "/v1/service_instances",
				ExternalID:   "",
			})
		})
	})
})
