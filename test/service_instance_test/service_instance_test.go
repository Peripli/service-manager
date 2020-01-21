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

	"github.com/gavv/httpexpect"

	"strconv"

	"github.com/Peripli/service-manager/test/testutil/service_instance"
	"github.com/gofrs/uuid"

	"net/http"
	"testing"

	"github.com/Peripli/service-manager/pkg/types"

	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/test/common"

	"github.com/Peripli/service-manager/test"

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

var _ = test.DescribeTestsFor(test.TestCase{
	API: web.ServiceInstancesURL,
	SupportedOps: []test.Op{
		test.Get, test.List, test.Delete, test.DeleteList, test.Patch,
	},
	MultitenancySettings: &test.MultitenancySettings{
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
	PatchResource:                          test.APIResourcePatch,
	AdditionalTests: func(ctx *common.TestContext) {
		Context("additional non-generic tests", func() {
			var (
				postInstanceRequest      common.Object
				expectedInstanceResponse common.Object

				servicePlanID        string
				anotherServicePlanID string

				instanceID string
			)

			createInstance := func(SM *common.SMExpect, expectedStatus int) {
				resp := SM.POST(web.ServiceInstancesURL).WithJSON(postInstanceRequest).
					Expect().
					Status(expectedStatus)

				if resp.Raw().StatusCode == http.StatusCreated {
					resp.JSON().Object().
						ContainsMap(expectedInstanceResponse).ContainsKey("id").
						ValueEqual("platform_id", types.SMPlatform)
				}
			}

			BeforeEach(func() {
				id, err := uuid.NewV4()
				if err != nil {
					panic(err)
				}

				instanceID = id.String()
				name := "test-instance"
				plans := generateServicePlanIDs(ctx, ctx.SMWithOAuth)
				servicePlanID = plans.Element(0).Object().Value("id").String().Raw()
				anotherServicePlanID = plans.Element(1).Object().Value("id").String().Raw()

				postInstanceRequest = common.Object{
					"id":               instanceID,
					"name":             name,
					"service_plan_id":  servicePlanID,
					"maintenance_info": "{}",
				}
				expectedInstanceResponse = common.Object{
					"id":               instanceID,
					"name":             name,
					"service_plan_id":  servicePlanID,
					"maintenance_info": "{}",
				}

			})

			AfterEach(func() {
				ctx.CleanupAdditionalResources()
			})

			Describe("GET", func() {
				var serviceInstance *types.ServiceInstance

				When("service instance contains tenant identifier in OSB context", func() {
					BeforeEach(func() {
						_, serviceInstance = service_instance.Prepare(ctx, ctx.TestPlatform.ID, "", fmt.Sprintf(`{"%s":"%s"}`, TenantIdentifier, TenantIDValue))
						_, err := ctx.SMRepository.Create(context.Background(), serviceInstance)
						Expect(err).ToNot(HaveOccurred())
					})

					It("labels instance with tenant identifier", func() {
						ctx.SMWithOAuth.GET(web.ServiceInstancesURL + "/" + serviceInstance.ID).Expect().
							Status(http.StatusOK).
							JSON().
							Object().Path(fmt.Sprintf("$.labels[%s][*]", TenantIdentifier)).Array().Contains(TenantIDValue)
					})
				})
				When("service instance doesn't contain tenant identifier in OSB context", func() {
					BeforeEach(func() {
						_, serviceInstance = service_instance.Prepare(ctx, ctx.TestPlatform.ID, "", "{}")
						_, err := ctx.SMRepository.Create(context.Background(), serviceInstance)
						Expect(err).ToNot(HaveOccurred())
					})

					It("doesn't label instance with tenant identifier", func() {
						obj := ctx.SMWithOAuth.GET(web.ServiceInstancesURL + "/" + serviceInstance.ID).Expect().
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
						_, serviceInstance = service_instance.Prepare(ctx, ctx.TestPlatform.ID, "", fmt.Sprintf(`{"%s":"%s"}`, TenantIdentifier, TenantIDValue))
						serviceInstance.DashboardURL = ""
						_, err := ctx.SMRepository.Create(context.Background(), serviceInstance)
						Expect(err).ToNot(HaveOccurred())
					})

					It("doesn't return dashboard_url", func() {
						ctx.SMWithOAuth.GET(web.ServiceInstancesURL + "/" + serviceInstance.ID).Expect().
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
							test.EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, servicePlanID, "")
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
							test.EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, postInstanceRequest["service_plan_id"].(string), "")
							createInstance(ctx.SMWithOAuth, http.StatusCreated)
						})
					}

					Context("when id  field is missing", func() {
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

				When("request body id field is invalid", func() {
					It("should return 400", func() {
						test.EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, postInstanceRequest["service_plan_id"].(string), "")
						postInstanceRequest["id"] = "instance/1"
						resp := ctx.SMWithOAuth.POST(web.ServiceInstancesURL).
							WithJSON(postInstanceRequest).
							Expect().Status(http.StatusBadRequest).JSON().Object()

						resp.Value("description").Equal("instance/1 contains invalid character(s)")
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
							test.EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, postInstanceRequest["service_plan_id"].(string), "")
							createInstance(ctx.SMWithOAuth, http.StatusCreated)
						})
					})
				})

				When("async query param", func() {
					It("succeeds", func() {
						test.EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, postInstanceRequest["service_plan_id"].(string), "")
						resp := ctx.SMWithOAuth.POST(web.ServiceInstancesURL).WithJSON(postInstanceRequest).
							WithQuery("async", "true").
							Expect().
							Status(http.StatusAccepted)

						test.ExpectOperation(ctx.SMWithOAuth, resp, types.SUCCEEDED)

						ctx.SMWithOAuth.GET(web.ServiceInstancesURL + "/" + instanceID).Expect().
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
							test.EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, servicePlanID, TenantIDValue)
							createInstance(ctx.SMWithOAuthForTenant, http.StatusCreated)
						})
					})

					When("plan has public visibility", func() {
						It("for global returns 201", func() {
							test.EnsurePublicPlanVisibility(ctx.SMRepository, servicePlanID)
							createInstance(ctx.SMWithOAuth, http.StatusCreated)
						})

						It("for tenant returns 201", func() {
							test.EnsurePublicPlanVisibility(ctx.SMRepository, servicePlanID)
							createInstance(ctx.SMWithOAuthForTenant, http.StatusCreated)
						})
					})
				})
			})

			Describe("PATCH", func() {
				When("content type is not JSON", func() {
					It("returns 415", func() {
						instanceID := fmt.Sprintf("%s", postInstanceRequest["id"])
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
						test.EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, postInstanceRequest["service_plan_id"].(string), "")
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
							test.EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, postInstanceRequest["service_plan_id"].(string), "")
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
							test.EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, postInstanceRequest["service_plan_id"].(string), "")
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
						test.EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, postInstanceRequest["service_plan_id"].(string), "")
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
							test.EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, servicePlanID, TenantIDValue)
							createInstance(ctx.SMWithOAuthForTenant, http.StatusCreated)

							ctx.SMWithOAuthForTenant.PATCH(web.ServiceInstancesURL + "/" + instanceID).
								WithJSON(common.Object{"service_plan_id": anotherServicePlanID}).
								Expect().Status(http.StatusNotFound)
						})
					})

					When("tenant has plan visibility", func() {
						It("returns 201", func() {
							test.EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, servicePlanID, TenantIDValue)
							createInstance(ctx.SMWithOAuthForTenant, http.StatusCreated)

							test.EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, anotherServicePlanID, TenantIDValue)
							ctx.SMWithOAuthForTenant.PATCH(web.ServiceInstancesURL + "/" + instanceID).
								WithJSON(common.Object{"service_plan_id": anotherServicePlanID}).
								Expect().Status(http.StatusOK)
						})
					})
				})

				Context("instance ownership", func() {
					When("tenant doesn't have ownership of instance", func() {
						It("returns 404", func() {
							test.EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, postInstanceRequest["service_plan_id"].(string), "")
							createInstance(ctx.SMWithOAuth, http.StatusCreated)

							ctx.SMWithOAuthForTenant.PATCH(web.ServiceInstancesURL + "/" + instanceID).
								WithJSON(common.Object{"service_plan_id": anotherServicePlanID}).
								Expect().Status(http.StatusNotFound)
						})
					})

					When("tenant has ownership of instance", func() {
						It("returns 200", func() {
							test.EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, servicePlanID, TenantIDValue)
							createInstance(ctx.SMWithOAuthForTenant, http.StatusCreated)

							ctx.SMWithOAuthForTenant.PATCH(web.ServiceInstancesURL + "/" + instanceID).
								WithJSON(common.Object{"platform_id": types.SMPlatform}).
								Expect().Status(http.StatusOK)
						})
					})
				})
			})

			Describe("DELETE", func() {
				Context("instance ownership", func() {
					When("tenant doesn't have ownership of instance", func() {
						It("returns 404", func() {
							test.EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, postInstanceRequest["service_plan_id"].(string), "")
							createInstance(ctx.SMWithOAuth, http.StatusCreated)

							ctx.SMWithOAuthForTenant.DELETE(web.ServiceInstancesURL + "/" + instanceID).
								Expect().Status(http.StatusNotFound)
						})
					})

					When("tenant doesn't have ownership of some instances in bulk delete", func() {
						It("returns 404", func() {
							test.EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, postInstanceRequest["service_plan_id"].(string), "")
							createInstance(ctx.SMWithOAuth, http.StatusCreated)

							ctx.SMWithOAuthForTenant.DELETE(web.ServiceInstancesURL).
								Expect().Status(http.StatusNotFound)
						})
					})

					When("tenant has ownership of instance", func() {
						It("returns 200", func() {
							test.EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, servicePlanID, TenantIDValue)
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
	instanceID, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}

	instanceReqBody := make(common.Object, 0)
	instanceReqBody["id"] = instanceID.String()
	instanceReqBody["name"] = "test-instance-" + instanceID.String()

	instanceReqBody["service_plan_id"] = generateServicePlanIDs(ctx, auth).First().Object().Value("id").String().Raw()

	test.EnsurePlanVisibility(ctx.SMRepository, TenantIdentifier, types.SMPlatform, instanceReqBody["service_plan_id"].(string), "")
	resp := auth.POST(web.ServiceInstancesURL).WithQuery("async", strconv.FormatBool(async)).WithJSON(instanceReqBody).Expect()

	var instance map[string]interface{}
	if async {
		instance = test.ExpectSuccessfulAsyncResourceCreation(resp, auth, instanceID.String(), web.ServiceInstancesURL)
	} else {
		instance = resp.Status(http.StatusCreated).JSON().Object().Raw()
	}

	return instance
}

func generateServicePlanIDs(ctx *common.TestContext, auth *common.SMExpect) *httpexpect.Array {
	cPaidPlan1 := common.GeneratePaidTestPlan()
	cPaidPlan2 := common.GeneratePaidTestPlan()
	cService := common.GenerateTestServiceWithPlans(cPaidPlan1, cPaidPlan2)
	catalog := common.NewEmptySBCatalog()
	catalog.AddService(cService)
	brokerID, _, _ := ctx.RegisterBrokerWithCatalog(catalog)

	so := auth.ListWithQuery(web.ServiceOfferingsURL, fmt.Sprintf("fieldQuery=broker_id eq '%s'", brokerID)).First()

	return auth.ListWithQuery(web.ServicePlansURL, "fieldQuery="+fmt.Sprintf("service_offering_id eq '%s'", so.Object().Value("id").String().Raw()))
}
