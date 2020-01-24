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

	"github.com/spf13/pflag"

	"github.com/gofrs/uuid"

	"github.com/gavv/httpexpect"

	"strconv"

	"net/http"
	"testing"

	"github.com/Peripli/service-manager/pkg/types"

	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/test/common"

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
	AdditionalTests: func(ctx *common.TestContext) {
		Context("additional non-generic tests", func() {
			var (
				postInstanceRequest      common.Object
				expectedInstanceResponse common.Object

				servicePlanID        string
				anotherServicePlanID string
				brokerID             string
				brokerServer         *common.BrokerServer

				instanceID string
			)

			createInstance := func(SM *common.SMExpect, expectedStatus int) {
				resp := SM.POST(web.ServiceInstancesURL).WithJSON(postInstanceRequest).
					Expect().
					Status(expectedStatus)

				if resp.Raw().StatusCode == http.StatusCreated {
					obj := resp.JSON().Object()

					obj.ContainsMap(expectedInstanceResponse).ContainsKey("id").
						ValueEqual("platform_id", types.SMPlatform)

					instanceID = obj.Value("id").String().Raw()
				}
			}

			BeforeEach(func() {
				var plans *httpexpect.Array
				brokerID, brokerServer, plans = prepareBrokerWithCatalog(ctx, ctx.SMWithOAuth)
				servicePlanID = plans.Element(0).Object().Value("id").String().Raw()
				anotherServicePlanID = plans.Element(1).Object().Value("id").String().Raw()
			})

			JustBeforeEach(func() {
				postInstanceRequest = common.Object{
					"name":             "test-instance",
					"service_plan_id":  servicePlanID,
					"maintenance_info": "{}",
				}
				expectedInstanceResponse = common.Object{
					"name":             "test-instance",
					"service_plan_id":  servicePlanID,
					"maintenance_info": "{}",
				}
			})

			AfterEach(func() {
				ctx.CleanupAdditionalResources()
			})

			Describe("GET", func() {
				When("service instance contains tenant identifier in OSB context", func() {
					BeforeEach(func() {
						EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, servicePlanID, TenantIDValue)
						createInstance(ctx.SMWithOAuthForTenant, http.StatusCreated)
					})

					It("labels instance with tenant identifier", func() {
						ctx.SMWithOAuthForTenant.GET(web.ServiceInstancesURL + "/" + instanceID).Expect().
							Status(http.StatusOK).
							JSON().
							Object().Path(fmt.Sprintf("$.labels[%s][*]", TenantIdentifier)).Array().Contains(TenantIDValue)
					})
				})
				When("service instance doesn't contain tenant identifier in OSB context", func() {
					BeforeEach(func() {
						EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, servicePlanID, "")
						createInstance(ctx.SMWithOAuth, http.StatusCreated)
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
				})

				When("service instance dashboard_url is not set", func() {
					BeforeEach(func() {
						postInstanceRequest["dashboard_url"] = ""
						EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, postInstanceRequest["service_plan_id"].(string), "")
						createInstance(ctx.SMWithOAuth, http.StatusCreated)
					})

					It("doesn't return dashboard_url", func() {
						ctx.SMWithOAuth.GET(web.ServiceInstancesURL + "/" + instanceID).Expect().
							Status(http.StatusOK).JSON().Object().NotContainsKey("dashboard_url")
					})
				})
			})

			Describe("POST", func() {
				When("content type is not JSON", func() {
					It("returns 415", func() {
						ctx.SMWithOAuth.POST(web.ServiceInstancesURL).WithText("text").
							Expect().
							Status(http.StatusUnsupportedMediaType).
							JSON().Object().
							Keys().Contains("error", "description")
					})
				})

				When("request body is not a valid JSON", func() {
					It("returns 400", func() {
						ctx.SMWithOAuth.POST(web.ServiceInstancesURL).
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
							delete(expectedInstanceResponse, field)
						})

						It("returns 400", func() {
							EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, servicePlanID, "")
							ctx.SMWithOAuth.POST(web.ServiceInstancesURL).WithJSON(postInstanceRequest).
								Expect().
								Status(http.StatusBadRequest).
								JSON().Object().
								Keys().Contains("error", "description")
						})
					}

					assertPOSTReturns201WhenFieldIsMissing := func(field string) {
						BeforeEach(func() {
							delete(postInstanceRequest, field)
							delete(expectedInstanceResponse, field)
						})

						It("returns 201", func() {
							EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, postInstanceRequest["service_plan_id"].(string), "")
							createInstance(ctx.SMWithOAuth, http.StatusCreated)
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
								Expect().Status(http.StatusBadRequest).JSON().Object()

							resp.Value("description").Equal("Providing platform_id property during provisioning/updating of a service instance is forbidden")
						})
					})

					Context("which is service-manager platform", func() {
						It("should return 200", func() {
							postInstanceRequest["platform_id"] = types.SMPlatform
							EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, postInstanceRequest["service_plan_id"].(string), "")
							createInstance(ctx.SMWithOAuth, http.StatusCreated)
						})
					})
				})

				When("async query param", func() {
					It("succeeds", func() {
						EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, postInstanceRequest["service_plan_id"].(string), "")
						resp := ctx.SMWithOAuth.POST(web.ServiceInstancesURL).WithJSON(postInstanceRequest).
							WithQuery("async", "true").
							Expect().
							Status(http.StatusAccepted)

						op, err := ExpectOperation(ctx.SMWithOAuth, resp, types.SUCCEEDED)
						Expect(err).To(BeNil())

						ctx.SMWithOAuth.GET(web.ServiceInstancesURL + "/" + op.Value("resource_id").String().Raw()).Expect().
							Status(http.StatusOK).
							JSON().Object().
							ContainsMap(expectedInstanceResponse).ContainsKey("id")
					})
				})

				Context("instance visibility", func() {
					When("tenant doesn't have plan visibility", func() {
						It("returns 404", func() {
							createInstance(ctx.SMWithOAuthForTenant, http.StatusNotFound)
						})
					})

					When("tenant has plan visibility", func() {
						It("returns 201", func() {
							EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, servicePlanID, TenantIDValue)
							createInstance(ctx.SMWithOAuthForTenant, http.StatusCreated)
						})
					})

					When("plan has public visibility", func() {
						It("for global returns 201", func() {
							EnsurePublicPlanVisibility(ctx.SMRepository, servicePlanID)
							createInstance(ctx.SMWithOAuth, http.StatusCreated)
						})

						It("for tenant returns 201", func() {
							EnsurePublicPlanVisibility(ctx.SMRepository, servicePlanID)
							createInstance(ctx.SMWithOAuthForTenant, http.StatusCreated)
						})
					})
				})

				//createOperation := func(instanceID string, category types.OperationCategory) {
				//	UUID, err := uuid.NewV4()
				//	Expect(err).ToNot(HaveOccurred())
				//	_, err = ctx.SMRepository.Create(context.TODO(), &types.Operation{
				//		Base: types.Base{
				//			ID:        UUID.String(),
				//			CreatedAt: time.Now(),
				//			UpdatedAt: time.Now(),
				//			Ready:     true,
				//		},
				//		Type:         category,
				//		State:        types.IN_PROGRESS,
				//		ResourceID:   instanceID,
				//		ResourceType: types.ServiceInstanceType,
				//		Reschedule:   false,
				//	})
				//
				//	Expect(err).ToNot(HaveOccurred())
				//}

				delayingHandler := func(done chan<- interface{}) func(rw http.ResponseWriter, req *http.Request) {
					return func(rw http.ResponseWriter, req *http.Request) {
						brokerDelay := 300 * time.Second
						timeoutContext, _ := context.WithTimeout(req.Context(), brokerDelay)
						<-timeoutContext.Done()
						common.SetResponse(rw, http.StatusTeapot, common.Object{})
						close(done)
					}
				}

				parameterizedHandler := func(statusCode int, responseBody string) func(rw http.ResponseWriter, _ *http.Request) {
					return func(rw http.ResponseWriter, _ *http.Request) {
						rw.Header().Set("Content-Type", "application/json")
						rw.WriteHeader(statusCode)
						rw.Write([]byte(responseBody))
					}
				}
				type testCase struct {
					async                     bool
					expectedSuccessStatusCode int
					verifyOperationSuccess    func()
				}

				//testCases := []testCase{
				//	{
				//		async:                     false,
				//		expectedSuccessStatusCode: http.StatusCreated,
				//		verifyOperationSuccess: func() {
				//
				//		},
				//	},
				//	{
				//		async:                     true,
				//		expectedSuccessStatusCode: http.StatusAccepted,
				//		verifyOperationSuccess: func() {
				//
				//		},
				//	},
				//}

				//for _, testCase := range testCases {
				//	testCase := testCase
				verifyInstanceCreated := func(instanceID string, ready bool) {
					timeoutDuration := 5 * time.Second
					tickerInterval := 100 * time.Millisecond
					ticker := time.NewTicker(tickerInterval)
					defer ticker.Stop()
					for {
						select {
						case <-time.After(timeoutDuration):
							Fail(fmt.Sprintf("instance with id %s did not appear in SM after %d seconds", instanceID, timeoutDuration))
						case <-ticker.C:
							instances := ctx.SMWithOAuthForTenant.ListWithQuery(web.ServiceInstancesURL, fmt.Sprintf("fieldQuery=id eq '%s'", instanceID))
							switch {
							case instances.Length().Raw() == 0:
								By(fmt.Sprintf("Could not find instance with id %s in SM. Retrying...", instanceID))
							case instances.Length().Raw() > 1:
								Fail(fmt.Sprintf("more than one instance with id %s was found in SM", instanceID))
							default:
								readyField := instances.First().Object().Value("ready").Boolean().Raw()
								if readyField != ready {
									Fail(fmt.Sprintf("Expected instance with id %s to be ready %t but ready was %t", instanceID, ready, readyField))
								}
								return
							}
						}
					}
				}

				When(fmt.Sprintf("async is %t", true), func() {
					var smInstanceID string
					var doneChannel chan<- interface{}
					BeforeEach(func() {
						EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, servicePlanID, TenantIDValue)
						doneChannel = make(chan<- interface{})
					})

					When("a create operation is already in progress", func() {
						BeforeEach(func() {
							brokerServer.ServiceInstanceHandler = parameterizedHandler(http.StatusAccepted, `{"async": true}`)
							brokerServer.ServiceInstanceLastOpHandler = delayingHandler(doneChannel)

							resp := ctx.SMWithOAuthForTenant.POST(web.ServiceInstancesURL).WithQuery("async", true).WithJSON(postInstanceRequest).
								Expect().Status(http.StatusAccepted)
							op, err := ExpectOperation(ctx.SMWithOAuth, resp, types.IN_PROGRESS)
							Expect(err).To(BeNil())
							smInstanceID = op.Value("resource_id").String().Raw()
							Expect(smInstanceID).ToNot(BeEmpty())

							verifyInstanceCreated(smInstanceID, false)
						}, 500)

						AfterEach(func() {
							close(doneChannel)
							brokerServer.ResetHandlers()
						})

						It("async updates fail with operation in progress", func() {
							ctx.SMWithOAuthForTenant.PATCH(web.ServiceInstancesURL+"/"+smInstanceID).WithQuery("async", true).WithJSON(common.Object{}).
								Expect().Status(http.StatusUnprocessableEntity)

						}, 500)

						It("sync updates fail with operation in progress", func() {
							ctx.SMWithOAuthForTenant.PATCH(web.ServiceInstancesURL+"/"+smInstanceID).WithQuery("async", false).WithJSON(common.Object{}).
								Expect().Status(http.StatusUnprocessableEntity)

						}, 500)

					})

					When("broker provision times out", func() {
						BeforeEach(func() {
							ctx = common.NewTestContextBuilderWithSecurity().WithEnvPreExtensions(func(set *pflag.FlagSet) {
								Expect(set.Set("httpclient.response_header_timeout", (500 * time.Millisecond).String())).ToNot(HaveOccurred())
							}).Build()
							brokerServer.ServiceInstanceHandler = delayingHandler(doneChannel)
						})

						AfterEach(func() {
							ctx.Cleanup()
						})

						FIt("does not store instance in SMDB and marks operation with failed adding an error", func() {
							resp := ctx.SMWithOAuthForTenant.POST(web.ServiceInstancesURL).WithQuery("async", true).WithJSON(postInstanceRequest).
								Expect().Status(http.StatusAccepted)
							op, err := ExpectOperation(ctx.SMWithOAuth, resp, types.FAILED)
							Expect(err).To(BeNil())
							smInstanceID = op.Value("resource_id").String().Raw()
							Expect(smInstanceID).ToNot(BeEmpty())
						})
					})

					When("broker does not exist", func() {
						It("does not store instance in SMDB and marks operation with failed adding an error", func() {

						})
					})

					When("broker is stopped", func() {
						It("does not store instance in SMDB and marks operation with failed adding an error", func() {

						})
					})

					When("broker responds with synchronous success", func() {
						It("stores instance as ready=true and marks operation as success, non rescheduable with empty deletion scheduled", func() {
							// trigger the verifier
						})
					})

					When("broker responds with asynchronous success", func() {
						It("polling broker last operation until operation succeeds and eventually marks operation as success", func() {
							//trigger the verifier func
						})

						When("job timeout is reached while polling", func() {

						})
					})

					When("broker responds with error that is incorrectly formatted osb error", func() {
						It("deletes the instance and marks the operation as failed, non rescheduable with empty deletion scheduled", func() {

						})
					})

					When("broker responds with error that is correctly formatted osb error that requires orphan mitigation", func() {
						When("broker orphan mitigation deprovision synchronously succeeds", func() {
							It("deletes the instance and marks the operation that triggered the orphan mitigation as failed with no deletion scheduled and not reschedulable", func() {

							})
						})

						When("broker orphan mitigation deprovision fails", func() {
							It("retries until deletion retry timeout is reached and eventually starts returning deletion timeout exceeded as response to api calls", func() {

							})
						})
					})
				})
				//}
			})

			Describe("PATCH", func() {
				When("content type is not JSON", func() {
					It("returns 415", func() {
						instanceID := fmt.Sprintf("%s", instanceID)
						ctx.SMWithOAuth.PATCH(web.ServiceInstancesURL+"/"+instanceID).
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
						ctx.SMWithOAuth.PATCH(web.ServiceInstancesURL+"/"+instanceID).
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
						createInstance(ctx.SMWithOAuth, http.StatusCreated)

						createdAt := "2015-01-01T00:00:00Z"

						ctx.SMWithOAuth.PATCH(web.ServiceInstancesURL+"/"+instanceID).
							WithJSON(common.Object{"created_at": createdAt}).
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
							createInstance(ctx.SMWithOAuth, http.StatusCreated)

							resp := ctx.SMWithOAuth.PATCH(web.ServiceInstancesURL + "/" + instanceID).
								WithJSON(common.Object{"platform_id": "test-platform-id"}).
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
							createInstance(ctx.SMWithOAuth, http.StatusCreated)

							ctx.SMWithOAuth.PATCH(web.ServiceInstancesURL + "/" + instanceID).
								WithJSON(common.Object{"platform_id": types.SMPlatform}).
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
						createInstance(ctx.SMWithOAuth, http.StatusCreated)

						for _, prop := range []string{"name", "maintenance_info"} {
							updatedBrokerJSON := common.Object{}
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
							createInstance(ctx.SMWithOAuthForTenant, http.StatusCreated)

							ctx.SMWithOAuthForTenant.PATCH(web.ServiceInstancesURL + "/" + instanceID).
								WithJSON(common.Object{"service_plan_id": anotherServicePlanID}).
								Expect().Status(http.StatusNotFound)
						})
					})

					When("tenant has plan visibility", func() {
						It("returns 201", func() {
							EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, servicePlanID, TenantIDValue)
							createInstance(ctx.SMWithOAuthForTenant, http.StatusCreated)

							EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, anotherServicePlanID, TenantIDValue)
							ctx.SMWithOAuthForTenant.PATCH(web.ServiceInstancesURL + "/" + instanceID).
								WithJSON(common.Object{"service_plan_id": anotherServicePlanID}).
								Expect().Status(http.StatusOK)
						})
					})
				})

				Context("instance ownership", func() {
					When("tenant doesn't have ownership of instance", func() {
						It("returns 404", func() {
							EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, postInstanceRequest["service_plan_id"].(string), "")
							createInstance(ctx.SMWithOAuth, http.StatusCreated)

							ctx.SMWithOAuthForTenant.PATCH(web.ServiceInstancesURL + "/" + instanceID).
								WithJSON(common.Object{"service_plan_id": anotherServicePlanID}).
								Expect().Status(http.StatusNotFound)
						})
					})

					When("tenant has ownership of instance", func() {
						It("returns 200", func() {
							EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, servicePlanID, TenantIDValue)
							createInstance(ctx.SMWithOAuthForTenant, http.StatusCreated)

							ctx.SMWithOAuthForTenant.PATCH(web.ServiceInstancesURL + "/" + instanceID).
								WithJSON(common.Object{"platform_id": types.SMPlatform}).
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

				Context("instance ownership", func() {
					When("tenant doesn't have ownership of instance", func() {
						It("returns 404", func() {
							EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, postInstanceRequest["service_plan_id"].(string), "")
							createInstance(ctx.SMWithOAuth, http.StatusCreated)

							ctx.SMWithOAuthForTenant.DELETE(web.ServiceInstancesURL + "/" + instanceID).
								Expect().Status(http.StatusNotFound)
						})
					})

					When("tenant has ownership of instance", func() {
						It("returns 200", func() {
							EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, servicePlanID, TenantIDValue)
							createInstance(ctx.SMWithOAuthForTenant, http.StatusCreated)

							ctx.SMWithOAuthForTenant.DELETE(web.ServiceInstancesURL + "/" + instanceID).
								Expect().Status(http.StatusOK)
						})
					})
				})
			})
		})
	},
})

func blueprint(ctx *common.TestContext, auth *common.SMExpect, async bool) common.Object {
	ID, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}

	instanceReqBody := make(common.Object, 0)
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

func prepareBrokerWithCatalog(ctx *common.TestContext, auth *common.SMExpect) (string, *common.BrokerServer, *httpexpect.Array) {
	cPaidPlan1 := common.GeneratePaidTestPlan()
	cPaidPlan2 := common.GeneratePaidTestPlan()
	cService := common.GenerateTestServiceWithPlans(cPaidPlan1, cPaidPlan2)
	catalog := common.NewEmptySBCatalog()
	catalog.AddService(cService)
	brokerID, _, server := ctx.RegisterBrokerWithCatalog(catalog)
	ctx.Servers[common.BrokerServerPrefix+brokerID] = server

	so := auth.ListWithQuery(web.ServiceOfferingsURL, fmt.Sprintf("fieldQuery=broker_id eq '%s'", brokerID)).First()

	return brokerID, server, auth.ListWithQuery(web.ServicePlansURL, "fieldQuery="+fmt.Sprintf("service_offering_id eq '%s'", so.Object().Value("id").String().Raw()))
}
