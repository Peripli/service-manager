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
	"sync/atomic"
	"time"

	"github.com/Peripli/service-manager/operations"
	"github.com/Peripli/service-manager/pkg/query"

	"github.com/Peripli/service-manager/pkg/env"
	"github.com/gofrs/uuid"

	"github.com/spf13/pflag"

	"github.com/Peripli/service-manager/pkg/util"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/gavv/httpexpect"

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
	TenantIdentifier       = "tenant"
	TenantIDValue          = "tenantID"
	MaximumPollingDuration = 2 //seconds
)

type testCase struct {
	async                           string
	expectedCreateSuccessStatusCode int
	expectedUpdateSuccessStatusCode int
	expectedDeleteSuccessStatusCode int
	expectedBrokerFailureStatusCode int
	expectedSMCrashStatusCode       int
}

func (t *testCase) responseByBrokerOrClientMode(expected int, statusByBrokerResponse int) int {
	if t.async != "" {
		return expected
	}
	return statusByBrokerResponse
}

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
	DisableTenantResources:                 false,
	StrictlyTenantScoped:                   true,
	ResourceBlueprint:                      blueprint,
	ResourceWithoutNullableFieldsBlueprint: blueprint,
	ResourcePropertiesToIgnore:             []string{"last_operation", "volume_mounts", "endpoints", "bind_resource", "credentials"},
	PatchResource:                          StorageResourcePatch,
	AdditionalTests: func(ctx *TestContext, t *TestCase) {
		Context("additional non-generic tests", func() {
			var (
				postBindingRequest   Object
				instanceID           string
				instanceName         string
				bindingID            string
				brokerID             string
				brokerServer         *BrokerServer
				servicePlanID        string
				serviceCatalogID     string
				servicePlanCatalogID string
				syncBindingResponse  Object
			)

			testCases := []testCase{
				{
					async:                           "false",
					expectedCreateSuccessStatusCode: http.StatusCreated,
					expectedDeleteSuccessStatusCode: http.StatusOK,
					expectedBrokerFailureStatusCode: http.StatusBadGateway,
					expectedSMCrashStatusCode:       http.StatusBadGateway,
				},
				{
					async:                           "true",
					expectedCreateSuccessStatusCode: http.StatusAccepted,
					expectedDeleteSuccessStatusCode: http.StatusAccepted,
					expectedBrokerFailureStatusCode: http.StatusAccepted,
					expectedSMCrashStatusCode:       http.StatusAccepted,
				},
				{
					async:                           "",
					expectedCreateSuccessStatusCode: http.StatusCreated,
					expectedUpdateSuccessStatusCode: http.StatusOK,
					expectedDeleteSuccessStatusCode: http.StatusOK,
					expectedBrokerFailureStatusCode: http.StatusBadGateway,
					expectedSMCrashStatusCode:       http.StatusBadGateway,
				},
			}

			createInstance := func(smClient *SMExpect, async bool, expectedStatusCode int) *httpexpect.Response {
				ID, err := uuid.NewV4()
				Expect(err).ToNot(HaveOccurred())
				postInstanceRequest := Object{
					"name":             "test-instance" + ID.String(),
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

			deleteInstance := func(smClient *SMExpect, async bool, expectedStatusCode int) *httpexpect.Response {
				return smClient.DELETE(web.ServiceInstancesURL+"/"+instanceID).
					WithQuery("async", async).
					Expect().
					Status(expectedStatusCode)
			}

			createBinding := func(SM *SMExpect, async string, expectedStatusCode int) *httpexpect.Response {
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

			deleteBinding := func(smClient *SMExpect, async string, expectedStatusCode int) *httpexpect.Response {
				return smClient.DELETE(web.ServiceBindingsURL+"/"+bindingID).
					WithQuery("async", async).
					Expect().
					Status(expectedStatusCode)
			}

			preparePrerequisitesWithMaxPollingDuration := func(maxPollingDuration int) {
				brokerID, brokerServer, servicePlanID, servicePlanCatalogID, serviceCatalogID = newServicePlanWithMaxPollingDuration(ctx, true, maxPollingDuration)
				brokerServer.ShouldRecordRequests(false)
				EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, servicePlanID, TenantIDValue)
				resp := createInstance(ctx.SMWithOAuthForTenant, false, http.StatusCreated)
				instanceName = resp.JSON().Object().Value("name").String().Raw()
				Expect(instanceName).ToNot(BeEmpty())

				postBindingRequest = Object{
					"name":                "test-binding",
					"service_instance_id": instanceID,
				}
				syncBindingResponse = Object{
					"async": false,
					"credentials": Object{
						"user":     "user",
						"password": "password",
					},
				}
			}

			preparePrerequisites := func() {
				preparePrerequisitesWithMaxPollingDuration(0)
			}

			BeforeEach(func() {
				preparePrerequisites()
			})

			JustBeforeEach(func() {
				postBindingRequest = Object{
					"name":                "test-binding",
					"service_instance_id": instanceID,
				}
			})

			AfterEach(func() {
				ctx.CleanupAdditionalResources()
			})

			Describe("get parameters", func() {
				When("service binding does not exist", func() {
					It("should return an error", func() {
						ctx.SMWithOAuthForTenant.GET(web.ServiceBindingsURL + "/" + bindingID + web.ParametersURL).Expect().
							Status(http.StatusNotFound)

					})
				})
				When("service binding exists", func() {
					var bindingRetrievable bool
					JustBeforeEach(func() {
						brokerServer.BindingHandlerFunc(http.MethodPut, http.MethodPut, func(req *http.Request) (int, map[string]interface{}) {
							return http.StatusCreated, Object{}
						})
						servicePlanID = findPlanIDForBrokerIDAndBindingRetrievable(ctx, brokerID, bindingRetrievable)
						EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, servicePlanID, TenantIDValue)
						createInstance(ctx.SMWithOAuthForTenant, false, http.StatusCreated)
						postBindingRequest["name"] = "test-binding-retrievable-name"
						postBindingRequest["service_instance_id"] = instanceID
						brokerServer.BindingHandlerFunc(http.MethodPut, http.MethodPut+"1", ParameterizedHandler(http.StatusCreated, syncBindingResponse))
						createBinding(ctx.SMWithOAuthForTenant, "false", http.StatusCreated)

					})

					Describe("not retrievable binding", func() {
						BeforeEach(func() {
							bindingRetrievable = false
						})
						It("should return an error", func() {
							ctx.SMWithOAuthForTenant.GET(web.ServiceBindingsURL + "/" + bindingID + web.ParametersURL).Expect().
								Status(http.StatusBadRequest)
						})

					})

					Describe("retrievable binding", func() {
						BeforeEach(func() {
							bindingRetrievable = true
						})
						When("async is requested", func() {
							It("Should return an error", func() {
								url := web.ServiceBindingsURL + "/" + bindingID + web.ParametersURL
								ctx.SMWithOAuthForTenant.GET(url).WithQuery("async", true).Expect().
									Status(http.StatusBadRequest)
							})

						})

						When("params are not readable", func() {
							BeforeEach(func() {
								bindingRetrievable = true
								postBindingRequest["parameters"] = map[string]string{
									"cat": "Freddy",
									"dog": "Lucy",
								}

								brokerServer.BindingHandlerFunc(http.MethodGet, http.MethodGet+"1", ParameterizedHandler(http.StatusOK, Object{
									"parameters":    "fdafds:fdasfd.f",
									"dashboard_url": "http://dashboard.com",
								}))
							})
							It("Should return an error", func() {
								ctx.SMWithOAuthForTenant.GET(web.ServiceBindingsURL + "/" + bindingID + web.ParametersURL).Expect().
									Status(http.StatusBadGateway)
							})
						})
						When("params are readable", func() {
							BeforeEach(func() {
								bindingRetrievable = true
								postBindingRequest["parameters"] = map[string]string{
									"cat": "Freddy",
									"dog": "Lucy",
								}

								brokerServer.BindingHandlerFunc(http.MethodGet, http.MethodGet+"1", ParameterizedHandler(http.StatusOK, Object{
									"parameters": map[string]string{
										"cat": "Freddy",
										"dog": "Lucy",
									},
									"dashboard_url": "http://dashboard.com",
								}))
							})

							It("Should return parameters", func() {
								response := ctx.SMWithOAuthForTenant.GET(web.ServiceBindingsURL + "/" + bindingID + web.ParametersURL).Expect()
								response.Status(http.StatusOK)
								jsonObject := response.JSON().Object()
								jsonObject.Value("cat").String().Equal("Freddy")
								jsonObject.Value("dog").String().Equal("Lucy")

							})
						})
					})
				})
			})

			Describe("GET", func() {
				When("service binding contains tenant identifier in OSB context", func() {
					BeforeEach(func() {
						createBinding(ctx.SMWithOAuthForTenant, "false", http.StatusCreated)
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
					Context(fmt.Sprintf("async = %s", testCase.async), func() {
						When("instance exists in a platform different from service manager", func() {
							const (
								brokerAPIVersionHeaderKey   = "X-Broker-API-Version"
								brokerAPIVersionHeaderValue = "2.13"
								SID                         = "abc1234"
							)

							BeforeEach(func() {
								EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, ctx.TestPlatform.ID, servicePlanID, TenantIDValue)

								ctx.SMWithBasic.PUT("/v1/osb/"+brokerID+"/v2/service_instances/"+SID).
									WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
									WithJSON(Object{
										"service_id": serviceCatalogID,
										"plan_id":    servicePlanCatalogID,
										"context": Object{
											TenantIdentifier: TenantIDValue,
										},
									}).
									Expect().Status(http.StatusCreated)

								ctx.SMWithOAuth.GET(web.ServiceInstancesURL + "/" + SID).
									Expect().
									Status(http.StatusOK).
									JSON().Object().Value("platform_id").Equal(ctx.TestPlatform.ID)
							})

							It("return 404", func() {
								postBindingRequest["service_instance_id"] = SID
								createBinding(ctx.SMWithOAuthForTenant, testCase.async, http.StatusNotFound)
							})
						})

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

						Context("when request body contains protected labels", func() {
							It("returns 400", func() {
								ctx.SMWithOAuthForTenant.POST(web.ServiceBindingsURL).
									WithQuery("async", testCase.async).
									WithHeader("Content-Type", "application/json").
									WithBytes([]byte(fmt.Sprintf(`{
										"name": "test-binding-name",
										"service_instance_id": "%s",
										"maintenance_info": {},
										"labels": {
											"%s":["test-tenant"]
										}
									}`, instanceID, TenantIdentifier))).
									Expect().
									Status(http.StatusBadRequest).
									JSON().Object().
									Keys().Contains("error", "description")
							})

							Context("when request body contains multiple label objects", func() {
								It("returns 400", func() {
									ctx.SMWithOAuthForTenant.POST(web.ServiceBindingsURL).
										WithQuery("async", testCase.async).
										WithHeader("Content-Type", "application/json").
										WithBytes([]byte(fmt.Sprintf(`{
										"name": "test-binding-name",
										"service_instance_id": "%s",
										"maintenance_info": {},
										"labels": {},
										"labels": {
											"%s":["test-tenant"]
										}
									}`, instanceID, TenantIdentifier))).
										Expect().
										Status(http.StatusBadRequest).
										JSON().Object().Value("description").String().Contains("invalid json: duplicate key labels")
								})
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
									resp := createBinding(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedCreateSuccessStatusCode)
									bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.CREATE,
										State:             types.SUCCEEDED,
										ResourceType:      types.ServiceBindingType,
										Reschedulable:     false,
										DeletionScheduled: false,
									})

									VerifyResourceExists(ctx.SMWithOAuthForTenant, ResourceExpectations{
										ID:    bindingID,
										Type:  types.ServiceBindingType,
										Ready: true,
									})
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
							BeforeEach(func() {
								brokerServer.BindingHandlerFunc(http.MethodPut, http.MethodPut+"1", func(req *http.Request) (int, map[string]interface{}) {
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
								createBinding(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedCreateSuccessStatusCode)
							})
						})

						Context("instance visibility", func() {
							When("tenant doesn't have ownership of instance", func() {
								BeforeEach(func() {
									createInstance(ctx.SMWithOAuthForTenant, false, http.StatusCreated)
								})

								It("returns 404", func() {
									otherTenantExpect := ctx.NewTenantExpect("tenancyClient", "other-tenant")
									createBinding(otherTenantExpect, testCase.async, http.StatusNotFound)
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
									VerifyResourceExists(ctx.SMWithOAuthForTenant, ResourceExpectations{
										ID:    bindingID,
										Type:  types.ServiceBindingType,
										Ready: true,
									})
								})
							})
						})

						Context("broker scenarios", func() {
							When("instance is not ready", func() {
								BeforeEach(func() {
									brokerServer.ServiceInstanceHandlerFunc(http.MethodPut, http.MethodPut, ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
									brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodPut, ParameterizedHandler(http.StatusInternalServerError, Object{}))
									resp := createInstance(ctx.SMWithOAuthForTenant, true, http.StatusAccepted)
									instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.CREATE,
										State:             types.FAILED,
										ResourceType:      types.ServiceInstanceType,
										Reschedulable:     true,
										DeletionScheduled: false,
									})

									VerifyResourceExists(ctx.SMWithOAuthForTenant, ResourceExpectations{
										ID:    instanceID,
										Type:  types.ServiceInstanceType,
										Ready: false,
									})
								})

								It("fails to create binding", func() {
									expectedStatusCode := testCase.expectedBrokerFailureStatusCode
									if testCase.async == "false" || testCase.async == "" {
										expectedStatusCode = http.StatusUnprocessableEntity
									}
									resp := createBinding(ctx.SMWithOAuthForTenant, testCase.async, expectedStatusCode)

									bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.CREATE,
										State:             types.FAILED,
										ResourceType:      types.ServiceBindingType,
										Reschedulable:     false,
										DeletionScheduled: false,
									})

									VerifyResourceDoesNotExist(ctx.SMWithOAuthForTenant, ResourceExpectations{
										ID:   bindingID,
										Type: types.ServiceBindingType,
									})
								})
							})

							When("instance is being deleted", func() {
								var doneChannel chan interface{}
								BeforeEach(func() {

									doneChannel = make(chan interface{})
									resp := createInstance(ctx.SMWithOAuthForTenant, false, http.StatusCreated)

									VerifyResourceExists(ctx.SMWithOAuthForTenant, ResourceExpectations{
										ID:    instanceID,
										Type:  types.ServiceInstanceType,
										Ready: true,
									})

									brokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete, ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
									brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodDelete, DelayingHandler(doneChannel))
									resp = deleteInstance(ctx.SMWithOAuthForTenant, true, http.StatusAccepted)
									instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.DELETE,
										State:             types.IN_PROGRESS,
										ResourceType:      types.ServiceInstanceType,
										Reschedulable:     true,
										DeletionScheduled: false,
									})

									VerifyResourceExists(ctx.SMWithOAuthForTenant, ResourceExpectations{
										ID:    instanceID,
										Type:  types.ServiceInstanceType,
										Ready: true,
									})
								})

								AfterEach(func() {
									close(doneChannel)
								})

								It("fails to create binding", func() {
									expectedStatusCode := testCase.expectedBrokerFailureStatusCode
									if testCase.async == "false" || testCase.async == "" {
										expectedStatusCode = http.StatusUnprocessableEntity
									}
									resp := createBinding(ctx.SMWithOAuthForTenant, testCase.async, expectedStatusCode)

									bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.CREATE,
										State:             types.FAILED,
										ResourceType:      types.ServiceBindingType,
										Reschedulable:     false,
										DeletionScheduled: false,
									})

									VerifyResourceDoesNotExist(ctx.SMWithOAuthForTenant, ResourceExpectations{
										ID:   bindingID,
										Type: types.ServiceBindingType,
									})
								})
							})

							When("plan is not bindable", func() {
								BeforeEach(func() {
									servicePlanID, _, _ = findPlanDetailsForBrokerID(ctx, brokerID, false)
									EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, servicePlanID, TenantIDValue)
									createInstance(ctx.SMWithOAuthForTenant, false, http.StatusCreated)
								})

								It("fails to create binding", func() {
									expectedStatusCode := testCase.expectedBrokerFailureStatusCode
									if testCase.async == "false" || testCase.async == "" {
										expectedStatusCode = http.StatusBadRequest
									}
									resp := createBinding(ctx.SMWithOAuthForTenant, testCase.async, expectedStatusCode)

									bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.CREATE,
										State:             types.FAILED,
										ResourceType:      types.ServiceBindingType,
										Reschedulable:     false,
										DeletionScheduled: false,
									})

									VerifyResourceDoesNotExist(ctx.SMWithOAuthForTenant, ResourceExpectations{
										ID:   bindingID,
										Type: types.ServiceBindingType,
									})
								})
							})

							for _, bindingRetrievable := range []bool{true, false} {
								bindingRetrievable := bindingRetrievable
								When(fmt.Sprintf("plan specifies binding_retrievable %t", bindingRetrievable), func() {
									BeforeEach(func() {
										brokerServer.BindingHandlerFunc(http.MethodPut, http.MethodPut, func(req *http.Request) (int, map[string]interface{}) {
											acceptsIncomplete := req.FormValue("accepts_incomplete")
											if len(acceptsIncomplete) == 0 {
												acceptsIncomplete = "false"
											}
											Expect(acceptsIncomplete).To(Equal(strconv.FormatBool(bindingRetrievable)))

											return http.StatusCreated, Object{}
										})
										servicePlanID = findPlanIDForBrokerIDAndBindingRetrievable(ctx, brokerID, bindingRetrievable)
										EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, servicePlanID, TenantIDValue)
										resp := createInstance(ctx.SMWithOAuthForTenant, false, http.StatusCreated)

										instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
											Category:          types.CREATE,
											State:             types.SUCCEEDED,
											ResourceType:      types.ServiceInstanceType,
											Reschedulable:     false,
											DeletionScheduled: false,
										})

										VerifyResourceExists(ctx.SMWithOAuthForTenant, ResourceExpectations{
											ID:    instanceID,
											Type:  types.ServiceInstanceType,
											Ready: true,
										})

										postBindingRequest["name"] = "test-binding-retrievable-name"
										postBindingRequest["service_instance_id"] = instanceID
									})

									It("successfully creates binding", func() {
										resp := createBinding(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedCreateSuccessStatusCode)

										bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
											Category:          types.CREATE,
											State:             types.SUCCEEDED,
											ResourceType:      types.ServiceBindingType,
											Reschedulable:     false,
											DeletionScheduled: false,
										})

										VerifyResourceExists(ctx.SMWithOAuthForTenant, ResourceExpectations{
											ID:    bindingID,
											Type:  types.ServiceBindingType,
											Ready: true,
										})
									})
								})
							}

							When("a create operation is already in progress", func() {
								var doneChannel chan interface{}

								BeforeEach(func() {
									doneChannel = make(chan interface{})
									brokerServer.BindingHandlerFunc(http.MethodPut, http.MethodPut+"1", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
									brokerServer.BindingLastOpHandlerFunc(http.MethodPut+"1", DelayingHandler(doneChannel))
									brokerServer.BindingLastOpHandlerFunc(http.MethodDelete+"1", ParameterizedHandler(http.StatusOK, Object{"state": "succeeded"}))

									resp := createBinding(ctx.SMWithOAuthForTenant, "true", http.StatusAccepted)

									bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.CREATE,
										State:             types.IN_PROGRESS,
										ResourceType:      types.ServiceBindingType,
										Reschedulable:     true,
										DeletionScheduled: false,
									})

									VerifyResourceExists(ctx.SMWithOAuthForTenant, ResourceExpectations{
										ID:    bindingID,
										Type:  types.ServiceBindingType,
										Ready: false,
									})
								})

								AfterEach(func() {
									close(doneChannel)
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

									VerifyResourceDoesNotExist(ctx.SMWithOAuthForTenant, ResourceExpectations{
										ID:   bindingID,
										Type: types.ServiceBindingType,
									})
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
									brokerServer.BindingHandlerFunc(http.MethodPut, http.MethodPut+"1", ParameterizedHandler(http.StatusCreated, syncBindingResponse))
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

									VerifyResourceExists(ctx.SMWithOAuthForTenant, ResourceExpectations{
										ID:    bindingID,
										Type:  types.ServiceBindingType,
										Ready: true,
									})
								})
							})

							When("broker responds with asynchronous success", func() {
								BeforeEach(func() {
									brokerServer.BindingHandlerFunc(http.MethodPut, http.MethodPut+"1", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
									brokerServer.BindingLastOpHandlerFunc(http.MethodPut+"1", MultiplePollsRequiredHandler("in progress", "succeeded"))
								})

								It("polling broker last operation until operation succeeds and eventually marks operation as success", func() {
									resp := createBinding(ctx.SMWithOAuthForTenant, testCase.async, testCase.responseByBrokerOrClientMode(testCase.expectedCreateSuccessStatusCode, http.StatusAccepted))

									bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.CREATE,
										State:             types.SUCCEEDED,
										ResourceType:      types.ServiceBindingType,
										Reschedulable:     false,
										DeletionScheduled: false,
									})

									VerifyResourceExists(ctx.SMWithOAuthForTenant, ResourceExpectations{
										ID:    bindingID,
										Type:  types.ServiceBindingType,
										Ready: true,
									})
								})

								When("maximum polling duration is reached while polling", func() {
									var newCtx *TestContext
									BeforeEach(func() {
										preparePrerequisitesWithMaxPollingDuration(MaximumPollingDuration)

										newCtx = t.ContextBuilder.WithEnvPreExtensions(func(set *pflag.FlagSet) {
											Expect(set.Set("operations.action_timeout", ((MaximumPollingDuration + 1) * time.Second).String())).ToNot(HaveOccurred())
										}).BuildWithoutCleanup()

										brokerServer.BindingHandlerFunc(http.MethodPut, http.MethodPut+"1", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
										brokerServer.BindingLastOpHandlerFunc(http.MethodPut+"1", ParameterizedHandler(http.StatusOK, Object{"state": "in progress"}))
									})

									AfterEach(func() {
										newCtx.CleanupAll(false)
									})

									When("orphan mitigation unbind synchronously succeeds", func() {
										BeforeEach(func() {
											brokerServer.BindingHandlerFunc(http.MethodDelete, http.MethodDelete+"3", ParameterizedHandler(http.StatusOK, Object{"async": false}))
										})

										It("verifies the binding and marks the operation that triggered the orphan mitigation as failed with no deletion scheduled and not reschedulable", func() {
											resp := createBinding(newCtx.SMWithOAuthForTenant, testCase.async, testCase.responseByBrokerOrClientMode(testCase.expectedBrokerFailureStatusCode, http.StatusAccepted))

											bindingID, _ = VerifyOperationExists(newCtx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.CREATE,
												State:             types.FAILED,
												ResourceType:      types.ServiceBindingType,
												Reschedulable:     false,
												DeletionScheduled: false,
											})

											VerifyResource(ctx.SMWithOAuthForTenant, ResourceExpectations{
												ID:   bindingID,
												Type: types.ServiceBindingType,
											}, testCase.async, true)
										})
									})

									When("broker orphan mitigation unbind synchronously fails with an error that will stop further orphan mitigation", func() {
										BeforeEach(func() {
											brokerServer.BindingHandlerFunc(http.MethodDelete, http.MethodDelete+"3", ParameterizedHandler(http.StatusBadRequest, Object{"error": "error"}))
										})

										It("verifies the binding is ready false and marks the operation with deletion scheduled", func() {
											resp := createBinding(newCtx.SMWithOAuthForTenant, testCase.async, testCase.responseByBrokerOrClientMode(testCase.expectedBrokerFailureStatusCode, http.StatusAccepted))

											bindingID, _ = VerifyOperationExists(newCtx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.CREATE,
												State:             types.FAILED,
												ResourceType:      types.ServiceBindingType,
												Reschedulable:     false,
												DeletionScheduled: true,
											})

											VerifyResource(ctx.SMWithOAuthForTenant, ResourceExpectations{
												ID:   bindingID,
												Type: types.ServiceBindingType,
											}, testCase.async, true)
										})
									})

									When("broker orphan mitigation unbind synchronously fails with an error that will continue further orphan mitigation and eventually succeed", func() {
										BeforeEach(func() {
											brokerServer.BindingHandlerFunc(http.MethodDelete, http.MethodDelete+"3", MultipleErrorsBeforeSuccessHandler(
												http.StatusInternalServerError, http.StatusOK,
												Object{"error": "error"}, Object{"async": false},
											))
										})

										It("verifies the binding and marks the operation that triggered the orphan mitigation as failed with no deletion scheduled and not reschedulable", func() {
											resp := createBinding(newCtx.SMWithOAuthForTenant, testCase.async, testCase.responseByBrokerOrClientMode(testCase.expectedBrokerFailureStatusCode, http.StatusAccepted))

											bindingID, _ = VerifyOperationExists(newCtx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.CREATE,
												State:             types.FAILED,
												ResourceType:      types.ServiceBindingType,
												Reschedulable:     false,
												DeletionScheduled: false,
											})

											VerifyResource(ctx.SMWithOAuthForTenant, ResourceExpectations{
												ID:   bindingID,
												Type: types.ServiceBindingType,
											}, testCase.async, true)
										})
									})

									When("orphan mitigation unbind asynchronously succeeds", func() {
										BeforeEach(func() {
											brokerServer.BindingHandlerFunc(http.MethodDelete, http.MethodDelete+"3", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
											brokerServer.BindingLastOpHandlerFunc(http.MethodDelete+"3", ParameterizedHandler(http.StatusOK, Object{"state": "succeeded"}))
										})

										It("deletes the binding and marks the operation that triggered the orphan mitigation as failed with no deletion scheduled and not reschedulable", func() {
											resp := createBinding(newCtx.SMWithOAuthForTenant, testCase.async, testCase.responseByBrokerOrClientMode(testCase.expectedBrokerFailureStatusCode, http.StatusAccepted))

											bindingID, _ = VerifyOperationExists(newCtx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.CREATE,
												State:             types.FAILED,
												ResourceType:      types.ServiceBindingType,
												Reschedulable:     false,
												DeletionScheduled: false,
											})

											VerifyResource(ctx.SMWithOAuthForTenant, ResourceExpectations{
												ID:   bindingID,
												Type: types.ServiceBindingType,
											}, testCase.async, true)
										})

									})
								})

								if testCase.async == "true" {
									When("action timeout is reached while polling", func() {
										var newCtx *TestContext
										BeforeEach(func() {
											newCtx = t.ContextBuilder.WithEnvPreExtensions(func(set *pflag.FlagSet) {
												Expect(set.Set("operations.action_timeout", (2 * time.Second).String())).ToNot(HaveOccurred())
											}).BuildWithoutCleanup()

											brokerServer.BindingHandlerFunc(http.MethodPut, http.MethodPut+"1", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
											brokerServer.BindingLastOpHandlerFunc(http.MethodPut+"1", ParameterizedHandler(http.StatusOK, Object{"state": "in progress"}))
										})

										AfterEach(func() {
											newCtx.CleanupAll(false)
										})

										It("stores binding as ready false and the operation as reschedulable in progress", func() {
											resp := createBinding(newCtx.SMWithOAuthForTenant, "true", http.StatusAccepted)

											bindingID, _ = VerifyOperationExists(newCtx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.CREATE,
												State:             types.IN_PROGRESS,
												ResourceType:      types.ServiceBindingType,
												Reschedulable:     true,
												DeletionScheduled: false,
											})

											VerifyResourceExists(newCtx.SMWithOAuthForTenant, ResourceExpectations{
												ID:    bindingID,
												Type:  types.ServiceBindingType,
												Ready: false,
											})
										})
									})

									When("SM crashes while polling", func() {
										var newSMCtx *TestContext
										var isBound atomic.Value

										BeforeEach(func() {
											newSMCtx = t.ContextBuilder.WithEnvPostExtensions(func(e env.Environment, servers map[string]FakeServer) {
												e.Set("server.shutdown_timeout", 1*time.Second)
												e.Set("operations.maintainer_retry_interval", 1*time.Second)
											}).BuildWithoutCleanup()

											brokerServer.BindingHandlerFunc(http.MethodPut, http.MethodPut+"1", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
											brokerServer.BindingLastOpHandlerFunc(http.MethodPut+"1", func(_ *http.Request) (int, map[string]interface{}) {
												if isBound.Load() != nil {
													return http.StatusOK, Object{"state": types.SUCCEEDED}
												} else {
													return http.StatusOK, Object{"state": types.IN_PROGRESS}
												}
											})

										})

										AfterEach(func() {
											newSMCtx.CleanupAll(false)
										})

										It("should start restart polling through maintainer and eventually binding is set to ready", func() {
											resp := createBinding(newSMCtx.SMWithOAuthForTenant, "true", http.StatusAccepted)

											operationExpectations := OperationExpectations{
												Category:          types.CREATE,
												State:             types.IN_PROGRESS,
												ResourceType:      types.ServiceBindingType,
												Reschedulable:     true,
												DeletionScheduled: false,
											}

											bindingID, _ = VerifyOperationExists(newSMCtx, resp.Header("Location").Raw(), operationExpectations)
											VerifyResourceExists(ctx.SMWithOAuthForTenant, ResourceExpectations{
												ID:    bindingID,
												Type:  types.ServiceBindingType,
												Ready: false,
											})

											newSMCtx.CleanupAll(false)
											isBound.Store(true)

											newSMCtx = t.ContextBuilder.WithEnvPostExtensions(func(e env.Environment, servers map[string]FakeServer) {
												e.Set("operations.action_timeout", 2*time.Second)
											}).BuildWithoutCleanup()

											operationExpectations.State = types.SUCCEEDED
											operationExpectations.Reschedulable = false

											bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), operationExpectations)
											VerifyResourceExists(ctx.SMWithOAuthForTenant, ResourceExpectations{
												ID:    bindingID,
												Type:  types.ServiceBindingType,
												Ready: true,
											})
										})
									})
								}

								When("polling responds with unexpected state and eventually with success state", func() {
									BeforeEach(func() {
										brokerServer.BindingHandlerFunc(http.MethodPut, http.MethodPut+"1", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
										brokerServer.BindingLastOpHandlerFunc(http.MethodPut+"1", MultiplePollsRequiredHandler("unknown", "succeeded"))
									})

									It("keeps polling and eventually updates the binding to ready true and operation to success", func() {
										resp := createBinding(ctx.SMWithOAuthForTenant, testCase.async, testCase.responseByBrokerOrClientMode(testCase.expectedCreateSuccessStatusCode, http.StatusAccepted))

										bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
											Category:          types.CREATE,
											State:             types.SUCCEEDED,
											ResourceType:      types.ServiceBindingType,
											Reschedulable:     false,
											DeletionScheduled: false,
										})
										VerifyResourceExists(ctx.SMWithOAuthForTenant, ResourceExpectations{
											ID:    bindingID,
											Type:  types.ServiceBindingType,
											Ready: true,
										})
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
											resp := createBinding(ctx.SMWithOAuthForTenant, testCase.async, testCase.responseByBrokerOrClientMode(testCase.expectedBrokerFailureStatusCode, http.StatusAccepted))

											bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.CREATE,
												State:             types.FAILED,
												ResourceType:      types.ServiceBindingType,
												Reschedulable:     false,
												DeletionScheduled: false,
											})

											VerifyResource(ctx.SMWithOAuthForTenant, ResourceExpectations{
												ID:   bindingID,
												Type: types.ServiceBindingType,
											}, testCase.async, true)

										})
									})

									When("broker orphan mitigation unbind synchronously fails with an error that will stop further orphan mitigation", func() {
										BeforeEach(func() {
											brokerServer.BindingHandlerFunc(http.MethodDelete, http.MethodDelete+"3", ParameterizedHandler(http.StatusBadRequest, Object{"error": "error"}))
										})

										It("keeps the binding with ready false and marks the operation with deletion scheduled", func() {
											resp := createBinding(ctx.SMWithOAuthForTenant, testCase.async, testCase.responseByBrokerOrClientMode(testCase.expectedBrokerFailureStatusCode, http.StatusAccepted))

											bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.CREATE,
												State:             types.FAILED,
												ResourceType:      types.ServiceBindingType,
												Reschedulable:     false,
												DeletionScheduled: true,
											})

											VerifyResource(ctx.SMWithOAuthForTenant, ResourceExpectations{
												ID:   bindingID,
												Type: types.ServiceBindingType,
											}, testCase.async, true)
										})
									})

									When("broker orphan mitigation unbind synchronously fails with an error that will continue further orphan mitigation and eventually succeed", func() {
										BeforeEach(func() {
											brokerServer.BindingHandlerFunc(http.MethodDelete, http.MethodDelete+"3", MultipleErrorsBeforeSuccessHandler(
												http.StatusInternalServerError, http.StatusOK,
												Object{"error": "error"}, Object{"async": false},
											))
										})

										It("deletes the binding and marks the operation that triggered the orphan mitigation as failed with no deletion scheduled and not reschedulable", func() {
											resp := createBinding(ctx.SMWithOAuthForTenant, testCase.async, testCase.responseByBrokerOrClientMode(testCase.expectedBrokerFailureStatusCode, http.StatusAccepted))

											bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.CREATE,
												State:             types.FAILED,
												ResourceType:      types.ServiceBindingType,
												Reschedulable:     false,
												DeletionScheduled: false,
											})

											VerifyResource(ctx.SMWithOAuthForTenant, ResourceExpectations{
												ID:   bindingID,
												Type: types.ServiceBindingType,
											}, testCase.async, true)
										})
									})
								})

								When("polling returns an unexpected status code", func() {
									BeforeEach(func() {
										brokerServer.BindingHandlerFunc(http.MethodPut, http.MethodPut+"3", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
										brokerServer.BindingLastOpHandlerFunc(http.MethodPut+"3", ParameterizedHandler(http.StatusInternalServerError, Object{"error": "error"}))
									})

									It("stores the binding as ready false and marks the operation as reschedulable", func() {
										resp := createBinding(ctx.SMWithOAuthForTenant, testCase.async, testCase.responseByBrokerOrClientMode(testCase.expectedBrokerFailureStatusCode, http.StatusAccepted))

										bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
											Category:          types.CREATE,
											State:             types.FAILED,
											ResourceType:      types.ServiceBindingType,
											Reschedulable:     true,
											DeletionScheduled: false,
										})

										VerifyResourceExists(ctx.SMWithOAuthForTenant, ResourceExpectations{
											ID:    bindingID,
											Type:  types.ServiceBindingType,
											Ready: false,
										})
									})
								})
							})

							if testCase.async == "true" {
								When("SM crashes after storing operation before storing resource", func() {
									var newSMCtx *TestContext
									var anotherSMCtx *TestContext

									BeforeEach(func() {
										newSMCtx = t.ContextBuilder.WithEnvPostExtensions(func(e env.Environment, servers map[string]FakeServer) {
											e.Set("server.shutdown_timeout", 1*time.Second)
											e.Set("operations.maintainer_retry_interval", 1*time.Second)
										}).BuildWithoutCleanup()

										brokerServer.BindingHandlerFunc(http.MethodDelete, http.MethodDelete+"3", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
										brokerServer.BindingLastOpHandlerFunc(http.MethodDelete+"3", func(_ *http.Request) (int, map[string]interface{}) {
											return http.StatusOK, Object{"state": types.SUCCEEDED}
										})
									})

									AfterEach(func() {
										newSMCtx.CleanupAll(false)
										if anotherSMCtx != nil {
											anotherSMCtx.CleanupAll(false)
										}
									})

									It("Should mark operation as failed and trigger orphan mitigation", func() {
										opChan := make(chan *types.Operation)
										defer close(opChan)

										opCriteria := []query.Criterion{
											query.ByField(query.EqualsOperator, "type", string(types.CREATE)),
											query.ByField(query.EqualsOperator, "state", string(types.IN_PROGRESS)),
											query.ByField(query.EqualsOperator, "resource_type", string(types.ServiceBindingType)),
											query.ByField(query.EqualsOperator, "reschedule", "false"),
											query.ByField(query.EqualsOperator, "deletion_scheduled", operations.ZeroTime),
										}

										go func() {
											for {
												object, err := ctx.SMRepository.Get(context.TODO(), types.OperationType, opCriteria...)
												if err == nil {
													newSMCtx.CleanupAll(false)
													opChan <- object.(*types.Operation)
													break
												}
											}
										}()

										createBinding(newSMCtx.SMWithOAuthForTenant, testCase.async, testCase.expectedSMCrashStatusCode)
										operation := <-opChan

										VerifyResourceDoesNotExist(ctx.SMWithOAuthForTenant, ResourceExpectations{
											ID:   operation.ResourceID,
											Type: types.ServiceBindingType,
										})

										anotherSMCtx = t.ContextBuilder.WithEnvPostExtensions(func(e env.Environment, servers map[string]FakeServer) {
											e.Set("operations.action_timeout", 2*time.Second)
											e.Set("operations.cleanup_interval", 2*time.Second)
										}).BuildWithoutCleanup()

										operationExpectation := OperationExpectations{
											Category:          types.CREATE,
											State:             types.FAILED,
											ResourceType:      types.ServiceBindingType,
											Reschedulable:     false,
											DeletionScheduled: false,
										}

										bindingID, _ = VerifyOperationExists(ctx, fmt.Sprintf("%s/%s%s/%s", web.ServiceBindingsURL, operation.ResourceID, web.ResourceOperationsURL, operation.ID), operationExpectation)
										VerifyResourceDoesNotExist(ctx.SMWithOAuthForTenant, ResourceExpectations{
											ID:   bindingID,
											Type: types.ServiceBindingType,
										})
									})
								})
							}

							When("bind responds with error due to stopped broker", func() {
								BeforeEach(func() {
									brokerServer.Close()
									delete(ctx.Servers, BrokerServerPrefix+brokerID)
								})

								It("verifies the binding in SMDB and marks operation with failed", func() {
									resp := createBinding(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)

									bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.CREATE,
										State:             types.FAILED,
										ResourceType:      types.ServiceBindingType,
										Reschedulable:     false,
										DeletionScheduled: false,
									})

									VerifyResource(ctx.SMWithOAuthForTenant, ResourceExpectations{
										ID:   bindingID,
										Type: types.ServiceBindingType,
									}, testCase.async, false)
								})
							})

							When("bind responds with error that does not require orphan mitigation", func() {
								BeforeEach(func() {
									brokerServer.BindingHandlerFunc(http.MethodPut, http.MethodPut+"3", ParameterizedHandler(http.StatusBadRequest, Object{"error": "error"}))
								})

								AfterEach(func() {
									brokerServer.ResetHandlers()
								})

								It("verifies the binding and marks the operation as failed, non rescheduable with empty deletion scheduled", func() {
									resp := createBinding(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)

									bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.CREATE,
										State:             types.FAILED,
										ResourceType:      types.ServiceBindingType,
										Reschedulable:     false,
										DeletionScheduled: false,
									})

									VerifyResource(ctx.SMWithOAuthForTenant, ResourceExpectations{
										ID:   bindingID,
										Type: types.ServiceBindingType,
									}, testCase.async, false)

								})
							})

							When("bind responds with error that requires orphan mitigation", func() {
								BeforeEach(func() {
									brokerServer.BindingHandlerFunc(http.MethodPut, http.MethodPut+"3", ParameterizedHandler(http.StatusInternalServerError, Object{"error": "error"}))
								})

								AfterEach(func() {
									brokerServer.ResetHandlers()
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

										VerifyResource(ctx.SMWithOAuthForTenant, ResourceExpectations{
											ID:   bindingID,
											Type: types.ServiceBindingType,
										}, testCase.async, false)
									})
								})

								if testCase.async == "true" {
									When("broker orphan mitigation unbind asynchronously keeps failing with an error while polling", func() {
										BeforeEach(func() {
											brokerServer.BindingHandlerFunc(http.MethodDelete, http.MethodDelete+"3", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
											brokerServer.BindingLastOpHandlerFunc(http.MethodDelete+"3", ParameterizedHandler(http.StatusBadRequest, Object{"error": "error"}))
										})

										AfterEach(func() {
											brokerServer.ResetHandlers()
										})

										It("keeps the binding as ready false and marks the operation as deletion scheduled", func() {
											resp := createBinding(ctx.SMWithOAuthForTenant, "true", http.StatusAccepted)

											bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.CREATE,
												State:             types.FAILED,
												ResourceType:      types.ServiceBindingType,
												Reschedulable:     true,
												DeletionScheduled: true,
											})

											VerifyResourceExists(ctx.SMWithOAuthForTenant, ResourceExpectations{
												ID:    bindingID,
												Type:  types.ServiceBindingType,
												Ready: false,
											})
										})
									})
								}

								When("SM crashes while orphan mitigating", func() {
									var newSMCtx *TestContext
									var isUnbound atomic.Value

									BeforeEach(func() {
										newSMCtx = t.ContextBuilder.WithEnvPostExtensions(func(e env.Environment, servers map[string]FakeServer) {
											e.Set("server.shutdown_timeout", 1*time.Second)
											e.Set("operations.maintainer_retry_interval", 1*time.Second)
										}).BuildWithoutCleanup()

										brokerServer.BindingHandlerFunc(http.MethodDelete, http.MethodDelete+"3", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
										brokerServer.BindingLastOpHandlerFunc(http.MethodDelete+"3", func(_ *http.Request) (int, map[string]interface{}) {
											if isUnbound.Load() != nil {
												return http.StatusOK, Object{"state": types.SUCCEEDED}
											} else {
												return http.StatusOK, Object{"state": types.IN_PROGRESS}
											}
										})
									})

									AfterEach(func() {
										newSMCtx.CleanupAll(false)
									})

									It("should restart orphan mitigation through maintainer and eventually succeeds", func() {
										resp := createBinding(newSMCtx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)

										operationExpectations := OperationExpectations{
											Category:          types.CREATE,
											State:             types.FAILED,
											ResourceType:      types.ServiceBindingType,
											Reschedulable:     true,
											DeletionScheduled: true,
										}

										bindingID, _ = VerifyOperationExists(newSMCtx, resp.Header("Location").Raw(), operationExpectations)

										newSMCtx.CleanupAll(false)
										isUnbound.Store(true)

										newSMCtx = t.ContextBuilder.WithEnvPostExtensions(func(e env.Environment, servers map[string]FakeServer) {
											e.Set("operations.action_timeout", 2*time.Second)
										}).BuildWithoutCleanup()

										operationExpectations.DeletionScheduled = false
										operationExpectations.Reschedulable = false

										bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), operationExpectations)

										VerifyResource(ctx.SMWithOAuthForTenant, ResourceExpectations{
											ID:   bindingID,
											Type: types.ServiceBindingType,
										}, testCase.async, false)
									})

								})

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

										VerifyResource(ctx.SMWithOAuthForTenant, ResourceExpectations{
											ID:   bindingID,
											Type: types.ServiceBindingType,
										}, testCase.async, false)
									})
								})
							})

							When("bind responds with error due to times out", func() {
								var doneChannel chan interface{}
								var newCtx *TestContext

								BeforeEach(func() {
									doneChannel = make(chan interface{})
									newCtx = t.ContextBuilder.WithEnvPreExtensions(func(set *pflag.FlagSet) {
										Expect(set.Set("httpclient.timeout", (1 * time.Second).String())).ToNot(HaveOccurred())
									}).BuildWithoutCleanup()

									brokerServer.BindingHandlerFunc(http.MethodPut, http.MethodPut+"1", DelayingHandler(doneChannel))
								})

								AfterEach(func() {
									newCtx.CleanupAll(false)
								})

								It("orphan mitigates the binding", func() {
									resp := createBinding(newCtx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)
									<-time.After(1100 * time.Millisecond)
									close(doneChannel)
									bindingID, _ = VerifyOperationExists(newCtx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.CREATE,
										State:             types.FAILED,
										ResourceType:      types.ServiceBindingType,
										Reschedulable:     false,
										DeletionScheduled: false,
									})

									VerifyResource(ctx.SMWithOAuthForTenant, ResourceExpectations{
										ID:   bindingID,
										Type: types.ServiceBindingType,
									}, testCase.async, false)
								})
							})
						})

						When("creating binding with same name", func() {
							JustBeforeEach(func() {
								postBindingRequest["name"] = "same-binding-name"
								resp := createBinding(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedCreateSuccessStatusCode)
								VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
									Category:          types.CREATE,
									State:             types.SUCCEEDED,
									ResourceType:      types.ServiceBindingType,
									Reschedulable:     false,
									DeletionScheduled: false,
								})
							})

							When("for the same service instance", func() {
								It("should reject", func() {
									statusCode := http.StatusAccepted
									if testCase.async == "false" || testCase.async == "" {
										statusCode = http.StatusConflict
									}
									resp := createBinding(ctx.SMWithOAuthForTenant, testCase.async, statusCode)
									VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.CREATE,
										State:             types.FAILED,
										ResourceType:      types.ServiceBindingType,
										Reschedulable:     false,
										DeletionScheduled: false,
										Error:             "binding with same name exists for instance with id",
									})
								})
							})

							When("for other service instance", func() {
								var otherInstanceID string

								JustBeforeEach(func() {
									otherInstanceID = createInstance(ctx.SMWithOAuthForTenant, false, http.StatusCreated).JSON().Object().Value("id").String().Raw()
									postBindingRequest["service_instance_id"] = otherInstanceID
								})

								It("should accept", func() {
									resp := createBinding(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedCreateSuccessStatusCode)
									VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.CREATE,
										State:             types.SUCCEEDED,
										ResourceType:      types.ServiceBindingType,
										Reschedulable:     false,
										DeletionScheduled: false,
									})
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
					Context(fmt.Sprintf("async = %s", testCase.async), func() {
						BeforeEach(func() {
							brokerServer.ShouldRecordRequests(true)
						})

						Context("instance ownership", func() {
							When("tenant doesn't have ownership of binding", func() {
								It("returns 404", func() {
									resp := createBinding(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedCreateSuccessStatusCode)
									bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.CREATE,
										State:             types.SUCCEEDED,
										ResourceType:      types.ServiceBindingType,
										Reschedulable:     false,
										DeletionScheduled: false,
									})
									VerifyResourceExists(ctx.SMWithOAuthForTenant, ResourceExpectations{
										ID:    bindingID,
										Type:  types.ServiceBindingType,
										Ready: true,
									})

									expectedCode := http.StatusNotFound
									if testCase.async == "true" {
										expectedCode = http.StatusAccepted
									}
									smWithOtherTenant := ctx.NewTenantExpect("tenancyClient", "other-tenant")
									deleteBinding(smWithOtherTenant, testCase.async, expectedCode)

									VerifyResourceExists(ctx.SMWithOAuthForTenant, ResourceExpectations{
										ID:    bindingID,
										Type:  types.ServiceBindingType,
										Ready: true,
									})

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
									VerifyResourceExists(ctx.SMWithOAuthForTenant, ResourceExpectations{
										ID:    bindingID,
										Type:  types.ServiceBindingType,
										Ready: true,
									})

									resp = deleteBinding(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedDeleteSuccessStatusCode)

									bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.DELETE,
										State:             types.SUCCEEDED,
										ResourceType:      types.ServiceBindingType,
										Reschedulable:     false,
										DeletionScheduled: false,
									})
									VerifyResourceDoesNotExist(ctx.SMWithOAuthForTenant, ResourceExpectations{
										ID:   bindingID,
										Type: types.ServiceBindingType,
									})
								})
							})
						})

						Context("broker scenarios", func() {
							BeforeEach(func() {
								resp := createBinding(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedCreateSuccessStatusCode)
								bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
									Category:          types.CREATE,
									State:             types.SUCCEEDED,
									ResourceType:      types.ServiceBindingType,
									Reschedulable:     false,
									DeletionScheduled: false,
								})
								VerifyResourceExists(ctx.SMWithOAuthForTenant, ResourceExpectations{
									ID:    bindingID,
									Type:  types.ServiceBindingType,
									Ready: true,
								})

							})

							for _, bindingRetrievable := range []bool{true, false} {
								bindingRetrievable := bindingRetrievable
								When(fmt.Sprintf("plan specifies binding_retrievable %t", bindingRetrievable), func() {
									BeforeEach(func() {
										brokerServer.BindingHandlerFunc(http.MethodDelete, http.MethodDelete, func(req *http.Request) (int, map[string]interface{}) {
											acceptsIncomplete := req.FormValue("accepts_incomplete")
											if len(acceptsIncomplete) == 0 {
												acceptsIncomplete = "false"
											}
											Expect(acceptsIncomplete).To(Equal(strconv.FormatBool(bindingRetrievable)))

											return http.StatusOK, Object{}
										})
										servicePlanID = findPlanIDForBrokerIDAndBindingRetrievable(ctx, brokerID, bindingRetrievable)
										EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, servicePlanID, TenantIDValue)

										resp := createInstance(ctx.SMWithOAuthForTenant, false, http.StatusCreated)

										instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
											Category:          types.CREATE,
											State:             types.SUCCEEDED,
											ResourceType:      types.ServiceInstanceType,
											Reschedulable:     false,
											DeletionScheduled: false,
										})

										VerifyResourceExists(ctx.SMWithOAuthForTenant, ResourceExpectations{
											ID:    instanceID,
											Type:  types.ServiceInstanceType,
											Ready: true,
										})

										postBindingRequest["name"] = "test-binding-retrievable-name"
										postBindingRequest["service_instance_id"] = instanceID
										resp = createBinding(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedCreateSuccessStatusCode)

										bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
											Category:          types.CREATE,
											State:             types.SUCCEEDED,
											ResourceType:      types.ServiceBindingType,
											Reschedulable:     false,
											DeletionScheduled: false,
										})

										VerifyResourceExists(ctx.SMWithOAuthForTenant, ResourceExpectations{
											ID:    bindingID,
											Type:  types.ServiceBindingType,
											Ready: true,
										})
									})

									It("successfully deletes binding", func() {
										resp := deleteBinding(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedDeleteSuccessStatusCode)

										bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
											Category:          types.DELETE,
											State:             types.SUCCEEDED,
											ResourceType:      types.ServiceBindingType,
											Reschedulable:     false,
											DeletionScheduled: false,
										})

										VerifyResourceDoesNotExist(ctx.SMWithOAuthForTenant, ResourceExpectations{
											ID:   bindingID,
											Type: types.ServiceBindingType,
										})
									})
								})
							}

							When("a delete operation is already in progress", func() {
								var doneChannel chan interface{}

								BeforeEach(func() {
									doneChannel = make(chan interface{})
									brokerServer.BindingHandlerFunc(http.MethodDelete, http.MethodDelete+"1", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
									brokerServer.BindingLastOpHandlerFunc(http.MethodDelete+"1", DelayingHandler(doneChannel))

									resp := deleteBinding(ctx.SMWithOAuthForTenant, "true", http.StatusAccepted)

									bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.DELETE,
										State:             types.IN_PROGRESS,
										ResourceType:      types.ServiceBindingType,
										Reschedulable:     true,
										DeletionScheduled: false,
									})

									VerifyResourceExists(ctx.SMWithOAuthForTenant, ResourceExpectations{
										ID:    bindingID,
										Type:  types.ServiceBindingType,
										Ready: true,
									})
								})

								AfterEach(func() {
									close(doneChannel)
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

									VerifyResourceDoesNotExist(ctx.SMWithOAuthForTenant, ResourceExpectations{
										ID:   bindingID,
										Type: types.ServiceBindingType,
									})
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

									VerifyResourceDoesNotExist(ctx.SMWithOAuthForTenant, ResourceExpectations{
										ID:   bindingID,
										Type: types.ServiceBindingType,
									})
								})
							})

							When("broker responds with asynchronous success", func() {
								BeforeEach(func() {
									brokerServer.BindingHandlerFunc(http.MethodDelete, http.MethodDelete+"1", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
									brokerServer.BindingLastOpHandlerFunc(http.MethodDelete+"1", MultiplePollsRequiredHandler("in progress", "succeeded"))
								})

								It("polling broker last operation until operation succeeds and eventually marks operation as success", func() {
									resp := deleteBinding(ctx.SMWithOAuthForTenant, testCase.async, testCase.responseByBrokerOrClientMode(testCase.expectedDeleteSuccessStatusCode, http.StatusAccepted))

									bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.DELETE,
										State:             types.SUCCEEDED,
										ResourceType:      types.ServiceBindingType,
										Reschedulable:     false,
										DeletionScheduled: false,
									})

									VerifyResourceDoesNotExist(ctx.SMWithOAuthForTenant, ResourceExpectations{
										ID:   bindingID,
										Type: types.ServiceBindingType,
									})
								})

								When("maximum polling duration is reached while polling", func() {
									var newCtx *TestContext
									BeforeEach(func() {
										preparePrerequisitesWithMaxPollingDuration(MaximumPollingDuration)

										resp := createBinding(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedCreateSuccessStatusCode)
										bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
											Category:          types.CREATE,
											State:             types.SUCCEEDED,
											ResourceType:      types.ServiceBindingType,
											Reschedulable:     false,
											DeletionScheduled: false,
										})
										VerifyResourceExists(ctx.SMWithOAuthForTenant, ResourceExpectations{
											ID:    bindingID,
											Type:  types.ServiceBindingType,
											Ready: true,
										})

										newCtx = t.ContextBuilder.WithEnvPreExtensions(func(set *pflag.FlagSet) {
											Expect(set.Set("operations.action_timeout", ((MaximumPollingDuration + 1) * time.Second).String())).ToNot(HaveOccurred())
										}).BuildWithoutCleanup()

										brokerServer.BindingHandlerFunc(http.MethodDelete, http.MethodDelete+"1", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
										brokerServer.BindingLastOpHandlerFunc(http.MethodDelete+"1", ParameterizedHandler(http.StatusOK, Object{"state": "in progress"}))
									})

									AfterEach(func() {
										newCtx.CleanupAll(false)
									})

									When("orphan mitigation unbind synchronously succeeds", func() {
										It("deletes the binding and marks the operation as success", func() {
											resp := deleteBinding(newCtx.SMWithOAuthForTenant, testCase.async, testCase.responseByBrokerOrClientMode(testCase.expectedBrokerFailureStatusCode, http.StatusAccepted))

											bindingID, _ = VerifyOperationExists(newCtx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.DELETE,
												State:             types.FAILED,
												ResourceType:      types.ServiceBindingType,
												Reschedulable:     false,
												DeletionScheduled: true,
											})

											brokerServer.BindingHandlerFunc(http.MethodDelete, http.MethodDelete+"2", ParameterizedHandler(http.StatusOK, Object{"async": false}))

											bindingID, _ = VerifyOperationExists(newCtx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.DELETE,
												State:             types.SUCCEEDED,
												ResourceType:      types.ServiceBindingType,
												Reschedulable:     false,
												DeletionScheduled: false,
											})

											VerifyResourceDoesNotExist(newCtx.SMWithOAuthForTenant, ResourceExpectations{
												ID:   bindingID,
												Type: types.ServiceBindingType,
											})
										})
									})

									When("broker orphan mitigation unbind synchronously fails with an unexpected error", func() {
										AfterEach(func() {
											brokerServer.ResetHandlers()
										})

										It("keeps the binding and marks the operation with deletion scheduled", func() {
											resp := deleteBinding(newCtx.SMWithOAuthForTenant, testCase.async, testCase.responseByBrokerOrClientMode(testCase.expectedBrokerFailureStatusCode, http.StatusAccepted))

											bindingID, _ = VerifyOperationExists(newCtx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.DELETE,
												State:             types.FAILED,
												ResourceType:      types.ServiceBindingType,
												Reschedulable:     false,
												DeletionScheduled: true,
											})

											brokerServer.BindingHandlerFunc(http.MethodDelete, http.MethodDelete+"2", ParameterizedHandler(http.StatusBadRequest, Object{"error": "error"}))

											bindingID, _ = VerifyOperationExists(newCtx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.DELETE,
												State:             types.FAILED,
												ResourceType:      types.ServiceBindingType,
												Reschedulable:     false,
												DeletionScheduled: true,
											})

											VerifyResourceExists(newCtx.SMWithOAuthForTenant, ResourceExpectations{
												ID:    bindingID,
												Type:  types.ServiceBindingType,
												Ready: true,
											})
										})
									})

									When("broker orphan mitigation unbind synchronously fails with an error that will continue further orphan mitigation and eventually succeed", func() {
										It("deletes the binding and marks the operation that triggered the orphan mitigation as failed with no deletion scheduled and not reschedulable", func() {
											resp := deleteBinding(newCtx.SMWithOAuthForTenant, testCase.async, testCase.responseByBrokerOrClientMode(testCase.expectedBrokerFailureStatusCode, http.StatusAccepted))

											bindingID, _ = VerifyOperationExists(newCtx, resp.Header("Location").Raw(), OperationExpectations{
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

											bindingID, _ = VerifyOperationExists(newCtx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.DELETE,
												State:             types.SUCCEEDED,
												ResourceType:      types.ServiceBindingType,
												Reschedulable:     false,
												DeletionScheduled: false,
											})

											VerifyResourceDoesNotExist(newCtx.SMWithOAuthForTenant, ResourceExpectations{
												ID:   bindingID,
												Type: types.ServiceBindingType,
											})
										})
									})
								})

								if testCase.async == "true" {
									When("SM crashes while polling", func() {
										var newSMCtx *TestContext
										var isBound atomic.Value

										BeforeEach(func() {
											newSMCtx = t.ContextBuilder.WithEnvPostExtensions(func(e env.Environment, servers map[string]FakeServer) {
												e.Set("server.shutdown_timeout", 1*time.Second)
												e.Set("operations.maintainer_retry_interval", 1*time.Second)
											}).BuildWithoutCleanup()

											brokerServer.BindingHandlerFunc(http.MethodDelete, http.MethodDelete+"1", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
											brokerServer.BindingLastOpHandlerFunc(http.MethodDelete+"1", func(_ *http.Request) (int, map[string]interface{}) {
												if isBound.Load() != nil {
													return http.StatusOK, Object{"state": types.SUCCEEDED}
												} else {
													return http.StatusOK, Object{"state": types.IN_PROGRESS}
												}
											})

										})

										AfterEach(func() {
											newSMCtx.CleanupAll(false)
										})

										It("should start restart polling through maintainer and eventually binding is set to ready", func() {
											resp := deleteBinding(newSMCtx.SMWithOAuthForTenant, "true", http.StatusAccepted)

											operationExpectations := OperationExpectations{
												Category:          types.DELETE,
												State:             types.IN_PROGRESS,
												ResourceType:      types.ServiceBindingType,
												Reschedulable:     true,
												DeletionScheduled: false,
											}

											bindingID, _ = VerifyOperationExists(newSMCtx, resp.Header("Location").Raw(), operationExpectations)
											VerifyResourceExists(ctx.SMWithOAuthForTenant, ResourceExpectations{
												ID:    bindingID,
												Type:  types.ServiceBindingType,
												Ready: true,
											})

											newSMCtx.CleanupAll(false)
											isBound.Store(true)

											newSMCtx = t.ContextBuilder.WithEnvPostExtensions(func(e env.Environment, servers map[string]FakeServer) {
												e.Set("operations.action_timeout", 2*time.Second)
											}).BuildWithoutCleanup()

											operationExpectations.State = types.SUCCEEDED
											operationExpectations.Reschedulable = false

											bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), operationExpectations)
											VerifyResourceDoesNotExist(ctx.SMWithOAuthForTenant, ResourceExpectations{
												ID:   bindingID,
												Type: types.ServiceBindingType,
											})
										})
									})
								}

								When("polling responds 410 GONE", func() {
									BeforeEach(func() {
										brokerServer.BindingHandlerFunc(http.MethodDelete, http.MethodDelete+"1", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
										brokerServer.BindingLastOpHandlerFunc(http.MethodDelete+"1", ParameterizedHandler(http.StatusGone, Object{}))
									})

									It("keeps polling and eventually deletes the binding and marks the operation as success", func() {
										resp := deleteBinding(ctx.SMWithOAuthForTenant, testCase.async, testCase.responseByBrokerOrClientMode(testCase.expectedDeleteSuccessStatusCode, http.StatusAccepted))

										bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
											Category:          types.DELETE,
											State:             types.SUCCEEDED,
											ResourceType:      types.ServiceBindingType,
											Reschedulable:     false,
											DeletionScheduled: false,
										})

										VerifyResourceDoesNotExist(ctx.SMWithOAuthForTenant, ResourceExpectations{
											ID:   bindingID,
											Type: types.ServiceBindingType,
										})
									})
								})

								When("polling responds with unexpected state and eventually with success state", func() {
									BeforeEach(func() {
										brokerServer.BindingHandlerFunc(http.MethodDelete, http.MethodDelete+"1", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
										brokerServer.BindingLastOpHandlerFunc(http.MethodDelete+"1", MultiplePollsRequiredHandler("unknown", "succeeded"))
									})

									It("keeps polling and eventually deletes the binding and marks the operation as success", func() {
										resp := deleteBinding(ctx.SMWithOAuthForTenant, testCase.async, testCase.responseByBrokerOrClientMode(testCase.expectedDeleteSuccessStatusCode, http.StatusAccepted))

										bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
											Category:          types.DELETE,
											State:             types.SUCCEEDED,
											ResourceType:      types.ServiceBindingType,
											Reschedulable:     false,
											DeletionScheduled: false,
										})

										VerifyResourceDoesNotExist(ctx.SMWithOAuthForTenant, ResourceExpectations{
											ID:   bindingID,
											Type: types.ServiceBindingType,
										})
									})
								})

								When("polling responds with unexpected state and eventually with failed state", func() {
									BeforeEach(func() {
										brokerServer.BindingHandlerFunc(http.MethodDelete, http.MethodDelete+"2", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
										brokerServer.BindingLastOpHandlerFunc(http.MethodDelete+"2", MultiplePollsRequiredHandler("unknown", "failed"))
									})

									When("orphan mitigation unbind synchronously succeeds", func() {
										It("deletes the binding and marks the operation as success", func() {
											resp := deleteBinding(ctx.SMWithOAuthForTenant, testCase.async, testCase.responseByBrokerOrClientMode(testCase.expectedBrokerFailureStatusCode, http.StatusAccepted))

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

											VerifyResourceDoesNotExist(ctx.SMWithOAuthForTenant, ResourceExpectations{
												ID:   bindingID,
												Type: types.ServiceBindingType,
											})
										})
									})

									When("broker orphan mitigation unbind synchronously fails with an unexpected error", func() {
										AfterEach(func() {
											brokerServer.ResetHandlers()
										})

										It("keeps the binding and marks the operation with deletion scheduled", func() {
											resp := deleteBinding(ctx.SMWithOAuthForTenant, testCase.async, testCase.responseByBrokerOrClientMode(testCase.expectedBrokerFailureStatusCode, http.StatusAccepted))

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

											VerifyResourceExists(ctx.SMWithOAuthForTenant, ResourceExpectations{
												ID:    bindingID,
												Type:  types.ServiceBindingType,
												Ready: true,
											})
										})
									})

									When("broker orphan mitigation unbind synchronously fails with an error that will continue further orphan mitigation and eventually succeed", func() {
										It("deletes the binding and marks the operation that triggered the orphan mitigation as failed with no deletion scheduled and not reschedulable", func() {
											resp := deleteBinding(ctx.SMWithOAuthForTenant, testCase.async, testCase.responseByBrokerOrClientMode(testCase.expectedBrokerFailureStatusCode, http.StatusAccepted))

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

											VerifyResourceDoesNotExist(ctx.SMWithOAuthForTenant, ResourceExpectations{
												ID:   bindingID,
												Type: types.ServiceBindingType,
											})
										})
									})
								})

								When("polling returns an unexpected status code", func() {
									BeforeEach(func() {
										brokerServer.BindingHandlerFunc(http.MethodDelete, http.MethodDelete+"3", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
										brokerServer.BindingLastOpHandlerFunc(http.MethodDelete+"3", ParameterizedHandler(http.StatusInternalServerError, Object{"error": "error"}))
									})

									It("keeps the binding and stores the operation as reschedulable", func() {
										resp := deleteBinding(ctx.SMWithOAuthForTenant, testCase.async, testCase.responseByBrokerOrClientMode(testCase.expectedBrokerFailureStatusCode, http.StatusAccepted))

										bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
											Category:          types.DELETE,
											State:             types.FAILED,
											ResourceType:      types.ServiceBindingType,
											Reschedulable:     true,
											DeletionScheduled: false,
										})

										VerifyResourceExists(ctx.SMWithOAuthForTenant, ResourceExpectations{
											ID:    bindingID,
											Type:  types.ServiceBindingType,
											Ready: true,
										})
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

									VerifyResourceExists(ctx.SMWithOAuthForTenant, ResourceExpectations{
										ID:    bindingID,
										Type:  types.ServiceBindingType,
										Ready: true,
									})
								})

								It("deletes the binding and marks operation with success when cascade and force are passed", func() {
									resp := ctx.SMWithOAuthForTenant.DELETE(web.ServiceBindingsURL+"/"+bindingID).
										WithQuery("async", testCase.async).
										WithQuery("force", true).
										WithQuery("cascade", true).
										Expect().
										Status(http.StatusAccepted)

									bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.DELETE,
										State:             types.SUCCEEDED,
										ResourceType:      types.ServiceBindingType,
										Reschedulable:     false,
										DeletionScheduled: false,
									})

									VerifyResourceDoesNotExist(ctx.SMWithOAuthForTenant, ResourceExpectations{
										ID:   bindingID,
										Type: types.ServiceBindingType,
									})
								})
							})

							When("unbind responds with error that does not require orphan mitigation", func() {
								BeforeEach(func() {
									brokerServer.BindingHandlerFunc(http.MethodDelete, http.MethodDelete+"3", ParameterizedHandler(http.StatusBadRequest, Object{"error": "error"}))
								})

								AfterEach(func() {
									brokerServer.ResetHandlers()
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

									VerifyResourceExists(ctx.SMWithOAuthForTenant, ResourceExpectations{
										ID:    bindingID,
										Type:  types.ServiceBindingType,
										Ready: true,
									})
								})

								It("deletes the binding and marks operation with success when cascade and force are passed", func() {
									resp := ctx.SMWithOAuthForTenant.DELETE(web.ServiceBindingsURL+"/"+bindingID).
										WithQuery("async", testCase.async).
										WithQuery("force", true).
										WithQuery("cascade", true).
										Expect().
										Status(http.StatusAccepted)

									bindingID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.DELETE,
										State:             types.SUCCEEDED,
										ResourceType:      types.ServiceBindingType,
										Reschedulable:     false,
										DeletionScheduled: false,
									})

									VerifyResourceDoesNotExist(ctx.SMWithOAuthForTenant, ResourceExpectations{
										ID:   bindingID,
										Type: types.ServiceBindingType,
									})
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

										VerifyResourceDoesNotExist(ctx.SMWithOAuthForTenant, ResourceExpectations{
											ID:   bindingID,
											Type: types.ServiceBindingType,
										})
									})
								})

								if testCase.async == "true" {
									When("broker orphan mitigation unbind asynchronously keeps failing with an error while polling", func() {
										AfterEach(func() {
											brokerServer.ResetHandlers()
										})

										It("keeps the binding and marks the operation as failed reschedulable with deletion scheduled", func() {
											resp := deleteBinding(ctx.SMWithOAuthForTenant, "true", http.StatusAccepted)

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

											VerifyResourceExists(ctx.SMWithOAuthForTenant, ResourceExpectations{
												ID:    bindingID,
												Type:  types.ServiceBindingType,
												Ready: true,
											})
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

										VerifyResourceDoesNotExist(ctx.SMWithOAuthForTenant, ResourceExpectations{
											ID:   bindingID,
											Type: types.ServiceBindingType,
										})
									})
								})
							})

							When("unbind responds with error due to times out", func() {
								var doneChannel chan interface{}
								var newCtx *TestContext

								BeforeEach(func() {
									doneChannel = make(chan interface{})
									newCtx = t.ContextBuilder.WithEnvPreExtensions(func(set *pflag.FlagSet) {
										Expect(set.Set("httpclient.timeout", (1 * time.Second).String())).ToNot(HaveOccurred())
									}).BuildWithoutCleanup()

									brokerServer.BindingHandlerFunc(http.MethodDelete, http.MethodDelete+"1", DelayingHandler(doneChannel))

								})

								AfterEach(func() {
									newCtx.CleanupAll(false)
								})

								It("orphan mitigates the binding", func() {
									resp := deleteBinding(newCtx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)
									<-time.After(1100 * time.Millisecond)
									close(doneChannel)
									bindingID, _ = VerifyOperationExists(newCtx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.DELETE,
										State:             types.FAILED,
										ResourceType:      types.ServiceBindingType,
										Reschedulable:     false,
										DeletionScheduled: true,
									})

									brokerServer.BindingHandlerFunc(http.MethodDelete, http.MethodDelete+"1", ParameterizedHandler(http.StatusOK, Object{"async": false}))

									bindingID, _ = VerifyOperationExists(newCtx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.DELETE,
										State:             types.SUCCEEDED,
										ResourceType:      types.ServiceBindingType,
										Reschedulable:     false,
										DeletionScheduled: false,
									})

									VerifyResourceDoesNotExist(newCtx.SMWithOAuthForTenant, ResourceExpectations{
										ID:   bindingID,
										Type: types.ServiceBindingType,
									})
								})
							})
						})
					})
				}
			})
		})
	},
})

