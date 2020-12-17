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
	"github.com/Peripli/service-manager/pkg/query"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Peripli/service-manager/pkg/env"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

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
	TenantIdentifier                   = "tenant"
	TenantIDValue                      = "tenantID"
	serviceNotSupportingContextUpdates = "serviceNotSupportingContextUpdatesID"
	service1CatalogID                  = "service1CatalogID"
	notRetrievableService              = "notRetrievableService"
	plan1CatalogID                     = "plan1CatalogID"
	planNotSupportingSMPlatform        = "planNotSupportingSmID"
	MaximumPollingDuration             = 2 // seconds
)

func checkInstance(req *http.Request) (int, map[string]interface{}) {
	body, err := util.BodyToBytes(req.Body)
	Expect(err).ToNot(HaveOccurred())
	tenantValue := gjson.GetBytes(body, "context."+TenantIdentifier).String()
	Expect(tenantValue).To(Equal(TenantIDValue))
	platformValue := gjson.GetBytes(body, "context.platform").String()
	Expect(platformValue).To(Equal(types.SMPlatform))

	return http.StatusCreated, Object{}
}

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
	SupportsCascadeDeleteOperations:        true,
	DisableTenantResources:                 false,
	StrictlyTenantScoped:                   true,
	ResourceBlueprint:                      blueprint,
	ResourceWithoutNullableFieldsBlueprint: blueprint,
	SubResourcesBlueprint:                  subResourcesBlueprint(),
	ResourcePropertiesToIgnore:             []string{"last_operation", "platform_id"},
	PatchResource:                          APIResourcePatch,
	AdditionalTests: func(ctx *TestContext, t *TestCase) {
		Context("additional non-generic tests", func() {
			var (
				postInstanceRequest    Object
				postInstanceRequestTLS Object
				patchInstanceRequest   Object

				servicePlanID               string
				servicePlanIDWithTLS        string
				anotherServicePlanCatalogID string
				anotherServicePlanID        string
				brokerID                    string
				brokerServer                *BrokerServer
				brokerServerWithTLS         *BrokerServer
				instanceID                  string
			)

			testCases := []testCase{
				{
					async:                           "false",
					expectedCreateSuccessStatusCode: http.StatusCreated,
					expectedUpdateSuccessStatusCode: http.StatusOK,
					expectedDeleteSuccessStatusCode: http.StatusOK,
					expectedBrokerFailureStatusCode: http.StatusBadGateway,
					expectedSMCrashStatusCode:       http.StatusBadGateway,
				},
				{
					async:                           "true",
					expectedCreateSuccessStatusCode: http.StatusAccepted,
					expectedUpdateSuccessStatusCode: http.StatusAccepted,
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

			createInstance := func(smClient *SMExpect, async string, expectedStatusCode int) *httpexpect.Response {
				resp := smClient.POST(web.ServiceInstancesURL).
					WithQuery("async", async).
					WithJSON(postInstanceRequest).
					Expect().Status(expectedStatusCode)

				if resp.Raw().StatusCode == http.StatusCreated {
					obj := resp.JSON().Object()

					obj.ContainsKey("id").
						ValueEqual("platform_id", types.SMPlatform)

					instanceID = obj.Value("id").String().Raw()
				}

				return resp
			}

			patchInstance := func(smClient *SMExpect, async string, instanceID string, expectedStatusCode int) *httpexpect.Response {
				return smClient.PATCH(web.ServiceInstancesURL+"/"+instanceID).
					WithQuery("async", async).
					WithJSON(patchInstanceRequest).
					Expect().Status(expectedStatusCode)
			}

			deleteInstance := func(smClient *SMExpect, async string, expectedStatusCode int) *httpexpect.Response {
				return smClient.DELETE(web.ServiceInstancesURL+"/"+instanceID).
					WithQuery("async", async).
					Expect().
					Status(expectedStatusCode)
			}

			verificationHandler := func(bodyExpectations map[string]string, code int) func(req *http.Request) (int, map[string]interface{}) {
				return func(req *http.Request) (int, map[string]interface{}) {
					body, err := util.BodyToBytes(req.Body)
					Expect(err).ToNot(HaveOccurred())
					for k, v := range bodyExpectations {
						actualBodyValue := gjson.GetBytes(body, k).String()
						Expect(actualBodyValue).To(Equal(v))
					}

					return code, Object{}
				}
			}

			preparePrerequisitesWithMaxPollingDuration := func(maxPollingDuration int) {
				ID, err := uuid.NewV4()
				Expect(err).ToNot(HaveOccurred())
				var plans *httpexpect.Array
				brokerUtils, plans := prepareBrokerWithCatalogAndPollingDuration(ctx, ctx.SMWithOAuth, maxPollingDuration)
				brokerID = brokerUtils.Broker.ID
				brokerUtils.BrokerWithTLS = ctx.RegisterBrokerWithRandomCatalogAndTLS(ctx.SMWithOAuth).BrokerWithTLS
				brokerServer = brokerUtils.Broker.BrokerServer
				brokerServerWithTLS = brokerUtils.BrokerWithTLS.BrokerServer
				brokerServerWithTLS.ShouldRecordRequests(false)
				brokerServer.ShouldRecordRequests(false)
				servicePlanID = plans.Element(0).Object().Value("id").String().Raw()
				anotherServicePlanCatalogID = plans.Element(1).Object().Value("catalog_id").String().Raw()
				anotherServicePlanID = plans.Element(1).Object().Value("id").String().Raw()

				postInstanceRequest = Object{
					"name":             "test-instance" + ID.String(),
					"service_plan_id":  servicePlanID,
					"maintenance_info": "{}",
				}

				prepareBrokerWithCatalog(ctx, ctx.SMWithOAuth)
				postInstanceRequestTLS, servicePlanIDWithTLS = brokerUtils.SetAuthContext(ctx.SMWithOAuth).
					GetServiceOfferings(brokerUtils.BrokerWithTLS.ID).GetServicePlans(0, "id").
					GetPlan(0, "id").
					GetAsServiceInstancePayload()

				patchInstanceRequest = Object{}
			}

			preparePrerequisites := func() {
				preparePrerequisitesWithMaxPollingDuration(0)
			}

			BeforeEach(func() {
				preparePrerequisites()
			})

			AfterEach(func() {
				ctx.CleanupAdditionalResources()
			})

			Describe("get parameters", func() {
				When("service instance does not exist", func() {
					It("should return an error", func() {
						ctx.SMWithOAuthForTenant.GET(web.ServiceInstancesURL + "/jkljlj" + web.ParametersURL).Expect().
							Status(http.StatusNotFound)
					})
				})

				When("service instance exists", func() {
					var instanceName string
					var serviceID string
					JustBeforeEach(func() {
						Expect(serviceID).ToNot(BeEmpty())
						planId := findPlanIDForCatalogID(ctx, brokerID, serviceID, plan1CatalogID)
						EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, planId, TenantIDValue)
						postInstanceRequest["service_plan_id"] = planId
						resp := createInstance(ctx.SMWithOAuthForTenant, "false", http.StatusCreated)
						instanceName = resp.JSON().Object().Value("name").String().Raw()
						Expect(instanceName).ToNot(BeEmpty())

					})
					Describe("not retrievable service instances", func() {
						BeforeEach(func() {
							serviceID = notRetrievableService
						})

						It("Should return an error", func() {
							ctx.SMWithOAuthForTenant.GET(web.ServiceInstancesURL + "/" + instanceID + web.ParametersURL).Expect().
								Status(http.StatusBadRequest)
						})

					})
					Describe("retrievable service instances", func() {
						BeforeEach(func() {
							serviceID = service1CatalogID
							postInstanceRequest["parameters"] = map[string]string{
								"cat": "Freddy",
								"dog": "Lucy",
							}

						})
						When("async operations is requested", func() {
							It("Should return an error", func() {
								url := web.ServiceInstancesURL + "/" + instanceID + web.ParametersURL
								ctx.SMWithOAuthForTenant.GET(url).WithQuery("async", true).Expect().
									Status(http.StatusBadRequest)
							})
						})
						When("parameters are not readable", func() {
							BeforeEach(func() {
								brokerServer.ServiceInstanceHandlerFunc(http.MethodGet, http.MethodGet+"1", ParameterizedHandler(http.StatusOK, Object{
									"parameters":    "mayamayamay:s",
									"dashboard_url": "http://dashboard.com",
								}))
							})
							It("should return an error", func() {
								ctx.SMWithOAuthForTenant.GET(web.ServiceInstancesURL + "/" + instanceID + web.ParametersURL).Expect().
									Status(http.StatusBadGateway)
							})
						})
						When("parameters are valid", func() {
							BeforeEach(func() {
								brokerServer.ServiceInstanceHandlerFunc(http.MethodGet, http.MethodGet+"1", ParameterizedHandler(http.StatusOK, Object{
									"parameters": map[string]string{
										"cat": "Freddy",
										"dog": "Lucy",
									},
									"dashboard_url": "http://dashboard.com",
								}))
							})

							It("should return parameters", func() {
								response := ctx.SMWithOAuthForTenant.GET(web.ServiceInstancesURL + "/" + instanceID + web.ParametersURL).Expect()
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
				var instanceName string

				When("service instance contains tenant identifier in OSB context", func() {
					BeforeEach(func() {
						EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, servicePlanID, TenantIDValue)
						resp := createInstance(ctx.SMWithOAuthForTenant, "", http.StatusCreated)
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

					It("returns OSB context with tenant as part of the instance using json query", func() {
						res := ctx.SMWithOAuthForTenant.GET(web.ServiceInstancesURL).
							WithQuery("fieldQuery", fmt.Sprintf("context/instance_name eq '%s' and context/platform eq '%s'", instanceName, types.SMPlatform)).
							Expect().
							Status(http.StatusOK).
							JSON().Object().Value("items").Array()
						res.Length().Equal(1)
						res.First().Object().Value("id").Equal(instanceID)
					})
				})

				When("service instance dashboard_url is not set", func() {
					BeforeEach(func() {
						postInstanceRequest["dashboard_url"] = ""
						EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, postInstanceRequest["service_plan_id"].(string), TenantIDValue)
						createInstance(ctx.SMWithOAuthForTenant, "false", http.StatusCreated)
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
					Context(fmt.Sprintf("async = '%s'", testCase.async), func() {
						When("content type is not JSON", func() {
							It("returns 415", func() {
								ctx.SMWithOAuth.POST(web.ServiceInstancesURL).
									WithQuery("async", testCase.async == "true").
									WithText("text").
									Expect().
									Status(http.StatusUnsupportedMediaType).
									JSON().Object().
									Keys().Contains("error", "description")
							})
						})

						When("Create service instance sm as a platform tls broker", func() {
							It("returns 202", func() {
								EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, servicePlanIDWithTLS, TenantIDValue)
								ctx.SMWithOAuthForTenant.POST(web.ServiceInstancesURL).
									WithQuery("async", true).
									WithJSON(postInstanceRequestTLS).
									Expect().Status(http.StatusAccepted)
							})

							It("returns 201", func() {
								EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, servicePlanIDWithTLS, TenantIDValue)
								ctx.SMWithOAuthForTenant.POST(web.ServiceInstancesURL).
									WithQuery("async", false).
									WithJSON(postInstanceRequestTLS).
									Expect().Status(http.StatusCreated)
							})
						})

						When("request body is not a valid JSON", func() {
							It("returns 400", func() {
								ctx.SMWithOAuth.POST(web.ServiceInstancesURL).
									WithQuery("async", testCase.async == "true").
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
								ctx.SMWithOAuthForTenant.POST(web.ServiceInstancesURL).
									WithQuery("async", testCase.async == "true").
									WithHeader("Content-Type", "application/json").
									WithBytes([]byte(fmt.Sprintf(`{
										"name": "test-instance-name",
										"service_plan_id": "%s",
										"maintenance_info": {},
										"labels": {
											"%s":["test-tenant"]
										}
									}`, servicePlanID, TenantIdentifier))).
									Expect().
									Status(http.StatusBadRequest).
									JSON().Object().
									Keys().Contains("error", "description")
							})

							Context("when request body contains multiple label objects", func() {
								It("returns 400", func() {
									ctx.SMWithOAuthForTenant.POST(web.ServiceInstancesURL).
										WithQuery("async", testCase.async == "true").
										WithHeader("Content-Type", "application/json").
										WithBytes([]byte(fmt.Sprintf(`{
										"name": "test-instance-name",
										"service_plan_id": "%s",
										"maintenance_info": {},
										"labels": {},
										"labels": {
											"%s":["test-tenant"]
										}
									}`, servicePlanID, TenantIdentifier))).
										Expect().
										Status(http.StatusBadRequest).
										JSON().Object().Value("description").String().Contains("invalid json: duplicate key labels")
								})
							})
						})

						When("a request body field is missing", func() {
							assertPOSTWhenFieldIsMissing := func(field string, expectedStatusCode int) {
								var servicePlanID string
								BeforeEach(func() {
									servicePlanID = postInstanceRequest["service_plan_id"].(string)
									delete(postInstanceRequest, field)
								})

								It("returns 4xx", func() {
									EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, servicePlanID, TenantIDValue)
									ctx.SMWithOAuthForTenant.POST(web.ServiceInstancesURL).
										WithJSON(postInstanceRequest).
										WithQuery("async", testCase.async == "true").
										Expect().
										Status(expectedStatusCode).
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

									VerifyResourceExists(ctx.SMWithOAuthForTenant, ResourceExpectations{
										ID:    instanceID,
										Type:  types.ServiceInstanceType,
										Ready: true,
									})
								})
							}

							Context("when id field is missing", func() {
								assertPOSTReturns201WhenFieldIsMissing("id")
							})

							Context("when name field is missing", func() {
								assertPOSTWhenFieldIsMissing("name", http.StatusBadRequest)
							})

							Context("when service_plan_id field is missing", func() {
								assertPOSTWhenFieldIsMissing("service_plan_id", http.StatusBadRequest)
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
									ctx.SMWithOAuthForTenant.POST(web.ServiceInstancesURL).
										WithJSON(postInstanceRequest).
										WithQuery("async", testCase.async).
										Expect().Status(http.StatusBadRequest).JSON().Object().Value("error").Equal("InvalidTransfer")
								})
							})

							Context("which is service-manager platform", func() {
								It(fmt.Sprintf("should return %d", testCase.expectedCreateSuccessStatusCode), func() {
									postInstanceRequest["platform_id"] = types.SMPlatform
									EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, postInstanceRequest["service_plan_id"].(string), TenantIDValue)
									createInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedCreateSuccessStatusCode)
								})
							})
						})

						Context("OSB context", func() {
							BeforeEach(func() {
								brokerServer.ServiceInstanceHandlerFunc(http.MethodPut, http.MethodPut+"1", checkInstance)
								brokerServerWithTLS.ServiceInstanceHandlerFunc(http.MethodPut, http.MethodPut+"1", checkInstance)
							})

							It("enriches the osb context with the tenant and sm platform", func() {
								EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, postInstanceRequest["service_plan_id"].(string), TenantIDValue)
								createInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedCreateSuccessStatusCode)
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
								It(fmt.Sprintf("for tenant returns %d", testCase.expectedCreateSuccessStatusCode), func() {
									EnsurePublicPlanVisibility(ctx.SMRepository, servicePlanID)
									createInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedCreateSuccessStatusCode)
								})
							})

							When("plan has public visibility and support specific platform", func() {
								It(fmt.Sprintf("for tenant returns %d", testCase.expectedCreateSuccessStatusCode), func() {
									EnsurePublicPlanVisibilityForPlatform(ctx.SMRepository, servicePlanID, types.SMPlatform)
									createInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedCreateSuccessStatusCode)
								})
							})

							When("creating instance with same name", func() {
								BeforeEach(func() {
									EnsurePublicPlanVisibility(ctx.SMRepository, servicePlanID)
									postInstanceRequest["name"] = "same-instance-name"
									resp := createInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedCreateSuccessStatusCode)

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
								})

								When("for the same tenant", func() {
									It("should reject", func() {
										statusCode := http.StatusAccepted
										if testCase.async == "false" || testCase.async == "" {
											statusCode = http.StatusConflict
										}

										resp := createInstance(ctx.SMWithOAuthForTenant, testCase.async, statusCode)

										instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
											Category:          types.CREATE,
											State:             types.FAILED,
											ResourceType:      types.ServiceInstanceType,
											Reschedulable:     false,
											DeletionScheduled: false,
										})

										VerifyResourceDoesNotExist(ctx.SMWithOAuthForTenant, ResourceExpectations{
											ID:   instanceID,
											Type: types.ServiceInstanceType,
										})
									})
								})

								When("for other tenant", func() {
									It("should accept", func() {
										otherTenantExpect := ctx.NewTenantExpect("tenancyClient", "other-tenant")
										resp := createInstance(otherTenantExpect, testCase.async, testCase.expectedCreateSuccessStatusCode)

										instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
											Category:          types.CREATE,
											State:             types.SUCCEEDED,
											ResourceType:      types.ServiceInstanceType,
											Reschedulable:     false,
											DeletionScheduled: false,
										})

										VerifyResourceExists(otherTenantExpect, ResourceExpectations{
											ID:    instanceID,
											Type:  types.ServiceInstanceType,
											Ready: true,
										})
									})
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

									resp := createInstance(ctx.SMWithOAuthForTenant, "true", http.StatusAccepted)

									instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.CREATE,
										State:             types.IN_PROGRESS,
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

								AfterEach(func() {
									close(doneChannel)
								})

								It("updates fail with operation in progress", func() {
									ctx.SMWithOAuthForTenant.PATCH(web.ServiceInstancesURL+"/"+instanceID).WithQuery("async", testCase.async == "true").WithJSON(Object{}).
										Expect().Status(http.StatusUnprocessableEntity)
								})

								It("deletes succeed", func() {
									resp := ctx.SMWithOAuthForTenant.DELETE(web.ServiceInstancesURL+"/"+instanceID).WithQuery("async", testCase.async == "true").
										Expect().StatusRange(httpexpect.Status2xx)

									instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.DELETE,
										State:             types.SUCCEEDED,
										ResourceType:      types.ServiceInstanceType,
										Reschedulable:     false,
										DeletionScheduled: false,
									})

									VerifyResourceDoesNotExist(ctx.SMWithOAuthForTenant, ResourceExpectations{
										ID:   instanceID,
										Type: types.ServiceInstanceType,
									})
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

									VerifyResourceExists(ctx.SMWithOAuthForTenant, ResourceExpectations{
										ID:    instanceID,
										Type:  types.ServiceInstanceType,
										Ready: true,
									})
								})
							})

							When("broker responds with asynchronous success", func() {
								BeforeEach(func() {
									brokerServer.ServiceInstanceHandlerFunc(http.MethodPut, http.MethodPut+"1", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
									brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodPut+"1", MultiplePollsRequiredHandler("in progress", "succeeded"))
								})

								It("polling broker last operation until operation succeeds and eventually marks operation as success", func() {
									resp := createInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.responseByBrokerOrClientMode(testCase.expectedCreateSuccessStatusCode, http.StatusAccepted))

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
								})

								When("maximum polling duration is reached while polling", func() {
									var newCtx *TestContext

									BeforeEach(func() {
										preparePrerequisitesWithMaxPollingDuration(MaximumPollingDuration)
										EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, servicePlanID, TenantIDValue)

										newCtx = t.ContextBuilder.WithEnvPreExtensions(func(set *pflag.FlagSet) {
											Expect(set.Set("operations.action_timeout", ((MaximumPollingDuration + 1) * time.Second).String())).ToNot(HaveOccurred())
										}).BuildWithoutCleanup()

										brokerServer.ServiceInstanceHandlerFunc(http.MethodPut, http.MethodPut+"1", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
										brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodPut+"1", ParameterizedHandler(http.StatusOK, Object{"state": "in progress"}))
									})

									AfterEach(func() {
										newCtx.CleanupAll(false)
									})

									When("orphan mitigation deprovision synchronously succeeds", func() {
										BeforeEach(func() {
											brokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"3", ParameterizedHandler(http.StatusOK, Object{"async": false}))
										})

										It("verifies the instance and marks the operation that triggered the orphan mitigation as failed with no deletion scheduled and not reschedulable", func() {
											resp := createInstance(newCtx.SMWithOAuthForTenant, testCase.async, testCase.responseByBrokerOrClientMode(testCase.expectedBrokerFailureStatusCode, http.StatusAccepted))

											instanceID, _ = VerifyOperationExists(newCtx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.CREATE,
												State:             types.FAILED,
												ResourceType:      types.ServiceInstanceType,
												Reschedulable:     false,
												DeletionScheduled: false,
											})

											VerifyResource(ctx.SMWithOAuthForTenant, ResourceExpectations{
												ID:   instanceID,
												Type: types.ServiceInstanceType,
											}, testCase.async, true)
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
											resp := createInstance(newCtx.SMWithOAuthForTenant, testCase.async, testCase.responseByBrokerOrClientMode(testCase.expectedBrokerFailureStatusCode, http.StatusAccepted))

											instanceID, _ = VerifyOperationExists(newCtx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.CREATE,
												State:             types.FAILED,
												ResourceType:      types.ServiceInstanceType,
												Reschedulable:     false,
												DeletionScheduled: true,
											})

											VerifyResource(ctx.SMWithOAuthForTenant, ResourceExpectations{
												ID:    instanceID,
												Type:  types.ServiceInstanceType,
												Ready: false,
											}, testCase.async, true)
										})
									})

									When("orphan mitigation deprovision asynchronously succeeds", func() {
										BeforeEach(func() {
											brokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"3", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
											brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodDelete+"3", ParameterizedHandler(http.StatusOK, Object{"state": "succeeded"}))
										})

										It("keeps the instance and marks the operation that triggered the orphan mitigation as failed with no deletion scheduled and not reschedulable", func() {

											resp := createInstance(newCtx.SMWithOAuthForTenant, testCase.async, testCase.responseByBrokerOrClientMode(testCase.expectedBrokerFailureStatusCode, http.StatusAccepted))

											instanceID, _ = VerifyOperationExists(newCtx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.CREATE,
												State:             types.FAILED,
												ResourceType:      types.ServiceInstanceType,
												Reschedulable:     false,
												DeletionScheduled: false,
											})

											VerifyResource(ctx.SMWithOAuthForTenant, ResourceExpectations{
												ID:   instanceID,
												Type: types.ServiceInstanceType,
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

											brokerServer.ServiceInstanceHandlerFunc(http.MethodPut, http.MethodPut+"1", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
											brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodPut+"1", ParameterizedHandler(http.StatusOK, Object{"state": "in progress"}))
										})

										AfterEach(func() {
											newCtx.CleanupAll(false)
										})

										It("stores instance as ready false and the operation as reschedulable in progress", func() {
											resp := createInstance(newCtx.SMWithOAuthForTenant, "true", http.StatusAccepted)

											instanceID, _ = VerifyOperationExists(newCtx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.CREATE,
												State:             types.IN_PROGRESS,
												ResourceType:      types.ServiceInstanceType,
												Reschedulable:     true,
												DeletionScheduled: false,
											})

											VerifyResourceExists(newCtx.SMWithOAuthForTenant, ResourceExpectations{
												ID:    instanceID,
												Type:  types.ServiceInstanceType,
												Ready: false,
											})
										})
									})

									When("SM crashes while polling", func() {
										var newSMCtx *TestContext
										var isProvisioned atomic.Value

										BeforeEach(func() {
											newSMCtx = t.ContextBuilder.WithEnvPostExtensions(func(e env.Environment, servers map[string]FakeServer) {
												e.Set("server.shutdown_timeout", 1*time.Second)
												e.Set("operations.maintainer_retry_interval", 1*time.Second)
											}).BuildWithoutCleanup()

											brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodPut+"1", func(_ *http.Request) (int, map[string]interface{}) {
												if isProvisioned.Load() != nil {
													return http.StatusOK, Object{"state": types.SUCCEEDED}
												} else {
													return http.StatusOK, Object{"state": types.IN_PROGRESS}
												}
											})
										})

										AfterEach(func() {
											newSMCtx.CleanupAll(false)
										})

										It("should restart polling through maintainer and eventually instance is set to ready", func() {
											resp := createInstance(newSMCtx.SMWithOAuthForTenant, testCase.async, testCase.expectedCreateSuccessStatusCode)

											operationExpectation := OperationExpectations{
												Category:          types.CREATE,
												State:             types.IN_PROGRESS,
												ResourceType:      types.ServiceInstanceType,
												Reschedulable:     true,
												DeletionScheduled: false,
											}

											instanceID, _ = VerifyOperationExists(newSMCtx, resp.Header("Location").Raw(), operationExpectation)
											VerifyResourceExists(newSMCtx.SMWithOAuthForTenant, ResourceExpectations{
												ID:    instanceID,
												Type:  types.ServiceInstanceType,
												Ready: false,
											})

											newSMCtx.CleanupAll(false)
											isProvisioned.Store(true)

											newSMCtx = t.ContextBuilder.WithEnvPostExtensions(func(e env.Environment, servers map[string]FakeServer) {
												e.Set("operations.action_timeout", 2*time.Second)
											}).BuildWithoutCleanup()

											operationExpectation.State = types.SUCCEEDED
											operationExpectation.Reschedulable = false

											instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), operationExpectation)
											VerifyResourceExists(ctx.SMWithOAuthForTenant, ResourceExpectations{
												ID:    instanceID,
												Type:  types.ServiceInstanceType,
												Ready: true,
											})
										})
									})
								}

								When("polling responds with unexpected state and eventually with success state", func() {
									BeforeEach(func() {
										brokerServer.ServiceInstanceHandlerFunc(http.MethodPut, http.MethodPut+"1", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
										brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodPut+"1", MultiplePollsRequiredHandler("unknown", "succeeded"))
									})

									It("keeps polling and eventually updates the instance to ready true and operation to success", func() {
										resp := createInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.responseByBrokerOrClientMode(testCase.expectedCreateSuccessStatusCode, http.StatusAccepted))

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
										It("verifies the instance and marks the operation that triggered the orphan mitigation as failed with no deletion scheduled and not reschedulable", func() {
											resp := createInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.responseByBrokerOrClientMode(testCase.expectedBrokerFailureStatusCode, http.StatusAccepted))

											instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.CREATE,
												State:             types.FAILED,
												ResourceType:      types.ServiceInstanceType,
												Reschedulable:     false,
												DeletionScheduled: false,
											})

											VerifyResource(ctx.SMWithOAuthForTenant, ResourceExpectations{
												ID:   instanceID,
												Type: types.ServiceInstanceType,
											}, testCase.async, true)
										})
									})

									When("broker orphan mitigation deprovision synchronously fails with an unexpected status code", func() {
										BeforeEach(func() {
											brokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"3", ParameterizedHandler(http.StatusBadRequest, Object{"error": "error"}))
										})

										AfterEach(func() {
											brokerServer.ResetHandlers()
										})

										It("verifies the instance with ready false and marks the operation with deletion scheduled", func() {
											resp := createInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.responseByBrokerOrClientMode(testCase.expectedBrokerFailureStatusCode, http.StatusAccepted))

											instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.CREATE,
												State:             types.FAILED,
												ResourceType:      types.ServiceInstanceType,
												Reschedulable:     false,
												DeletionScheduled: true,
											})

											VerifyResource(ctx.SMWithOAuthForTenant, ResourceExpectations{
												ID:   instanceID,
												Type: types.ServiceInstanceType,
											}, testCase.async, true)
										})
									})

									When("broker orphan mitigation deprovision synchronously fails with an error that will continue further orphan mitigation and eventually succeed", func() {
										BeforeEach(func() {
											brokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"3", MultipleErrorsBeforeSuccessHandler(
												http.StatusInternalServerError, http.StatusOK,
												Object{"error": "error"}, Object{"async": false},
											))
										})

										It("verifies the instance and marks the operation that triggered the orphan mitigation as failed with no deletion scheduled and not reschedulable", func() {
											resp := createInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.responseByBrokerOrClientMode(testCase.expectedBrokerFailureStatusCode, http.StatusAccepted))

											instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.CREATE,
												State:             types.FAILED,
												ResourceType:      types.ServiceInstanceType,
												Reschedulable:     false,
												DeletionScheduled: false,
											})

											VerifyResource(ctx.SMWithOAuthForTenant, ResourceExpectations{
												ID:   instanceID,
												Type: types.ServiceInstanceType,
											}, testCase.async, true)
										})
									})
								})

								When("polling returns an unexpected status code", func() {
									BeforeEach(func() {
										brokerServer.ServiceInstanceHandlerFunc(http.MethodPut, http.MethodPut+"3", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
										brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodPut+"3", ParameterizedHandler(http.StatusInternalServerError, Object{"error": "error"}))
									})

									It("stores the instance as ready false and marks the operation as reschedulable", func() {
										resp := createInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.responseByBrokerOrClientMode(testCase.expectedBrokerFailureStatusCode, http.StatusAccepted))

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
								})

								When("broker unavailable during polling", func() {

									It("polling proceeds until success on 500", func() {
										brokerServer.ServiceInstanceHandlerFunc(http.MethodPut, http.MethodPut+"3", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
										brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodPut+"3", MultipleErrorsBeforeSuccessHandler(
											http.StatusServiceUnavailable, http.StatusOK,
											Object{"error": "error"}, Object{"state": "succeeded"},
										))

										resp := createInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.responseByBrokerOrClientMode(testCase.expectedCreateSuccessStatusCode, http.StatusAccepted))

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
									})
									It("polling proceeds until success on 404", func() {
										brokerServer.ServiceInstanceHandlerFunc(http.MethodPut, http.MethodPut+"3", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
										brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodPut+"3", MultipleErrorsBeforeSuccessHandler(
											http.StatusNotFound, http.StatusOK,
											Object{"error": "error"}, Object{"state": "succeeded"},
										))

										resp := createInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.responseByBrokerOrClientMode(testCase.expectedCreateSuccessStatusCode, http.StatusAccepted))

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

										brokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"3", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
										brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodDelete+"3", func(_ *http.Request) (int, map[string]interface{}) {
											return http.StatusOK, Object{"state": "succeeded"}
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
											query.ByField(query.EqualsOperator, "resource_type", string(types.ServiceInstanceType)),
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

										createInstance(newSMCtx.SMWithOAuthForTenant, testCase.async, testCase.expectedSMCrashStatusCode)
										operation := <-opChan

										VerifyResourceDoesNotExist(ctx.SMWithOAuthForTenant, ResourceExpectations{
											ID:   operation.ResourceID,
											Type: types.ServiceInstanceType,
										})

										anotherSMCtx = t.ContextBuilder.WithEnvPostExtensions(func(e env.Environment, servers map[string]FakeServer) {
											e.Set("operations.action_timeout", 2*time.Second)
											e.Set("operations.cleanup_interval", 2*time.Second)
										}).BuildWithoutCleanup()

										operationExpectation := OperationExpectations{
											Category:          types.CREATE,
											State:             types.FAILED,
											ResourceType:      types.ServiceInstanceType,
											Reschedulable:     false,
											DeletionScheduled: false,
										}

										instanceID, _ = VerifyOperationExists(ctx, fmt.Sprintf("%s/%s%s/%s", web.ServiceInstancesURL, operation.ResourceID, web.ResourceOperationsURL, operation.ID), operationExpectation)
										VerifyResourceDoesNotExist(ctx.SMWithOAuthForTenant, ResourceExpectations{
											ID:   instanceID,
											Type: types.ServiceInstanceType,
										})
									})
								})
							}

							When("provision responds with error due to stopped broker", func() {
								BeforeEach(func() {
									brokerServer.Close()
									delete(ctx.Servers, BrokerServerPrefix+brokerID)
								})

								It("verifies instance in SMDB and marks operation with failed", func() {
									resp := createInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)

									instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.CREATE,
										State:             types.FAILED,
										ResourceType:      types.ServiceInstanceType,
										Reschedulable:     false,
										DeletionScheduled: false,
									})

									VerifyResource(ctx.SMWithOAuthForTenant, ResourceExpectations{
										ID:   instanceID,
										Type: types.ServiceInstanceType,
									}, testCase.async, false)
								})
							})

							When("provision responds with error that does not require orphan mitigation", func() {
								BeforeEach(func() {
									brokerServer.ServiceInstanceHandlerFunc(http.MethodPut, http.MethodPut+"3", ParameterizedHandler(http.StatusBadRequest, Object{"error": "error"}))
								})

								It("verifies the instance and marks the operation as failed, non rescheduable with empty deletion scheduled", func() {
									resp := createInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)

									instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.CREATE,
										State:             types.FAILED,
										ResourceType:      types.ServiceInstanceType,
										Reschedulable:     false,
										DeletionScheduled: false,
									})

									VerifyResource(ctx.SMWithOAuthForTenant, ResourceExpectations{
										ID:   instanceID,
										Type: types.ServiceInstanceType,
									}, testCase.async, false)
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
										resp := createInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)

										instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
											Category:          types.CREATE,
											State:             types.FAILED,
											ResourceType:      types.ServiceInstanceType,
											Reschedulable:     false,
											DeletionScheduled: false,
										})

										VerifyResource(ctx.SMWithOAuthForTenant, ResourceExpectations{
											ID:   instanceID,
											Type: types.ServiceInstanceType,
										}, testCase.async, false)
									})

									When("maximum deletion timeout has been reached", func() {
										var newCtx *TestContext

										BeforeEach(func() {
											newCtx = t.ContextBuilder.WithEnvPreExtensions(func(set *pflag.FlagSet) {
												Expect(set.Set("operations.reconciliation_operation_timeout", (2 * time.Millisecond).String())).ToNot(HaveOccurred())
											}).BuildWithoutCleanup()
										})

										AfterEach(func() {
											newCtx.CleanupAll(false)
										})

										It("verifies the instance as ready false and marks the operation as deletion scheduled", func() {
											resp := createInstance(newCtx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)

											instanceID, _ = VerifyOperationExists(newCtx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.CREATE,
												State:             types.FAILED,
												ResourceType:      types.ServiceInstanceType,
												Reschedulable:     false,
												DeletionScheduled: true,
											})

											VerifyResource(ctx.SMWithOAuthForTenant, ResourceExpectations{
												ID:    instanceID,
												Ready: false,
												Type:  types.ServiceInstanceType,
											}, testCase.async, false)
										})
									})
								})

								if testCase.async == "true" {
									When("broker orphan mitigation deprovision asynchronously keeps failing with an error while polling", func() {
										BeforeEach(func() {
											brokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"3", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
											brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodDelete+"3", ParameterizedHandler(http.StatusBadRequest, Object{"error": "error"}))
										})

										It("keeps the instance as ready false and marks the operation as deletion scheduled", func() {
											resp := createInstance(ctx.SMWithOAuthForTenant, "true", http.StatusAccepted)

											instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.CREATE,
												State:             types.FAILED,
												ResourceType:      types.ServiceInstanceType,
												Reschedulable:     true,
												DeletionScheduled: true,
											})

											VerifyResourceExists(ctx.SMWithOAuthForTenant, ResourceExpectations{
												ID:    instanceID,
												Type:  types.ServiceInstanceType,
												Ready: false,
											})
										})
									})
								}

								When("SM crashes while orphan mitigating", func() {
									var newSMCtx *TestContext
									var isDeprovisioned atomic.Value

									BeforeEach(func() {
										newSMCtx = t.ContextBuilder.WithEnvPostExtensions(func(e env.Environment, servers map[string]FakeServer) {
											e.Set("server.shutdown_timeout", 1*time.Second)
											e.Set("operations.maintainer_retry_interval", 1*time.Second)
										}).BuildWithoutCleanup()

										brokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"3", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
										brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodDelete+"3", func(_ *http.Request) (int, map[string]interface{}) {
											if isDeprovisioned.Load() != nil {
												return http.StatusOK, Object{"state": "succeeded"}
											} else {
												return http.StatusOK, Object{"state": "in progress"}
											}
										})
									})

									AfterEach(func() {
										newSMCtx.CleanupAll(false)
									})

									It("should restart orphan mitigation through maintainer and eventually succeeds", func() {
										resp := createInstance(newSMCtx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)

										operationExpectations := OperationExpectations{
											Category:          types.CREATE,
											State:             types.FAILED,
											ResourceType:      types.ServiceInstanceType,
											Reschedulable:     true,
											DeletionScheduled: true,
										}

										instanceID, _ = VerifyOperationExists(newSMCtx, resp.Header("Location").Raw(), operationExpectations)

										newSMCtx.CleanupAll(false)
										isDeprovisioned.Store(true)

										newSMCtx = t.ContextBuilder.WithEnvPostExtensions(func(e env.Environment, servers map[string]FakeServer) {
											e.Set("operations.action_timeout", 1*time.Second)
										}).BuildWithoutCleanup()

										operationExpectations.DeletionScheduled = false
										operationExpectations.Reschedulable = false
										instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), operationExpectations)
										VerifyResource(ctx.SMWithOAuthForTenant, ResourceExpectations{
											ID:   instanceID,
											Type: types.ServiceInstanceType,
										}, testCase.async, false)
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
										resp := createInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)

										instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
											Category:          types.CREATE,
											State:             types.FAILED,
											ResourceType:      types.ServiceInstanceType,
											Reschedulable:     false,
											DeletionScheduled: false,
											Error:             "Failed provisioning request context",
										})
										VerifyResource(ctx.SMWithOAuthForTenant, ResourceExpectations{
											ID:   instanceID,
											Type: types.ServiceInstanceType,
										}, testCase.async, false)
									})
								})
							})

							When("provision responds with error due to time out", func() {
								var doneChannel chan interface{}
								var newCtx *TestContext

								BeforeEach(func() {
									doneChannel = make(chan interface{})
									newCtx = t.ContextBuilder.WithEnvPreExtensions(func(set *pflag.FlagSet) {
										Expect(set.Set("httpclient.timeout", (2 * time.Second).String())).ToNot(HaveOccurred())
									}).BuildWithoutCleanup()

									brokerServer.ServiceInstanceHandlerFunc(http.MethodPut, http.MethodPut+"1", DelayingHandler(doneChannel))
									brokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"1", ParameterizedHandler(http.StatusOK, Object{}))
								})

								AfterEach(func() {
									newCtx.CleanupAll(false)
								})

								It("orphan mitigates the instance", func() {
									resp := createInstance(newCtx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)
									<-time.After(2100 * time.Millisecond)
									close(doneChannel)
									instanceID, _ = VerifyOperationExists(newCtx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.CREATE,
										State:             types.FAILED,
										ResourceType:      types.ServiceInstanceType,
										Reschedulable:     false,
										DeletionScheduled: false,
									})

									VerifyResource(ctx.SMWithOAuthForTenant, ResourceExpectations{
										ID:   instanceID,
										Type: types.ServiceInstanceType,
									}, testCase.async, false)
								})
							})
						})
					})
				}
			})

			Describe("PATCH", func() {
				for _, testCase := range testCases {
					testCase := testCase
					Context(fmt.Sprintf("async = %s", testCase.async), func() {
						When("instance is missing", func() {
							It("returns 404", func() {
								ctx.SMWithOAuthForTenant.PATCH(web.ServiceInstancesURL+"/no_such_id").
									WithQuery("async", testCase.async == "true").
									WithJSON(postInstanceRequest).
									Expect().Status(http.StatusNotFound).
									JSON().Object().
									Keys().Contains("error", "description")
							})
						})

						When("instance exists in a platform different from service manager", func() {
							const (
								brokerAPIVersionHeaderKey   = "X-Broker-API-Version"
								brokerAPIVersionHeaderValue = "2.13"
								SID                         = "abc1234"
							)

							var serviceID string
							var planID string
							var testCtx *TestContext

							BeforeEach(func() {
								serviceID = ""
								planID = ""
								testCtx = ctx
								brokerServer.ServiceInstanceHandlerFunc(http.MethodPut, http.MethodPut, verificationHandler(map[string]string{
									"context." + TenantIdentifier: TenantIDValue,
								}, http.StatusCreated))
								brokerServer.BindingHandlerFunc(http.MethodPut, http.MethodPut, verificationHandler(map[string]string{
									"context." + TenantIdentifier: TenantIDValue,
								}, http.StatusCreated))
								brokerServer.ServiceInstanceHandlerFunc(http.MethodPatch, http.MethodPatch, verificationHandler(map[string]string{
									"context." + TenantIdentifier: TenantIDValue,
								}, http.StatusOK))
							})

							JustBeforeEach(func() {
								Expect(serviceID).ToNot(BeEmpty())
								Expect(planID).ToNot(BeEmpty())
								EnsurePlanVisibility(testCtx.SMRepository, TenantIdentifier, testCtx.TestPlatform.ID, findPlanIDForCatalogID(testCtx, brokerID, serviceID, planID), TenantIDValue)

								testCtx.SMWithBasic.PUT("/v1/osb/"+brokerID+"/v2/service_instances/"+SID).
									WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
									WithJSON(Object{
										"service_id": serviceID,
										"plan_id":    planID,
										"context": Object{
											TenantIdentifier: TenantIDValue,
										},
									}).
									Expect().Status(http.StatusCreated)

								testCtx.SMWithOAuth.GET(web.ServiceInstancesURL + "/" + SID).
									Expect().
									Status(http.StatusOK).
									JSON().Object().Value("platform_id").Equal(testCtx.TestPlatform.ID)
							})

							When("platform_id provided in request body", func() {
								BeforeEach(func() {
									serviceID = service1CatalogID
									planID = plan1CatalogID
								})

								When("transfer instance is disabled", func() {
									It("should return 400", func() {
										testCtx.SMWithOAuthForTenant.PATCH(web.ServiceInstancesURL+"/"+SID).
											WithQuery("async", testCase.async == "true").
											WithJSON(Object{"platform_id": "service-manager"}).
											Expect().Status(http.StatusBadRequest).
											JSON().Object().Value("error").Equal("TransferDisabled")

										objAfterOp := VerifyResourceExists(testCtx.SMWithOAuthForTenant, ResourceExpectations{
											ID:    SID,
											Type:  types.ServiceInstanceType,
											Ready: true,
										})

										objAfterOp.Value("platform_id").Equal(testCtx.TestPlatform.ID)
									})
								})

								When("transfer instance is enabled", func() {
									BeforeEach(func() {
										testCtx = t.ContextBuilder.WithEnvPreExtensions(func(set *pflag.FlagSet) {
											Expect(set.Set("api.enable_instance_transfer", "true")).ToNot(HaveOccurred())
										}).WithBasicAuthPlatformName("inner-testCtx-basic-credentials").BuildWithCleanup(false)
									})

									AfterEach(func() {
										testCtx.CleanupAll(false)
									})

									Context("which is not service-manager platform", func() {
										It("should return 400", func() {
											testCtx.SMWithOAuthForTenant.PATCH(web.ServiceInstancesURL+"/"+SID).
												WithQuery("async", testCase.async == "true").
												WithJSON(Object{"platform_id": "another-platform-id"}).
												Expect().Status(http.StatusBadRequest)

											objAfterOp := VerifyResourceExists(testCtx.SMWithOAuthForTenant, ResourceExpectations{
												ID:    SID,
												Type:  types.ServiceInstanceType,
												Ready: true,
											})

											objAfterOp.Value("platform_id").Equal(testCtx.TestPlatform.ID)
										})
									})

									Context("which is service-manager platform", func() {
										Context("when plan does not support the platform", func() {
											BeforeEach(func() {
												serviceID = service1CatalogID
												planID = planNotSupportingSMPlatform
											})

											It("should return 400", func() {
												testCtx.SMWithOAuthForTenant.PATCH(web.ServiceInstancesURL+"/"+SID).
													WithQuery("async", testCase.async == "true").
													WithJSON(Object{"platform_id": types.SMPlatform}).
													Expect().Status(http.StatusBadRequest).
													JSON().Object().Value("error").Equal("UnsupportedPlatform")
											})
										})

										Context("when service does not support context updates", func() {
											BeforeEach(func() {
												serviceID = serviceNotSupportingContextUpdates
												planID = plan1CatalogID
											})

											It("should return 400", func() {
												testCtx.SMWithOAuthForTenant.PATCH(web.ServiceInstancesURL+"/"+SID).
													WithQuery("async", testCase.async == "true").
													WithJSON(Object{"platform_id": types.SMPlatform}).
													Expect().Status(http.StatusBadRequest).
													JSON().Object().Value("error").Equal("UnsupportedContextUpdate")
											})
										})

										Context("when plan supports the platform and service supports context updates", func() {
											BeforeEach(func() {
												serviceID = service1CatalogID
												planID = plan1CatalogID
											})

											It("should return 2xx and allow management of the transferred instance in SMaaP but not in old platform", func() {
												var bindingID string

												By("verify patch request for instance transfer to SMaaP succeeds")
												resp := testCtx.SMWithOAuthForTenant.PATCH(web.ServiceInstancesURL+"/"+SID).
													WithQuery("async", testCase.async == "true").
													WithJSON(Object{"platform_id": types.SMPlatform}).
													Expect().Status(testCase.expectedUpdateSuccessStatusCode)

												instanceID, _ = VerifyOperationExists(testCtx, resp.Header("Location").Raw(), OperationExpectations{
													Category:          types.UPDATE,
													State:             types.SUCCEEDED,
													ResourceType:      types.ServiceInstanceType,
													Reschedulable:     false,
													DeletionScheduled: false,
												})

												Expect(instanceID).To(Equal(SID))

												objAfterOp := VerifyResourceExists(testCtx.SMWithOAuthForTenant, ResourceExpectations{
													ID:    instanceID,
													Type:  types.ServiceInstanceType,
													Ready: true,
												})

												By("verify instance is transferred to SMaaP")
												objAfterOp.Value("platform_id").Equal(types.SMPlatform)

												By("verify instance updates in SMaaP still works after transfer")
												resp = testCtx.SMWithOAuthForTenant.PATCH(web.ServiceInstancesURL+"/"+SID).
													WithQuery("async", testCase.async == "true").
													WithJSON(Object{}).
													Expect().Status(testCase.expectedUpdateSuccessStatusCode)

												instanceID, _ = VerifyOperationExists(testCtx, resp.Header("Location").Raw(), OperationExpectations{
													Category:          types.UPDATE,
													State:             types.SUCCEEDED,
													ResourceType:      types.ServiceInstanceType,
													Reschedulable:     false,
													DeletionScheduled: false,
												})

												By("verify instance updates old platform does not work after transfer")
												testCtx.SMWithBasic.PATCH("/v1/osb/"+brokerID+"/v2/service_instances/"+SID).
													WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
													WithJSON(Object{
														"service_id": service1CatalogID,
														"plan_id":    plan1CatalogID,
													}).
													Expect().Status(http.StatusNotFound)

												By("verify instance binds in SMaaP still works after transfer")
												resp = testCtx.SMWithOAuthForTenant.POST(web.ServiceBindingsURL).
													WithQuery("async", testCase.async == "true").
													WithJSON(Object{
														"name":                "binding-to-transferred-instance",
														"service_instance_id": SID,
													}).
													Expect().
													Status(testCase.expectedCreateSuccessStatusCode)

												bindingID, _ = VerifyOperationExists(testCtx, resp.Header("Location").Raw(), OperationExpectations{
													Category:          types.CREATE,
													State:             types.SUCCEEDED,
													ResourceType:      types.ServiceBindingType,
													Reschedulable:     false,
													DeletionScheduled: false,
												})

												By("verify instance unbind in SMaaP still works after transfer")
												resp = testCtx.SMWithOAuthForTenant.DELETE(web.ServiceBindingsURL+"/"+bindingID).
													WithQuery("async", testCase.async == "true").
													Expect().
													Status(testCase.expectedDeleteSuccessStatusCode)

												VerifyOperationExists(testCtx, resp.Header("Location").Raw(), OperationExpectations{
													Category:          types.DELETE,
													State:             types.SUCCEEDED,
													ResourceType:      types.ServiceBindingType,
													Reschedulable:     false,
													DeletionScheduled: false,
												})

												VerifyResourceDoesNotExist(testCtx.SMWithOAuthForTenant, ResourceExpectations{
													ID:   bindingID,
													Type: types.ServiceBindingType,
												})

												By("verify instance binds in old platform does not work after transfer")
												testCtx.SMWithBasic.PUT("/v1/osb/"+brokerID+"/v2/service_instances/"+SID+"/service_bindings/"+bindingID).
													WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
													WithJSON(Object{
														"service_id": service1CatalogID,
														"plan_id":    plan1CatalogID,
													}).
													Expect().Status(http.StatusNotFound)

												By("verify instance unbind in old platform does not after transfer")
												testCtx.SMWithBasic.DELETE("/v1/osb/"+brokerID+"/v2/service_instances/"+SID+"/service_bindings/"+bindingID).
													WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
													WithJSON(Object{
														"service_id": service1CatalogID,
														"plan_id":    plan1CatalogID,
													}).
													Expect().Status(http.StatusNotFound)

												By("verify instance deprovision in old platform does not after transfer")
												testCtx.SMWithBasic.DELETE("/v1/osb/"+brokerID+"/v2/service_instances/"+SID).
													WithJSON(Object{
														"service_id": service1CatalogID,
														"plan_id":    plan1CatalogID,
													}).
													WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
													Expect().Status(http.StatusNotFound)

												By("verify instance deprovision in SMaaP still works after transfer")
												resp = testCtx.SMWithOAuthForTenant.DELETE(web.ServiceInstancesURL+"/"+SID).
													WithQuery("async", testCase.async == "true").
													WithJSON(Object{}).
													Expect().Status(testCase.expectedDeleteSuccessStatusCode)

												instanceID, _ = VerifyOperationExists(testCtx, resp.Header("Location").Raw(), OperationExpectations{
													Category:          types.DELETE,
													State:             types.SUCCEEDED,
													ResourceType:      types.ServiceInstanceType,
													Reschedulable:     false,
													DeletionScheduled: false,
												})

												VerifyResourceDoesNotExist(testCtx.SMWithOAuthForTenant, ResourceExpectations{
													ID:   instanceID,
													Type: types.ServiceInstanceType,
												})
											})
										})
									})
								})
							})

							When("platform_id is not provided in request body", func() {
								BeforeEach(func() {
									serviceID = service1CatalogID
									planID = plan1CatalogID
								})

								It("returns 404", func() {
									testCtx.SMWithOAuthForTenant.PATCH(web.ServiceInstancesURL+"/"+SID).
										WithQuery("async", testCase.async == "true").
										WithJSON(Object{}).
										Expect().Status(http.StatusNotFound)
								})
							})
						})

						When("instance exists in service manager platform", func() {
							var testCtx *TestContext

							BeforeEach(func() {
								testCtx = ctx
							})

							JustBeforeEach(func() {
								EnsurePlanVisibility(testCtx.SMRepository, TenantIdentifier, types.SMPlatform, postInstanceRequest["service_plan_id"].(string), TenantIDValue)
								resp := createInstance(testCtx.SMWithOAuthForTenant, testCase.async, testCase.expectedCreateSuccessStatusCode)

								instanceID, _ = VerifyOperationExists(testCtx, resp.Header("Location").Raw(), OperationExpectations{
									Category:          types.CREATE,
									State:             types.SUCCEEDED,
									ResourceType:      types.ServiceInstanceType,
									Reschedulable:     false,
									DeletionScheduled: false,
								})

								VerifyResourceExists(testCtx.SMWithOAuthForTenant, ResourceExpectations{
									ID:    instanceID,
									Type:  types.ServiceInstanceType,
									Ready: true,
								})
							})

							When("content type is not JSON", func() {
								It("returns 415", func() {
									testCtx.SMWithOAuth.PATCH(web.ServiceInstancesURL+"/"+instanceID).
										WithQuery("async", testCase.async == "true").
										WithText("text").
										Expect().Status(http.StatusUnsupportedMediaType).
										JSON().Object().
										Keys().Contains("error", "description")
								})
							})

							When("request body is not valid JSON", func() {
								It("returns 400", func() {
									testCtx.SMWithOAuth.PATCH(web.ServiceInstancesURL+"/"+instanceID).
										WithQuery("async", testCase.async == "true").
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
									createdAt := "2015-01-01T00:00:00Z"

									resp := testCtx.SMWithOAuthForTenant.PATCH(web.ServiceInstancesURL+"/"+instanceID).
										WithJSON(Object{"created_at": createdAt}).
										WithQuery("async", testCase.async == "true").
										Expect().
										Status(testCase.expectedUpdateSuccessStatusCode)

									instanceID, _ = VerifyOperationExists(testCtx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.UPDATE,
										State:             types.SUCCEEDED,
										ResourceType:      types.ServiceInstanceType,
										Reschedulable:     false,
										DeletionScheduled: false,
									})

									objAfterUpdate := VerifyResourceExists(testCtx.SMWithOAuthForTenant, ResourceExpectations{
										ID:    instanceID,
										Type:  types.ServiceInstanceType,
										Ready: true,
									})

									objAfterUpdate.
										ContainsKey("created_at").
										ValueNotEqual("created_at", createdAt)
								})
							})

							When("platform_id provided in body", func() {
								AfterEach(func() {
									objAfterUpdate := VerifyResourceExists(ctx.SMWithOAuthForTenant, ResourceExpectations{
										ID:    instanceID,
										Type:  types.ServiceInstanceType,
										Ready: true,
									})

									objAfterUpdate.
										ContainsKey("platform_id").
										ValueEqual("platform_id", types.SMPlatform)
								})

								When("transfer instance is disabled", func() {
									It("should return 400", func() {
										testCtx.SMWithOAuthForTenant.PATCH(web.ServiceInstancesURL+"/"+instanceID).
											WithQuery("async", testCase.async == "true").
											WithJSON(Object{"platform_id": "service-manager"}).
											Expect().Status(http.StatusBadRequest).
											JSON().Object().Value("error").Equal("TransferDisabled")

										VerifyResourceExists(testCtx.SMWithOAuthForTenant, ResourceExpectations{
											ID:    instanceID,
											Type:  types.ServiceInstanceType,
											Ready: true,
										})
									})
								})

								When("transfer instance is enabled", func() {
									BeforeEach(func() {
										testCtx = t.ContextBuilder.WithEnvPreExtensions(func(set *pflag.FlagSet) {
											Expect(set.Set("api.enable_instance_transfer", "true")).ToNot(HaveOccurred())
										}).WithBasicAuthPlatformName("inner-testtestCtx-basic-credentials").BuildWithCleanup(false)
									})

									AfterEach(func() {
										testCtx.CleanupAll(false)
									})

									Context("which is not service-manager platform", func() {
										It("should return 400", func() {
											testCtx.SMWithOAuth.PATCH(web.ServiceInstancesURL+"/"+instanceID).
												WithQuery("async", testCase.async == "true").
												WithJSON(Object{"platform_id": "test-platform-id"}).
												Expect().Status(http.StatusBadRequest)
										})
									})

									Context("which is service-manager platform", func() {
										It("should return 200", func() {
											resp := testCtx.SMWithOAuthForTenant.PATCH(web.ServiceInstancesURL+"/"+instanceID).
												WithQuery("async", testCase.async == "true").
												WithJSON(Object{"platform_id": types.SMPlatform}).
												Expect().Status(testCase.expectedUpdateSuccessStatusCode)

											instanceID, _ = VerifyOperationExists(testCtx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.UPDATE,
												State:             types.SUCCEEDED,
												ResourceType:      types.ServiceInstanceType,
												Reschedulable:     false,
												DeletionScheduled: false,
											})
										})
									})
								})
							})

							When("fields are updated one by one", func() {
								It("returns 200", func() {
									for _, prop := range []string{"name", "maintenance_info", "service_plan_id"} {
										updatedBrokerJSON := Object{}
										if prop == "service_plan_id" {
											EnsurePlanVisibility(testCtx.SMRepository, TenantIdentifier, types.SMPlatform, anotherServicePlanID, TenantIDValue)
											updatedBrokerJSON[prop] = anotherServicePlanID
										} else {
											updatedBrokerJSON[prop] = "updated-" + prop
										}

										resp := testCtx.SMWithOAuthForTenant.PATCH(web.ServiceInstancesURL+"/"+instanceID).
											WithQuery("async", testCase.async == "true").
											WithJSON(updatedBrokerJSON).
											Expect().
											Status(testCase.expectedUpdateSuccessStatusCode)

										instanceID, _ = VerifyOperationExists(testCtx, resp.Header("Location").Raw(), OperationExpectations{
											Category:          types.UPDATE,
											State:             types.SUCCEEDED,
											ResourceType:      types.ServiceInstanceType,
											Reschedulable:     false,
											DeletionScheduled: false,
										})

										objAfterUpdate := VerifyResourceExists(testCtx.SMWithOAuthForTenant, ResourceExpectations{
											ID:    instanceID,
											Type:  types.ServiceInstanceType,
											Ready: true,
										})

										objAfterUpdate.
											ContainsMap(updatedBrokerJSON)
									}
								})
							})

							Context("OSB context", func() {
								BeforeEach(func() {
									brokerServer.ServiceInstanceHandlerFunc(http.MethodPatch, http.MethodPatch+"1", func(req *http.Request) (int, map[string]interface{}) {
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
									testCtx.SMWithOAuthForTenant.PATCH(web.ServiceInstancesURL+"/"+instanceID).
										WithQuery("async", testCase.async == "true").
										WithJSON(Object{}).
										Expect().Status(testCase.expectedBrokerFailureStatusCode)
								})
							})

							Context("instance visibility", func() {
								When("tenant doesn't have plan visibility", func() {
									It("returns 404", func() {
										EnsurePlanVisibilityDoesNotExist(testCtx.SMRepository, TenantIdentifier, types.SMPlatform, anotherServicePlanID, TenantIDValue)

										testCtx.SMWithOAuthForTenant.PATCH(web.ServiceInstancesURL+"/"+instanceID).
											WithQuery("async", testCase.async == "true").
											WithJSON(Object{"service_plan_id": anotherServicePlanID}).
											Expect().Status(http.StatusNotFound)
									})
								})

								When("tenant has plan visibility", func() {
									It("returns success", func() {
										EnsurePlanVisibility(testCtx.SMRepository, TenantIdentifier, types.SMPlatform, anotherServicePlanID, TenantIDValue)
										resp := testCtx.SMWithOAuthForTenant.PATCH(web.ServiceInstancesURL+"/"+instanceID).
											WithQuery("async", testCase.async == "true").
											WithJSON(Object{"service_plan_id": anotherServicePlanID}).
											Expect().Status(testCase.expectedUpdateSuccessStatusCode)

										instanceID, _ = VerifyOperationExists(testCtx, resp.Header("Location").Raw(), OperationExpectations{
											Category:          types.UPDATE,
											State:             types.SUCCEEDED,
											ResourceType:      types.ServiceInstanceType,
											Reschedulable:     false,
											DeletionScheduled: false,
										})

										objAfterUpdate := VerifyResourceExists(testCtx.SMWithOAuthForTenant, ResourceExpectations{
											ID:    instanceID,
											Type:  types.ServiceInstanceType,
											Ready: true,
										})

										objAfterUpdate.
											Value("service_plan_id").Equal(anotherServicePlanID)
									})
								})
							})

							Context("instance ownership", func() {
								When("tenant doesn't have ownership of instance", func() {
									It("returns 404", func() {
										otherTenantExpect := testCtx.NewTenantExpect("tenancyClient", "other-tenant")
										otherTenantExpect.PATCH(web.ServiceInstancesURL+"/"+instanceID).
											WithQuery("async", testCase.async == "true").
											WithJSON(Object{"service_plan_id": anotherServicePlanID}).
											Expect().Status(http.StatusNotFound)
									})
								})

								When("tenant has ownership of instance", func() {
									It("returns 200", func() {
										resp := testCtx.SMWithOAuthForTenant.PATCH(web.ServiceInstancesURL+"/"+instanceID).
											WithQuery("async", testCase.async == "true").
											WithJSON(Object{}).
											Expect().Status(testCase.expectedUpdateSuccessStatusCode)

										instanceID, _ = VerifyOperationExists(testCtx, resp.Header("Location").Raw(), OperationExpectations{
											Category:          types.UPDATE,
											State:             types.SUCCEEDED,
											ResourceType:      types.ServiceInstanceType,
											Reschedulable:     false,
											DeletionScheduled: false,
										})

										VerifyResourceExists(testCtx.SMWithOAuthForTenant, ResourceExpectations{
											ID:    instanceID,
											Type:  types.ServiceInstanceType,
											Ready: true,
										})
									})
								})
							})

							When("changing instance name to existing instance name", func() {
								Context("same tenant", func() {
									It("fails to update", func() {
										instance1ID := instanceID
										postInstanceRequest["name"] = "instance2"
										resp := createInstance(testCtx.SMWithOAuthForTenant, testCase.async, testCase.expectedCreateSuccessStatusCode)

										instance2ID, _ := VerifyOperationExists(testCtx, resp.Header("Location").Raw(), OperationExpectations{
											Category:          types.CREATE,
											State:             types.SUCCEEDED,
											ResourceType:      types.ServiceInstanceType,
											Reschedulable:     false,
											DeletionScheduled: false,
										})

										VerifyResourceExists(testCtx.SMWithOAuthForTenant, ResourceExpectations{
											ID:    instance2ID,
											Type:  types.ServiceInstanceType,
											Ready: true,
										})

										resp = testCtx.SMWithOAuthForTenant.PATCH(web.ServiceInstancesURL+"/"+instance1ID).
											WithQuery("async", false).
											WithJSON(Object{"name": "instance2"}).
											Expect()

										VerifyOperationExists(testCtx, resp.Header("Location").Raw(), OperationExpectations{
											Category:          types.UPDATE,
											State:             types.FAILED,
											ResourceType:      types.ServiceInstanceType,
											Reschedulable:     false,
											DeletionScheduled: false,
										})

										objAfterUpdate := VerifyResourceExists(testCtx.SMWithOAuthForTenant, ResourceExpectations{
											ID:    instance1ID,
											Type:  types.ServiceInstanceType,
											Ready: true,
										})

										objAfterUpdate.
											ValueNotEqual("name", "instance2")
									})
								})

								Context("different tenants", func() {
									It("succeeds to update", func() {
										EnsurePublicPlanVisibility(testCtx.SMRepository, servicePlanID)

										postInstanceRequest["name"] = "instance1"
										otherTenant := testCtx.NewTenantExpect("tenancyClient", "other-tenant")
										resp := createInstance(otherTenant, testCase.async, testCase.expectedCreateSuccessStatusCode)
										instance1ID, _ := VerifyOperationExists(testCtx, resp.Header("Location").Raw(), OperationExpectations{
											Category:          types.CREATE,
											State:             types.SUCCEEDED,
											ResourceType:      types.ServiceInstanceType,
											Reschedulable:     false,
											DeletionScheduled: false,
										})

										VerifyResourceExists(otherTenant, ResourceExpectations{
											ID:    instance1ID,
											Type:  types.ServiceInstanceType,
											Ready: true,
										})

										postInstanceRequest["name"] = "instance2"
										resp = createInstance(testCtx.SMWithOAuthForTenant, testCase.async, testCase.expectedCreateSuccessStatusCode)

										instance2ID, _ := VerifyOperationExists(testCtx, resp.Header("Location").Raw(), OperationExpectations{
											Category:          types.CREATE,
											State:             types.SUCCEEDED,
											ResourceType:      types.ServiceInstanceType,
											Reschedulable:     false,
											DeletionScheduled: false,
										})

										VerifyResourceExists(testCtx.SMWithOAuthForTenant, ResourceExpectations{
											ID:    instance2ID,
											Type:  types.ServiceInstanceType,
											Ready: true,
										})

										resp = testCtx.SMWithOAuthForTenant.PATCH(web.ServiceInstancesURL+"/"+instance2ID).
											WithQuery("async", testCase.async == "true").
											WithJSON(Object{"name": "instance1"}).
											Expect().Status(testCase.expectedUpdateSuccessStatusCode)

										instance2ID, _ = VerifyOperationExists(testCtx, resp.Header("Location").Raw(), OperationExpectations{
											Category:          types.UPDATE,
											State:             types.SUCCEEDED,
											ResourceType:      types.ServiceInstanceType,
											Reschedulable:     false,
											DeletionScheduled: false,
										})

										objAfterUpdate := VerifyResourceExists(testCtx.SMWithOAuthForTenant, ResourceExpectations{
											ID:    instance2ID,
											Type:  types.ServiceInstanceType,
											Ready: true,
										})

										objAfterUpdate.
											ValueEqual("name", "instance1")
									})
								})
							})

							Context("broker scenarios", func() {
								When("dashboard_url is changed from broker", func() {
									const updatedDashboardURL = "http://new_dashboard_url"

									BeforeEach(func() {
										brokerServer.ServiceInstanceHandlerFunc(http.MethodPatch, http.MethodPatch+"1", ParameterizedHandler(http.StatusAccepted, Object{
											"async":         true,
											"dashboard_url": updatedDashboardURL,
										}))
										brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodPatch+"1", ParameterizedHandler(http.StatusOK, Object{
											"state": "succeeded",
										}))
									})

									It("should update it", func() {
										resp := testCtx.SMWithOAuthForTenant.PATCH(web.ServiceInstancesURL+"/"+instanceID).
											WithQuery("async", testCase.async == "true").
											WithJSON(Object{}).
											Expect().
											Status(testCase.expectedUpdateSuccessStatusCode)

										instanceID, _ = VerifyOperationExists(testCtx, resp.Header("Location").Raw(), OperationExpectations{
											Category:          types.UPDATE,
											State:             types.SUCCEEDED,
											ResourceType:      types.ServiceInstanceType,
											Reschedulable:     false,
											DeletionScheduled: false,
										})

										objAfterUpdate := VerifyResourceExists(testCtx.SMWithOAuthForTenant, ResourceExpectations{
											ID:    instanceID,
											Type:  types.ServiceInstanceType,
											Ready: true,
										})

										objAfterUpdate.
											ContainsKey("dashboard_url").
											ValueEqual("dashboard_url", updatedDashboardURL)
									})
								})

								When("service plan id is updated", func() {
									It("propagates the update to the broker", func() {
										brokerServer.ServiceInstanceHandlerFunc(http.MethodPatch, http.MethodPatch+"1",
											verificationHandler(map[string]string{
												"plan_id":          anotherServicePlanCatalogID,
												"context.platform": types.SMPlatform,
											}, http.StatusOK))

										EnsurePlanVisibility(testCtx.SMRepository, TenantIdentifier, types.SMPlatform, anotherServicePlanID, TenantIDValue)

										patchInstanceRequest["service_plan_id"] = anotherServicePlanID
										patchInstance(testCtx.SMWithOAuthForTenant, testCase.async, instanceID, testCase.expectedUpdateSuccessStatusCode)
									})
								})

								When("parameters are updated", func() {
									It("propagates the update to the broker", func() {
										patchInstanceRequest["parameters"] = map[string]string{
											"newParamKey": "newParamValue",
										}
										brokerServer.ServiceInstanceHandlerFunc(http.MethodPatch, http.MethodPatch+"1",
											verificationHandler(map[string]string{
												"parameters":       `{"newParamKey":"newParamValue"}`,
												"context.platform": types.SMPlatform,
											}, http.StatusOK))

										patchInstance(testCtx.SMWithOAuthForTenant, testCase.async, instanceID, testCase.expectedUpdateSuccessStatusCode)
									})
								})

								When("an update operation is already in progress", func() {
									var doneChannel chan interface{}

									JustBeforeEach(func() {
										doneChannel = make(chan interface{})

										brokerServer.ServiceInstanceHandlerFunc(http.MethodPatch, http.MethodPatch+"1", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
										brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodPatch+"1", DelayingHandler(doneChannel))

										resp := patchInstance(testCtx.SMWithOAuthForTenant, "true", instanceID, http.StatusAccepted)

										instanceID, _ = VerifyOperationExists(testCtx, resp.Header("Location").Raw(), OperationExpectations{
											Category:          types.UPDATE,
											State:             types.IN_PROGRESS,
											ResourceType:      types.ServiceInstanceType,
											Reschedulable:     true,
											DeletionScheduled: false,
										})

										VerifyResourceExists(testCtx.SMWithOAuthForTenant, ResourceExpectations{
											ID:    instanceID,
											Type:  types.ServiceInstanceType,
											Ready: true,
										})
									})

									AfterEach(func() {
										close(doneChannel)
									})

									It("updates fail with operation in progress", func() {
										patchInstance(testCtx.SMWithOAuthForTenant, testCase.async, instanceID, http.StatusUnprocessableEntity)
									})

									It("deletes succeed", func() {
										resp := testCtx.SMWithOAuthForTenant.DELETE(web.ServiceInstancesURL+"/"+instanceID).WithQuery("async", testCase.async == "true").
											Expect().StatusRange(httpexpect.Status2xx)

										instanceID, _ = VerifyOperationExists(testCtx, resp.Header("Location").Raw(), OperationExpectations{
											Category:          types.DELETE,
											State:             types.SUCCEEDED,
											ResourceType:      types.ServiceInstanceType,
											Reschedulable:     false,
											DeletionScheduled: false,
										})

										VerifyResourceDoesNotExist(testCtx.SMWithOAuthForTenant, ResourceExpectations{
											ID:   instanceID,
											Type: types.ServiceInstanceType,
										})
									})
								})

								When("plan does not exist", func() {
									BeforeEach(func() {
										patchInstanceRequest["service_plan_id"] = "non-existing-id"
									})

									It("update fails", func() {
										patchInstance(testCtx.SMWithOAuthForTenant, testCase.async, instanceID, http.StatusNotFound)
									})
								})

								When("broker responds with synchronous success", func() {
									BeforeEach(func() {
										brokerServer.ServiceInstanceHandlerFunc(http.MethodPatch, http.MethodPatch+"1", ParameterizedHandler(http.StatusOK, Object{"async": false}))
									})

									It("stores instance as ready=true and the operation as success, non rescheduable with no deletion scheduled", func() {
										resp := patchInstance(testCtx.SMWithOAuthForTenant, testCase.async, instanceID, testCase.expectedUpdateSuccessStatusCode)

										instanceID, _ = VerifyOperationExists(testCtx, resp.Header("Location").Raw(), OperationExpectations{
											Category:          types.UPDATE,
											State:             types.SUCCEEDED,
											ResourceType:      types.ServiceInstanceType,
											Reschedulable:     false,
											DeletionScheduled: false,
										})

										VerifyResourceExists(testCtx.SMWithOAuthForTenant, ResourceExpectations{
											ID:    instanceID,
											Type:  types.ServiceInstanceType,
											Ready: true,
										})
									})
								})

								When("broker responds with asynchronous success", func() {
									BeforeEach(func() {
										brokerServer.ServiceInstanceHandlerFunc(http.MethodPatch, http.MethodPatch+"1", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
										brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodPatch+"1", MultiplePollsRequiredHandler("in progress", "succeeded"))
									})

									It("polling broker last operation until operation succeeds and eventually marks operation as success", func() {
										resp := patchInstance(testCtx.SMWithOAuthForTenant, testCase.async, instanceID, testCase.responseByBrokerOrClientMode(testCase.expectedUpdateSuccessStatusCode, http.StatusAccepted))

										instanceID, _ = VerifyOperationExists(testCtx, resp.Header("Location").Raw(), OperationExpectations{
											Category:          types.UPDATE,
											State:             types.SUCCEEDED,
											ResourceType:      types.ServiceInstanceType,
											Reschedulable:     false,
											DeletionScheduled: false,
										})

										VerifyResourceExists(testCtx.SMWithOAuthForTenant, ResourceExpectations{
											ID:    instanceID,
											Type:  types.ServiceInstanceType,
											Ready: true,
										})
									})

									When("maximum polling duration is reached while polling", func() {
										JustBeforeEach(func() {
											preparePrerequisitesWithMaxPollingDuration(MaximumPollingDuration)

											EnsurePlanVisibility(testCtx.SMRepository, TenantIdentifier, types.SMPlatform, postInstanceRequest["service_plan_id"].(string), TenantIDValue)
											resp := createInstance(testCtx.SMWithOAuthForTenant, testCase.async, testCase.expectedCreateSuccessStatusCode)

											instanceID, _ = VerifyOperationExists(testCtx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.CREATE,
												State:             types.SUCCEEDED,
												ResourceType:      types.ServiceInstanceType,
												Reschedulable:     false,
												DeletionScheduled: false,
											})

											VerifyResourceExists(testCtx.SMWithOAuthForTenant, ResourceExpectations{
												ID:    instanceID,
												Type:  types.ServiceInstanceType,
												Ready: true,
											})

											testCtx = t.ContextBuilder.WithEnvPreExtensions(func(set *pflag.FlagSet) {
												Expect(set.Set("operations.action_timeout", ((MaximumPollingDuration + 5) * time.Second).String())).ToNot(HaveOccurred())
												Expect(set.Set("api.enable_instance_transfer", "true")).ToNot(HaveOccurred())
											}).BuildWithoutCleanup()

											brokerServer.ServiceInstanceHandlerFunc(http.MethodPatch, http.MethodPatch+"1", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
											brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodPatch+"1", ParameterizedHandler(http.StatusOK, Object{"state": "in progress"}))
										})

										AfterEach(func() {
											testCtx.CleanupAll(false)
										})

										It("keeps instance as ready true and stores the operation as failed", func() {
											resp := patchInstance(testCtx.SMWithOAuthForTenant, "true", instanceID, http.StatusAccepted)

											instanceID, _ = VerifyOperationExists(testCtx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.UPDATE,
												State:             types.FAILED,
												ResourceType:      types.ServiceInstanceType,
												Reschedulable:     false,
												DeletionScheduled: false,
											})

											VerifyResourceExists(testCtx.SMWithOAuthForTenant, ResourceExpectations{
												ID:    instanceID,
												Type:  types.ServiceInstanceType,
												Ready: true,
											})
										})
									})

									if testCase.async == "true" {
										When("action timeout is reached while polling", func() {
											var newtestCtx *TestContext

											BeforeEach(func() {
												newtestCtx = t.ContextBuilder.WithEnvPreExtensions(func(set *pflag.FlagSet) {
													Expect(set.Set("operations.action_timeout", (2 * time.Second).String())).ToNot(HaveOccurred())
												}).BuildWithoutCleanup()

												brokerServer.ServiceInstanceHandlerFunc(http.MethodPatch, http.MethodPatch+"1", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
												brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodPatch+"1", ParameterizedHandler(http.StatusOK, Object{"state": "in progress"}))
											})

											AfterEach(func() {
												newtestCtx.CleanupAll(false)
											})

											It("stores instance as ready true and the operation as reschedulable in progress", func() {
												resp := patchInstance(newtestCtx.SMWithOAuthForTenant, "true", instanceID, http.StatusAccepted)

												instanceID, _ = VerifyOperationExists(newtestCtx, resp.Header("Location").Raw(), OperationExpectations{
													Category:          types.UPDATE,
													State:             types.IN_PROGRESS,
													ResourceType:      types.ServiceInstanceType,
													Reschedulable:     true,
													DeletionScheduled: false,
												})

												VerifyResourceExists(newtestCtx.SMWithOAuthForTenant, ResourceExpectations{
													ID:    instanceID,
													Type:  types.ServiceInstanceType,
													Ready: true,
												})
											})
										})
									}

									When("polling responds with unexpected state and eventually with success state", func() {
										BeforeEach(func() {
											brokerServer.ServiceInstanceHandlerFunc(http.MethodPatch, http.MethodPatch+"1", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
											brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodPatch+"1", MultiplePollsRequiredHandler("unknown", "succeeded"))
										})

										It("keeps polling and eventually updates the instance to ready true and operation to success", func() {
											resp := patchInstance(testCtx.SMWithOAuthForTenant, testCase.async, instanceID, testCase.responseByBrokerOrClientMode(testCase.expectedUpdateSuccessStatusCode, http.StatusAccepted))

											instanceID, _ = VerifyOperationExists(testCtx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.UPDATE,
												State:             types.SUCCEEDED,
												ResourceType:      types.ServiceInstanceType,
												Reschedulable:     false,
												DeletionScheduled: false,
											})

											VerifyResourceExists(testCtx.SMWithOAuthForTenant, ResourceExpectations{
												ID:    instanceID,
												Type:  types.ServiceInstanceType,
												Ready: true,
											})
										})
									})

									When("polling responds with unexpected state and eventually with failed state", func() {
										BeforeEach(func() {
											brokerServer.ServiceInstanceHandlerFunc(http.MethodPatch, http.MethodPatch+"2", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
											brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodPatch+"2", MultiplePollsRequiredHandler("unknown", "failed"))
										})

										It("keeps the instance and marks the operation as failed with no deletion scheduled and not reschedulable", func() {
											resp := patchInstance(testCtx.SMWithOAuthForTenant, testCase.async, instanceID, testCase.responseByBrokerOrClientMode(testCase.expectedBrokerFailureStatusCode, http.StatusAccepted))

											instanceID, _ = VerifyOperationExists(testCtx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.UPDATE,
												State:             types.FAILED,
												ResourceType:      types.ServiceInstanceType,
												Reschedulable:     false,
												DeletionScheduled: false,
											})

											VerifyResourceExists(testCtx.SMWithOAuthForTenant, ResourceExpectations{
												ID:    instanceID,
												Type:  types.ServiceInstanceType,
												Ready: true,
											})
										})

										When("polling returns an unexpected status code", func() {
											BeforeEach(func() {
												brokerServer.ServiceInstanceHandlerFunc(http.MethodPatch, http.MethodPatch+"3", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
												brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodPatch+"3", ParameterizedHandler(http.StatusInternalServerError, Object{"error": "error"}))
											})

											It("stores the instance as ready true and marks the operation as reschedulable", func() {
												resp := patchInstance(testCtx.SMWithOAuthForTenant, testCase.async, instanceID, testCase.responseByBrokerOrClientMode(testCase.expectedBrokerFailureStatusCode, http.StatusAccepted))

												instanceID, _ = VerifyOperationExists(testCtx, resp.Header("Location").Raw(), OperationExpectations{
													Category:          types.UPDATE,
													State:             types.FAILED,
													ResourceType:      types.ServiceInstanceType,
													Reschedulable:     true,
													DeletionScheduled: false,
												})

												VerifyResourceExists(testCtx.SMWithOAuthForTenant, ResourceExpectations{
													ID:    instanceID,
													Type:  types.ServiceInstanceType,
													Ready: true,
												})
											})
										})
									})

									When("broker responds with error due to stopped broker", func() {
										It("keeps the instance and marks operation with failed", func() {
											brokerServer.Close()
											delete(testCtx.Servers, BrokerServerPrefix+brokerID)

											resp := patchInstance(testCtx.SMWithOAuthForTenant, testCase.async, instanceID, testCase.expectedBrokerFailureStatusCode)

											instanceID, _ = VerifyOperationExists(testCtx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.UPDATE,
												State:             types.FAILED,
												ResourceType:      types.ServiceInstanceType,
												Reschedulable:     false,
												DeletionScheduled: false,
											})

											VerifyResourceExists(testCtx.SMWithOAuthForTenant, ResourceExpectations{
												ID:    instanceID,
												Type:  types.ServiceInstanceType,
												Ready: true,
											})
										})
									})

									When("broker responds with error", func() {
										BeforeEach(func() {
											brokerServer.ServiceInstanceHandlerFunc(http.MethodPatch, http.MethodPatch+"3", ParameterizedHandler(http.StatusInternalServerError, Object{"error": "error"}))
										})

										It("keeps the instance as ready true and marks the operation as failed", func() {
											resp := patchInstance(testCtx.SMWithOAuthForTenant, testCase.async, instanceID, testCase.expectedBrokerFailureStatusCode)

											instanceID, _ = VerifyOperationExists(testCtx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.UPDATE,
												State:             types.FAILED,
												ResourceType:      types.ServiceInstanceType,
												Reschedulable:     false,
												DeletionScheduled: false,
											})

											VerifyResourceExists(testCtx.SMWithOAuthForTenant, ResourceExpectations{
												ID:    instanceID,
												Type:  types.ServiceInstanceType,
												Ready: true,
											})
										})

									})
								})
							})
						})
						if testCase.async == "true" {
							When("instance that has failed to create exists in the service manager platform", func() {
								BeforeEach(func() {
									EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, servicePlanID, TenantIDValue)
									brokerServer.ServiceInstanceHandlerFunc(http.MethodPut, http.MethodPut+"1", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
									brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodPut+"1", ParameterizedHandler(
										http.StatusOK, Object{"state": "failed"},
									))

									resp := createInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedCreateSuccessStatusCode)

									instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
										Category:          types.CREATE,
										State:             types.FAILED,
										ResourceType:      types.ServiceInstanceType,
										Reschedulable:     false,
										DeletionScheduled: false,
									})

									VerifyResourceExists(ctx.SMWithOAuthForTenant, ResourceExpectations{
										ID:   instanceID,
										Type: types.ServiceInstanceType,
									})
								})

								It("patch should fail", func() {
									ctx.SMWithOAuthForTenant.PATCH(web.ServiceInstancesURL+"/"+instanceID).
										WithQuery("async", testCase.async == "true").
										WithJSON(Object{"name": "instance2"}).
										Expect().Status(http.StatusForbidden)
								})
							})
						}
					})
				}
			})

			Describe("DELETE", func() {
				It("returns 405 for bulk delete", func() {
					ctx.SMWithOAuthForTenant.DELETE(web.ServiceInstancesURL).
						Expect().Status(http.StatusMethodNotAllowed)
				})

				for _, testCase := range testCases {
					testCase := testCase

					Context(fmt.Sprintf("async = %s", testCase.async), func() {
						BeforeEach(func() {
							brokerServer.ShouldRecordRequests(true)
						})

						AfterEach(func() {
							brokerServer.ResetHandlers()
							ctx.SMWithOAuth.DELETE(web.ServiceInstancesURL + "/" + instanceID).Expect()
							ctx.SMWithOAuthForTenant.DELETE(web.ServiceInstancesURL + "/" + instanceID).Expect()
						})

						When("instance exists in a platform different from service manager", func() {
							const (
								brokerAPIVersionHeaderKey   = "X-Broker-API-Version"
								brokerAPIVersionHeaderValue = "2.13"
								SID                         = "abc1234"
							)

							BeforeEach(func() {
								brokerServer.ServiceInstanceHandlerFunc(http.MethodPut, http.MethodPut, verificationHandler(map[string]string{
									"context." + TenantIdentifier: TenantIDValue,
								}, http.StatusCreated))

								EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, ctx.TestPlatform.ID, findPlanIDForCatalogID(ctx, brokerID, service1CatalogID, plan1CatalogID), TenantIDValue)
								instanceID = SID
								ctx.SMWithBasic.PUT("/v1/osb/"+brokerID+"/v2/service_instances/"+SID).
									WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
									WithJSON(Object{
										"service_id": service1CatalogID,
										"plan_id":    plan1CatalogID,
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

							It("is successfully deleted", func() {
								requestsBeforeDeletion := len(brokerServer.ServiceInstanceEndpointRequests)
								resp := deleteInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedDeleteSuccessStatusCode)
								instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
									Category:          types.DELETE,
									State:             types.SUCCEEDED,
									ResourceType:      types.ServiceInstanceType,
									Reschedulable:     false,
									DeletionScheduled: false,
								})
								VerifyResourceDoesNotExist(ctx.SMWithOAuthForTenant, ResourceExpectations{
									ID:   instanceID,
									Type: types.ServiceInstanceType,
								})
								requestsAfterDeletion := len(brokerServer.ServiceInstanceEndpointRequests)

								Expect(requestsAfterDeletion - requestsBeforeDeletion).To(Equal(1))
							})

						})

						When("instance exists in service manager platform", func() {
							Context("instance ownership", func() {
								When("tenant doesn't have ownership of instance", func() {
									It("returns 404", func() {
										EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, postInstanceRequest["service_plan_id"].(string), TenantIDValue)
										resp := createInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedCreateSuccessStatusCode)
										instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
											Category:          types.CREATE,
											State:             types.SUCCEEDED,
											ResourceType:      types.ServiceInstanceType,
											Reschedulable:     false,
											DeletionScheduled: false,
										})
										expectedCode := http.StatusNotFound
										if testCase.async == "true" {
											expectedCode = http.StatusAccepted
										}
										otherTenantExpect := ctx.NewTenantExpect("tenancyClient", "other-tenant")
										deleteInstance(otherTenantExpect, testCase.async, expectedCode)
									})
								})

								When("tenant has ownership of instance", func() {
									It("returns 200", func() {
										EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, servicePlanID, TenantIDValue)
										resp := createInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedCreateSuccessStatusCode)
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
										VerifyResourceDoesNotExist(ctx.SMWithOAuthForTenant, ResourceExpectations{
											ID:   instanceID,
											Type: types.ServiceInstanceType,
										})
									})
								})
							})

							Context("broker scenarios", func() {
								BeforeEach(func() {
									EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, servicePlanID, TenantIDValue)
									resp := createInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedCreateSuccessStatusCode)

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
								})

								When("a delete operation is already in progress", func() {
									var doneChannel chan interface{}

									BeforeEach(func() {
										doneChannel = make(chan interface{})
										brokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"1", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
										brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodDelete+"1", DelayingHandler(doneChannel))

										resp := deleteInstance(ctx.SMWithOAuthForTenant, "true", http.StatusAccepted)

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

									It("updates fail with operation in progress", func() {
										ctx.SMWithOAuthForTenant.PATCH(web.ServiceInstancesURL+"/"+instanceID).WithQuery("async", testCase.async).WithJSON(Object{}).
											Expect().Status(http.StatusUnprocessableEntity)
									})

									It("deletes fail with operation in progress", func() {
										ctx.SMWithOAuthForTenant.DELETE(web.ServiceInstancesURL+"/"+instanceID).WithQuery("async", testCase.async).
											Expect().Status(http.StatusUnprocessableEntity)
									})
								})

								When("maximum polling duration is reached while polling", func() {
									var newCtx *TestContext

									BeforeEach(func() {
										preparePrerequisitesWithMaxPollingDuration(MaximumPollingDuration)

										EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, servicePlanID, TenantIDValue)
										resp := createInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.expectedCreateSuccessStatusCode)

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

										newCtx = t.ContextBuilder.WithEnvPreExtensions(func(set *pflag.FlagSet) {
											Expect(set.Set("operations.action_timeout", ((MaximumPollingDuration + 1) * time.Second).String())).ToNot(HaveOccurred())
										}).BuildWithoutCleanup()

										brokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"1", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
										brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodDelete+"1", ParameterizedHandler(http.StatusOK, Object{"state": "in progress"}))
									})

									AfterEach(func() {
										newCtx.CleanupAll(false)
									})

									When("orphan mitigation deprovision synchronously succeeds", func() {
										It("deletes the instance and marks the operation as success", func() {
											resp := deleteInstance(newCtx.SMWithOAuthForTenant, testCase.async, testCase.responseByBrokerOrClientMode(testCase.expectedBrokerFailureStatusCode, http.StatusAccepted))

											instanceID, _ = VerifyOperationExists(newCtx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.DELETE,
												State:             types.FAILED,
												ResourceType:      types.ServiceInstanceType,
												Reschedulable:     false,
												DeletionScheduled: true,
											})

											brokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"2", ParameterizedHandler(http.StatusOK, Object{"async": false}))

											instanceID, _ = VerifyOperationExists(newCtx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.DELETE,
												State:             types.SUCCEEDED,
												ResourceType:      types.ServiceInstanceType,
												Reschedulable:     false,
												DeletionScheduled: false,
											})

											VerifyResourceDoesNotExist(newCtx.SMWithOAuthForTenant, ResourceExpectations{
												ID:   instanceID,
												Type: types.ServiceInstanceType,
											})
										})
									})

									When("broker orphan mitigation deprovision synchronously fails with an unexpected error", func() {
										It("keeps in the instance and marks the operation with deletion scheduled", func() {
											resp := deleteInstance(newCtx.SMWithOAuthForTenant, testCase.async, testCase.responseByBrokerOrClientMode(testCase.expectedBrokerFailureStatusCode, http.StatusAccepted))

											instanceID, _ = VerifyOperationExists(newCtx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.DELETE,
												State:             types.FAILED,
												ResourceType:      types.ServiceInstanceType,
												Reschedulable:     false,
												DeletionScheduled: true,
											})

											brokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"2", ParameterizedHandler(http.StatusBadRequest, Object{"error": "error"}))

											instanceID, _ = VerifyOperationExists(newCtx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.DELETE,
												State:             types.FAILED,
												ResourceType:      types.ServiceInstanceType,
												Reschedulable:     false,
												DeletionScheduled: true,
											})

											VerifyResourceExists(newCtx.SMWithOAuthForTenant, ResourceExpectations{
												ID:    instanceID,
												Type:  types.ServiceInstanceType,
												Ready: true,
											})
										})
									})

									When("broker orphan mitigation deprovision synchronously fails with an error that will continue further orphan mitigation and eventually succeed", func() {
										It("deletes the instance and marks the operation that triggered the orphan mitigation as failed with no deletion scheduled and not reschedulable", func() {
											resp := deleteInstance(newCtx.SMWithOAuthForTenant, testCase.async, testCase.responseByBrokerOrClientMode(testCase.expectedBrokerFailureStatusCode, http.StatusAccepted))

											instanceID, _ = VerifyOperationExists(newCtx, resp.Header("Location").Raw(), OperationExpectations{
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

											instanceID, _ = VerifyOperationExists(newCtx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.DELETE,
												State:             types.SUCCEEDED,
												ResourceType:      types.ServiceInstanceType,
												Reschedulable:     false,
												DeletionScheduled: false,
											})

											VerifyResourceDoesNotExist(newCtx.SMWithOAuthForTenant, ResourceExpectations{
												ID:   instanceID,
												Type: types.ServiceInstanceType,
											})
										})
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
										if testCase.async == "true" {
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

										VerifyResourceExists(ctx.SMWithOAuthForTenant, ResourceExpectations{
											ID:    instanceID,
											Type:  types.ServiceInstanceType,
											Ready: true,
										})
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

										VerifyResourceDoesNotExist(ctx.SMWithOAuthForTenant, ResourceExpectations{
											ID:   instanceID,
											Type: types.ServiceInstanceType,
										})
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

										VerifyResourceDoesNotExist(ctx.SMWithOAuthForTenant, ResourceExpectations{
											ID:   instanceID,
											Type: types.ServiceInstanceType,
										})
									})
								})

								When("broker responds with asynchronous success", func() {
									BeforeEach(func() {
										brokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"1", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
										brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodDelete+"1", MultiplePollsRequiredHandler("in progress", "succeeded"))
									})

									It("polling broker last operation until operation succeeds and eventually marks operation as success", func() {
										resp := deleteInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.responseByBrokerOrClientMode(testCase.expectedDeleteSuccessStatusCode, http.StatusAccepted))

										instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
											Category:          types.DELETE,
											State:             types.SUCCEEDED,
											ResourceType:      types.ServiceInstanceType,
											Reschedulable:     false,
											DeletionScheduled: false,
										})

										VerifyResourceDoesNotExist(ctx.SMWithOAuthForTenant, ResourceExpectations{
											ID:   instanceID,
											Type: types.ServiceInstanceType,
										})
									})

									if testCase.async == "true" {
										When("SM crashes while polling", func() {
											var newSMCtx *TestContext
											var isDeprovisioned atomic.Value

											BeforeEach(func() {
												newSMCtx = t.ContextBuilder.WithEnvPostExtensions(func(e env.Environment, servers map[string]FakeServer) {
													e.Set("server.shutdown_timeout", 1*time.Second)
													e.Set("operations.maintainer_retry_interval", 1*time.Second)
												}).BuildWithoutCleanup()

												brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodDelete+"1", func(_ *http.Request) (int, map[string]interface{}) {
													if isDeprovisioned.Load() != nil {
														return http.StatusOK, Object{"state": "succeeded"}
													} else {
														return http.StatusOK, Object{"state": "in progress"}
													}
												})
											})

											AfterEach(func() {
												newSMCtx.CleanupAll(false)
											})

											It("should restart polling through maintainer and eventually deletes the instance", func() {
												resp := deleteInstance(newSMCtx.SMWithOAuthForTenant, "true", http.StatusAccepted)

												operationExpectations := OperationExpectations{
													Category:          types.DELETE,
													State:             types.IN_PROGRESS,
													ResourceType:      types.ServiceInstanceType,
													Reschedulable:     true,
													DeletionScheduled: false,
												}

												instanceID, _ = VerifyOperationExists(newSMCtx, resp.Header("Location").Raw(), operationExpectations)
												VerifyResourceExists(newSMCtx.SMWithOAuthForTenant, ResourceExpectations{
													ID:    instanceID,
													Type:  types.ServiceInstanceType,
													Ready: true,
												})

												newSMCtx.CleanupAll(false)
												isDeprovisioned.Store(true)

												newSMCtx = t.ContextBuilder.WithEnvPostExtensions(func(e env.Environment, servers map[string]FakeServer) {
													e.Set("operations.action_timeout", 2*time.Second)
												}).BuildWithoutCleanup()

												operationExpectations.State = types.SUCCEEDED
												operationExpectations.Reschedulable = false

												instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), operationExpectations)
												VerifyResourceDoesNotExist(ctx.SMWithOAuthForTenant, ResourceExpectations{
													ID:   instanceID,
													Type: types.ServiceInstanceType,
												})

											})
										})
									}

									When("polling responds 410 GONE", func() {
										BeforeEach(func() {
											brokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"1", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
											brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodDelete+"1", ParameterizedHandler(http.StatusGone, Object{}))
										})

										It("keeps polling and eventually deletes the instance and marks the operation as success", func() {
											resp := deleteInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.responseByBrokerOrClientMode(testCase.expectedDeleteSuccessStatusCode, http.StatusAccepted))

											instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.DELETE,
												State:             types.SUCCEEDED,
												ResourceType:      types.ServiceInstanceType,
												Reschedulable:     false,
												DeletionScheduled: false,
											})

											VerifyResourceDoesNotExist(ctx.SMWithOAuthForTenant, ResourceExpectations{
												ID:   instanceID,
												Type: types.ServiceInstanceType,
											})
										})
									})

									When("polling responds with unexpected state and eventually with success state", func() {
										BeforeEach(func() {
											brokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"1", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
											brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodDelete+"1", MultiplePollsRequiredHandler("unknown", "succeeded"))
										})

										It("keeps polling and eventually deletes the instance and marks the operation as success", func() {
											resp := deleteInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.responseByBrokerOrClientMode(testCase.expectedDeleteSuccessStatusCode, http.StatusAccepted))

											instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.DELETE,
												State:             types.SUCCEEDED,
												ResourceType:      types.ServiceInstanceType,
												Reschedulable:     false,
												DeletionScheduled: false,
											})

											VerifyResourceDoesNotExist(ctx.SMWithOAuthForTenant, ResourceExpectations{
												ID:   instanceID,
												Type: types.ServiceInstanceType,
											})
										})
									})

									When("polling responds with unexpected state and eventually with failed state", func() {
										BeforeEach(func() {
											brokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"2", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
											brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodDelete+"2", MultiplePollsRequiredHandler("unknown", "failed"))
										})

										When("orphan mitigation deprovision synchronously succeeds", func() {
											It("deletes the instance and marks the operation as success", func() {
												resp := deleteInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.responseByBrokerOrClientMode(testCase.expectedBrokerFailureStatusCode, http.StatusAccepted))

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

												VerifyResourceDoesNotExist(ctx.SMWithOAuthForTenant, ResourceExpectations{
													ID:   instanceID,
													Type: types.ServiceInstanceType,
												})
											})
										})

										When("broker orphan mitigation deprovision synchronously fails with an unexpected error", func() {
											It("keeps in the instance and marks the operation with deletion scheduled", func() {
												resp := deleteInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.responseByBrokerOrClientMode(testCase.expectedBrokerFailureStatusCode, http.StatusAccepted))

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

												VerifyResourceExists(ctx.SMWithOAuthForTenant, ResourceExpectations{
													ID:    instanceID,
													Type:  types.ServiceInstanceType,
													Ready: true,
												})
											})
										})

										When("broker orphan mitigation deprovision synchronously fails with an error that will continue further orphan mitigation and eventually succeed", func() {
											It("deletes the instance and marks the operation that triggered the orphan mitigation as failed with no deletion scheduled and not reschedulable", func() {
												resp := deleteInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.responseByBrokerOrClientMode(testCase.expectedBrokerFailureStatusCode, http.StatusAccepted))

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

												VerifyResourceDoesNotExist(ctx.SMWithOAuthForTenant, ResourceExpectations{
													ID:   instanceID,
													Type: types.ServiceInstanceType,
												})
											})
										})

										When("maximum deletion timout has been reached", func() {
											var newCtx *TestContext

											BeforeEach(func() {
												newCtx = t.ContextBuilder.WithEnvPreExtensions(func(set *pflag.FlagSet) {
													Expect(set.Set("operations.reconciliation_operation_timeout", (2 * time.Second).String())).ToNot(HaveOccurred())
												}).BuildWithoutCleanup()
											})

											AfterEach(func() {
												newCtx.CleanupAll(false)
											})

											It("keeps the instance as ready false and marks the operation as deletion scheduled", func() {
												resp := deleteInstance(newCtx.SMWithOAuthForTenant, testCase.async, testCase.responseByBrokerOrClientMode(testCase.expectedBrokerFailureStatusCode, http.StatusAccepted))

												instanceID, _ = VerifyOperationExists(newCtx, resp.Header("Location").Raw(), OperationExpectations{
													Category:          types.DELETE,
													State:             types.FAILED,
													ResourceType:      types.ServiceInstanceType,
													Reschedulable:     false,
													DeletionScheduled: true,
												})

												VerifyResourceExists(newCtx.SMWithOAuthForTenant, ResourceExpectations{
													ID:    instanceID,
													Type:  types.ServiceInstanceType,
													Ready: true,
												})
											})
										})
									})

									When("polling returns an unexpected status code", func() {
										BeforeEach(func() {
											brokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"3", ParameterizedHandler(http.StatusAccepted, Object{"async": true}))
											brokerServer.ServiceInstanceLastOpHandlerFunc(http.MethodDelete+"3", ParameterizedHandler(http.StatusInternalServerError, Object{"error": "error"}))
										})

										It("keeps the instance and stores the operation as reschedulable", func() {
											resp := deleteInstance(ctx.SMWithOAuthForTenant, testCase.async, testCase.responseByBrokerOrClientMode(testCase.expectedBrokerFailureStatusCode, http.StatusAccepted))

											instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.DELETE,
												State:             types.FAILED,
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
									})
								})

								When("deprovision responds with error due to stopped broker", func() {
									JustBeforeEach(func() {
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

										VerifyResourceExists(ctx.SMWithOAuthForTenant, ResourceExpectations{
											ID:    instanceID,
											Type:  types.ServiceInstanceType,
											Ready: true,
										})
									})

									When("cascade=true and force=true are passed", func() {
										var bindingID string
										BeforeEach(func() {
											resp := ctx.SMWithOAuthForTenant.POST(web.ServiceBindingsURL).
												WithJSON(Object{"name": "test-binding", "service_instance_id": instanceID}).
												Expect().
												Status(http.StatusCreated)
											bindingID = resp.JSON().Object().Value("id").String().Raw()
										})

										It("deletes the instance and its bindings and marks operation with success", func() {
											resp := ctx.SMWithOAuthForTenant.DELETE(web.ServiceInstancesURL+"/"+instanceID).
												WithQuery("async", testCase.async).
												WithQuery("force", true).
												WithQuery("cascade", true).
												Expect().
												Status(http.StatusAccepted)

											By("validating instance delete operation exists with status pending")
											instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.DELETE,
												State:             types.PENDING,
												ResourceType:      types.ServiceInstanceType,
												Reschedulable:     false,
												DeletionScheduled: false,
											})

											By("validating binding does not exist")
											VerifyResourceDoesNotExist(ctx.SMWithOAuthForTenant, ResourceExpectations{
												ID:   bindingID,
												Type: types.ServiceBindingType,
											})

											By("validating instance does not exist")
											VerifyResourceDoesNotExist(ctx.SMWithOAuthForTenant, ResourceExpectations{
												ID:   instanceID,
												Type: types.ServiceInstanceType,
											})
										})
									})

								})

								When("deprovision responds with error that does not require orphan mitigation", func() {
									JustBeforeEach(func() {
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

										VerifyResourceExists(ctx.SMWithOAuthForTenant, ResourceExpectations{
											ID:    instanceID,
											Type:  types.ServiceInstanceType,
											Ready: true,
										})
									})

									When("cascade=true and force=true are passed", func() {
										var bindingID string
										BeforeEach(func() {
											resp := ctx.SMWithOAuthForTenant.POST(web.ServiceBindingsURL).
												WithJSON(Object{"name": "test-binding", "service_instance_id": instanceID}).
												Expect().
												Status(http.StatusCreated)

											bindingID = resp.JSON().Object().Value("id").String().Raw()
										})

										It("deletes the instance and its bindings and marks operation with success", func() {
											resp := ctx.SMWithOAuthForTenant.DELETE(web.ServiceInstancesURL+"/"+instanceID).
												WithQuery("async", testCase.async).
												WithQuery("force", true).
												WithQuery("cascade", true).
												Expect().
												Status(http.StatusAccepted)

											By("validating instance delete operation exists with status pending")
											instanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
												Category:          types.DELETE,
												State:             types.PENDING,
												ResourceType:      types.ServiceInstanceType,
												Reschedulable:     false,
												DeletionScheduled: false,
											})

											By("validating binding does not exist")
											VerifyResourceDoesNotExist(ctx.SMWithOAuthForTenant, ResourceExpectations{
												ID:   bindingID,
												Type: types.ServiceBindingType,
											})

											By("validating instance does not exist")
											VerifyResourceDoesNotExist(ctx.SMWithOAuthForTenant, ResourceExpectations{
												ID:   instanceID,
												Type: types.ServiceInstanceType,
											})
										})
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

											VerifyResourceDoesNotExist(ctx.SMWithOAuthForTenant, ResourceExpectations{
												ID:   instanceID,
												Type: types.ServiceInstanceType,
											})
										})
									})

									if testCase.async == "true" {
										When("broker orphan mitigation deprovision asynchronously keeps failing with an error while polling", func() {
											It("keeps the instance and marks the operation as failed reschedulable with deletion scheduled", func() {
												resp := deleteInstance(ctx.SMWithOAuthForTenant, "true", http.StatusAccepted)

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

												VerifyResourceExists(ctx.SMWithOAuthForTenant, ResourceExpectations{
													ID:    instanceID,
													Type:  types.ServiceInstanceType,
													Ready: true,
												})
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

											VerifyResourceDoesNotExist(ctx.SMWithOAuthForTenant, ResourceExpectations{
												ID:   instanceID,
												Type: types.ServiceInstanceType,
											})
										})
									})
								})

								When("deprovision responds with error due to times out", func() {
									var newSMCtx *TestContext
									var doneChannel chan interface{}

									BeforeEach(func() {
										doneChannel = make(chan interface{})

										newSMCtx = t.ContextBuilder.WithEnvPreExtensions(func(set *pflag.FlagSet) {
											Expect(set.Set("httpclient.timeout", (1 * time.Second).String())).ToNot(HaveOccurred())
										}).BuildWithoutCleanup()

										brokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"1", DelayingHandler(doneChannel))
									})

									AfterEach(func() {
										newSMCtx.CleanupAll(false)
									})

									It("orphan mitigates the instance", func() {
										resp := deleteInstance(newSMCtx.SMWithOAuthForTenant, testCase.async, testCase.expectedBrokerFailureStatusCode)
										<-time.After(1100 * time.Millisecond)
										close(doneChannel)

										instanceID, _ = VerifyOperationExists(newSMCtx, resp.Header("Location").Raw(), OperationExpectations{
											Category:          types.DELETE,
											State:             types.FAILED,
											ResourceType:      types.ServiceInstanceType,
											Reschedulable:     false,
											DeletionScheduled: true,
										})

										brokerServer.ServiceInstanceHandlerFunc(http.MethodDelete, http.MethodDelete+"1", ParameterizedHandler(http.StatusOK, Object{"async": false}))

										instanceID, _ = VerifyOperationExists(newSMCtx, resp.Header("Location").Raw(), OperationExpectations{
											Category:          types.DELETE,
											State:             types.SUCCEEDED,
											ResourceType:      types.ServiceInstanceType,
											Reschedulable:     false,
											DeletionScheduled: false,
										})

										VerifyResourceDoesNotExist(ctx.SMWithOAuthForTenant, ResourceExpectations{
											ID:   instanceID,
											Type: types.ServiceInstanceType,
										})
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

func blueprint(ctx *TestContext, auth *SMExpect, async bool) Object {
	ID, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}

	instanceReqBody := make(Object)
	instanceReqBody["name"] = "test-service-instance-" + ID.String()
	_, array := prepareBrokerWithCatalog(ctx, auth)
	instanceReqBody["service_plan_id"] = array.First().Object().Value("id").String().Raw()

	EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, instanceReqBody["service_plan_id"].(string), TenantIDValue)
	resp := ctx.SMWithOAuthForTenant.POST(web.ServiceInstancesURL).WithQuery("async", strconv.FormatBool(async)).WithJSON(instanceReqBody).Expect()

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

	return instance
}

func subResourcesBlueprint() func(ctx *TestContext, auth *SMExpect, async bool, platformID string, resourceType types.ObjectType, instance Object) {
	return func(ctx *TestContext, auth *SMExpect, async bool, platformID string, resourceType types.ObjectType, instance Object) {

		resp := ctx.SMWithOAuthForTenant.POST(web.ServiceBindingsURL).
			WithQuery("async", strconv.FormatBool(async)).
			WithJSON(Object{
				"name":                "test-service-binding",
				"service_instance_id": instance["id"],
			}).Expect()

		if async {
			resp.Status(http.StatusAccepted)
		} else {
			resp.Status(http.StatusCreated)
		}

		id, _ := VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
			Category:          types.CREATE,
			State:             types.SUCCEEDED,
			ResourceType:      types.ServiceBindingType,
			Reschedulable:     false,
			DeletionScheduled: false,
		})

		VerifyResourceExists(ctx.SMWithOAuthForTenant, ResourceExpectations{
			ID:    id,
			Type:  types.ServiceBindingType,
			Ready: true,
		}).Raw()

	}
}

