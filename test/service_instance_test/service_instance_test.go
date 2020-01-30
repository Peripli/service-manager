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

package service_test

import (
	"context"
	"fmt"
	"time"

	"github.com/tidwall/gjson"

	"github.com/Peripli/service-manager/pkg/util"

	"github.com/spf13/pflag"

	"github.com/gofrs/uuid"

	"github.com/gavv/httpexpect"

	"strconv"

	"net/http"
	"testing"

	"github.com/Peripli/service-manager/pkg/types"

	"github.com/Peripli/service-manager/pkg/web"
	. "github.com/Peripli/service-manager/test/common"

	. "github.com/Peripli/service-manager/test"

	. "github.com/onsi/ginkgo"

	. "github.com/onsi/gomega"
)

func TestServiceInstances(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Instances Tests Suite")
}

const (
	TenantIdentifier = "tenant"
	TenantIDValue    = "tenantID"
)

var _ = DescribeTestsFor(TestCase{
	API: web.ServiceInstancesURL,
	SupportedOps: []Op{
		Get, List, Delete, Patch,
	},
	MultitenancySettings: &MultitenancySettings{
		ClientID:           "tenancyClient",
		ClientIDTokenClaim: "cid",
		TenantTokenClaim:   "zid",
		LabelKey:           TenantIdentifier,
		TokenClaims: map[string]interface{}{
			"cid": "tenancyClient",
			"zid": TenantIDValue,
		},
	},
	ResourceType:                           types.ServiceInstanceType,
	SupportsAsyncOperations:                true,
	DisableTenantResources:                 true,
	ResourceBlueprint:                      blueprint,
	ResourceWithoutNullableFieldsBlueprint: blueprint,
	ResourcePropertiesToIgnore:             []string{"platform_id"},
	PatchResource:                          APIResourcePatch,
	AdditionalTests: func(ctx *TestContext) {
		Context("additional non-generic tests", func() {
			var (
				postInstanceRequest Object

				servicePlanID        string
				anotherServicePlanID string
				brokerID             string
				brokerServer         *BrokerServer
				instanceID           string
			)

			type testCase struct {
				async                           bool
				expectedCreateSuccessStatusCode int
				expectedDeleteSuccessStatusCode int
				expectedBrokerFailureStatusCode int
			}

			testCases := []testCase{
				{
					async:                           false,
					expectedCreateSuccessStatusCode: http.StatusCreated,
					expectedDeleteSuccessStatusCode: http.StatusOK,
					expectedBrokerFailureStatusCode: http.StatusBadGateway,
				},
				{
					async:                           true,
					expectedCreateSuccessStatusCode: http.StatusAccepted,
					expectedDeleteSuccessStatusCode: http.StatusAccepted,
					expectedBrokerFailureStatusCode: http.StatusAccepted,
				},
			}

			createInstance := func(smClient *SMExpect, async bool, expectedStatusCode int) *httpexpect.Response {
				resp := smClient.POST(web.ServiceInstancesURL).WithQuery("async", async).WithJSON(postInstanceRequest).
					Expect().Status(expectedStatusCode)

				if resp.Raw().StatusCode == http.StatusCreated {
					obj := resp.JSON().Object()

					obj.ContainsKey("id").
						ValueEqual("platform_id", types.SMPlatform)

					instanceID = obj.Value("id").String().Raw()
				}

				return resp
			}

			deleteInstance := func(smClient *SMExpect, async bool, expectedStatusCode int) *httpexpect.Response {
				return smClient.DELETE(web.ServiceInstancesURL+"/"+instanceID).
					WithQuery("async", async).
					WithJSON(postInstanceRequest).
					Expect().
					Status(expectedStatusCode)
			}

			verifyInstanceExists := func(instanceID string, ready bool) {
				timeoutDuration := 15 * time.Second
				tickerInterval := 100 * time.Millisecond
				ticker := time.NewTicker(tickerInterval)
				timeout := time.After(timeoutDuration)
				defer ticker.Stop()
				for {
					select {
					case <-timeout:
						Fail(fmt.Sprintf("instance with id %s did not appear in SM after %.0f seconds", instanceID, timeoutDuration.Seconds()))
					case <-ticker.C:
						instances := ctx.SMWithOAuthForTenant.ListWithQuery(web.ServiceInstancesURL, fmt.Sprintf("fieldQuery=id eq '%s'", instanceID))
						switch {
						case instances.Length().Raw() == 0:
							By(fmt.Sprintf("Could not find instance with id %s in SM. Retrying...", instanceID))
						case instances.Length().Raw() > 1:
							Fail(fmt.Sprintf("more than one instance with id %s was found in SM", instanceID))
						default:
							instanceObject := instances.First().Object()
							//instanceObject.Path(fmt.Sprintf("$.labels[%s][*]", TenantIdentifier)).Array().Contains(TenantIDValue)
							readyField := instanceObject.Value("ready").Boolean().Raw()
							if readyField != ready {
								Fail(fmt.Sprintf("Expected instance with id %s to be ready %t but ready was %t", instanceID, ready, readyField))
							}
							return
						}
					}
				}
			}

			verifyInstanceDoesNotExist := func(instanceID string) {
				timeoutDuration := 15 * time.Second
				tickerInterval := 100 * time.Millisecond
				ticker := time.NewTicker(tickerInterval)
				timeout := time.After(timeoutDuration)

				defer ticker.Stop()
				for {
					select {
					case <-timeout:
						Fail(fmt.Sprintf("instance with id %s was still in SM after %.0f seconds", instanceID, timeoutDuration.Seconds()))
					case <-ticker.C:
						resp := ctx.SMWithOAuthForTenant.GET(web.ServiceInstancesURL + "/" + instanceID).
							Expect().Raw()
						if resp.StatusCode != http.StatusNotFound {
							By(fmt.Sprintf("Found instance with id %s but it should be deleted. Retrying...", instanceID))
						} else {
							return
						}
					}
				}
			}

			BeforeEach(func() {
				var plans *httpexpect.Array
				brokerID, brokerServer, plans = prepareBrokerWithCatalog(ctx, ctx.SMWithOAuth)
				servicePlanID = plans.Element(0).Object().Value("id").String().Raw()
				anotherServicePlanID = plans.Element(1).Object().Value("id").String().Raw()
				postInstanceRequest = Object{
					"name":             "test-instance",
					"service_plan_id":  servicePlanID,
					"maintenance_info": "{}",
				}
			})

			AfterEach(func() {
				ctx.CleanupAdditionalResources()
			})

			Describe("GET", func() {
				var instanceName string
				BeforeEach(func() {

				})
				When("service instance contains tenant identifier in OSB context", func() {
					BeforeEach(func() {
						EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, servicePlanID, TenantIDValue)
						resp := createInstance(ctx.SMWithOAuthForTenant, false, http.StatusCreated)
						instanceName = resp.JSON().Object().Value("name").String().Raw()
						Expect(instanceName).ToNot(BeEmpty())
					})

					It("labels instance with tenant identifier", func() {
						ctx.SMWithOAuthForTenant.GET(web.ServiceInstancesURL + "/" + instanceID).Expect().
							Status(http.StatusOK).
							JSON().
							Object().Path(fmt.Sprintf("$.labels[%s][*]", TenantIdentifier)).Array().Contains(TenantIDValue)
					})

					It("returns OSB context with tenant as part of the instance", func() {
						ctx.SMWithOAuthForTenant.GET(web.ServiceInstancesURL + "/" + instanceID).Expect().
							Status(http.StatusOK).
							JSON().
							Object().Value("context").Object().Equal(map[string]interface{}{
							"platform":       types.SMPlatform,
							"instance_name":  instanceName,
							TenantIdentifier: TenantIDValue,
						})
					})
				})

				When("service instance doesn't contain tenant identifier in OSB context", func() {
					BeforeEach(func() {
						EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, servicePlanID, "")
						resp := createInstance(ctx.SMWithOAuth, false, http.StatusCreated)
						instanceName = resp.JSON().Object().Value("name").String().Raw()
						Expect(instanceName).ToNot(BeEmpty())
					})

					It("doesn't label instance with tenant identifier", func() {
						obj := ctx.SMWithOAuth.GET(web.ServiceInstancesURL + "/" + instanceID).Expect().
							Status(http.StatusOK).JSON().Object()

						objMap := obj.Raw()
						objLabels, exist := objMap["labels"]
						if exist {
							labels := objLabels.(map[string]interface{})
							_, tenantLabelExists := labels[TenantIdentifier]
							Expect(tenantLabelExists).To(BeFalse())
						}
					})

					It("returns OSB context with no tenant as part of the instance", func() {
						ctx.SMWithOAuth.GET(web.ServiceInstancesURL + "/" + instanceID).Expect().
							Status(http.StatusOK).
							JSON().
							Object().Value("context").Object().Equal(map[string]interface{}{
							"platform":      types.SMPlatform,
							"instance_name": instanceName,
						})
					})
				})

				When("service instance dashboard_url is not set", func() {
					BeforeEach(func() {
						postInstanceRequest["dashboard_url"] = ""
						EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, postInstanceRequest["service_plan_id"].(string), "")
						createInstance(ctx.SMWithOAuth, false, http.StatusCreated)
					})

					It("doesn't return dashboard_url", func() {
						ctx.SMWithOAuth.GET(web.ServiceInstancesURL + "/" + instanceID).Expect().
							Status(http.StatusOK).JSON().Object().NotContainsKey("dashboard_url")
					})
				})
			})

			Describe("POST", func() {
				for _, testCase := range testCases {
					testCase := testCase
					Context(fmt.Sprintf("async = %t", testCase.async), func() {
						When("content type is not JSON", func() {
							It("returns 415", func() {
								ctx.SMWithOAuth.POST(web.ServiceInstancesURL).
									WithQuery("async", testCase.async).
									WithText("text").
									Expect().
									Status(http.StatusUnsupportedMediaType).
									JSON().Object().
									Keys().Contains("error", "description")
							})
						})

						When("request body is not a valid JSON", func() {
							It("returns 400", func() {
								ctx.SMWithOAuth.POST(web.ServiceInstancesURL).
									WithQuery("async", testCase.async).
									WithText("invalid json").
									WithHeader("content-type", "application/json").
									Expect().
									Status(http.StatusBadRequest).
									JSON().Object().
									Keys().Contains("error", "description")
							})
						})

						When("a request body field is missing", func() {
							assertPOSTReturns400WhenFieldIsMissing := func(field string) {
								var servicePlanID string
								BeforeEach(func() {
									servicePlanID = postInstanceRequest["service_plan_id"].(string)
									delete(postInstanceRequest, field)
								})

								It("returns 400", func() {
									EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, servicePlanID, "")
									ctx.SMWithOAuth.POST(web.ServiceInstancesURL).
										WithJSON(postInstanceRequest).
										WithQuery("async", testCase.async).
										Expect().
										Status(http.StatusBadRequest).
										JSON().Object().
										Keys().Contains("error", "description")
								})
							}

							assertPOSTReturns201WhenFieldIsMissing := func(field string) {
								BeforeEach(func() {
									delete(postInstanceRequest, field)
								})

								It("returns 201", func() {
									EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, postInstanceRequest["service_plan_id"].(string), TenantIDValue)
									resp := createInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedCreateSuccessStatusCode)
									instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.CREATE,
										State:             types.SUCCEEDED,
										ResourceType:      types.ServiceInstanceType,
										Reschedulable:     false,
										DeletionScheduled: false,
									})

									verifyInstanceExists(instanceID, true)
								})
							}

							Context("when id field is missing", func() {
								assertPOSTReturns201WhenFieldIsMissing("id")
							})

							Context("when name field is missing", func() {
								assertPOSTReturns400WhenFieldIsMissing("name")
							})

							Context("when service_plan_id field is missing", func() {
								assertPOSTReturns400WhenFieldIsMissing("service_plan_id")
							})

							Context("when maintenance_info field is missing", func() {
								assertPOSTReturns201WhenFieldIsMissing("maintenance_info")
							})
						})

						When("request body id field is provided", func() {
							It("should return 400", func() {
								EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, postInstanceRequest["service_plan_id"].(string), "")
								postInstanceRequest["id"] = "test-instance-id"
								resp := ctx.SMWithOAuth.POST(web.ServiceInstancesURL).
									WithQuery("async", testCase.async).
									WithJSON(postInstanceRequest).
									Expect().Status(http.StatusBadRequest).JSON().Object()

								Expect(resp.Value("description").String().Raw()).To(ContainSubstring("providing specific resource id is forbidden"))
							})
						})

						When("request body platform_id field is provided", func() {
							Context("which is not service-manager platform", func() {
								It("should return 400", func() {
									postInstanceRequest["platform_id"] = "test-platform-id"
									resp := ctx.SMWithOAuth.POST(web.ServiceInstancesURL).
										WithJSON(postInstanceRequest).
										WithQuery("async", testCase.async).
										Expect().Status(http.StatusBadRequest).JSON().Object()

									resp.Value("description").Equal("Providing platform_id property during provisioning/updating of a service instance is forbidden")
								})
							})

							Context("which is service-manager platform", func() {
								It(fmt.Sprintf("should return %d", testCase.expectedCreateSuccessStatusCode), func() {
									postInstanceRequest["platform_id"] = types.SMPlatform
									EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, postInstanceRequest["service_plan_id"].(string), "")
									createInstance(ctx.SMWithOAuth, testCase.async, testCase.expectedCreateSuccessStatusCode)
								})
							})
						})

						Context("OSB context", func() {
							It("enriches the osb context with the tenant and sm platform", func() {
								EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, postInstanceRequest["service_plan_id"].(string), TenantIDValue)
								createInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedCreateSuccessStatusCode)
								for _, provisionRequest := range brokerServer.ServiceInstanceEndpointRequests {
									body, err := util.BodyToBytes(provisionRequest.Body)
									Expect(err).ToNot(HaveOccurred())
									tenantValue := gjson.GetBytes(body, "context."+TenantIdentifier).String()
									Expect(tenantValue).To(Equal(TenantIDValue))
									platformValue := gjson.GetBytes(body, "context.platform").String()
									Expect(platformValue).To(Equal(types.SMPlatform))
								}
							})
						})

						Context("instance visibility", func() {
							When("tenant doesn't have plan visibility", func() {
								It("returns 404", func() {
									createInstance(ctx.SMWithOAuthForTenant, testCase.async, http.StatusNotFound)
								})
							})

							When("tenant has plan visibility", func() {
								It(fmt.Sprintf("returns %d", testCase.expectedCreateSuccessStatusCode), func() {
									EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, servicePlanID, TenantIDValue)
									createInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedCreateSuccessStatusCode)
								})
							})

							When("plan has public visibility", func() {
								It(fmt.Sprintf("for global returns %d", testCase.expectedCreateSuccessStatusCode), func() {
									EnsurePublicPlanVisibility(ctx.SMRepository, servicePlanID)
									createInstance(ctx.SMWithOAuth, testCase.async, testCase.expectedCreateSuccessStatusCode)
								})

								It(fmt.Sprintf("for tenant returns %d", testCase.expectedCreateSuccessStatusCode), func() {
									EnsurePublicPlanVisibility(ctx.SMRepository, servicePlanID)
									createInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedCreateSuccessStatusCode)
								})
							})
						})

						Context("broker scenarios", func() {
							var doneChannel chan interface{}
							var cancelCtx context.Context
							var cancelFunc context.CancelFunc

							BeforeEach(func() {
								EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, servicePlanID, TenantIDValue)
								doneChannel = make(chan interface{})
								cancelCtx, cancelFunc = context.WithCancel(context.Background())
							})

							AfterEach(func() {
								close(doneChannel)
								brokerServer.ResetHandlers()
							})

							When("a create operation is already in progress", func() {
								BeforeEach(func() {
									brokerServer.ServiceInstanceHandlerFunc(http.MethodPut, http.MethodPut+"1", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
									brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodPut+"1", DelayingHandler(doneChannel, cancelFunc))

									resp := createInstance(ctx.SMWithOAuthForTenant, true, http.StatusAccepted)

									instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.CREATE,
										State:             types.IN_PROGRESS,
										ResourceType:      types.ServiceInstanceType,
										Reschedulable:     true,
										DeletionScheduled: false,
									})

									verifyInstanceExists(instanceID, false)
								})

								It("updates fail with operation in progress", func() {
									ctx.SMWithOAuthForTenant.PATCH(web.ServiceInstancesURL+"/"+instanceID).WithQuery("async", testCase.async).WithJSON(Object{}).
										Expect().Status(http.StatusUnprocessableEntity)
								})

								It("deletes succeed", func() {
									resp := ctx.SMWithOAuthForTenant.DELETE(web.ServiceInstancesURL+"/"+instanceID).WithQuery("async", testCase.async).
										Expect().StatusRange(httpexpect.Status2xx)

									instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.DELETE,
										State:             types.SUCCEEDED,
										ResourceType:      types.ServiceInstanceType,
										Reschedulable:     false,
										DeletionScheduled: false,
									})

									verifyInstanceDoesNotExist(instanceID)
								})
							})

							When("plan does not exist", func() {
								BeforeEach(func() {
									postInstanceRequest["service_plan_id"] = "non-existing-id"
								})

								It("provision fails", func() {
									createInstance(ctx.SMWithOAuthForTenant, testCase.async, http.StatusNotFound)
								})
							})

							When("broker responds with synchronous success", func() {
								BeforeEach(func() {
									brokerServer.ServiceInstanceHandlerFunc(http.MethodPut, http.MethodPut+"1", ParameterizedHandler(http.StatusCreated, Object{"async": false}))
								})

								It("stores instance as ready=true and the operation as success, non rescheduable with no deletion scheduled", func() {
									resp := createInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedCreateSuccessStatusCode)

									instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.CREATE,
										State:             types.SUCCEEDED,
										ResourceType:      types.ServiceInstanceType,
										Reschedulable:     false,
										DeletionScheduled: false,
									})

									verifyInstanceExists(instanceID, true)
								})
							})

							When("broker responds with asynchronous success", func() {
								BeforeEach(func() {
									brokerServer.ServiceInstanceHandlerFunc(http.MethodPut, http.MethodPut+"1", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
									brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodPut+"1", MultiplePollsRequiredHandler("in progress", "succeeded"))
								})

								It("polling broker last operation until operation succeeds and eventually marks operation as success", func() {
									resp := createInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedCreateSuccessStatusCode)

									instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.CREATE,
										State:             types.SUCCEEDED,
										ResourceType:      types.ServiceInstanceType,
										Reschedulable:     false,
										DeletionScheduled: false,
									})

									verifyInstanceExists(instanceID, true)
								})

								if testCase.async {
									When("job timeout is reached while polling", func() {
										var oldCtx *TestContext
										BeforeEach(func() {
											oldCtx = ctx
											ctx = NewTestContextBuilderWithSecurity().WithEnvPreExtensions(func(set *pflag.FlagSet) {
												Expect(set.Set("operations.job_timeout", (2 * time.Second).String())).ToNot(HaveOccurred())
											}).Build()

											var plans *httpexpect.Array
											brokerID, brokerServer, plans = prepareBrokerWithCatalog(ctx, ctx.SMWithOAuth)
											servicePlanID = plans.Element(0).Object().Value("id").String().Raw()

											postInstanceRequest = Object{
												"name":             "test-instance",
												"service_plan_id":  servicePlanID,
												"maintenance_info": "{}",
											}

											brokerServer.ServiceInstanceHandlerFunc(http.MethodPut, http.MethodPut+"1", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
											brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodPut+"1", ParameterizedHandler(http.StatusOK, Object{"state": "in progress"}))
										})

										AfterEach(func() {
											ctx.SMRepository.Delete(context.TODO(), types.OperationType)
											DeleteInstance(ctx, instanceID, servicePlanID)
											ctx.SMWithOAuth.DELETE(web.ServiceBrokersURL + "/" + brokerID).Expect()
											delete(ctx.Servers, BrokerServerPrefix+brokerID)
											brokerServer.Close()
											ctx = oldCtx
										})

										It("stores instance as ready false and the operation as reschedulable in progress", func() {
											resp := createInstance(ctx.SMWithOAuthForTenant, true, http.StatusAccepted)

											instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.CREATE,
												State:             types.IN_PROGRESS,
												ResourceType:      types.ServiceInstanceType,
												Reschedulable:     true,
												DeletionScheduled: false,
											})

											verifyInstanceExists(instanceID, false)
										})
									})
								}

								When("polling responds with unexpected state and eventually with success state", func() {
									BeforeEach(func() {
										brokerServer.ServiceInstanceHandlerFunc(http.MethodPut, http.MethodPut+"1", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
										brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodPut+"1", MultiplePollsRequiredHandler("unknown", "succeeded"))
									})

									It("keeps polling and eventually updates the instance to ready true and operation to success", func() {
										resp := createInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedCreateSuccessStatusCode)

										instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
											Category:          types.CREATE,
											State:             types.SUCCEEDED,
											ResourceType:      types.ServiceInstanceType,
											Reschedulable:     false,
											DeletionScheduled: false,
										})
										verifyInstanceExists(instanceID, true)
									})
								})

								When("polling responds with unexpected state and eventually with failed state", func() {
									BeforeEach(func() {
										brokerServer.ServiceInstanceHandlerFunc(http.MethodPut, http.MethodPut+"2", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
										brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodPut+"2", MultiplePollsRequiredHandler("unknown", "failed"))
									})

									When("orphan mitigation deprovision synchronously succeeds", func() {
										BeforeEach(func() {
											brokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"3", ParameterizedHandler(http.StatusOK, Object{"async": false}))
										})

										It("deletes the instance and marks the operation that triggered the orphan mitigation as failed with no deletion scheduled and not reschedulable", func() {
											resp := createInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)

											instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.CREATE,
												State:             types.FAILED,
												ResourceType:      types.ServiceInstanceType,
												Reschedulable:     false,
												DeletionScheduled: false,
											})

											verifyInstanceDoesNotExist(instanceID)
										})
									})

									When("broker orphan mitigation deprovision synchronously fails with an error that will stop further orphan mitigation", func() {
										BeforeEach(func() {
											brokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"3", ParameterizedHandler(http.StatusBadRequest, Object{"error": "error"}))
										})

										It("keeps in the instance with ready false and marks the operation with deletion scheduled", func() {
											resp := createInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)

											instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.CREATE,
												State:             types.FAILED,
												ResourceType:      types.ServiceInstanceType,
												Reschedulable:     false,
												DeletionScheduled: true,
											})

											verifyInstanceExists(instanceID, !testCase.async)
										})
									})

									When("broker orphan mitigation deprovision synchronously fails with an error that will continue further orphan mitigation and eventually succeed", func() {
										BeforeEach(func() {
											brokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"3", MultipleErrorsBeforeSuccessHandler(
												http.StatusInternalServerError, http.StatusOK,
												Object{"error": "error"}, Object{"async": false},
											))
										})

										It("deletes the instance and marks the operation that triggered the orphan mitigation as failed with no deletion scheduled and not reschedulable", func() {
											resp := createInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)

											instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.CREATE,
												State:             types.FAILED,
												ResourceType:      types.ServiceInstanceType,
												Reschedulable:     false,
												DeletionScheduled: false,
											})

											verifyInstanceDoesNotExist(instanceID)
										})
									})
								})

								When("polling returns an unexpected status code", func() {
									BeforeEach(func() {
										brokerServer.ServiceInstanceHandlerFunc(http.MethodPut, http.MethodPut+"3", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
										brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodPut+"3", ParameterizedHandler(http.StatusInternalServerError, Object{"error": "error"}))
									})

									It("stores the instance as ready false and marks the operation as reschedulable", func() {
										resp := createInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)

										instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
											Category:          types.CREATE,
											State:             types.FAILED,
											ResourceType:      types.ServiceInstanceType,
											Reschedulable:     true,
											DeletionScheduled: false,
										})

										verifyInstanceExists(instanceID, !testCase.async)
									})
								})

								When("broker stops while polling", func() {
									BeforeEach(func() {
										brokerServer.ServiceInstanceHandlerFunc(http.MethodPut, http.MethodPut+"3", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
										brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodPut+"3", DelayingHandler(doneChannel, cancelFunc))
									})

									It("keeps the instance as ready false and marks the operation as failed reschedulable with no orphan mitigation", func() {
										resp := createInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)

										<-cancelCtx.Done()
										brokerServer.Close()
										delete(ctx.Servers, BrokerServerPrefix+brokerID)

										instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
											Category:          types.CREATE,
											State:             types.FAILED,
											ResourceType:      types.ServiceInstanceType,
											Reschedulable:     true,
											DeletionScheduled: false,
										})

										verifyInstanceExists(instanceID, !testCase.async)
									})
								})
							})

							When("provision responds with error due to stopped broker", func() {
								BeforeEach(func() {
									brokerServer.Close()
									delete(ctx.Servers, BrokerServerPrefix+brokerID)
								})

								It("does not store instance in SMDB and marks operation with failed", func() {
									resp := createInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)

									instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.CREATE,
										State:             types.FAILED,
										ResourceType:      types.ServiceInstanceType,
										Reschedulable:     false,
										DeletionScheduled: false,
									})

									verifyInstanceDoesNotExist(instanceID)
								})
							})

							When("provision responds with error that does not require orphan mitigation", func() {
								BeforeEach(func() {
									brokerServer.ServiceInstanceHandlerFunc(http.MethodPut, http.MethodPut+"3", ParameterizedHandler(http.StatusBadRequest, Object{"error": "error"}))
								})

								It("does not store the instance and marks the operation as failed, non rescheduable with empty deletion scheduled", func() {
									resp := createInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)

									instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.CREATE,
										State:             types.FAILED,
										ResourceType:      types.ServiceInstanceType,
										Reschedulable:     false,
										DeletionScheduled: false,
									})

									verifyInstanceDoesNotExist(instanceID)
								})
							})

							When("provision responds with error that requires orphan mitigation", func() {
								BeforeEach(func() {
									brokerServer.ServiceInstanceHandlerFunc(http.MethodPut, http.MethodPut+"3", ParameterizedHandler(http.StatusInternalServerError, Object{"error": "error"}))
								})

								When("orphan mitigation deprovision asynchronously succeeds", func() {
									BeforeEach(func() {
										brokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"3", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
										brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodDelete+"3", ParameterizedHandler(http.StatusOK, Object{"state": "succeeded"}))
									})

									It("deletes the instance and marks the operation that triggered the orphan mitigation as failed with no deletion scheduled and not reschedulable", func() {
										resp := createInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)

										instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
											Category:          types.CREATE,
											State:             types.FAILED,
											ResourceType:      types.ServiceInstanceType,
											Reschedulable:     false,
											DeletionScheduled: false,
										})

										verifyInstanceDoesNotExist(instanceID)
									})
								})

								if testCase.async {
									When("broker orphan mitigation deprovision asynchronously keeps failing with an error while polling", func() {
										BeforeEach(func() {
											brokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"3", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
											brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodDelete+"3", ParameterizedHandler(http.StatusBadRequest, Object{"error": "error"}))
										})

										It("keeps the instance as ready false and marks the operation as deletion scheduled", func() {
											resp := createInstance(ctx.SMWithOAuthForTenant, true, http.StatusAccepted)

											instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.CREATE,
												State:             types.FAILED,
												ResourceType:      types.ServiceInstanceType,
												Reschedulable:     true,
												DeletionScheduled: true,
											})

											verifyInstanceExists(instanceID, false)
										})
									})
								}

								When("broker orphan mitigation deprovision asynchronously fails with an error that will continue further orphan mitigation and eventually succeed", func() {
									BeforeEach(func() {
										brokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"3", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))

										brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodDelete+"3", MultipleErrorsBeforeSuccessHandler(
											http.StatusOK, http.StatusOK,
											Object{"state": "failed"}, Object{"state": "succeeded"},
										))
									})

									It("deletes the instance and marks the operation that triggered the orphan mitigation as failed with no deletion scheduled and not reschedulable", func() {
										resp := createInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)

										instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
											Category:          types.CREATE,
											State:             types.FAILED,
											ResourceType:      types.ServiceInstanceType,
											Reschedulable:     false,
											DeletionScheduled: false,
										})

										verifyInstanceDoesNotExist(instanceID)
									})
								})
							})

							When("provision responds with error due to times out", func() {
								var oldCtx *TestContext
								BeforeEach(func() {
									oldCtx = ctx
									ctx = NewTestContextBuilderWithSecurity().WithEnvPreExtensions(func(set *pflag.FlagSet) {
										Expect(set.Set("httpclient.response_header_timeout", (1 * time.Second).String())).ToNot(HaveOccurred())
									}).Build()

									var plans *httpexpect.Array
									brokerID, brokerServer, plans = prepareBrokerWithCatalog(ctx, ctx.SMWithOAuth)
									servicePlanID = plans.Element(0).Object().Value("id").String().Raw()
									brokerServer.ServiceInstanceHandlerFunc(http.MethodPut, http.MethodPut+"1", DelayingHandler(doneChannel, cancelFunc))
									postInstanceRequest = Object{
										"name":             "test-instance",
										"service_plan_id":  servicePlanID,
										"maintenance_info": "{}",
									}
								})

								AfterEach(func() {
									ctx.SMRepository.Delete(context.TODO(), types.OperationType)
									DeleteInstance(ctx, instanceID, servicePlanID)
									ctx.SMWithOAuth.DELETE(web.ServiceBrokersURL + "/" + brokerID).Expect()
									delete(ctx.Servers, BrokerServerPrefix+brokerID)
									brokerServer.Close()
									ctx = oldCtx
								})

								It("orphan mitigates the instance", func() {
									resp := createInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)

									instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.CREATE,
										State:             types.FAILED,
										ResourceType:      types.ServiceInstanceType,
										Reschedulable:     false,
										DeletionScheduled: false,
									})

									verifyInstanceDoesNotExist(instanceID)
								})
							})
						})
					})
				}
			})

			Describe("PATCH", func() {
				When("content type is not JSON", func() {
					It("returns 415", func() {
						ctx.SMWithOAuth.PATCH(web.ServiceInstancesURL+"/instance-id").
							WithText("text").
							Expect().Status(http.StatusUnsupportedMediaType).
							JSON().Object().
							Keys().Contains("error", "description")
					})
				})

				When("instance is missing", func() {
					It("returns 404", func() {
						ctx.SMWithOAuth.PATCH(web.ServiceInstancesURL+"/no_such_id").
							WithJSON(postInstanceRequest).
							Expect().Status(http.StatusNotFound).
							JSON().Object().
							Keys().Contains("error", "description")
					})
				})

				When("request body is not valid JSON", func() {
					It("returns 400", func() {
						ctx.SMWithOAuth.PATCH(web.ServiceInstancesURL+"/instance-id").
							WithText("invalid json").
							WithHeader("content-type", "application/json").
							Expect().
							Status(http.StatusBadRequest).
							JSON().Object().
							Keys().Contains("error", "description")
					})
				})

				When("created_at provided in body", func() {
					It("should not change created at", func() {
						EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, postInstanceRequest["service_plan_id"].(string), "")
						createInstance(ctx.SMWithOAuth, false, http.StatusCreated)

						createdAt := "2015-01-01T00:00:00Z"

						ctx.SMWithOAuth.PATCH(web.ServiceInstancesURL+"/"+instanceID).
							WithJSON(Object{"created_at": createdAt}).
							Expect().
							Status(http.StatusOK).JSON().Object().
							ContainsKey("created_at").
							ValueNotEqual("created_at", createdAt)

						ctx.SMWithOAuth.GET(web.ServiceInstancesURL+"/"+instanceID).
							Expect().
							Status(http.StatusOK).JSON().Object().
							ContainsKey("created_at").
							ValueNotEqual("created_at", createdAt)
					})
				})

				When("platform_id provided in body", func() {
					Context("which is not service-manager platform", func() {
						It("should return 400", func() {
							EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, postInstanceRequest["service_plan_id"].(string), "")
							createInstance(ctx.SMWithOAuth, false, http.StatusCreated)

							resp := ctx.SMWithOAuth.PATCH(web.ServiceInstancesURL + "/" + instanceID).
								WithJSON(Object{"platform_id": "test-platform-id"}).
								Expect().Status(http.StatusBadRequest).JSON().Object()

							resp.Value("description").Equal("Providing platform_id property during provisioning/updating of a service instance is forbidden")

							ctx.SMWithOAuth.GET(web.ServiceInstancesURL+"/"+instanceID).
								Expect().
								Status(http.StatusOK).JSON().Object().
								ContainsKey("platform_id").
								ValueEqual("platform_id", types.SMPlatform)
						})
					})

					Context("which is service-manager platform", func() {
						It("should return 200", func() {
							EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, postInstanceRequest["service_plan_id"].(string), "")
							createInstance(ctx.SMWithOAuth, false, http.StatusCreated)

							ctx.SMWithOAuth.PATCH(web.ServiceInstancesURL + "/" + instanceID).
								WithJSON(Object{"platform_id": types.SMPlatform}).
								Expect().Status(http.StatusOK).JSON().Object()

							ctx.SMWithOAuth.GET(web.ServiceInstancesURL+"/"+instanceID).
								Expect().
								Status(http.StatusOK).JSON().Object().
								ContainsKey("platform_id").
								ValueEqual("platform_id", types.SMPlatform)

						})
					})
				})

				When("fields are updated one by one", func() {
					It("returns 200", func() {
						EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, postInstanceRequest["service_plan_id"].(string), "")
						createInstance(ctx.SMWithOAuth, false, http.StatusCreated)

						for _, prop := range []string{"name", "maintenance_info"} {
							updatedBrokerJSON := Object{}
							updatedBrokerJSON[prop] = "updated-" + prop
							ctx.SMWithOAuth.PATCH(web.ServiceInstancesURL + "/" + instanceID).
								WithJSON(updatedBrokerJSON).
								Expect().
								Status(http.StatusOK).
								JSON().Object().
								ContainsMap(updatedBrokerJSON)

							ctx.SMWithOAuth.GET(web.ServiceInstancesURL + "/" + instanceID).
								Expect().
								Status(http.StatusOK).
								JSON().Object().
								ContainsMap(updatedBrokerJSON)

						}
					})
				})

				Context("instance visibility", func() {
					When("tenant doesn't have plan visibility", func() {
						It("returns 404", func() {
							EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, servicePlanID, TenantIDValue)
							createInstance(ctx.SMWithOAuthForTenant, false, http.StatusCreated)

							ctx.SMWithOAuthForTenant.PATCH(web.ServiceInstancesURL + "/" + instanceID).
								WithJSON(Object{"service_plan_id": anotherServicePlanID}).
								Expect().Status(http.StatusNotFound)
						})
					})

					When("tenant has plan visibility", func() {
						It("returns 201", func() {
							EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, servicePlanID, TenantIDValue)
							createInstance(ctx.SMWithOAuthForTenant, false, http.StatusCreated)

							EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, anotherServicePlanID, TenantIDValue)
							ctx.SMWithOAuthForTenant.PATCH(web.ServiceInstancesURL + "/" + instanceID).
								WithJSON(Object{"service_plan_id": anotherServicePlanID}).
								Expect().Status(http.StatusOK)
						})
					})
				})

				Context("instance ownership", func() {
					When("tenant doesn't have ownership of instance", func() {
						It("returns 404", func() {
							EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, postInstanceRequest["service_plan_id"].(string), "")
							createInstance(ctx.SMWithOAuth, false, http.StatusCreated)

							ctx.SMWithOAuthForTenant.PATCH(web.ServiceInstancesURL + "/" + instanceID).
								WithJSON(Object{"service_plan_id": anotherServicePlanID}).
								Expect().Status(http.StatusNotFound)
						})
					})

					When("tenant has ownership of instance", func() {
						It("returns 200", func() {
							EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, servicePlanID, TenantIDValue)
							createInstance(ctx.SMWithOAuthForTenant, false, http.StatusCreated)

							ctx.SMWithOAuthForTenant.PATCH(web.ServiceInstancesURL + "/" + instanceID).
								WithJSON(Object{"platform_id": types.SMPlatform}).
								Expect().Status(http.StatusOK)
						})
					})
				})
			})

			Describe("DELETE", func() {
				It("returns 405 for bulk delete", func() {
					ctx.SMWithOAuthForTenant.DELETE(web.ServiceInstancesURL).
						Expect().Status(http.StatusMethodNotAllowed)
				})

				for _, testCase := range testCases {
					testCase := testCase

					Context(fmt.Sprintf("async = %t", testCase.async), func() {
						Context("instance ownership", func() {
							When("tenant doesn't have ownership of instance", func() {
								It("returns 404", func() {
									EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, postInstanceRequest["service_plan_id"].(string), "")
									createInstance(ctx.SMWithOAuth, testCase.async, testCase.expectedCreateSuccessStatusCode)
									expectedCode := http.StatusNotFound
									if testCase.async {
										expectedCode = http.StatusAccepted
									}
									deleteInstance(ctx.SMWithOAuthForTenant, testCase.async, expectedCode)
									instanceID, _ = VerifyOperationExists(ctx, "", OperationExpectations{
										Category:          types.DELETE,
										State:             types.FAILED,
										ResourceType:      types.ServiceInstanceType,
										Reschedulable:     false,
										DeletionScheduled: false,
									})
								})
							})

							When("tenant has ownership of instance", func() {
								It("returns 200", func() {
									EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, servicePlanID, TenantIDValue)
									createInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedCreateSuccessStatusCode)

									deleteInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedDeleteSuccessStatusCode)
								})
							})
						})

						Context("broker scenarios", func() {
							var doneChannel chan interface{}
							var cancelCtx context.Context
							var cancelFunc context.CancelFunc
							BeforeEach(func() {
								EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, servicePlanID, TenantIDValue)
								doneChannel = make(chan interface{})
								cancelCtx, cancelFunc = context.WithCancel(context.Background())
								resp := createInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedCreateSuccessStatusCode)

								instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
									Category:          types.CREATE,
									State:             types.SUCCEEDED,
									ResourceType:      types.ServiceInstanceType,
									Reschedulable:     false,
									DeletionScheduled: false,
								})

								verifyInstanceExists(instanceID, true)
							})

							AfterEach(func() {
								close(doneChannel)
								brokerServer.ResetHandlers()
							})

							When("a delete operation is already in progress", func() {
								BeforeEach(func() {
									brokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"1", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
									brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodDelete+"1", DelayingHandler(doneChannel, cancelFunc))

									resp := deleteInstance(ctx.SMWithOAuthForTenant, true, http.StatusAccepted)

									instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.DELETE,
										State:             types.IN_PROGRESS,
										ResourceType:      types.ServiceInstanceType,
										Reschedulable:     true,
										DeletionScheduled: false,
									})

									verifyInstanceExists(instanceID, true)
								})

								It("updates fail with operation in progress", func() {
									ctx.SMWithOAuthForTenant.PATCH(web.ServiceInstancesURL+"/"+instanceID).WithQuery("async", testCase.async).WithJSON(Object{}).
										Expect().Status(http.StatusUnprocessableEntity)
								})

								It("deletes fail with operation in progress", func() {
									ctx.SMWithOAuthForTenant.DELETE(web.ServiceInstancesURL+"/"+instanceID).WithQuery("async", testCase.async).
										Expect().Status(http.StatusUnprocessableEntity)
								})
							})

							When("binding exists for the instance", func() {
								It("fails to delete it and marks the operation as failed", func() {
									ctx.SMWithOAuthForTenant.POST(web.ServiceBindingsURL).
										WithQuery("async", false).
										WithJSON(Object{
											"name":                "test-service-binding",
											"service_instance_id": instanceID,
										}).
										Expect().
										Status(http.StatusCreated)

									expectedStatus := http.StatusBadRequest
									if testCase.async {
										expectedStatus = http.StatusAccepted
									}
									resp := deleteInstance(ctx.SMWithOAuthForTenant, testCase.async, expectedStatus)

									instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.DELETE,
										State:             types.FAILED,
										ResourceType:      types.ServiceInstanceType,
										Reschedulable:     false,
										DeletionScheduled: false,
									})

									verifyInstanceExists(instanceID, true)
								})
							})

							When("broker responds with synchronous success", func() {
								BeforeEach(func() {
									brokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"1", ParameterizedHandler(http.StatusOK, Object{"async": false}))
								})

								It("deletes the instance and stores a delete succeeded operation", func() {
									resp := deleteInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedDeleteSuccessStatusCode)

									instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.DELETE,
										State:             types.SUCCEEDED,
										ResourceType:      types.ServiceInstanceType,
										Reschedulable:     false,
										DeletionScheduled: false,
									})

									verifyInstanceDoesNotExist(instanceID)
								})
							})

							When("broker responds with 410 GONE", func() {
								BeforeEach(func() {
									brokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"1", ParameterizedHandler(http.StatusGone, Object{}))
								})

								It("deletes the instance and stores a delete succeeded operation", func() {
									resp := deleteInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedDeleteSuccessStatusCode)

									instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.DELETE,
										State:             types.SUCCEEDED,
										ResourceType:      types.ServiceInstanceType,
										Reschedulable:     false,
										DeletionScheduled: false,
									})

									verifyInstanceDoesNotExist(instanceID)
								})
							})

							When("broker responds with asynchronous success", func() {
								BeforeEach(func() {
									brokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"1", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
									brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodDelete+"1", MultiplePollsRequiredHandler("in progress", "succeeded"))
								})

								It("polling broker last operation until operation succeeds and eventually marks operation as success", func() {
									resp := deleteInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedDeleteSuccessStatusCode)

									instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.DELETE,
										State:             types.SUCCEEDED,
										ResourceType:      types.ServiceInstanceType,
										Reschedulable:     false,
										DeletionScheduled: false,
									})

									verifyInstanceDoesNotExist(instanceID)
								})

								When("polling responds 410 GONE", func() {
									BeforeEach(func() {
										brokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"1", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
										brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodDelete+"1", ParameterizedHandler(http.StatusGone, Object{}))
									})

									It("keeps polling and eventually deletes the binding and marks the operation as success", func() {
										resp := deleteInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedDeleteSuccessStatusCode)

										instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
											Category:          types.DELETE,
											State:             types.SUCCEEDED,
											ResourceType:      types.ServiceInstanceType,
											Reschedulable:     false,
											DeletionScheduled: false,
										})

										verifyInstanceDoesNotExist(instanceID)
									})
								})

								When("polling responds with unexpected state and eventually with success state", func() {
									BeforeEach(func() {
										brokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"1", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
										brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodDelete+"1", MultiplePollsRequiredHandler("unknown", "succeeded"))
									})

									It("keeps polling and eventually deletes the instance and marks the operation as success", func() {
										resp := deleteInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedDeleteSuccessStatusCode)

										instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
											Category:          types.DELETE,
											State:             types.SUCCEEDED,
											ResourceType:      types.ServiceInstanceType,
											Reschedulable:     false,
											DeletionScheduled: false,
										})

										verifyInstanceDoesNotExist(instanceID)
									})
								})

								When("polling responds with unexpected state and eventually with failed state", func() {
									BeforeEach(func() {
										brokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"2", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
										brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodDelete+"2", MultiplePollsRequiredHandler("unknown", "failed"))
									})

									When("orphan mitigation deprovision synchronously succeeds", func() {
										It("deletes the instance and marks the operation as success", func() {
											resp := deleteInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)

											instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.DELETE,
												State:             types.FAILED,
												ResourceType:      types.ServiceInstanceType,
												Reschedulable:     false,
												DeletionScheduled: true,
											})

											brokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"2", ParameterizedHandler(http.StatusOK, Object{"async": false}))

											instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.DELETE,
												State:             types.SUCCEEDED,
												ResourceType:      types.ServiceInstanceType,
												Reschedulable:     false,
												DeletionScheduled: false,
											})

											verifyInstanceDoesNotExist(instanceID)
										})
									})

									When("broker orphan mitigation deprovision synchronously fails with an unexpected error", func() {
										It("keeps in the instance and marks the operation with deletion scheduled", func() {
											resp := deleteInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)

											instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.DELETE,
												State:             types.FAILED,
												ResourceType:      types.ServiceInstanceType,
												Reschedulable:     false,
												DeletionScheduled: true,
											})

											brokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"2", ParameterizedHandler(http.StatusBadRequest, Object{"error": "error"}))

											instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.DELETE,
												State:             types.FAILED,
												ResourceType:      types.ServiceInstanceType,
												Reschedulable:     false,
												DeletionScheduled: true,
											})

											verifyInstanceExists(instanceID, true)
										})
									})

									When("broker orphan mitigation deprovision synchronously fails with an error that will continue further orphan mitigation and eventually succeed", func() {
										It("deletes the instance and marks the operation that triggered the orphan mitigation as failed with no deletion scheduled and not reschedulable", func() {
											resp := deleteInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)

											instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.DELETE,
												State:             types.FAILED,
												ResourceType:      types.ServiceInstanceType,
												Reschedulable:     false,
												DeletionScheduled: true,
											})

											brokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"2", MultipleErrorsBeforeSuccessHandler(
												http.StatusInternalServerError, http.StatusOK,
												Object{"error": "error"}, Object{"async": false},
											))

											instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.DELETE,
												State:             types.SUCCEEDED,
												ResourceType:      types.ServiceInstanceType,
												Reschedulable:     false,
												DeletionScheduled: false,
											})

											verifyInstanceDoesNotExist(instanceID)
										})
									})
								})

								When("polling returns an unexpected status code", func() {
									BeforeEach(func() {
										brokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"3", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
										brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodDelete+"3", ParameterizedHandler(http.StatusInternalServerError, Object{"error": "error"}))
									})

									It("keeps the instance and stores the operation as reschedulable", func() {
										resp := deleteInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)

										instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
											Category:          types.DELETE,
											State:             types.FAILED,
											ResourceType:      types.ServiceInstanceType,
											Reschedulable:     true,
											DeletionScheduled: false,
										})

										verifyInstanceExists(instanceID, true)
									})
								})

								When("broker stops while polling", func() {
									BeforeEach(func() {
										brokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"3", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
										brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodDelete+"3", DelayingHandler(doneChannel, cancelFunc))
									})

									It("keeps the instance and stores the operation as reschedulable", func() {
										resp := deleteInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)

										<-cancelCtx.Done()
										brokerServer.Close()
										delete(ctx.Servers, BrokerServerPrefix+brokerID)

										instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
											Category:          types.DELETE,
											State:             types.FAILED,
											ResourceType:      types.ServiceInstanceType,
											Reschedulable:     true,
											DeletionScheduled: false,
										})

										verifyInstanceExists(instanceID, true)
									})
								})
							})

							When("deprovision responds with error due to stopped broker", func() {
								BeforeEach(func() {
									brokerServer.Close()
									delete(ctx.Servers, BrokerServerPrefix+brokerID)
								})

								It("keeps the instance and marks operation with failed", func() {
									resp := deleteInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)

									instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.DELETE,
										State:             types.FAILED,
										ResourceType:      types.ServiceInstanceType,
										Reschedulable:     false,
										DeletionScheduled: false,
									})

									verifyInstanceExists(instanceID, true)
								})
							})

							When("deprovision responds with error that does not require orphan mitigation", func() {
								BeforeEach(func() {
									brokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"3", ParameterizedHandler(http.StatusBadRequest, Object{"error": "error"}))
								})

								It("keeps the instance and marks the operation as failed", func() {
									resp := deleteInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)
									instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.DELETE,
										State:             types.FAILED,
										ResourceType:      types.ServiceInstanceType,
										Reschedulable:     false,
										DeletionScheduled: false,
									})

									verifyInstanceExists(instanceID, true)
								})
							})

							When("deprovision responds with error that requires orphan mitigation", func() {
								BeforeEach(func() {
									brokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"3", ParameterizedHandler(http.StatusInternalServerError, Object{"error": "error"}))
								})

								When("orphan mitigation deprovision asynchronously succeeds", func() {
									It("deletes the instance and marks the operation as success", func() {
										resp := deleteInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)

										instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
											Category:          types.DELETE,
											State:             types.FAILED,
											ResourceType:      types.ServiceInstanceType,
											Reschedulable:     false,
											DeletionScheduled: true,
										})

										brokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"3", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
										brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodDelete+"3", ParameterizedHandler(http.StatusOK, Object{"state": "succeeded"}))

										instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
											Category:          types.DELETE,
											State:             types.SUCCEEDED,
											ResourceType:      types.ServiceInstanceType,
											Reschedulable:     false,
											DeletionScheduled: false,
										})

										verifyInstanceDoesNotExist(instanceID)
									})
								})

								if testCase.async {
									When("broker orphan mitigation deprovision asynchronously keeps failing with an error while polling", func() {
										It("keeps the instance and marks the operation as failed reschedulable with deletion scheduled", func() {
											resp := deleteInstance(ctx.SMWithOAuthForTenant, true, http.StatusAccepted)

											instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.DELETE,
												State:             types.FAILED,
												ResourceType:      types.ServiceInstanceType,
												Reschedulable:     false,
												DeletionScheduled: true,
											})

											brokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"3", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
											brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodDelete+"3", ParameterizedHandler(http.StatusBadRequest, Object{"error": "error"}))

											instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.DELETE,
												State:             types.FAILED,
												ResourceType:      types.ServiceInstanceType,
												Reschedulable:     true,
												DeletionScheduled: true,
											})

											verifyInstanceExists(instanceID, true)
										})
									})
								}

								When("broker orphan mitigation deprovision asynchronously fails with an error that will continue further orphan mitigation and eventually succeed", func() {
									It("deletes the instance and marks the operation as success", func() {
										resp := deleteInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)

										instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
											Category:          types.DELETE,
											State:             types.FAILED,
											ResourceType:      types.ServiceInstanceType,
											Reschedulable:     false,
											DeletionScheduled: true,
										})

										brokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"3", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
										brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodDelete+"3", MultipleErrorsBeforeSuccessHandler(
											http.StatusOK, http.StatusOK,
											Object{"state": "failed"}, Object{"state": "succeeded"},
										))

										instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
											Category:          types.DELETE,
											State:             types.SUCCEEDED,
											ResourceType:      types.ServiceInstanceType,
											Reschedulable:     false,
											DeletionScheduled: false,
										})

										verifyInstanceDoesNotExist(instanceID)
									})
								})
							})

							When("deprovision responds with error due to times out", func() {
								var oldCtx *TestContext
								BeforeEach(func() {
									oldCtx = ctx
									ctx = NewTestContextBuilderWithSecurity().WithEnvPreExtensions(func(set *pflag.FlagSet) {
										Expect(set.Set("httpclient.response_header_timeout", (1 * time.Second).String())).ToNot(HaveOccurred())
									}).Build()

									var plans *httpexpect.Array
									brokerID, brokerServer, plans = prepareBrokerWithCatalog(ctx, ctx.SMWithOAuth)
									servicePlanID = plans.Element(0).Object().Value("id").String().Raw()
									postInstanceRequest = Object{
										"name":             "test-instance",
										"service_plan_id":  servicePlanID,
										"maintenance_info": "{}",
									}

									EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, servicePlanID, TenantIDValue)
									resp := createInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedCreateSuccessStatusCode)

									instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.CREATE,
										State:             types.SUCCEEDED,
										ResourceType:      types.ServiceInstanceType,
										Reschedulable:     false,
										DeletionScheduled: false,
									})

									verifyInstanceExists(instanceID, true)

									brokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"1", DelayingHandler(doneChannel, cancelFunc))

								})

								AfterEach(func() {
									ctx.SMRepository.Delete(context.TODO(), types.OperationType)
									DeleteInstance(ctx, instanceID, servicePlanID)
									ctx.SMWithOAuth.DELETE(web.ServiceBrokersURL + "/" + brokerID).Expect()
									delete(ctx.Servers, BrokerServerPrefix+brokerID)
									brokerServer.Close()
									ctx = oldCtx
								})

								It("orphan mitigates the instance", func() {
									resp := deleteInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)

									instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.DELETE,
										State:             types.FAILED,
										ResourceType:      types.ServiceInstanceType,
										Reschedulable:     false,
										DeletionScheduled: true,
									})

									brokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"1", ParameterizedHandler(http.StatusOK, Object{"async": false}))

									instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.DELETE,
										State:             types.SUCCEEDED,
										ResourceType:      types.ServiceInstanceType,
										Reschedulable:     false,
										DeletionScheduled: false,
									})

									verifyInstanceDoesNotExist(instanceID)
								})
							})
						})
					})
				}
			})
		})
	},
})