func blueprint(ctx *TestContext, _ *SMExpect, async bool) Object {
	_, _, servicePlanID, _, _ := newServicePlan(ctx, true)
	EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, servicePlanID, TenantIDValue)
	ID, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}
	resp := ctx.SMWithOAuthForTenant.POST(web.ServiceInstancesURL).
		WithQuery("async", strconv.FormatBool(async)).
		WithJSON(Object{
			"name":             "test-service-instance" + ID.String(),
			"service_plan_id":  servicePlanID,
			"maintenance_info": "{}",
		}).Expect()

	var instance map[string]interface{}
	if async {
		resp.Status(http.StatusAccepted)
	} else {
		resp.Status(http.StatusCreated)
	}

	id, _ := VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
		Category:          types.CREATE,
		State:             types.SUCCEEDED,
		ResourceType:      types.ServiceInstanceType,
		Reschedulable:     false,
		DeletionScheduled: false,
	})

	instance = VerifyResourceExists(ctx.SMWithOAuthForTenant, ResourceExpectations{
		ID:    id,
		Type:  types.ServiceInstanceType,
		Ready: true,
	}).Raw()

	resp = ctx.SMWithOAuthForTenant.POST(web.ServiceBindingsURL).
		WithQuery("async", strconv.FormatBool(async)).
		WithJSON(Object{
			"name":                "test-service-binding",
			"service_instance_id": instance["id"],
		}).Expect()

	var binding map[string]interface{}
	if async {
		resp.Status(http.StatusAccepted)
	} else {
		resp.Status(http.StatusCreated)
	}

	id, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
		Category:          types.CREATE,
		State:             types.SUCCEEDED,
		ResourceType:      types.ServiceBindingType,
		Reschedulable:     false,
		DeletionScheduled: false,
	})

	binding = VerifyResourceExists(ctx.SMWithOAuthForTenant, ResourceExpectations{
		ID:    id,
		Type:  types.ServiceBindingType,
		Ready: true,
	}).Raw()

	delete(binding, "credentials")
	return binding
}