func prepareBrokerWithCatalog(ctx *TestContext, auth *SMExpect) (*BrokerUtils, *httpexpect.Array) {
	return prepareBrokerWithCatalogAndPollingDuration(ctx, auth, 0)
}

func prepareBrokerWithCatalogAndPollingDuration(ctx *TestContext, auth *SMExpect, maxPollingDuration int) (*BrokerUtils, *httpexpect.Array) {
	cPaidPlan1 := GenerateTestPlanWithID(plan1CatalogID)
	cPaidPlan1, err := sjson.Set(cPaidPlan1, "maximum_polling_duration", maxPollingDuration)
	if err != nil {
		panic(err)
	}
	cPaidPlan2 := GeneratePaidTestPlan()
	cPaidPlan2, err = sjson.Set(cPaidPlan2, "maximum_polling_duration", maxPollingDuration)
	if err != nil {
		panic(err)
	}
	planNotSupportingSM := GenerateTestPlanWithID(planNotSupportingSMPlatform)
	planNotSupportingSM, err = sjson.Set(planNotSupportingSM, "metadata.supportedPlatforms.-1", "kubernetes")
	if err != nil {
		panic(err)
	}

	cService := GenerateTestServiceWithPlansWithID(service1CatalogID, cPaidPlan1, cPaidPlan2, planNotSupportingSM)
	cPaidPlan3 := GenerateTestPlanWithID(plan1CatalogID)
	cService2 := GenerateTestServiceWithPlansWithID(serviceNotSupportingContextUpdates, cPaidPlan3)
	cService2, err = sjson.Set(cService2, "allow_context_updates", false)
	if err != nil {
		panic(err)
	}
	cPaidPlan4 := GenerateTestPlanWithID(plan1CatalogID)
	cService3 := GenerateTestServiceWithPlansWithID(notRetrievableService, cPaidPlan4)
	cService3, err = sjson.Set(cService3, "instances_retrievable", false)
	if err != nil {
		panic(err)
	}
	catalog := NewEmptySBCatalog()
	catalog.AddService(cService)
	catalog.AddService(cService2)
	catalog.AddService(cService3)
	brokerUtils := ctx.RegisterBrokerWithCatalog(catalog)
	brokerID := brokerUtils.Broker.ID
	server := brokerUtils.Broker.BrokerServer

	ctx.Servers[BrokerServerPrefix+brokerID] = server
	so := auth.ListWithQuery(web.ServiceOfferingsURL, fmt.Sprintf("fieldQuery=broker_id eq '%s'", brokerID)).First()

	return brokerUtils, auth.ListWithQuery(web.ServicePlansURL, "fieldQuery="+fmt.Sprintf("service_offering_id eq '%s'", so.Object().Value("id").String().Raw()))
}

func findPlanIDForCatalogID(ctx *TestContext, brokerID, catalogServiceID, catalogPlanID string) string {
	resp := ctx.SMWithOAuth.ListWithQuery(web.ServiceOfferingsURL, fmt.Sprintf("fieldQuery=broker_id eq '%s' and catalog_id eq '%s'", brokerID, catalogServiceID))
	soIDs := make([]string, 0)
	for _, item := range resp.Iter() {
		soID := item.Object().Value("id").String().Raw()
		Expect(soID).ToNot(BeEmpty())
		soIDs = append(soIDs, soID)
	}

	return ctx.SMWithOAuth.ListWithQuery(web.ServicePlansURL, fmt.Sprintf("fieldQuery=catalog_id eq '%s' and service_offering_id in ('%s')", catalogPlanID, strings.Join(soIDs, "','"))).
		First().Object().Value("id").String().Raw()
}