func blueprint(ctx *TestContext, auth *SMExpect, async bool) Object {
	ID, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}

	instanceReqBody := make(Object, 0)
	instanceReqBody["name"] = "test-service-instance-" + ID.String()
	_, _, array := prepareBrokerWithCatalog(ctx, auth)
	instanceReqBody["service_plan_id"] = array.First().Object().Value("id").String().Raw()

	EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, instanceReqBody["service_plan_id"].(string), "")
	resp := auth.POST(web.ServiceInstancesURL).WithQuery("async", strconv.FormatBool(async)).WithJSON(instanceReqBody).Expect()

	var instance map[string]interface{}
	if async {
		instance = ExpectSuccessfulAsyncResourceCreation(resp, auth, web.ServiceInstancesURL)
	} else {
		instance = resp.Status(http.StatusCreated).JSON().Object().Raw()
	}

	return instance
}

func prepareBrokerWithCatalog(ctx *TestContext, auth *SMExpect) (string, *BrokerServer, *httpexpect.Array) {
	cPaidPlan1 := GeneratePaidTestPlan()
	cPaidPlan2 := GeneratePaidTestPlan()
	cService := GenerateTestServiceWithPlans(cPaidPlan1, cPaidPlan2)
	catalog := NewEmptySBCatalog()
	catalog.AddService(cService)
	brokerID, _, server := ctx.RegisterBrokerWithCatalog(catalog)
	ctx.Servers[BrokerServerPrefix+brokerID] = server

	so := auth.ListWithQuery(web.ServiceOfferingsURL, fmt.Sprintf("fieldQuery=broker_id eq '%s'", brokerID)).First()

	return brokerID, server, auth.ListWithQuery(web.ServicePlansURL, "fieldQuery="+fmt.Sprintf("service_offering_id eq '%s'", so.Object().Value("id").String().Raw()))
}