func newServicePlan(ctx *TestContext, bindable bool) (string, *BrokerServer, string, string, string) {
	return newServicePlanWithMaxPollingDuration(ctx, bindable, 0)
}

func newServicePlanWithMaxPollingDuration(ctx *TestContext, bindable bool, maxPollingDuration int) (string, *BrokerServer, string, string, string) {
	cPaidPlan1 := GeneratePaidTestPlan()
	cPaidPlan1, err := sjson.Set(cPaidPlan1, "maximum_polling_duration", maxPollingDuration)
	if err != nil {
		panic(err)
	}
	cPaidPlan2 := GeneratePaidTestPlan()
	cPaidPlan2, err = sjson.Set(cPaidPlan2, "bindable", false)
	if err != nil {
		panic(err)
	}
	cService := GenerateTestServiceWithPlans(cPaidPlan1, cPaidPlan2)

	freePlan := GenerateFreeTestPlan()
	service2 := GenerateTestServiceWithPlans(freePlan)
	service2, err = sjson.Set(service2, "bindings_retrievable", false)
	if err != nil {
		panic(err)
	}

	catalog := NewEmptySBCatalog()
	catalog.AddService(cService)
	catalog.AddService(service2)

	brokerID, _, server := ctx.RegisterBrokerWithCatalog(catalog).GetBrokerAsParams()
	ctx.Servers[BrokerServerPrefix+brokerID] = server

	servicePlanID, servicePlanCatalogID, serviceCatalogID := findPlanDetailsForBrokerID(ctx, brokerID, bindable)
	return brokerID, server, servicePlanID, servicePlanCatalogID, serviceCatalogID
}

