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

package service_binding_test

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/spf13/pflag"

	"github.com/Peripli/service-manager/pkg/util"
	"github.com/tidwall/gjson"

	"github.com/gavv/httpexpect"

	"github.com/Peripli/service-manager/pkg/query"

	"testing"

	"github.com/Peripli/service-manager/pkg/types"

	"github.com/Peripli/service-manager/pkg/web"
	. "github.com/Peripli/service-manager/test/common"

	. "github.com/Peripli/service-manager/test"

	. "github.com/onsi/ginkgo"

	. "github.com/onsi/gomega"
)

func TestServiceBindings(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Bindings Tests Suite")
}

const (
	TenantIdentifier = "tenant"
	TenantIDValue    = "tenantID"
)

var _ = DescribeTestsFor(TestCase{
	API: web.ServiceBindingsURL,
	SupportedOps: []Op{
		Get, List, Delete,
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
	ResourceType:                           types.ServiceBindingType,
	SupportsAsyncOperations:                true,
	DisableTenantResources:                 true,
	ResourceBlueprint:                      blueprint,
	ResourceWithoutNullableFieldsBlueprint: blueprint,
	ResourcePropertiesToIgnore:             []string{"volume_mounts", "endpoints", "bind_resource", "credentials"},
	PatchResource:                          StorageResourcePatch,
	AdditionalTests: func(ctx *TestContext) {
		Context("additional non-generic tests", func() {
			var (
				postBindingRequest  Object
				instanceID          string
				instanceName        string
				instanceOperationID string
				bindingID           string
				bindingOperationID  string
				brokerID            string
				brokerServer        *BrokerServer
				servicePlanID       string
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
				postInstanceRequest := Object{
					"name":             "test-instance",
					"service_plan_id":  servicePlanID,
					"maintenance_info": "{}",
				}

				resp := smClient.POST(web.ServiceInstancesURL).
					WithQuery("async", async).
					WithJSON(postInstanceRequest).
					Expect().
					Status(expectedStatusCode)

				if resp.Raw().StatusCode == http.StatusCreated {
					obj := resp.JSON().Object()

					obj.ContainsKey("id").
						ValueEqual("platform_id", types.SMPlatform)

					instanceID = obj.Value("id").String().Raw()
				}

				return resp
			}

			createBinding := func(SM *SMExpect, async bool, expectedStatusCode int) *httpexpect.Response {
				resp := SM.POST(web.ServiceBindingsURL).
					WithQuery("async", async).
					WithJSON(postBindingRequest).
					Expect().
					Status(expectedStatusCode)
				obj := resp.JSON().Object()

				if expectedStatusCode == http.StatusCreated {
					obj.ContainsKey("id")
					bindingID = obj.Value("id").String().Raw()
				}

				return resp
			}

			deleteBinding := func(smClient *SMExpect, async bool, expectedStatusCode int) *httpexpect.Response {
				return smClient.DELETE(web.ServiceBindingsURL+"/"+bindingID).
					WithQuery("async", async).
					Expect().
					Status(expectedStatusCode)
			}

			verifyBindingExists := func(smClient *SMExpect, bindingID string, ready bool) {
				timeoutDuration := 25 * time.Second
				tickerInterval := 100 * time.Millisecond
				ticker := time.NewTicker(tickerInterval)
				timeout := time.After(timeoutDuration)
				defer ticker.Stop()
				for {
					select {
					case <-timeout:
						Fail(fmt.Sprintf("binding with id %s did not appear in SM after %.0f seconds", bindingID, timeoutDuration.Seconds()))
					case <-ticker.C:
						bindings := smClient.ListWithQuery(web.ServiceBindingsURL, fmt.Sprintf("fieldQuery=id eq '%s'", bindingID))
						switch {
						case bindings.Length().Raw() == 0:
							By(fmt.Sprintf("Could not find binding with id %s in SM. Retrying...", bindingID))
						case bindings.Length().Raw() > 1:
							Fail(fmt.Sprintf("more than one binding with id %s was found in SM", bindingID))
						default:
							bindingObject := bindings.First().Object()
							//bindingObject.Path(fmt.Sprintf("$.labels[%s][*]", TenantIdentifier)).Array().Contains(TenantIDValue)
							readyField := bindingObject.Value("ready").Boolean().Raw()
							if readyField != ready {
								Fail(fmt.Sprintf("Expected binding with id %s to be ready %t but ready was %t", bindingID, ready, readyField))
							}
							return
						}
					}
				}
			}

			verifyBindingDoesNotExist := func(smClient *SMExpect, bindingID string) {
				timeoutDuration := 25 * time.Second
				tickerInterval := 100 * time.Millisecond
				ticker := time.NewTicker(tickerInterval)
				timeout := time.After(timeoutDuration)

				defer ticker.Stop()
				for {
					select {
					case <-timeout:
						Fail(fmt.Sprintf("binding with id %s was still in SM after %.0f seconds", bindingID, timeoutDuration.Seconds()))
					case <-ticker.C:
						resp := smClient.GET(web.ServiceBindingsURL + "/" + bindingID).
							Expect().Raw()
						if resp.StatusCode != http.StatusNotFound {
							By(fmt.Sprintf("Found binding with id %s but it should be deleted. Retrying...", bindingID))
						} else {
							return
						}
					}
				}
			}

			BeforeEach(func() {
				brokerID, brokerServer, servicePlanID = newServicePlan(ctx)
				EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, servicePlanID, TenantIDValue)
				resp := createInstance(ctx.SMWithOAuthForTenant, false, http.StatusCreated)
				instanceName = resp.JSON().Object().Value("name").String().Raw()
				Expect(instanceName).ToNot(BeEmpty())

				postBindingRequest = Object{
					"name":                "test-binding",
					"service_instance_id": instanceID,
				}
			})

			JustBeforeEach(func() {
				postBindingRequest = Object{
					"name":                "test-binding",
					"service_instance_id": instanceID,
				}
			})

			AfterEach(func() {
				ctx.SMRepository.Delete(context.TODO(), types.OperationType, query.ByField(query.InOperator, "id", bindingOperationID, instanceOperationID))
				DeleteBinding(ctx, bindingID, instanceID)
				DeleteInstance(ctx, instanceID, servicePlanID)
				ctx.SMWithOAuth.DELETE(web.ServiceBrokersURL + "/" + brokerID).Expect()
				delete(ctx.Servers, BrokerServerPrefix+brokerID)
				brokerServer.Close()
			})

			FDescribe("GET", func() {
				When("service binding contains tenant identifier in OSB context", func() {
					BeforeEach(func() {
						createBinding(ctx.SMWithOAuthForTenant, false, http.StatusCreated)
					})

					It("labels instance with tenant identifier", func() {
						ctx.SMWithOAuthForTenant.GET(web.ServiceBindingsURL + "/" + bindingID).Expect().
							Status(http.StatusOK).
							JSON().
							Object().Path(fmt.Sprintf("$.labels[%s][*]", TenantIdentifier)).Array().Contains(TenantIDValue)
					})

					It("returns OSB context with no tenant as part of the binding", func() {
						ctx.SMWithOAuthForTenant.GET(web.ServiceBindingsURL + "/" + bindingID).Expect().
							Status(http.StatusOK).
							JSON().
							Object().Value("context").Object().Equal(map[string]interface{}{
							"platform":      types.SMPlatform,
							"instance_name": instanceName,
						})
					})
				})

				When("service binding doesn't contain tenant identifier in OSB context", func() {
					BeforeEach(func() {
						createBinding(ctx.SMWithOAuth, false, http.StatusCreated)
					})

					It("doesn't label instance with tenant identifier", func() {
						obj := ctx.SMWithOAuth.GET(web.ServiceBindingsURL + "/" + bindingID).Expect().
							Status(http.StatusOK).JSON().Object()

						objMap := obj.Raw()
						objLabels, exist := objMap["labels"]
						if exist {
							labels := objLabels.(map[string]interface{})
							_, tenantLabelExists := labels[TenantIdentifier]
							Expect(tenantLabelExists).To(BeFalse())
						}
					})

					It("returns OSB context with tenant as part of the binding", func() {
						ctx.SMWithOAuth.GET(web.ServiceBindingsURL + "/" + bindingID).Expect().
							Status(http.StatusOK).
							JSON().
							Object().Value("context").Object().Equal(map[string]interface{}{
							"platform":       types.SMPlatform,
							"instance_name":  instanceName,
							TenantIdentifier: TenantIDValue,
						})
					})
				})
			})

			Describe("POST", func() {
				for _, testCase := range testCases {
					testCase := testCase
					Context(fmt.Sprintf("async = %t", testCase.async), func() {
						Context("when content type is not JSON", func() {
							It("returns 415", func() {
								ctx.SMWithOAuth.POST(web.ServiceBindingsURL).
									WithQuery("async", testCase.async).
									WithText("text").
									Expect().
									Status(http.StatusUnsupportedMediaType).
									JSON().Object().
									Keys().Contains("error", "description")
							})
						})

						Context("when request body is not a valid JSON", func() {
							It("returns 400", func() {
								ctx.SMWithOAuth.POST(web.ServiceBindingsURL).
									WithQuery("async", testCase.async).
									WithText("invalid json").
									WithHeader("content-type", "application/json").
									Expect().
									Status(http.StatusBadRequest).
									JSON().Object().
									Keys().Contains("error", "description")
							})
						})

						Context("when a request body field is missing", func() {
							assertPOSTReturns400WhenFieldIsMissing := func(field string) {
								JustBeforeEach(func() {
									delete(postBindingRequest, field)
								})

								It("returns 400", func() {
									ctx.SMWithOAuth.POST(web.ServiceBindingsURL).
										WithQuery("async", testCase.async).
										WithJSON(postBindingRequest).
										Expect().
										Status(http.StatusBadRequest).
										JSON().Object().
										Keys().Contains("error", "description")
								})
							}

							assertPOSTReturns201WhenFieldIsMissing := func(field string) {
								JustBeforeEach(func() {
									delete(postBindingRequest, field)
								})

								It("returns 201", func() {
									resp := createBinding(ctx.SMWithOAuth, testCase.async, testCase.expectedCreateSuccessStatusCode)
									bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.CREATE,
										State:             types.SUCCEEDED,
										ResourceType:      types.ServiceBindingType,
										Reschedulable:     false,
										DeletionScheduled: false,
									})

									verifyBindingExists(ctx.SMWithOAuth, bindingID, true)
								})
							}

							Context("when id field is missing", func() {
								assertPOSTReturns201WhenFieldIsMissing("id")
							})

							Context("when name field is missing", func() {
								assertPOSTReturns400WhenFieldIsMissing("name")
							})

							Context("when service_instance_id field is missing", func() {
								assertPOSTReturns400WhenFieldIsMissing("service_instance_id")
							})

						})

						Context("when request body id field is provided", func() {
							It("should return 400", func() {
								postBindingRequest["id"] = "test-binding-id"
								resp := ctx.SMWithOAuth.
									POST(web.ServiceBindingsURL).
									WithQuery("async", testCase.async).
									WithJSON(postBindingRequest).
									Expect().
									Status(http.StatusBadRequest).JSON().Object()
								Expect(resp.Value("description").String().Raw()).To(ContainSubstring("providing specific resource id is forbidden"))
							})
						})

						Context("OSB context", func() {
							It("enriches the osb context with the tenant and sm platform", func() {
								createBinding(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedCreateSuccessStatusCode)
								for _, bindRequest := range brokerServer.BindingEndpointRequests {
									body, err := util.BodyToBytes(bindRequest.Body)
									Expect(err).ToNot(HaveOccurred())
									tenantValue := gjson.GetBytes(body, "context."+TenantIdentifier).String()
									Expect(tenantValue).To(Equal(TenantIDValue))
									platformValue := gjson.GetBytes(body, "context.platform").String()
									Expect(platformValue).To(Equal(types.SMPlatform))
								}
							})
						})

						Context("instance visibility", func() {
							When("tenant doesn't have ownership of instance", func() {
								BeforeEach(func() {
									createInstance(ctx.SMWithOAuth, false, http.StatusCreated)
								})

								It("returns 404", func() {
									createBinding(ctx.SMWithOAuthForTenant, testCase.async, http.StatusNotFound)
								})
							})

							When("tenant has ownership of instance", func() {
								It("returns 201", func() {
									resp := createBinding(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedCreateSuccessStatusCode)
									bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.CREATE,
										State:             types.SUCCEEDED,
										ResourceType:      types.ServiceBindingType,
										Reschedulable:     false,
										DeletionScheduled: false,
									})
									verifyBindingExists(ctx.SMWithOAuthForTenant, bindingID, true)
								})
							})
						})

						Context("broker scenarios", func() {
							var doneChannel chan interface{}
							var cancelCtx context.Context
							var cancelFunc context.CancelFunc

							BeforeEach(func() {
								doneChannel = make(chan interface{})
								cancelCtx, cancelFunc = context.WithCancel(context.Background())
							})

							AfterEach(func() {
								close(doneChannel)
								brokerServer.ResetHandlers()
							})

							When("instance creation is still in progress", func() {
								It("fails to create binding", func() {

								})
							})

							When("instance creation and orphan mitigation failed", func() {
								It("fails to create binding", func() {

								})
							})

							When("plan is not bindable", func() {
								It("fails to create binding", func() {

								})
							})

							When("a create operation is already in progress", func() {
								BeforeEach(func() {
									brokerServer.BindingHandlerFunc(http.MethodPut, http.MethodPut+"1", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
									brokerServer.BindingLastOpHandlerFunc(http.MethodPut+"1", DelayingHandler(doneChannel, cancelFunc))

									resp := createBinding(ctx.SMWithOAuthForTenant, true, http.StatusAccepted)

									bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.CREATE,
										State:             types.IN_PROGRESS,
										ResourceType:      types.ServiceBindingType,
										Reschedulable:     true,
										DeletionScheduled: false,
									})

									verifyBindingExists(ctx.SMWithOAuthForTenant, bindingID, false)
								})

								It("deletes succeed", func() {
									resp := ctx.SMWithOAuthForTenant.DELETE(web.ServiceBindingsURL+"/"+bindingID).WithQuery("async", testCase.async).
										Expect().StatusRange(httpexpect.Status2xx)

									bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.DELETE,
										State:             types.SUCCEEDED,
										ResourceType:      types.ServiceBindingType,
										Reschedulable:     false,
										DeletionScheduled: false,
									})

									verifyBindingDoesNotExist(ctx.SMWithOAuthForTenant, bindingID)
								})
							})

							When("instance does not exist", func() {
								JustBeforeEach(func() {
									postBindingRequest["service_instance_id"] = "non-existing-id"
								})

								It("bind fails", func() {
									createBinding(ctx.SMWithOAuthForTenant, testCase.async, http.StatusNotFound)
								})
							})

							When("broker responds with synchronous success", func() {
								BeforeEach(func() {
									brokerServer.BindingHandlerFunc(http.MethodPut, http.MethodPut+"1", ParameterizedHandler(http.StatusCreated, Object{"async": false}))
								})

								It("stores binding as ready=true and the operation as success, non rescheduable with no deletion scheduled", func() {
									resp := createBinding(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedCreateSuccessStatusCode)

									bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.CREATE,
										State:             types.SUCCEEDED,
										ResourceType:      types.ServiceBindingType,
										Reschedulable:     false,
										DeletionScheduled: false,
									})

									verifyBindingExists(ctx.SMWithOAuthForTenant, bindingID, true)
								})
							})

							When("broker responds with asynchronous success", func() {
								BeforeEach(func() {
									brokerServer.BindingHandlerFunc(http.MethodPut, http.MethodPut+"1", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
									brokerServer.BindingLastOpHandlerFunc(http.MethodPut+"1", MultiplePollsRequiredHandler("in progress", "succeeded"))
								})

								It("polling broker last operation until operation succeeds and eventually marks operation as success", func() {
									resp := createBinding(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedCreateSuccessStatusCode)

									bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.CREATE,
										State:             types.SUCCEEDED,
										ResourceType:      types.ServiceBindingType,
										Reschedulable:     false,
										DeletionScheduled: false,
									})

									verifyBindingExists(ctx.SMWithOAuthForTenant, bindingID, true)
								})

								if testCase.async {
									When("job timeout is reached while polling", func() {
										var oldCtx *TestContext
										BeforeEach(func() {
											oldCtx = ctx
											ctx = NewTestContextBuilderWithSecurity().WithEnvPreExtensions(func(set *pflag.FlagSet) {
												Expect(set.Set("operations.job_timeout", (2 * time.Second).String())).ToNot(HaveOccurred())
											}).Build()

											brokerID, brokerServer, servicePlanID = newServicePlan(ctx)
											EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, servicePlanID, TenantIDValue)
											createInstance(ctx.SMWithOAuthForTenant, false, http.StatusCreated)

											brokerServer.BindingHandlerFunc(http.MethodPut, http.MethodPut+"1", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
											brokerServer.BindingLastOpHandlerFunc(http.MethodPut+"1", ParameterizedHandler(http.StatusOK, Object{"state": "in progress"}))
										})

										AfterEach(func() {
											ctx.SMRepository.Delete(context.TODO(), types.OperationType)
											DeleteBinding(ctx, bindingID, instanceID)
											DeleteInstance(ctx, instanceID, servicePlanID)
											ctx.SMWithOAuth.DELETE(web.ServiceBrokersURL + "/" + brokerID).Expect()
											delete(ctx.Servers, BrokerServerPrefix+brokerID)
											brokerServer.Close()
											ctx = oldCtx
										})

										It("stores binding as ready false and the operation as reschedulable in progress", func() {
											resp := createBinding(ctx.SMWithOAuthForTenant, true, http.StatusAccepted)

											bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.CREATE,
												State:             types.IN_PROGRESS,
												ResourceType:      types.ServiceBindingType,
												Reschedulable:     true,
												DeletionScheduled: false,
											})

											verifyBindingExists(ctx.SMWithOAuthForTenant, bindingID, false)
										})
									})
								}

								When("polling responds with unexpected state and eventually with success state", func() {
									BeforeEach(func() {
										brokerServer.BindingHandlerFunc(http.MethodPut, http.MethodPut+"1", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
										brokerServer.BindingLastOpHandlerFunc(http.MethodPut+"1", MultiplePollsRequiredHandler("unknown", "succeeded"))
									})

									It("keeps polling and eventually updates the binding to ready true and operation to success", func() {
										resp := createBinding(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedCreateSuccessStatusCode)

										bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
											Category:          types.CREATE,
											State:             types.SUCCEEDED,
											ResourceType:      types.ServiceBindingType,
											Reschedulable:     false,
											DeletionScheduled: false,
										})
										verifyBindingExists(ctx.SMWithOAuthForTenant, bindingID, true)
									})
								})

								When("polling responds with unexpected state and eventually with failed state", func() {
									BeforeEach(func() {
										brokerServer.BindingHandlerFunc(http.MethodPut, http.MethodPut+"2", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
										brokerServer.BindingLastOpHandlerFunc(http.MethodPut+"2", MultiplePollsRequiredHandler("unknown", "failed"))
									})

									When("orphan mitigation unbind synchronously succeeds", func() {
										BeforeEach(func() {
											brokerServer.BindingHandlerFunc(http.MethodDelete, http.MethodDelete+"3", ParameterizedHandler(http.StatusOK, Object{"async": false}))
										})

										It("deletes the binding and marks the operation that triggered the orphan mitigation as failed with no deletion scheduled and not reschedulable", func() {
											resp := createBinding(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)

											bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.CREATE,
												State:             types.FAILED,
												ResourceType:      types.ServiceBindingType,
												Reschedulable:     false,
												DeletionScheduled: false,
											})

											verifyBindingDoesNotExist(ctx.SMWithOAuthForTenant, bindingID)
										})
									})

									When("broker orphan mitigation unbind synchronously fails with an error that will stop further orphan mitigation", func() {
										BeforeEach(func() {
											brokerServer.BindingHandlerFunc(http.MethodDelete, http.MethodDelete+"3", ParameterizedHandler(http.StatusBadRequest, Object{"error": "error"}))
										})

										It("keeps the binding with ready false and marks the operation with deletion scheduled", func() {
											resp := createBinding(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)

											bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.CREATE,
												State:             types.FAILED,
												ResourceType:      types.ServiceBindingType,
												Reschedulable:     false,
												DeletionScheduled: true,
											})

											verifyBindingExists(ctx.SMWithOAuthForTenant, bindingID, !testCase.async)
										})
									})

									When("broker orphan mitigation unbind synchronously fails with an error that will continue further orphan mitigation and eventually succeed", func() {
										BeforeEach(func() {
											brokerServer.BindingHandlerFunc(http.MethodDelete, http.MethodDelete+"3", MultipleErrorsBeforeSuccessHandler(
												http.StatusInternalServerError, http.StatusOK,
												Object{"error": "error"}, Object{"async": "false"},
											))
										})

										It("deletes the binding and marks the operation that triggered the orphan mitigation as failed with no deletion scheduled and not reschedulable", func() {
											resp := createBinding(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)

											bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.CREATE,
												State:             types.FAILED,
												ResourceType:      types.ServiceBindingType,
												Reschedulable:     false,
												DeletionScheduled: false,
											})

											verifyBindingDoesNotExist(ctx.SMWithOAuthForTenant, bindingID)
										})
									})
								})

								When("polling returns an unexpected status code", func() {
									BeforeEach(func() {
										brokerServer.BindingHandlerFunc(http.MethodPut, http.MethodPut+"3", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
										brokerServer.BindingLastOpHandlerFunc(http.MethodPut+"3", ParameterizedHandler(http.StatusInternalServerError, Object{"error": "error"}))
									})

									It("stores the binding as ready false and marks the operation as reschedulable", func() {
										resp := createBinding(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)

										bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
											Category:          types.CREATE,
											State:             types.FAILED,
											ResourceType:      types.ServiceBindingType,
											Reschedulable:     true,
											DeletionScheduled: false,
										})

										verifyBindingExists(ctx.SMWithOAuthForTenant, bindingID, !testCase.async)
									})
								})

								When("broker stops while polling", func() {
									BeforeEach(func() {
										brokerServer.BindingHandlerFunc(http.MethodPut, http.MethodPut+"3", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
										brokerServer.BindingLastOpHandlerFunc(http.MethodPut+"3", DelayingHandler(doneChannel, cancelFunc))
									})

									It("keeps the binding as ready false and marks the operation as failed reschedulable with no orphan mitigation", func() {
										resp := createBinding(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)

										<-cancelCtx.Done()
										brokerServer.Close()
										delete(ctx.Servers, BrokerServerPrefix+brokerID)

										bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
											Category:          types.CREATE,
											State:             types.FAILED,
											ResourceType:      types.ServiceBindingType,
											Reschedulable:     true,
											DeletionScheduled: false,
										})

										verifyBindingExists(ctx.SMWithOAuthForTenant, bindingID, !testCase.async)
									})
								})
							})

							When("bind responds with error due to stopped broker", func() {
								BeforeEach(func() {
									brokerServer.Close()
									delete(ctx.Servers, BrokerServerPrefix+brokerID)
								})

								It("does not store binding in SMDB and marks operation with failed", func() {
									resp := createBinding(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)

									bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.CREATE,
										State:             types.FAILED,
										ResourceType:      types.ServiceBindingType,
										Reschedulable:     false,
										DeletionScheduled: false,
									})

									verifyBindingDoesNotExist(ctx.SMWithOAuthForTenant, bindingID)
								})
							})

							When("bind responds with error that does not require orphan mitigation", func() {
								BeforeEach(func() {
									brokerServer.BindingHandlerFunc(http.MethodPut, http.MethodPut+"3", ParameterizedHandler(http.StatusBadRequest, Object{"error": "error"}))
								})

								It("does not store the binding and marks the operation as failed, non rescheduable with empty deletion scheduled", func() {
									resp := createBinding(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)

									bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.CREATE,
										State:             types.FAILED,
										ResourceType:      types.ServiceBindingType,
										Reschedulable:     false,
										DeletionScheduled: false,
									})

									verifyBindingDoesNotExist(ctx.SMWithOAuthForTenant, bindingID)
								})
							})

							When("bind responds with error that requires orphan mitigation", func() {
								BeforeEach(func() {
									brokerServer.BindingHandlerFunc(http.MethodPut, http.MethodPut+"3", ParameterizedHandler(http.StatusInternalServerError, Object{"error": "error"}))
								})

								When("orphan mitigation unbind asynchronously succeeds", func() {
									BeforeEach(func() {
										brokerServer.BindingHandlerFunc(http.MethodDelete, http.MethodDelete+"3", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
										brokerServer.BindingLastOpHandlerFunc(http.MethodDelete+"3", ParameterizedHandler(http.StatusOK, Object{"state": "succeeded"}))
									})

									It("deletes the binding and marks the operation that triggered the orphan mitigation as failed with no deletion scheduled and not reschedulable", func() {
										resp := createBinding(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)

										bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
											Category:          types.CREATE,
											State:             types.FAILED,
											ResourceType:      types.ServiceBindingType,
											Reschedulable:     false,
											DeletionScheduled: false,
										})

										verifyBindingDoesNotExist(ctx.SMWithOAuthForTenant, bindingID)
									})
								})

								if testCase.async {
									When("broker orphan mitigation unbind asynchronously keeps failing with an error while polling", func() {
										BeforeEach(func() {
											brokerServer.BindingHandlerFunc(http.MethodDelete, http.MethodDelete+"3", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
											brokerServer.BindingLastOpHandlerFunc(http.MethodDelete+"3", ParameterizedHandler(http.StatusBadRequest, Object{"error": "error"}))
										})

										It("keeps the binding as ready false and marks the operation as deletion scheduled", func() {
											resp := createBinding(ctx.SMWithOAuthForTenant, true, http.StatusAccepted)

											bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.CREATE,
												State:             types.FAILED,
												ResourceType:      types.ServiceBindingType,
												Reschedulable:     true,
												DeletionScheduled: true,
											})

											verifyBindingExists(ctx.SMWithOAuthForTenant, bindingID, false)
										})
									})
								}

								When("broker orphan mitigation unbind asynchronously fails with an error that will continue further orphan mitigation and eventually succeed", func() {
									BeforeEach(func() {
										brokerServer.BindingHandlerFunc(http.MethodDelete, http.MethodDelete+"3", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))

										brokerServer.BindingLastOpHandlerFunc(http.MethodDelete+"3", MultipleErrorsBeforeSuccessHandler(
											http.StatusOK, http.StatusOK,
											Object{"state": "failed"}, Object{"state": "succeeded"},
										))
									})

									It("deletes the binding and marks the operation that triggered the orphan mitigation as failed with no deletion scheduled and not reschedulable", func() {
										resp := createBinding(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)

										bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
											Category:          types.CREATE,
											State:             types.FAILED,
											ResourceType:      types.ServiceBindingType,
											Reschedulable:     false,
											DeletionScheduled: false,
										})

										verifyBindingDoesNotExist(ctx.SMWithOAuthForTenant, bindingID)
									})
								})
							})

							When("bind responds with error due to times out", func() {
								var oldCtx *TestContext
								BeforeEach(func() {
									oldCtx = ctx
									ctx = NewTestContextBuilderWithSecurity().WithEnvPreExtensions(func(set *pflag.FlagSet) {
										Expect(set.Set("httpclient.response_header_timeout", (1 * time.Second).String())).ToNot(HaveOccurred())
									}).Build()

									brokerID, brokerServer, servicePlanID = newServicePlan(ctx)
									EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, servicePlanID, TenantIDValue)
									createInstance(ctx.SMWithOAuthForTenant, false, http.StatusCreated)

									brokerServer.BindingHandlerFunc(http.MethodPut, http.MethodPut+"1", DelayingHandler(doneChannel, cancelFunc))
								})

								AfterEach(func() {
									ctx.SMRepository.Delete(context.TODO(), types.OperationType)
									DeleteBinding(ctx, bindingID, instanceID)
									DeleteInstance(ctx, instanceID, servicePlanID)
									ctx.SMWithOAuth.DELETE(web.ServiceBrokersURL + "/" + brokerID).Expect()
									delete(ctx.Servers, BrokerServerPrefix+brokerID)
									brokerServer.Close()
									ctx = oldCtx
								})

								It("orphan mitigates the binding", func() {
									resp := createBinding(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)

									bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.CREATE,
										State:             types.FAILED,
										ResourceType:      types.ServiceBindingType,
										Reschedulable:     false,
										DeletionScheduled: false,
									})

									verifyBindingDoesNotExist(ctx.SMWithOAuthForTenant, bindingID)
								})
							})
						})

					})
				}
			})

			Describe("DELETE", func() {
				It("returns 405 for bulk delete", func() {
					ctx.SMWithOAuthForTenant.DELETE(web.ServiceBindingsURL).
						Expect().Status(http.StatusMethodNotAllowed)
				})

				for _, testCase := range testCases {
					testCase := testCase
					Context(fmt.Sprintf("async = %t", testCase.async), func() {
						Context("instance ownership", func() {
							When("tenant doesn't have ownership of binding", func() {
								It("returns 404", func() {
									resp := createBinding(ctx.SMWithOAuth, testCase.async, testCase.expectedCreateSuccessStatusCode)
									bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.CREATE,
										State:             types.SUCCEEDED,
										ResourceType:      types.ServiceBindingType,
										Reschedulable:     false,
										DeletionScheduled: false,
									})
									verifyBindingExists(ctx.SMWithOAuth, bindingID, true)

									expectedCode := http.StatusNotFound
									if testCase.async {
										expectedCode = http.StatusAccepted
									}
									deleteBinding(ctx.SMWithOAuthForTenant, testCase.async, expectedCode)

									verifyBindingExists(ctx.SMWithOAuth, bindingID, true)

								})
							})

							When("tenant has ownership of instance", func() {
								It("returns 200", func() {
									resp := createBinding(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedCreateSuccessStatusCode)
									bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.CREATE,
										State:             types.SUCCEEDED,
										ResourceType:      types.ServiceBindingType,
										Reschedulable:     false,
										DeletionScheduled: false,
									})
									verifyBindingExists(ctx.SMWithOAuthForTenant, bindingID, true)

									resp = deleteBinding(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedDeleteSuccessStatusCode)

									bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.DELETE,
										State:             types.SUCCEEDED,
										ResourceType:      types.ServiceBindingType,
										Reschedulable:     false,
										DeletionScheduled: false,
									})
									verifyBindingDoesNotExist(ctx.SMWithOAuthForTenant, bindingID)
								})
							})
						})
						Context("broker scenarios", func() {
							var doneChannel chan interface{}
							var cancelCtx context.Context
							var cancelFunc context.CancelFunc

							BeforeEach(func() {
								doneChannel = make(chan interface{})
								cancelCtx, cancelFunc = context.WithCancel(context.Background())

								resp := createBinding(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedCreateSuccessStatusCode)
								bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
									Category:          types.CREATE,
									State:             types.SUCCEEDED,
									ResourceType:      types.ServiceBindingType,
									Reschedulable:     false,
									DeletionScheduled: false,
								})
								verifyBindingExists(ctx.SMWithOAuth, bindingID, true)

							})

							AfterEach(func() {
								close(doneChannel)
							})

							When("a delete operation is already in progress", func() {
								BeforeEach(func() {
									brokerServer.BindingHandlerFunc(http.MethodDelete, http.MethodDelete+"1", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
									brokerServer.BindingLastOpHandlerFunc(http.MethodDelete+"1", DelayingHandler(doneChannel, cancelFunc))

									resp := deleteBinding(ctx.SMWithOAuthForTenant, true, http.StatusAccepted)

									bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.DELETE,
										State:             types.IN_PROGRESS,
										ResourceType:      types.ServiceBindingType,
										Reschedulable:     true,
										DeletionScheduled: false,
									})

									verifyBindingExists(ctx.SMWithOAuthForTenant, bindingID, true)
								})

								It("deletes fail with operation in progress", func() {
									deleteBinding(ctx.SMWithOAuthForTenant, testCase.async, http.StatusUnprocessableEntity)
								})
							})

							When("broker responds with synchronous success", func() {
								BeforeEach(func() {
									brokerServer.BindingHandlerFunc(http.MethodDelete, http.MethodDelete+"1", ParameterizedHandler(http.StatusOK, Object{"async": false}))
								})

								It("deletes the binding and stores a delete succeeded operation", func() {
									resp := deleteBinding(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedDeleteSuccessStatusCode)

									bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.DELETE,
										State:             types.SUCCEEDED,
										ResourceType:      types.ServiceBindingType,
										Reschedulable:     false,
										DeletionScheduled: false,
									})

									verifyBindingDoesNotExist(ctx.SMWithOAuthForTenant, bindingID)
								})
							})

							When("broker responds with 410 GONE", func() {
								BeforeEach(func() {
									brokerServer.BindingHandlerFunc(http.MethodDelete, http.MethodDelete+"1", ParameterizedHandler(http.StatusGone, Object{}))
								})

								It("deletes the instance and stores a delete succeeded operation", func() {
									resp := deleteBinding(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedDeleteSuccessStatusCode)

									bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.DELETE,
										State:             types.SUCCEEDED,
										ResourceType:      types.ServiceBindingType,
										Reschedulable:     false,
										DeletionScheduled: false,
									})

									verifyBindingDoesNotExist(ctx.SMWithOAuthForTenant, bindingID)
								})
							})

							When("broker responds with asynchronous success", func() {
								BeforeEach(func() {
									brokerServer.BindingHandlerFunc(http.MethodDelete, http.MethodDelete+"1", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
									brokerServer.BindingLastOpHandlerFunc(http.MethodDelete+"1", MultiplePollsRequiredHandler("in progress", "succeeded"))
								})

								It("polling broker last operation until operation succeeds and eventually marks operation as success", func() {
									resp := deleteBinding(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedDeleteSuccessStatusCode)

									bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.DELETE,
										State:             types.SUCCEEDED,
										ResourceType:      types.ServiceBindingType,
										Reschedulable:     false,
										DeletionScheduled: false,
									})

									verifyBindingDoesNotExist(ctx.SMWithOAuthForTenant, bindingID)
								})

								When("polling responds 410 GONE", func() {
									BeforeEach(func() {
										brokerServer.BindingHandlerFunc(http.MethodDelete, http.MethodDelete+"1", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
										brokerServer.BindingLastOpHandlerFunc(http.MethodDelete+"1", ParameterizedHandler(http.StatusGone, Object{}))
									})

									It("keeps polling and eventually deletes the binding and marks the operation as success", func() {
										resp := deleteBinding(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedDeleteSuccessStatusCode)

										bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
											Category:          types.DELETE,
											State:             types.SUCCEEDED,
											ResourceType:      types.ServiceBindingType,
											Reschedulable:     false,
											DeletionScheduled: false,
										})

										verifyBindingDoesNotExist(ctx.SMWithOAuthForTenant, bindingID)
									})
								})

								When("polling responds with unexpected state and eventually with success state", func() {
									BeforeEach(func() {
										brokerServer.BindingHandlerFunc(http.MethodDelete, http.MethodDelete+"1", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
										brokerServer.BindingLastOpHandlerFunc(http.MethodDelete+"1", MultiplePollsRequiredHandler("unknown", "succeeded"))
									})

									It("keeps polling and eventually deletes the binding and marks the operation as success", func() {
										resp := deleteBinding(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedDeleteSuccessStatusCode)

										bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
											Category:          types.DELETE,
											State:             types.SUCCEEDED,
											ResourceType:      types.ServiceBindingType,
											Reschedulable:     false,
											DeletionScheduled: false,
										})

										verifyBindingDoesNotExist(ctx.SMWithOAuthForTenant, bindingID)
									})
								})

								When("polling responds with unexpected state and eventually with failed state", func() {
									BeforeEach(func() {
										brokerServer.BindingHandlerFunc(http.MethodDelete, http.MethodDelete+"2", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
										brokerServer.BindingLastOpHandlerFunc(http.MethodDelete+"2", MultiplePollsRequiredHandler("unknown", "failed"))
									})

									When("orphan mitigation unbind synchronously succeeds", func() {
										It("deletes the binding and marks the operation as success", func() {
											resp := deleteBinding(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)

											bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.DELETE,
												State:             types.FAILED,
												ResourceType:      types.ServiceBindingType,
												Reschedulable:     false,
												DeletionScheduled: true,
											})

											brokerServer.BindingHandlerFunc(http.MethodDelete, http.MethodDelete+"2", ParameterizedHandler(http.StatusOK, Object{"async": false}))

											bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.DELETE,
												State:             types.SUCCEEDED,
												ResourceType:      types.ServiceBindingType,
												Reschedulable:     false,
												DeletionScheduled: false,
											})

											verifyBindingDoesNotExist(ctx.SMWithOAuthForTenant, bindingID)
										})
									})

									When("broker orphan mitigation unbind synchronously fails with an unexpected error", func() {
										It("keeps the binding and marks the operation with deletion scheduled", func() {
											resp := deleteBinding(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)

											bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.DELETE,
												State:             types.FAILED,
												ResourceType:      types.ServiceBindingType,
												Reschedulable:     false,
												DeletionScheduled: true,
											})

											brokerServer.BindingHandlerFunc(http.MethodDelete, http.MethodDelete+"2", ParameterizedHandler(http.StatusBadRequest, Object{"error": "error"}))

											bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.DELETE,
												State:             types.FAILED,
												ResourceType:      types.ServiceBindingType,
												Reschedulable:     false,
												DeletionScheduled: true,
											})

											verifyBindingExists(ctx.SMWithOAuthForTenant, bindingID, true)
										})
									})

									When("broker orphan mitigation unbind synchronously fails with an error that will continue further orphan mitigation and eventually succeed", func() {
										It("deletes the binding and marks the operation that triggered the orphan mitigation as failed with no deletion scheduled and not reschedulable", func() {
											resp := deleteBinding(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)

											bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.DELETE,
												State:             types.FAILED,
												ResourceType:      types.ServiceBindingType,
												Reschedulable:     false,
												DeletionScheduled: true,
											})

											brokerServer.BindingHandlerFunc(http.MethodDelete, http.MethodDelete+"2", MultipleErrorsBeforeSuccessHandler(
												http.StatusInternalServerError, http.StatusOK,
												Object{"error": "error"}, Object{"async": false},
											))

											bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.DELETE,
												State:             types.SUCCEEDED,
												ResourceType:      types.ServiceBindingType,
												Reschedulable:     false,
												DeletionScheduled: false,
											})

											verifyBindingDoesNotExist(ctx.SMWithOAuthForTenant, bindingID)
										})
									})
								})

								When("polling returns an unexpected status code", func() {
									BeforeEach(func() {
										brokerServer.BindingHandlerFunc(http.MethodDelete, http.MethodDelete+"3", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
										brokerServer.BindingLastOpHandlerFunc(http.MethodDelete+"3", ParameterizedHandler(http.StatusInternalServerError, Object{"error": "error"}))
									})

									It("keeps the binding and stores the operation as reschedulable", func() {
										resp := deleteBinding(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)

										bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
											Category:          types.DELETE,
											State:             types.FAILED,
											ResourceType:      types.ServiceBindingType,
											Reschedulable:     true,
											DeletionScheduled: false,
										})

										verifyBindingExists(ctx.SMWithOAuthForTenant, bindingID, true)
									})
								})

								When("broker stops while polling", func() {
									BeforeEach(func() {
										brokerServer.BindingHandlerFunc(http.MethodDelete, http.MethodDelete+"3", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
										brokerServer.BindingLastOpHandlerFunc(http.MethodDelete+"3", DelayingHandler(doneChannel, cancelFunc))
									})

									It("keeps the binding and stores the operation as reschedulable", func() {
										resp := deleteBinding(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)

										<-cancelCtx.Done()
										brokerServer.Close()
										delete(ctx.Servers, BrokerServerPrefix+brokerID)

										bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
											Category:          types.DELETE,
											State:             types.FAILED,
											ResourceType:      types.ServiceBindingType,
											Reschedulable:     true,
											DeletionScheduled: false,
										})

										verifyBindingExists(ctx.SMWithOAuthForTenant, bindingID, true)
									})
								})
							})

							When("unbind responds with error due to stopped broker", func() {
								BeforeEach(func() {
									brokerServer.Close()
									delete(ctx.Servers, BrokerServerPrefix+brokerID)
								})

								It("keeps the binding and marks operation with failed", func() {
									resp := deleteBinding(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)

									bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.DELETE,
										State:             types.FAILED,
										ResourceType:      types.ServiceBindingType,
										Reschedulable:     false,
										DeletionScheduled: false,
									})

									verifyBindingExists(ctx.SMWithOAuthForTenant, bindingID, true)
								})
							})

							When("unbind responds with error that does not require orphan mitigation", func() {
								BeforeEach(func() {
									brokerServer.BindingHandlerFunc(http.MethodDelete, http.MethodDelete+"3", ParameterizedHandler(http.StatusBadRequest, Object{"error": "error"}))
								})

								It("keeps the binding and marks the operation as failed", func() {
									resp := deleteBinding(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)
									bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.DELETE,
										State:             types.FAILED,
										ResourceType:      types.ServiceBindingType,
										Reschedulable:     false,
										DeletionScheduled: false,
									})

									verifyBindingExists(ctx.SMWithOAuthForTenant, bindingID, true)
								})
							})

							When("unbind responds with error that requires orphan mitigation", func() {
								BeforeEach(func() {
									brokerServer.BindingHandlerFunc(http.MethodDelete, http.MethodDelete+"3", ParameterizedHandler(http.StatusInternalServerError, Object{"error": "error"}))
								})

								When("orphan mitigation unbind asynchronously succeeds", func() {
									It("deletes the binding and marks the operation as success", func() {
										resp := deleteBinding(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)

										bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
											Category:          types.DELETE,
											State:             types.FAILED,
											ResourceType:      types.ServiceBindingType,
											Reschedulable:     false,
											DeletionScheduled: true,
										})

										brokerServer.BindingHandlerFunc(http.MethodDelete, http.MethodDelete+"3", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
										brokerServer.BindingLastOpHandlerFunc(http.MethodDelete+"3", ParameterizedHandler(http.StatusOK, Object{"state": "succeeded"}))

										bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
											Category:          types.DELETE,
											State:             types.SUCCEEDED,
											ResourceType:      types.ServiceBindingType,
											Reschedulable:     false,
											DeletionScheduled: false,
										})

										verifyBindingDoesNotExist(ctx.SMWithOAuthForTenant, bindingID)
									})
								})

								if testCase.async {
									When("broker orphan mitigation unbind asynchronously keeps failing with an error while polling", func() {
										It("keeps the binding and marks the operation as failed reschedulable with deletion scheduled", func() {
											resp := deleteBinding(ctx.SMWithOAuthForTenant, true, http.StatusAccepted)

											bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.DELETE,
												State:             types.FAILED,
												ResourceType:      types.ServiceBindingType,
												Reschedulable:     false,
												DeletionScheduled: true,
											})

											brokerServer.BindingHandlerFunc(http.MethodDelete, http.MethodDelete+"3", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
											brokerServer.BindingLastOpHandlerFunc(http.MethodDelete+"3", ParameterizedHandler(http.StatusBadRequest, Object{"error": "error"}))

											bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.DELETE,
												State:             types.FAILED,
												ResourceType:      types.ServiceBindingType,
												Reschedulable:     true,
												DeletionScheduled: true,
											})

											verifyBindingExists(ctx.SMWithOAuthForTenant, bindingID, true)
										})
									})
								}

								When("broker orphan mitigation unbind asynchronously fails with an error that will continue further orphan mitigation and eventually succeed", func() {
									It("deletes the binding and marks the operation as success", func() {
										resp := deleteBinding(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)

										bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
											Category:          types.DELETE,
											State:             types.FAILED,
											ResourceType:      types.ServiceBindingType,
											Reschedulable:     false,
											DeletionScheduled: true,
										})

										brokerServer.BindingHandlerFunc(http.MethodDelete, http.MethodDelete+"3", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
										brokerServer.BindingLastOpHandlerFunc(http.MethodDelete+"3", MultipleErrorsBeforeSuccessHandler(
											http.StatusOK, http.StatusOK,
											Object{"state": "failed"}, Object{"state": "succeeded"},
										))

										bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
											Category:          types.DELETE,
											State:             types.SUCCEEDED,
											ResourceType:      types.ServiceBindingType,
											Reschedulable:     false,
											DeletionScheduled: false,
										})

										verifyBindingDoesNotExist(ctx.SMWithOAuthForTenant, bindingID)
									})
								})
							})

							When("unbind responds with error due to times out", func() {
								var oldCtx *TestContext
								BeforeEach(func() {
									oldCtx = ctx
									ctx = NewTestContextBuilderWithSecurity().WithEnvPreExtensions(func(set *pflag.FlagSet) {
										Expect(set.Set("httpclient.response_header_timeout", (1 * time.Second).String())).ToNot(HaveOccurred())
									}).Build()

									brokerID, brokerServer, servicePlanID = newServicePlan(ctx)
									EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, servicePlanID, TenantIDValue)
									createInstance(ctx.SMWithOAuthForTenant, false, http.StatusCreated)
									postBindingRequest["service_instance_id"] = instanceID
									createBinding(ctx.SMWithOAuthForTenant, false, http.StatusCreated)

									brokerServer.BindingHandlerFunc(http.MethodDelete, http.MethodDelete+"1", DelayingHandler(doneChannel, cancelFunc))

								})

								AfterEach(func() {
									ctx.SMRepository.Delete(context.TODO(), types.OperationType)
									DeleteBinding(ctx, bindingID, instanceID)
									DeleteInstance(ctx, instanceID, servicePlanID)
									ctx.SMWithOAuth.DELETE(web.ServiceBrokersURL + "/" + brokerID).Expect()
									delete(ctx.Servers, BrokerServerPrefix+brokerID)
									brokerServer.Close()
									ctx = oldCtx
								})

								It("orphan mitigates the binding", func() {
									resp := deleteBinding(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)

									bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.DELETE,
										State:             types.FAILED,
										ResourceType:      types.ServiceBindingType,
										Reschedulable:     false,
										DeletionScheduled: true,
									})

									brokerServer.BindingHandlerFunc(http.MethodDelete, http.MethodDelete+"1", ParameterizedHandler(http.StatusOK, Object{"async": false}))

									bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.DELETE,
										State:             types.SUCCEEDED,
										ResourceType:      types.ServiceBindingType,
										Reschedulable:     false,
										DeletionScheduled: false,
									})

									verifyBindingDoesNotExist(ctx.SMWithOAuthForTenant, bindingID)
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
	_, _, servicePlanID := newServicePlan(ctx)
	EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, servicePlanID, "")
	resp := ctx.SMWithOAuth.POST(web.ServiceInstancesURL).
		WithQuery("async", strconv.FormatBool(async)).
		WithJSON(Object{
			"name":             "test-service-instance",
			"service_plan_id":  servicePlanID,
			"maintenance_info": "{}",
		}).Expect()

	var instance map[string]interface{}
	if async {
		instance = ExpectSuccessfulAsyncResourceCreation(resp, auth, web.ServiceInstancesURL)
	} else {
		instance = resp.Status(http.StatusCreated).JSON().Object().Raw()
	}

	resp = ctx.SMWithOAuth.POST(web.ServiceBindingsURL).
		WithQuery("async", strconv.FormatBool(async)).
		WithJSON(Object{
			"name":                "test-service-binding",
			"service_instance_id": instance["id"],
		}).Expect()

	var binding map[string]interface{}
	if async {
		binding = ExpectSuccessfulAsyncResourceCreation(resp, auth, web.ServiceBindingsURL)
	} else {
		binding = resp.Status(http.StatusCreated).JSON().Object().Raw()
	}

	delete(binding, "credentials")
	return binding
}

func newServicePlan(ctx *TestContext) (string, *BrokerServer, string) {
	brokerID, _, brokerServer := ctx.RegisterBrokerWithCatalog(NewRandomSBCatalog())
	ctx.Servers[BrokerServerPrefix+brokerID] = brokerServer
	so := ctx.SMWithOAuth.ListWithQuery(web.ServiceOfferingsURL, fmt.Sprintf("fieldQuery=broker_id eq '%s'", brokerID)).First()
	servicePlanID := ctx.SMWithOAuth.ListWithQuery(web.ServicePlansURL, "fieldQuery="+fmt.Sprintf("service_offering_id eq '%s'", so.Object().Value("id").String().Raw())).
		First().Object().Value("id").String().Raw()
	return brokerID, brokerServer, servicePlanID
}
