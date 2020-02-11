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
	"github.com/Peripli/service-manager/operations"
	"github.com/Peripli/service-manager/pkg/env"
	"github.com/Peripli/service-manager/pkg/query"
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
	DisableTenantResources:                 false,
	StrictlyTenantScoped:                   true,
	ResourceBlueprint:                      blueprint,
	ResourceWithoutNullableFieldsBlueprint: blueprint,
	ResourcePropertiesToIgnore:             []string{"platform_id"},
	PatchResource:                          APIResourcePatch,
	AdditionalTests: func(ctx *TestContext, t *TestCase) {
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
				expectedSMCrashStatusCode       int
			}

			testCases := []testCase{
				{
					async:                           false,
					expectedCreateSuccessStatusCode: http.StatusCreated,
					expectedDeleteSuccessStatusCode: http.StatusOK,
					expectedBrokerFailureStatusCode: http.StatusBadGateway,
					expectedSMCrashStatusCode:       http.StatusBadGateway,
				},
				{
					async:                           true,
					expectedCreateSuccessStatusCode: http.StatusAccepted,
					expectedDeleteSuccessStatusCode: http.StatusAccepted,
					expectedBrokerFailureStatusCode: http.StatusAccepted,
					expectedSMCrashStatusCode:       http.StatusAccepted,
				},
			}

			createInstance := func(smClient *SMExpect, expectedStatusCode int) *httpexpect.Response {
				resp := smClient.POST(web.ServiceInstancesURL).WithJSON(postInstanceRequest).
					Expect().Status(expectedStatusCode)

				return resp
			}

			createInstanceWithAsync := func(smClient *SMExpect, async bool, expectedStatusCode int) *httpexpect.Response {
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
					Expect().
					Status(expectedStatusCode)
			}

			verifyInstanceExists := func(ctx *TestContext, instanceID string, ready bool) {
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
							readyField := instanceObject.Value("ready").Boolean().Raw()
							if readyField != ready {
								By(fmt.Sprintf("Expected instance with id %s to be ready %t but ready was %t. Retrying...", instanceID, ready, readyField))
							} else {
								return
							}
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
				brokerServer.ShouldRecordRequests(false)
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

				When("service instance contains tenant identifier in OSB context", func() {
					BeforeEach(func() {
						EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, servicePlanID, TenantIDValue)
						resp := createInstanceWithAsync(ctx.SMWithOAuthForTenant, false, http.StatusCreated)
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

				When("service instance dashboard_url is not set", func() {
					BeforeEach(func() {
						postInstanceRequest["dashboard_url"] = ""
						EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, postInstanceRequest["service_plan_id"].(string), TenantIDValue)
						createInstanceWithAsync(ctx.SMWithOAuthForTenant, false, http.StatusCreated)
					})

					It("doesn't return dashboard_url", func() {
						ctx.SMWithOAuthForTenant.GET(web.ServiceInstancesURL + "/" + instanceID).Expect().
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
									resp := createInstanceWithAsync(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedCreateSuccessStatusCode)
									instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.CREATE,
										State:             types.SUCCEEDED,
										ResourceType:      types.ServiceInstanceType,
										Reschedulable:     false,
										DeletionScheduled: false,
									})

									verifyInstanceExists(ctx, instanceID, true)
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
									resp := ctx.SMWithOAuthForTenant.POST(web.ServiceInstancesURL).
										WithJSON(postInstanceRequest).
										WithQuery("async", testCase.async).
										Expect().Status(http.StatusBadRequest).JSON().Object()

									resp.Value("description").Equal("Providing platform_id property during provisioning/updating of a service instance is forbidden")
								})
							})

							Context("which is service-manager platform", func() {
								It(fmt.Sprintf("should return %d", testCase.expectedCreateSuccessStatusCode), func() {
									postInstanceRequest["platform_id"] = types.SMPlatform
									EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, postInstanceRequest["service_plan_id"].(string), TenantIDValue)
									createInstanceWithAsync(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedCreateSuccessStatusCode)
								})
							})
						})

						Context("OSB context", func() {
							BeforeEach(func() {
								brokerServer.ServiceInstanceHandlerFunc(http.MethodPut, http.MethodPut+"1", func(req *http.Request) (int, map[string]interface{}) {
									body, err := util.BodyToBytes(req.Body)
									Expect(err).ToNot(HaveOccurred())
									tenantValue := gjson.GetBytes(body, "context."+TenantIdentifier).String()
									Expect(tenantValue).To(Equal(TenantIDValue))
									platformValue := gjson.GetBytes(body, "context.platform").String()
									Expect(platformValue).To(Equal(types.SMPlatform))

									return http.StatusCreated, Object{}
								})
							})

							It("enriches the osb context with the tenant and sm platform", func() {
								EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, postInstanceRequest["service_plan_id"].(string), TenantIDValue)
								createInstanceWithAsync(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedCreateSuccessStatusCode)
							})
						})

						Context("instance visibility", func() {
							When("tenant doesn't have plan visibility", func() {
								It("returns 404", func() {
									createInstanceWithAsync(ctx.SMWithOAuthForTenant, testCase.async, http.StatusNotFound)
								})
							})

							When("tenant has plan visibility", func() {
								It(fmt.Sprintf("returns %d", testCase.expectedCreateSuccessStatusCode), func() {
									EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, servicePlanID, TenantIDValue)
									createInstanceWithAsync(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedCreateSuccessStatusCode)
								})
							})

							When("plan has public visibility", func() {
								It(fmt.Sprintf("for tenant returns %d", testCase.expectedCreateSuccessStatusCode), func() {
									EnsurePublicPlanVisibility(ctx.SMRepository, servicePlanID)
									createInstanceWithAsync(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedCreateSuccessStatusCode)
								})
							})
						})

						Context("broker scenarios", func() {
							BeforeEach(func() {
								EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, servicePlanID, TenantIDValue)
							})

							When("a create operation is already in progress", func() {
								var doneChannel chan interface{}

								BeforeEach(func() {
									doneChannel = make(chan interface{})

									brokerServer.ServiceInstanceHandlerFunc(http.MethodPut, http.MethodPut+"1", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
									brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodPut+"1", DelayingHandler(doneChannel))

									resp := createInstanceWithAsync(ctx.SMWithOAuthForTenant, true, http.StatusAccepted)

									instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.CREATE,
										State:             types.IN_PROGRESS,
										ResourceType:      types.ServiceInstanceType,
										Reschedulable:     true,
										DeletionScheduled: false,
									})

									verifyInstanceExists(ctx, instanceID, false)
								})

								AfterEach(func() {
									close(doneChannel)
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
									createInstanceWithAsync(ctx.SMWithOAuthForTenant, testCase.async, http.StatusNotFound)
								})
							})

							When("SM crashes after storing operation before storing resource", func() {
								var newCtx *TestContext

								postHookWithShutdownTimeout := func() func(e env.Environment, servers map[string]FakeServer) {
									return func(e env.Environment, servers map[string]FakeServer) {
										e.Set("server.shutdown_timeout", 1*time.Second)
										e.Set("httpclient.response_header_timeout", 1*time.Second)
									}
								}

								BeforeEach(func() {
									ctxMaintainerBuilder := t.ContextBuilder.WithEnvPostExtensions(postHookWithShutdownTimeout())
									newCtx = ctxMaintainerBuilder.BuildWithoutCleanup()

									brokerServer.ServiceInstanceHandlerFunc(http.MethodPut, http.MethodPut+"3", func(_ *http.Request) (int, map[string]interface{}) {
										defer newCtx.CleanupAll(false)
										return http.StatusOK, Object{"state": "in progress"}
									})

									brokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"3", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
									brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodDelete+"3", func(_ *http.Request) (int, map[string]interface{}) {
										return http.StatusOK, Object{"state": "succeeded"}
									})
								})

								It("Should mark operation as failed and trigger orphan mitigation", func() {
									opChan := make(chan *types.Operation)
									defer close(opChan)

									opCriteria := []query.Criterion{
										query.ByField(query.EqualsOperator, "type", string(types.CREATE)),
										query.ByField(query.EqualsOperator, "state", string(types.IN_PROGRESS)),
										query.ByField(query.EqualsOperator, "resource_type", string(types.ServiceInstanceType)),
										query.ByField(query.EqualsOperator, "reschedule", "false"),
										query.ByField(query.EqualsOperator, "deletion_scheduled", operations.ZeroTime),
									}

									go func() {
										for {
											object, err := ctx.SMRepository.Get(context.TODO(), types.OperationType, opCriteria...)
											if err == nil {
												opChan <- object.(*types.Operation)
												break
											}
										}
									}()

									createInstanceWithAsync(newCtx.SMWithOAuthForTenant, testCase.async, testCase.expectedSMCrashStatusCode)
									operation := <-opChan

									verifyInstanceDoesNotExist(operation.ResourceID)

									operationExpectation := OperationExpectations{
										Category:          types.CREATE,
										State:             types.FAILED,
										ResourceType:      types.ServiceInstanceType,
										Reschedulable:     false,
										DeletionScheduled: false,
									}

									instanceID, _ = VerifyOperationExists(ctx, fmt.Sprintf("%s/%s%s/%s", web.ServiceInstancesURL, operation.ResourceID, web.OperationsURL, operation.ID), operationExpectation)
									verifyInstanceDoesNotExist(instanceID)
								})
							})

							When("broker responds with synchronous success", func() {
								BeforeEach(func() {
									brokerServer.ServiceInstanceHandlerFunc(http.MethodPut, http.MethodPut+"1", ParameterizedHandler(http.StatusCreated, Object{"async": false}))
								})

								It("stores instance as ready=true and the operation as success, non rescheduable with no deletion scheduled", func() {
									resp := createInstanceWithAsync(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedCreateSuccessStatusCode)

									instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.CREATE,
										State:             types.SUCCEEDED,
										ResourceType:      types.ServiceInstanceType,
										Reschedulable:     false,
										DeletionScheduled: false,
									})

									verifyInstanceExists(ctx, instanceID, true)
								})
							})

							When("broker responds with asynchronous success", func() {
								BeforeEach(func() {
									brokerServer.ServiceInstanceHandlerFunc(http.MethodPut, http.MethodPut+"1", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
									brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodPut+"1", MultiplePollsRequiredHandler("in progress", "succeeded"))
								})

								It("polling broker last operation until operation succeeds and eventually marks operation as success", func() {
									resp := createInstanceWithAsync(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedCreateSuccessStatusCode)

									instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.CREATE,
										State:             types.SUCCEEDED,
										ResourceType:      types.ServiceInstanceType,
										Reschedulable:     false,
										DeletionScheduled: false,
									})

									verifyInstanceExists(ctx, instanceID, true)
								})

								if testCase.async {
									When("job timeout is reached while polling", func() {
										var oldCtx *TestContext

										BeforeEach(func() {
											oldCtx = ctx
											ctx = NewTestContextBuilderWithSecurity().WithEnvPreExtensions(func(set *pflag.FlagSet) {
												Expect(set.Set("operations.job_timeout", (2 * time.Second).String())).ToNot(HaveOccurred())
											}).BuildWithoutCleanup()

											brokerServer.ServiceInstanceHandlerFunc(http.MethodPut, http.MethodPut+"1", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
											brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodPut+"1", ParameterizedHandler(http.StatusOK, Object{"state": "in progress"}))
										})

										AfterEach(func() {
											ctx = oldCtx
										})

										It("stores instance as ready false and the operation as reschedulable in progress", func() {
											resp := createInstanceWithAsync(ctx.SMWithOAuthForTenant, true, http.StatusAccepted)

											instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.CREATE,
												State:             types.IN_PROGRESS,
												ResourceType:      types.ServiceInstanceType,
												Reschedulable:     true,
												DeletionScheduled: false,
											})

											verifyInstanceExists(ctx, instanceID, false)
										})
									})

									When("SM crashes while polling", func() {
										var newCtx *TestContext
										var isProvisioned = false

										postHookWithShutdownTimeout := func() func(e env.Environment, servers map[string]FakeServer) {
											return func(e env.Environment, servers map[string]FakeServer) {
												e.Set("server.shutdown_timeout", 1*time.Second)
											}
										}

										BeforeEach(func() {
											ctxMaintainerBuilder := t.ContextBuilder.WithEnvPostExtensions(postHookWithShutdownTimeout())
											newCtx = ctxMaintainerBuilder.BuildWithoutCleanup()

											brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodPut+"1", func(_ *http.Request) (int, map[string]interface{}) {
												if isProvisioned {
													return http.StatusOK, Object{"state": "succeeded"}
												} else {
													return http.StatusOK, Object{"state": "in progress"}
												}
											})
										})

										It("should start restart polling through maintainer and eventually instance is set to ready", func() {
											resp := createInstanceWithAsync(newCtx.SMWithOAuthForTenant, testCase.async, testCase.expectedCreateSuccessStatusCode)

											operationExpectation := OperationExpectations{
												Category:          types.CREATE,
												State:             types.IN_PROGRESS,
												ResourceType:      types.ServiceInstanceType,
												Reschedulable:     true,
												DeletionScheduled: false,
											}

											instanceID, _ = VerifyOperationExists(newCtx, resp.Header("Location").Raw(), operationExpectation)
											verifyInstanceExists(newCtx, instanceID, false)

											newCtx.CleanupAll(false)

											isProvisioned = true

											operationExpectation.State = types.SUCCEEDED
											operationExpectation.Reschedulable = false

											instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), operationExpectation)
											verifyInstanceExists(ctx, instanceID, true)
										})
									})
								}

								When("polling responds with unexpected state and eventually with success state", func() {
									BeforeEach(func() {
										brokerServer.ServiceInstanceHandlerFunc(http.MethodPut, http.MethodPut+"1", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
										brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodPut+"1", MultiplePollsRequiredHandler("unknown", "succeeded"))
									})

									It("keeps polling and eventually updates the instance to ready true and operation to success", func() {
										resp := createInstanceWithAsync(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedCreateSuccessStatusCode)

										instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
											Category:          types.CREATE,
											State:             types.SUCCEEDED,
											ResourceType:      types.ServiceInstanceType,
											Reschedulable:     false,
											DeletionScheduled: false,
										})
										verifyInstanceExists(ctx, instanceID, true)
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
											resp := createInstanceWithAsync(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)

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

									When("broker orphan mitigation deprovision synchronously fails with an unexpected status code", func() {
										BeforeEach(func() {
											brokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"3", ParameterizedHandler(http.StatusBadRequest, Object{"error": "error"}))
										})

										AfterEach(func() {
											brokerServer.ResetHandlers()
										})

										It("keeps in the instance with ready false and marks the operation with deletion scheduled", func() {
											resp := createInstanceWithAsync(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)

											instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.CREATE,
												State:             types.FAILED,
												ResourceType:      types.ServiceInstanceType,
												Reschedulable:     false,
												DeletionScheduled: true,
											})

											verifyInstanceExists(ctx, instanceID, false)
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
											resp := createInstanceWithAsync(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)

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
										resp := createInstanceWithAsync(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)

										instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
											Category:          types.CREATE,
											State:             types.FAILED,
											ResourceType:      types.ServiceInstanceType,
											Reschedulable:     true,
											DeletionScheduled: false,
										})

										verifyInstanceExists(ctx, instanceID, false)
									})
								})
							})

							When("provision responds with error due to stopped broker", func() {
								BeforeEach(func() {
									brokerServer.Close()
									delete(ctx.Servers, BrokerServerPrefix+brokerID)
								})

								It("does not store instance in SMDB and marks operation with failed", func() {
									resp := createInstanceWithAsync(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)

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
									resp := createInstanceWithAsync(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)

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

								AfterEach(func() {
									brokerServer.ResetHandlers()
								})

								When("orphan mitigation deprovision asynchronously succeeds", func() {
									BeforeEach(func() {
										brokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"3", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
										brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodDelete+"3", ParameterizedHandler(http.StatusOK, Object{"state": "succeeded"}))
									})

									It("deletes the instance and marks the operation that triggered the orphan mitigation as failed with no deletion scheduled and not reschedulable", func() {
										resp := createInstanceWithAsync(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)

										instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
											Category:          types.CREATE,
											State:             types.FAILED,
											ResourceType:      types.ServiceInstanceType,
											Reschedulable:     false,
											DeletionScheduled: false,
										})

										verifyInstanceDoesNotExist(instanceID)
									})

									When("maximum deletion timout has been reached", func() {
										var oldCtx *TestContext
										BeforeEach(func() {
											oldCtx = ctx
											ctx = NewTestContextBuilderWithSecurity().WithEnvPreExtensions(func(set *pflag.FlagSet) {
												Expect(set.Set("operations.reconciliation_operation_timeout", (2 * time.Millisecond).String())).ToNot(HaveOccurred())
											}).BuildWithoutCleanup()
										})

										AfterEach(func() {
											ctx = oldCtx
										})

										It("keeps the instance as ready false and marks the operation as deletion scheduled", func() {
											resp := createInstanceWithAsync(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)

											instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.CREATE,
												State:             types.FAILED,
												ResourceType:      types.ServiceInstanceType,
												Reschedulable:     false,
												DeletionScheduled: true,
											})

											verifyInstanceExists(ctx, instanceID, false)
										})
									})
								})

								if testCase.async {
									When("broker orphan mitigation deprovision asynchronously keeps failing with an error while polling", func() {
										BeforeEach(func() {
											brokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"3", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
											brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodDelete+"3", ParameterizedHandler(http.StatusBadRequest, Object{"error": "error"}))
										})

										It("keeps the instance as ready false and marks the operation as deletion scheduled", func() {
											resp := createInstanceWithAsync(ctx.SMWithOAuthForTenant, true, http.StatusAccepted)

											instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.CREATE,
												State:             types.FAILED,
												ResourceType:      types.ServiceInstanceType,
												Reschedulable:     true,
												DeletionScheduled: true,
											})

											verifyInstanceExists(ctx, instanceID, false)
										})
									})
								}

								When("SM crashes while orphan mitigating", func() {
									var newCtx *TestContext
									var isDeprovisioned = false

									postHookWithShutdownTimeout := func() func(e env.Environment, servers map[string]FakeServer) {
										return func(e env.Environment, servers map[string]FakeServer) {
											e.Set("server.shutdown_timeout", 1*time.Second)
										}
									}

									BeforeEach(func() {
										ctxMaintainerBuilder := t.ContextBuilder.WithEnvPostExtensions(postHookWithShutdownTimeout())
										newCtx = ctxMaintainerBuilder.BuildWithoutCleanup()

										brokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"3", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
										brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodDelete+"3", func(_ *http.Request) (int, map[string]interface{}) {
											if isDeprovisioned {
												return http.StatusOK, Object{"state": "succeeded"}
											} else {
												return http.StatusOK, Object{"state": "in progress"}
											}
										})
									})

									It("should restart orphan mitigation through maintainer and eventually succeeds", func() {
										resp := createInstanceWithAsync(newCtx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)

										operationExpectations := OperationExpectations{
											Category:          types.CREATE,
											State:             types.FAILED,
											ResourceType:      types.ServiceInstanceType,
											Reschedulable:     true,
											DeletionScheduled: true,
										}

										instanceID, _ = VerifyOperationExists(newCtx, resp.Header("Location").Raw(), operationExpectations)

										newCtx.CleanupAll(false)

										isDeprovisioned = true

										operationExpectations.DeletionScheduled = false
										operationExpectations.Reschedulable = false
										instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), operationExpectations)

										verifyInstanceDoesNotExist(instanceID)
									})

								})

								When("broker orphan mitigation deprovision asynchronously fails with an error that will continue further orphan mitigation and eventually succeed", func() {
									BeforeEach(func() {
										brokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"3", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))

										brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodDelete+"3", MultipleErrorsBeforeSuccessHandler(
											http.StatusOK, http.StatusOK,
											Object{"state": "failed"}, Object{"state": "succeeded"},
										))
									})

									It("deletes the instance and marks the operation that triggered the orphan mitigation as failed with no deletion scheduled and not reschedulable", func() {
										resp := createInstanceWithAsync(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)

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

							When("provision responds with error due to time out", func() {
								var doneChannel chan interface{}
								var oldCtx *TestContext

								BeforeEach(func() {
									oldCtx = ctx
									doneChannel = make(chan interface{})
									ctx = NewTestContextBuilderWithSecurity().WithEnvPreExtensions(func(set *pflag.FlagSet) {
										Expect(set.Set("httpclient.response_header_timeout", (1 * time.Second).String())).ToNot(HaveOccurred())
									}).BuildWithoutCleanup()

									brokerServer.ServiceInstanceHandlerFunc(http.MethodPut, http.MethodPut+"1", DelayingHandler(doneChannel))
									brokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"1", ParameterizedHandler(http.StatusOK, Object{}))
								})

								AfterEach(func() {
									ctx = oldCtx
								})

								It("orphan mitigates the instance", func() {
									resp := createInstanceWithAsync(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)
									<-time.After(1100 * time.Millisecond)
									close(doneChannel)
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
						ctx.SMWithOAuthForTenant.PATCH(web.ServiceInstancesURL+"/no_such_id").
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
						EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, postInstanceRequest["service_plan_id"].(string), TenantIDValue)
						resp := createInstance(ctx.SMWithOAuthForTenant, http.StatusAccepted)
						instance := ExpectSuccessfulAsyncResourceCreation(resp, ctx.SMWithOAuth, web.ServiceInstancesURL)
						instanceID := instance["id"].(string)

						createdAt := "2015-01-01T00:00:00Z"

						resp = ctx.SMWithOAuthForTenant.PATCH(web.ServiceInstancesURL + "/" + instanceID).
							WithJSON(Object{"created_at": createdAt}).
							Expect().
							Status(http.StatusAccepted)

						instance = ExpectSuccessfulAsyncResourceCreation(resp, ctx.SMWithOAuth, web.ServiceInstancesURL)
						Expect(instance["created_at"].(string)).ToNot(Equal(createdAt))

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
							EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, postInstanceRequest["service_plan_id"].(string), TenantIDValue)
							createInstanceWithAsync(ctx.SMWithOAuthForTenant, false, http.StatusCreated)

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
							EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, postInstanceRequest["service_plan_id"].(string), TenantIDValue)
							resp := createInstance(ctx.SMWithOAuthForTenant, http.StatusAccepted)
							instance := ExpectSuccessfulAsyncResourceCreation(resp, ctx.SMWithOAuth, web.ServiceInstancesURL)
							instanceID := instance["id"].(string)

							resp = ctx.SMWithOAuthForTenant.PATCH(web.ServiceInstancesURL + "/" + instanceID).
								WithJSON(Object{"platform_id": types.SMPlatform}).
								Expect().Status(http.StatusAccepted)

							instance = ExpectSuccessfulAsyncResourceCreation(resp, ctx.SMWithOAuth, web.ServiceInstancesURL)
							Expect(instance["platform_id"].(string)).To(Equal(types.SMPlatform))

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
						EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, postInstanceRequest["service_plan_id"].(string), TenantIDValue)
						resp := createInstance(ctx.SMWithOAuthForTenant, http.StatusAccepted)
						instance := ExpectSuccessfulAsyncResourceCreation(resp, ctx.SMWithOAuth, web.ServiceInstancesURL)
						instanceID := instance["id"].(string)

						for _, prop := range []string{"name", "maintenance_info"} {
							updatedBrokerJSON := Object{}
							updatedBrokerJSON[prop] = "updated-" + prop
							resp = ctx.SMWithOAuthForTenant.PATCH(web.ServiceInstancesURL + "/" + instanceID).
								WithJSON(updatedBrokerJSON).
								Expect().
								Status(http.StatusAccepted)

							ExpectSuccessfulAsyncResourceCreation(resp, ctx.SMWithOAuth, web.ServiceInstancesURL)

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
							createInstanceWithAsync(ctx.SMWithOAuthForTenant, false, http.StatusCreated)

							ctx.SMWithOAuthForTenant.PATCH(web.ServiceInstancesURL + "/" + instanceID).
								WithJSON(Object{"service_plan_id": anotherServicePlanID}).
								Expect().Status(http.StatusNotFound)
						})
					})

					When("tenant has plan visibility", func() {
						It("returns 201", func() {
							EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, servicePlanID, TenantIDValue)
							resp := createInstance(ctx.SMWithOAuthForTenant, http.StatusAccepted)
							instance := ExpectSuccessfulAsyncResourceCreation(resp, ctx.SMWithOAuth, web.ServiceInstancesURL)
							instanceID := instance["id"].(string)

							EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, anotherServicePlanID, TenantIDValue)
							resp = ctx.SMWithOAuthForTenant.PATCH(web.ServiceInstancesURL + "/" + instanceID).
								WithJSON(Object{"service_plan_id": anotherServicePlanID}).
								Expect().Status(http.StatusAccepted)

							ExpectSuccessfulAsyncResourceCreation(resp, ctx.SMWithOAuth, web.ServiceInstancesURL)
						})
					})
				})

				Context("instance ownership", func() {
					When("tenant doesn't have ownership of instance", func() {
						It("returns 404", func() {
							EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, postInstanceRequest["service_plan_id"].(string), TenantIDValue)
							createInstanceWithAsync(ctx.SMWithOAuthForTenant, false, http.StatusCreated)

							otherTenantExpect := ctx.NewTenantExpect("other-tenant")
							otherTenantExpect.PATCH(web.ServiceInstancesURL + "/" + instanceID).
								WithJSON(Object{"service_plan_id": anotherServicePlanID}).
								Expect().Status(http.StatusNotFound)
						})
					})

					When("tenant has ownership of instance", func() {
						It("returns 200", func() {
							EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, servicePlanID, TenantIDValue)
							resp := createInstance(ctx.SMWithOAuthForTenant, http.StatusAccepted)
							instance := ExpectSuccessfulAsyncResourceCreation(resp, ctx.SMWithOAuth, web.ServiceInstancesURL)
							instanceID := instance["id"].(string)

							resp = ctx.SMWithOAuthForTenant.PATCH(web.ServiceInstancesURL + "/" + instanceID).
								WithJSON(Object{"platform_id": types.SMPlatform}).
								Expect().Status(http.StatusAccepted)

							ExpectSuccessfulAsyncResourceCreation(resp, ctx.SMWithOAuth, web.ServiceInstancesURL)
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
						BeforeEach(func() {
							brokerServer.ShouldRecordRequests(true)
						})

						AfterEach(func() {
							brokerServer.ResetHandlers()
							ctx.SMWithOAuth.DELETE(web.ServiceInstancesURL + "/" + instanceID).Expect()
							ctx.SMWithOAuthForTenant.DELETE(web.ServiceInstancesURL + "/" + instanceID).Expect()
						})

						Context("instance ownership", func() {
							When("tenant doesn't have ownership of instance", func() {
								It("returns 404", func() {
									EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, postInstanceRequest["service_plan_id"].(string), TenantIDValue)
									resp := createInstanceWithAsync(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedCreateSuccessStatusCode)
									instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.CREATE,
										State:             types.SUCCEEDED,
										ResourceType:      types.ServiceInstanceType,
										Reschedulable:     false,
										DeletionScheduled: false,
									})
									expectedCode := http.StatusNotFound
									if testCase.async {
										expectedCode = http.StatusAccepted
									}
									otherTenantExpect := ctx.NewTenantExpect("other-tenant")
									deleteInstance(otherTenantExpect, testCase.async, expectedCode)
								})
							})

							When("tenant has ownership of instance", func() {
								It("returns 200", func() {
									EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, servicePlanID, TenantIDValue)
									resp := createInstanceWithAsync(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedCreateSuccessStatusCode)
									instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.CREATE,
										State:             types.SUCCEEDED,
										ResourceType:      types.ServiceInstanceType,
										Reschedulable:     false,
										DeletionScheduled: false,
									})

									resp = deleteInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedDeleteSuccessStatusCode)
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

						Context("broker scenarios", func() {
							BeforeEach(func() {
								EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, servicePlanID, TenantIDValue)
								resp := createInstanceWithAsync(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedCreateSuccessStatusCode)

								instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
									Category:          types.CREATE,
									State:             types.SUCCEEDED,
									ResourceType:      types.ServiceInstanceType,
									Reschedulable:     false,
									DeletionScheduled: false,
								})

								verifyInstanceExists(ctx, instanceID, true)
							})

							When("a delete operation is already in progress", func() {
								var doneChannel chan interface{}

								BeforeEach(func() {
									doneChannel = make(chan interface{})
									brokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"1", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
									brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodDelete+"1", DelayingHandler(doneChannel))

									resp := deleteInstance(ctx.SMWithOAuthForTenant, true, http.StatusAccepted)

									instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.DELETE,
										State:             types.IN_PROGRESS,
										ResourceType:      types.ServiceInstanceType,
										Reschedulable:     true,
										DeletionScheduled: false,
									})

									verifyInstanceExists(ctx, instanceID, true)
								})

								AfterEach(func() {
									close(doneChannel)
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
								var bindingID string

								AfterEach(func() {
									ctx.SMWithOAuthForTenant.DELETE(web.ServiceBindingsURL + "/" + bindingID).
										Expect().StatusRange(httpexpect.Status2xx)
								})

								It("fails to delete it and marks the operation as failed", func() {
									bindingID = ctx.SMWithOAuthForTenant.POST(web.ServiceBindingsURL).
										WithQuery("async", false).
										WithJSON(Object{
											"name":                "test-service-binding",
											"service_instance_id": instanceID,
										}).
										Expect().
										Status(http.StatusCreated).JSON().Object().Value("id").String().Raw()

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

									verifyInstanceExists(ctx, instanceID, true)
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

								if testCase.async {
									When("SM crashes while polling", func() {
										var newCtx *TestContext
										var isDeprovisioned = false

										postHookWithShutdownTimeout := func() func(e env.Environment, servers map[string]FakeServer) {
											return func(e env.Environment, servers map[string]FakeServer) {
												e.Set("server.shutdown_timeout", 1*time.Second)
											}
										}

										BeforeEach(func() {
											ctxMaintainerBuilder := t.ContextBuilder.WithEnvPostExtensions(postHookWithShutdownTimeout())
											newCtx = ctxMaintainerBuilder.BuildWithoutCleanup()

											brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodDelete+"1", func(_ *http.Request) (int, map[string]interface{}) {
												if isDeprovisioned {
													return http.StatusOK, Object{"state": "succeeded"}
												} else {
													return http.StatusOK, Object{"state": "in progress"}
												}
											})
										})

										It("should restart polling through maintainer and eventually deletes the instance", func() {
											resp := deleteInstance(newCtx.SMWithOAuthForTenant, true, http.StatusAccepted)

											operationExpectations := OperationExpectations{
												Category:          types.DELETE,
												State:             types.IN_PROGRESS,
												ResourceType:      types.ServiceInstanceType,
												Reschedulable:     true,
												DeletionScheduled: false,
											}

											instanceID, _ = VerifyOperationExists(newCtx, resp.Header("Location").Raw(), operationExpectations)
											verifyInstanceExists(newCtx, instanceID, true)

											newCtx.CleanupAll(false)

											isDeprovisioned = true

											operationExpectations.State = types.SUCCEEDED
											operationExpectations.Reschedulable = false

											instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), operationExpectations)
											verifyInstanceDoesNotExist(instanceID)

										})
									})
								}

								When("polling responds 410 GONE", func() {
									BeforeEach(func() {
										brokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"1", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
										brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodDelete+"1", ParameterizedHandler(http.StatusGone, Object{}))
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

											verifyInstanceExists(ctx, instanceID, true)
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

									When("maximum deletion timout has been reached", func() {
										var oldCtx *TestContext
										BeforeEach(func() {
											oldCtx = ctx
											ctx = NewTestContextBuilderWithSecurity().WithEnvPreExtensions(func(set *pflag.FlagSet) {
												Expect(set.Set("operations.reconciliation_operation_timeout", (2 * time.Millisecond).String())).ToNot(HaveOccurred())
											}).BuildWithoutCleanup()
										})

										AfterEach(func() {
											ctx = oldCtx
										})

										It("keeps the instance as ready false and marks the operation as deletion scheduled", func() {
											resp := deleteInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)

											instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.DELETE,
												State:             types.FAILED,
												ResourceType:      types.ServiceInstanceType,
												Reschedulable:     false,
												DeletionScheduled: true,
											})

											verifyInstanceExists(ctx, instanceID, true)
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

										verifyInstanceExists(ctx, instanceID, true)
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

									verifyInstanceExists(ctx, instanceID, true)
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

									verifyInstanceExists(ctx, instanceID, true)
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

											verifyInstanceExists(ctx, instanceID, true)
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
								var newCtx *TestContext
								var doneChannel chan interface{}

								BeforeEach(func() {
									doneChannel = make(chan interface{})

									newCtx = t.ContextBuilder.WithEnvPreExtensions(func(set *pflag.FlagSet) {
										Expect(set.Set("httpclient.response_header_timeout", (1 * time.Second).String())).ToNot(HaveOccurred())
									}).BuildWithoutCleanup()

									brokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"1", DelayingHandler(doneChannel))
								})

								AfterEach(func() {
									newCtx.CleanupAll(false)
								})

								It("orphan mitigates the instance", func() {
									resp := deleteInstance(newCtx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)
									<-time.After(1100 * time.Millisecond)
									close(doneChannel)

									instanceID, _ = VerifyOperationExists(newCtx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.DELETE,
										State:             types.FAILED,
										ResourceType:      types.ServiceInstanceType,
										Reschedulable:     false,
										DeletionScheduled: true,
									})

									brokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"1", ParameterizedHandler(http.StatusOK, Object{"async": false}))

									instanceID, _ = VerifyOperationExists(newCtx, resp.Header("Location").Raw(), OperationExpectations{
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

	EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, instanceReqBody["service_plan_id"].(string), TenantIDValue)
	resp := ctx.SMWithOAuthForTenant.POST(web.ServiceInstancesURL).WithQuery("async", strconv.FormatBool(async)).WithJSON(instanceReqBody).Expect()

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