func findPlanDetailsForBrokerID(ctx *TestContext, brokerID string, bindable bool) (string, string, string) {
	so := ctx.SMWithOAuth.ListWithQuery(web.ServiceOfferingsURL, fmt.Sprintf("fieldQuery=broker_id eq '%s'", brokerID)).First()
	serviceCatalogID := so.Object().Value("catalog_id").String().Raw()
	resp := ctx.SMWithOAuth.ListWithQuery(web.ServicePlansURL, "fieldQuery="+fmt.Sprintf("service_offering_id eq '%s' and bindable eq %t", so.Object().Value("id").String().Raw(), bindable)).
		First().Object()

	servicePlanID := resp.Value("id").String().Raw()
	servicePlanCatalogID := resp.Value("catalog_id").String().Raw()

	return servicePlanID, servicePlanCatalogID, serviceCatalogID
}

func findPlanIDForBrokerIDAndBindingRetrievable(ctx *TestContext, brokerID string, bindingRetrievable bool) string {
	so := ctx.SMWithOAuth.ListWithQuery(web.ServiceOfferingsURL, fmt.Sprintf("fieldQuery=broker_id eq '%s' and bindings_retrievable eq %t", brokerID, bindingRetrievable)).First()
	servicePlanID := ctx.SMWithOAuth.ListWithQuery(web.ServicePlansURL, "fieldQuery="+fmt.Sprintf("service_offering_id eq '%s'", so.Object().Value("id").String().Raw())).
		First().Object().Value("id").String().Raw()
	Expect(servicePlanID).ToNot(BeEmpty())

	return servicePlanID
}
