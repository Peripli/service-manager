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

			createInstance := func(smClient *SMExpect, async bool, expectedStatusCode int) {
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
				createInstance(ctx.SMWithOAuthForTenant, false, http.StatusCreated)
			})

			JustBeforeEach(func() {
				postBindingRequest = Object{
					"name":                "test-binding",
					"service_instance_id": instanceID,
				}
			})

			AfterEach(func() {
				ctx.SMRepository.Delete(context.TODO(), types.OperationType, query.ByField(query.InOperator, "id", bindingOperationID, instanceOperationID))
				DeleteInstance(ctx, instanceID, servicePlanID)
				DeleteBinding(ctx, bindingID, instanceID)
				ctx.SMWithOAuth.DELETE(web.ServiceBrokersURL + "/" + brokerID).Expect()
				delete(ctx.Servers, BrokerServerPrefix+brokerID)
				brokerServer.Close()
			})

			Describe("GET", func() {
				When("service binding contains tenant identifier in OSB context", func() {
					It("labels instance with tenant identifier", func() {
						createBinding(ctx.SMWithOAuthForTenant, false, http.StatusCreated)

						ctx.SMWithOAuthForTenant.GET(web.ServiceBindingsURL + "/" + bindingID).Expect().
							Status(http.StatusOK).
							JSON().
							Object().Path(fmt.Sprintf("$.labels[%s][*]", TenantIdentifier)).Array().Contains(TenantIDValue)
					})
				})

				When("service binding doesn't contain tenant identifier in OSB context", func() {
					It("doesn't label instance with tenant identifier", func() {
						createBinding(ctx.SMWithOAuth, false, http.StatusCreated)

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

							Context("when id  field is missing", func() {
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
								for _, provisionRequest := range brokerServer.BindingEndpointRequests {
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
