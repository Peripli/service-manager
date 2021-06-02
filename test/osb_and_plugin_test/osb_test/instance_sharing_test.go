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
	"encoding/json"
	"fmt"
	"github.com/Peripli/service-manager/pkg/instance_sharing"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/storage"
	"github.com/Peripli/service-manager/test"
	"github.com/Peripli/service-manager/test/common"
	"github.com/gofrs/uuid"
	"net/http"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/web"
	. "github.com/Peripli/service-manager/test/common"
	"github.com/gavv/httpexpect"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Instance Sharing", func() {
	var platform *types.Platform
	var platformJSON common.Object

	JustBeforeEach(func() {
		instanceSharingBrokerServer.ShouldRecordRequests(true)

		platform = common.RegisterPlatformInSM(platformJSON, ctx.SMWithOAuth, map[string]string{})

		instanceSharingUtils.SetAuthContext(ctx.SMWithOAuth).AddPlanVisibilityForPlatform(utils.SelectBroker(&utils.BrokerWithTLS).GetPlanCatalogId(0, 0), platform.ID, organizationGUID)
		instanceSharingUtils.SetAuthContext(ctx.SMWithOAuth).AddPlanVisibilityForPlatform(shareablePlanCatalogID, platform.ID, organizationGUID)

		SMWithBasic := &common.SMExpect{Expect: ctx.SM.Builder(func(req *httpexpect.Request) {
			username, password := platform.Credentials.Basic.Username, platform.Credentials.Basic.Password
			req.WithBasicAuth(username, password).WithClient(ctx.HttpClient)
		})}

		username, password := test.RegisterBrokerPlatformCredentials(SMWithBasic, instanceSharingBrokerID)
		instanceSharingUtils.SetAuthContext(SMWithBasic).RegisterPlatformToBroker(username, password, utils.BrokerWithTLS.ID)
		ctx.SMWithBasic.SetBasicCredentials(ctx, username, password)
	})

	AfterEach(func() {
		err := ctx.SMRepository.Delete(context.TODO(), types.BrokerPlatformCredentialType,
			query.ByField(query.EqualsOperator, "platform_id", platform.ID))
		Expect(err).ToNot(HaveOccurred())

		ctx.SMWithOAuth.DELETE(web.VisibilitiesURL + "?fieldQuery=" + fmt.Sprintf("platform_id eq '%s'", platform.ID))
		ctx.SMWithOAuth.DELETE(web.PlatformsURL + "/" + platform.ID).Expect().Status(http.StatusOK)
	})

	Describe("PROVISION", func() {
		When("provisioning a reference instance in cf platform", func() {
			var sharedInstanceID, referenceInstanceID string
			BeforeEach(func() {
				platformJSON = common.MakePlatform("cf-platform", "cf-platform", "cloudfoundry", "test-platform-cf")
				instanceSharingBrokerServer.ServiceInstanceHandler = parameterizedHandler(http.StatusCreated, `{}`)
				instanceSharingUtils.BrokerWithTLS.BrokerServer.ServiceInstanceHandler = parameterizedHandler(http.StatusCreated, `{}`)
			})
			JustBeforeEach(func() {
				_, sharedInstanceID = createAndShareInstance(false)
				VerifyResourceExists(ctx.SMWithOAuthForTenant, ResourceExpectations{
					ID:    sharedInstanceID,
					Type:  types.ServiceInstanceType,
					Ready: true,
				})
				verifyOperationExists(operationExpectations{
					Type:         types.CREATE,
					State:        types.SUCCEEDED,
					ResourceID:   sharedInstanceID,
					ResourceType: "/v1/service_instances",
					ExternalID:   "",
				})
			})
			JustAfterEach(func() {
				verifyOperationExists(operationExpectations{
					Type:         types.CREATE,
					State:        types.SUCCEEDED,
					ResourceID:   referenceInstanceID,
					ResourceType: "/v1/service_instances",
					ExternalID:   "",
				})
				referenceInstanceObject := VerifyResourceExists(ctx.SMWithOAuthForTenant, ResourceExpectations{
					ID:    referenceInstanceID,
					Type:  types.ServiceInstanceType,
					Ready: true,
				})
				referenceInstanceObject.ContainsKey("platform_id").
					ValueEqual("platform_id", platform.ID)
				referenceInstanceObject.ContainsKey(instance_sharing.ReferencedInstanceIDKey).
					ValueEqual(instance_sharing.ReferencedInstanceIDKey, sharedInstanceID)
			})
			It("returns 201", func() {
				_, referenceInstanceID = createReferenceInstance(platform.ID, instance_sharing.ReferencedInstanceIDKey, sharedInstanceID, false)
			})
			It("returns 202", func() {
				_, referenceInstanceID = createReferenceInstance(platform.ID, instance_sharing.ReferencedInstanceIDKey, sharedInstanceID, true)
			})
		})
		When("provisioning a reference instance in K8S platform", func() {
			BeforeEach(func() {
				platformJSON = common.MakePlatform("k8s-platform", "k8s-platform", "kubernetes", "test-platform-k8s")
			})
			It("creates reference instance successfully", func() {
				createSharedInstanceAndReference(platform, false)
			})
		})
		When("provision request contains shared property", func() {
			BeforeEach(func() {
				platformJSON = common.MakePlatform("k8s-platform", "k8s-platform", "kubernetes", "test-platform-k8s")
			})
			It("should fail provisioning an instance with shared property in the request", func() {
				UUID, err := uuid.NewV4()
				if err != nil {
					panic(err)
				}
				instanceGuid := UUID.String()

				resp := ctx.SMWithBasic.PUT(instanceSharingBrokerPath+"/v2/service_instances/"+instanceGuid).
					WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
					WithQuery(acceptsIncompleteKey, false).
					WithJSON(Object{
						"shared":     true,
						"service_id": service2CatalogID,
						"plan_id":    shareablePlanCatalogID,
						"context": map[string]string{
							"platform": platform.ID,
						},
						"organization_guid": organizationGUID,
						"space_guid":        instanceSharingSpaceGUID,
						"maintenance_info": map[string]string{
							"version": "old",
						},
					}).
					Expect().Status(http.StatusBadRequest)

				VerifyResourceDoesNotExist(ctx.SMWithOAuthForTenant, ResourceExpectations{
					ID:    instanceGuid,
					Type:  types.ServiceInstanceType,
					Ready: true,
				})
				resp.JSON().Object().
					ContainsKey("description").
					ValueEqual("description", util.HandleInstanceSharingError(util.ErrInvalidProvisionRequestWithSharedProperty, "").Error())

			})
		})
		When(fmt.Sprintf("provision request contains %s property", instance_sharing.ReferencedInstanceIDKey), func() {
			var sharedInstanceID string
			var referencePlan *types.ServicePlan
			BeforeEach(func() {
				platformJSON = common.MakePlatform("k8s-platform", "k8s-platform", "kubernetes", "test-platform-k8s")
			})
			JustBeforeEach(func() {
				_, sharedInstanceID = createAndShareInstance(false)
				VerifyResourceExists(ctx.SMWithOAuthForTenant, ResourceExpectations{
					ID:    sharedInstanceID,
					Type:  types.ServiceInstanceType,
					Ready: true,
				})
				verifyOperationExists(operationExpectations{
					Type:         types.CREATE,
					State:        types.SUCCEEDED,
					ResourceID:   sharedInstanceID,
					ResourceType: "/v1/service_instances",
					ExternalID:   "",
				})
				referencePlan = GetReferencePlanOfExistingPlan(ctx, "catalog_id", shareablePlanCatalogID)
				utils.SetAuthContext(ctx.SMWithOAuth).AddPlanVisibilityForPlatform(referencePlan.CatalogID, platform.ID, organizationGUID)
			})
			It("should fail", func() {
				UUID, err := uuid.NewV4()
				if err != nil {
					panic(err)
				}
				instanceGuid := UUID.String()

				resp := ctx.SMWithBasic.PUT(instanceSharingBrokerPath+"/v2/service_instances/"+instanceGuid).
					WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
					WithQuery(acceptsIncompleteKey, false).
					WithJSON(Object{
						instance_sharing.ReferencedInstanceIDKey: sharedInstanceID,
						"service_id":                             service2CatalogID,
						"plan_id":                                referencePlan.CatalogID,
						"context": map[string]string{
							"platform": platform.ID,
						},
						"organization_guid": organizationGUID,
						"space_guid":        instanceSharingSpaceGUID,
						"maintenance_info": map[string]string{
							"version": "old",
						},
					}).
					Expect().Status(http.StatusBadRequest)

				VerifyResourceDoesNotExist(ctx.SMWithOAuthForTenant, ResourceExpectations{
					ID:    instanceGuid,
					Type:  types.ServiceInstanceType,
					Ready: true,
				})
				resp.JSON().Object().
					ContainsKey("description").
					ValueEqual("description", util.HandleInstanceSharingError(util.ErrMissingOrInvalidReferenceParameter, instance_sharing.ReferencedInstanceIDKey).Error())

			})
		})
		Context("selectors", func() {
			When("provisioning reference instance with selectors", func() {
				var resp *httpexpect.Response
				var sharedInstanceID, referenceInstanceID string
				//var sharedInstance *types.ServiceInstance
				//var referencePlan *types.ServicePlan
				//var sharedPlan *types.ServicePlan
				BeforeEach(func() {
					platformJSON = common.MakePlatform("cf-platform", "cf-platform", "cloudfoundry", "test-platform-cf")
					instanceSharingBrokerServer.ServiceInstanceHandler = parameterizedHandler(http.StatusCreated, `{}`)
					instanceSharingUtils.BrokerWithTLS.BrokerServer.ServiceInstanceHandler = parameterizedHandler(http.StatusCreated, `{}`)
				})
				JustBeforeEach(func() {
					_, sharedInstanceID = createAndShareInstance(false)
					VerifyResourceExists(ctx.SMWithOAuthForTenant, ResourceExpectations{
						ID:    sharedInstanceID,
						Type:  types.ServiceInstanceType,
						Ready: true,
					})
					verifyOperationExists(operationExpectations{
						Type:         types.CREATE,
						State:        types.SUCCEEDED,
						ResourceID:   sharedInstanceID,
						ResourceType: "/v1/service_instances",
						ExternalID:   "",
					})
					//sharedInstance, _ = GetInstanceObjectByID(ctx, sharedInstanceID)
					//referencePlan = GetReferencePlanOfExistingPlan(ctx, "catalog_id", shareablePlanCatalogID)
					//sharedPlan = GetPlanByKey(ctx, "catalog_id", shareablePlanCatalogID)
				})
				JustAfterEach(func() {
					referenceInstanceID, _ = VerifyOperationExists(ctx, resp.Header("Location").Raw(), OperationExpectations{
						Category:          types.CREATE,
						State:             types.SUCCEEDED,
						ResourceType:      types.ServiceInstanceType,
						Reschedulable:     false,
						DeletionScheduled: false,
					})
					VerifyResourceExists(ctx.SMWithOAuthForTenant, ResourceExpectations{
						ID:    referenceInstanceID,
						Type:  types.ServiceInstanceType,
						Ready: true,
					})
					deleteInstance(referenceInstanceID, http.StatusOK)
				})
				It("creates reference instance by plan selector", func() {
					shareablePlan := GetPlanByKey(ctx, "catalog_id", shareablePlanCatalogID)
					resp, referenceInstanceID = createReferenceInstance(platform.ID, instance_sharing.ReferencePlanNameSelector, shareablePlan.CatalogName, false)
				})
				It("creates reference instance by name selector", func() {
					sharedInstance, _ := GetInstanceObjectByID(ctx, sharedInstanceID)
					resp, referenceInstanceID = createReferenceInstance(platform.ID, instance_sharing.ReferenceInstanceNameSelector, sharedInstance.Name, false)
				})
				It("creates reference instance by global (*) pointer to a shared instance", func() {
					resp, referenceInstanceID = createReferenceInstance(platform.ID, instance_sharing.ReferencedInstanceIDKey, "*", false)
				})
				It("creates reference instance by label selector", func() {
					labelSelector := Object{TenantIdentifier: Array{TenantValue}}
					resp, referenceInstanceID = createReferenceInstance(platform.ID, instance_sharing.ReferenceLabelSelector, labelSelector, false)
				})
				It("creates reference instance by combination of selectors", func() {
					sharedInstance, _ := GetInstanceObjectByID(ctx, sharedInstanceID)
					shareablePlan := GetPlanByKey(ctx, "catalog_id", shareablePlanCatalogID)

					UUID, err := uuid.NewV4()
					if err != nil {
						panic(err)
					}
					instanceID := UUID.String()

					referencePlan := GetReferencePlanOfExistingPlan(ctx, "catalog_id", shareablePlanCatalogID)
					referenceProvisionBody := buildReferenceProvisionBody(referencePlan.CatalogID, platform.ID)
					referenceProvisionBody["parameters"] = Object{
						TenantIdentifier: Array{TenantValue},
						instance_sharing.ReferenceInstanceNameSelector: sharedInstance.Name,
						instance_sharing.ReferencePlanNameSelector:     shareablePlan.CatalogName,
					}
					utils.SetAuthContext(ctx.SMWithOAuth).AddPlanVisibilityForPlatform(referencePlan.CatalogID, platform.ID, organizationGUID)
					resp = ctx.SMWithBasic.PUT(instanceSharingBrokerPath+"/v2/service_instances/"+instanceID).
						WithQuery(acceptsIncompleteKey, false).
						WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
						WithJSON(referenceProvisionBody).
						Expect().Status(http.StatusCreated)
					resp.Body().Contains("{}")
				})
				When("has more than one shared instance but only 1 is qualified by the selectors", func() {
					JustBeforeEach(func() {
						_, sharedInstanceID2 := createAndShareInstance(false)
						// rename instance to avoid multiple results:
						sharedInstance2, _ := GetInstanceObjectByID(ctx, sharedInstanceID2)
						sharedInstance2.Name = "shared-instance-2"
						ctx.SMRepository.Update(context.TODO(), sharedInstance2, types.LabelChanges{})

						_, sharedInstanceID3 := createAndShareInstance(false)
						// rename instance to avoid multiple results:
						sharedInstance3, _ := GetInstanceObjectByID(ctx, sharedInstanceID3)
						sharedInstance3.Name = "shared-instance-2"
						sharedInstance3.SetLabels(types.Labels{
							TenantIdentifier: {TenantValue},
							"type":           {"dev"},
						})
						ctx.SMRepository.Update(context.TODO(), sharedInstance3, types.LabelChanges{})
						VerifyResourceExists(ctx.SMWithOAuthForTenant, ResourceExpectations{
							ID:    sharedInstanceID2,
							Type:  types.ServiceInstanceType,
							Ready: true,
						})
						verifyOperationExists(operationExpectations{
							Type:         types.CREATE,
							State:        types.SUCCEEDED,
							ResourceID:   sharedInstanceID2,
							ResourceType: "/v1/service_instances",
							ExternalID:   "",
						})
					})
					It("creates reference instance by combination of selectors", func() {
						sharedInstance, _ := GetInstanceObjectByID(ctx, sharedInstanceID)
						shareablePlan := GetPlanByKey(ctx, "catalog_id", shareablePlanCatalogID)

						UUID, err := uuid.NewV4()
						if err != nil {
							panic(err)
						}
						instanceID := UUID.String()

						referencePlan := GetReferencePlanOfExistingPlan(ctx, "catalog_id", shareablePlanCatalogID)
						referenceProvisionBody := buildReferenceProvisionBody(referencePlan.CatalogID, platform.ID)
						referenceProvisionBody["parameters"] = Object{
							TenantIdentifier: Array{TenantValue},
							instance_sharing.ReferenceInstanceNameSelector: sharedInstance.Name,
							instance_sharing.ReferencePlanNameSelector:     shareablePlan.CatalogName,
						}
						utils.SetAuthContext(ctx.SMWithOAuth).AddPlanVisibilityForPlatform(referencePlan.CatalogID, platform.ID, organizationGUID)
						resp = ctx.SMWithBasic.PUT(instanceSharingBrokerPath+"/v2/service_instances/"+instanceID).
							WithQuery(acceptsIncompleteKey, false).
							WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
							WithJSON(referenceProvisionBody).
							Expect().Status(http.StatusCreated)
						resp.Body().Contains("{}")
					})
				})
			})
		})
	})

	Describe("DEPROVISION", func() {
		Context("reference instance", func() {
			var referenceInstanceID string
			JustBeforeEach(func() {
				_, referenceInstanceID = createSharedInstanceAndReference(platform, false)
			})
			When("deleting a reference instance in cf platform", func() {
				BeforeEach(func() {
					platformJSON = common.MakePlatform("cf-platform", "cf-platform", "cloudfoundry", "test-platform-cf")
				})
				AfterEach(func() {
					VerifyResourceDoesNotExist(ctx.SMWithOAuthForTenant, ResourceExpectations{
						ID:   referenceInstanceID,
						Type: types.ServiceInstanceType,
					})
				})
				It("deletes reference instance successfully", func() {
					deleteInstance(referenceInstanceID, http.StatusOK)
				})
			})
			When("deleting a reference instance in K8S platform", func() {
				BeforeEach(func() {
					platformJSON = common.MakePlatform("k8s-platform", "k8s-platform", "kubernetes", "test-platform-k8s")
				})
				It("deletes reference instance successfully", func() {
					deleteInstance(referenceInstanceID, http.StatusOK)
				})
			})
		})
		Context("shared instance", func() {
			var sharedInstanceID string
			BeforeEach(func() {
				platformJSON = common.MakePlatform("cf-platform", "cf-platform", "cloudfoundry", "test-platform-cf")
			})
			JustBeforeEach(func() {
				_, sharedInstanceID = createAndShareInstance(false)
				VerifyResourceExists(ctx.SMWithOAuthForTenant, ResourceExpectations{
					ID:    sharedInstanceID,
					Type:  types.ServiceInstanceType,
					Ready: true,
				})
			})
			When("a shared instance has existing references", func() {
				var referenceInstanceID string
				JustBeforeEach(func() {
					_, referenceInstanceID = createReferenceInstance(platform.ID, instance_sharing.ReferencedInstanceIDKey, sharedInstanceID, false)
					VerifyResourceExists(ctx.SMWithOAuthForTenant, ResourceExpectations{
						ID:    referenceInstanceID,
						Type:  types.ServiceInstanceType,
						Ready: true,
					})
				})
				AfterEach(func() {
					deleteInstance(referenceInstanceID, http.StatusOK)
					VerifyResourceDoesNotExist(ctx.SMWithOAuthForTenant, ResourceExpectations{
						ID:   referenceInstanceID,
						Type: types.ServiceInstanceType,
					})
					deleteInstance(sharedInstanceID, http.StatusOK)
					VerifyResourceDoesNotExist(ctx.SMWithOAuthForTenant, ResourceExpectations{
						ID:   sharedInstanceID,
						Type: types.ServiceInstanceType,
					})
				})
				It("fails to delete the shared instance due to existing references", func() {
					deleteInstance(sharedInstanceID, http.StatusBadRequest)
					VerifyResourceExists(ctx.SMWithOAuthForTenant, ResourceExpectations{
						ID:    referenceInstanceID,
						Type:  types.ServiceInstanceType,
						Ready: true,
					})
					VerifyResourceExists(ctx.SMWithOAuthForTenant, ResourceExpectations{
						ID:    sharedInstanceID,
						Type:  types.ServiceInstanceType,
						Ready: true,
					})
				})
			})
			When("a shared instance does not have references", func() {
				AfterEach(func() {
					VerifyResourceDoesNotExist(ctx.SMWithOAuthForTenant, ResourceExpectations{
						ID:   sharedInstanceID,
						Type: types.ServiceInstanceType,
					})
				})
				It("deletes the shared instance successfully", func() {
					deleteInstance(sharedInstanceID, http.StatusOK)
				})
			})
		})
	})

	Describe("PATCH", func() {
		When("updating a reference instance in cf platform", func() {
			var referenceInstanceID string
			BeforeEach(func() {
				platformJSON = common.MakePlatform("cf-platform", "cf-platform", "cloudfoundry", "test-platform-cf")
			})
			It("should fail updating the reference instance plan", func() {
				_, referenceInstanceID = createSharedInstanceAndReference(platform, false)
				referencePlan := GetReferencePlanOfExistingPlan(ctx, "catalog_id", shareablePlanCatalogID)

				body := generateUpdateRequestBody(service2CatalogID, referencePlan.CatalogID, shareablePlanCatalogID, "new-reference-name")()
				resp := updateInstance(referenceInstanceID, body, http.StatusBadRequest)
				resp.JSON().Object().Equal(util.HandleInstanceSharingError(util.ErrChangingPlanOfReferenceInstance, referenceInstanceID))
			})
			It("should fail updating the reference instance parameters", func() {
				_, referenceInstanceID = createSharedInstanceAndReference(platform, false)
				referencePlan := GetReferencePlanOfExistingPlan(ctx, "catalog_id", shareablePlanCatalogID)

				body := generateUpdateRequestBody(service2CatalogID, referencePlan.CatalogID, referencePlan.CatalogID, "new-reference-name")()
				body["parameters"] = map[string]string{
					instance_sharing.ReferencedInstanceIDKey: "fake-guid",
				}
				resp := updateInstance(referenceInstanceID, body, http.StatusBadRequest)
				resp.JSON().Object().Equal(util.HandleInstanceSharingError(util.ErrChangingParametersOfReferenceInstance, referenceInstanceID))
			})
			It("should succeed updating the reference instance name", func() {
				_, referenceInstanceID = createSharedInstanceAndReference(platform, false)
				referencePlan := GetReferencePlanOfExistingPlan(ctx, "catalog_id", shareablePlanCatalogID)
				body := generateUpdateRequestBody(service2CatalogID, referencePlan.CatalogID, referencePlan.CatalogID, "new-reference-name")()
				updateInstance(referenceInstanceID, body, http.StatusOK)
			})
		})
		When("updating a shared instance in cf platform", func() {
			platformID := "cf-platform"
			var sharedInstanceID string
			BeforeEach(func() {
				platformJSON = common.MakePlatform(platformID, platformID, "cloudfoundry", "test-platform-cf")
			})
			JustBeforeEach(func() {
				_, sharedInstanceID = createAndShareInstance(false)
				VerifyResourceExists(ctx.SMWithOAuthForTenant, ResourceExpectations{
					ID:    sharedInstanceID,
					Type:  types.ServiceInstanceType,
					Ready: true,
				})
			})
			It("should succeed updating the shared instance name", func() {
				body := generateUpdateRequestBody(service2CatalogID, shareablePlanCatalogID, shareablePlanCatalogID, "new-shared-instance-name")()
				updateInstance(sharedInstanceID, body, http.StatusOK)
			})
			It("should fail updating the shared instance plan to a non shareable plan", func() {
				referencePlan := GetReferencePlanOfExistingPlan(ctx, "catalog_id", shareablePlanCatalogID)
				utils.SetAuthContext(ctx.SMWithOAuth).AddPlanVisibilityForPlatform(referencePlan.CatalogID, platformID, organizationGUID)

				body := generateUpdateRequestBody(service2CatalogID, shareablePlanCatalogID, referencePlan.CatalogID, "renamed-shared-instance")()
				resp := updateInstance(sharedInstanceID, body, http.StatusBadRequest)
				resp.JSON().Object().Equal(util.HandleInstanceSharingError(util.ErrNewPlanDoesNotSupportInstanceSharing, sharedInstanceID))
			})
			It("should succeed updating the shared instance plan to a new shareable plan", func() {
				utils.SetAuthContext(ctx.SMWithOAuth).AddPlanVisibilityForPlatform(shareablePlan2CatalogID, platformID, organizationGUID)

				body := generateUpdateRequestBody(service2CatalogID, shareablePlanCatalogID, shareablePlan2CatalogID, "renamed-shared-instance")()
				updateInstance(sharedInstanceID, body, http.StatusOK)
				dbNewPlanObject, _ := storage.GetObjectByField(context.TODO(), ctx.SMRepository, types.ServicePlanType, "catalog_id", shareablePlan2CatalogID)
				newPlanObject := dbNewPlanObject.(*types.ServicePlan)
				instance, _ := GetInstanceObjectByID(ctx, sharedInstanceID)
				Expect(newPlanObject.ID).To(Equal(instance.ServicePlanID))
				Expect(newPlanObject.CatalogID).To(Equal(shareablePlan2CatalogID))
			})
		})
	})

	Describe("POLL BINDING", func() {
		When("broker returns async response", func() {
			var sharedInstanceID, referenceInstanceID, referenceBindingID string
			BeforeEach(func() {
				platformJSON = common.MakePlatform("cf-platform", "cf-platform", "cloudfoundry", "test-platform-cf")
				instanceSharingBrokerServer.BindingHandlerFunc("PUT", "async-put-response", func(req *http.Request) (int, map[string]interface{}) {
					Expect(req.RequestURI)
					return http.StatusAccepted, Object{
						"async": true,
					}
				})
				instanceSharingBrokerServer.BindingHandlerFunc("GET", "async-get-response", func(req *http.Request) (int, map[string]interface{}) {
					Expect(req.RequestURI)
					return http.StatusOK, Object{
						"async": true,
						"credentials": Object{
							"user":     "user",
							"password": "password",
						},
					}
				})
			})
			JustBeforeEach(func() {
				sharedInstanceID, referenceInstanceID = createSharedInstanceAndReference(platform, false)
				referenceBindingID = createBinding(referenceInstanceID, sharedInstanceID, http.StatusAccepted, "true")
			})
			It("returns the credentials of the binding", func() {
				// 1. Create binding -> expected to get 202 status code.
				// 2. Polling request -> expected to get status code OK (200) with operation of succeeded.
				// 3. Get binding -> binding is created and retrieved from the server.
				path := fmt.Sprintf("%s/v2/service_instances/%s/service_bindings/%s/last_operation", instanceSharingBrokerPath, referenceInstanceID, referenceBindingID)
				ctx.SMWithBasic.GET(path).
					WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
					Expect().Status(http.StatusOK)

				path = fmt.Sprintf("/v2/service_instances/%s/service_bindings/%s/last_operation", sharedInstanceID, referenceBindingID)
				Expect(instanceSharingBrokerServer.LastRequest.RequestURI).To(Equal(path))

				path = fmt.Sprintf("%s/v2/service_instances/%s/service_bindings/%s", instanceSharingBrokerPath, referenceInstanceID, referenceBindingID)
				object := ctx.SMWithBasic.GET(path).
					WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
					Expect().Status(http.StatusOK).JSON().Object()
				object.ContainsKey("credentials")

				path = fmt.Sprintf("/v2/service_instances/%s/service_bindings/%s", sharedInstanceID, referenceBindingID)
				Expect(instanceSharingBrokerServer.LastRequest.RequestURI).To(Equal(path))
			})
		})
	})

	Describe("BIND", func() {
		When("binding a shared instance", func() {
			var sharedInstanceID string
			BeforeEach(func() {
				platformJSON = common.MakePlatform("cf-platform", "cf-platform", "cloudfoundry", "test-platform-cf")
			})
			JustBeforeEach(func() {
				sharedInstanceID, _ = createSharedInstanceAndReference(platform, false)
			})
			When("broker returns async response", func() {
				BeforeEach(func() {
					instanceSharingBrokerServer.BindingHandlerFunc("PUT", "async-response", func(req *http.Request) (int, map[string]interface{}) {
						return http.StatusAccepted, Object{
							"async": true,
						}
					})
				})
				It("returns 202", func() {
					createBinding(sharedInstanceID, sharedInstanceID, http.StatusAccepted, "true")
				})
			})
			It("creates bindings successfully async=true", func() {
				bindingID := createBinding(sharedInstanceID, sharedInstanceID, http.StatusCreated, "true")

				lastRequest := instanceSharingBrokerServer.LastRequest
				Expect(lastRequest.RequestURI).To(ContainSubstring(sharedInstanceID))
				Expect(lastRequest.Method).To(ContainSubstring("PUT"))

				ctx.SMWithOAuth.GET(web.ServiceBindingsURL+"/"+bindingID).
					Expect().
					Status(http.StatusOK).
					JSON().
					Object().
					ContainsKey("service_instance_id").
					ValueEqual("service_instance_id", sharedInstanceID)

				// verify not communicating the service broker after the get request.
				lastRequest = instanceSharingBrokerServer.LastRequest
				Expect(lastRequest.RequestURI).To(ContainSubstring(sharedInstanceID))
				Expect(lastRequest.Method).To(ContainSubstring("PUT"))
			})
			It("creates bindings successfully async=false", func() {
				bindingID := createBinding(sharedInstanceID, sharedInstanceID, http.StatusCreated, "false")

				lastRequest := instanceSharingBrokerServer.LastRequest
				Expect(lastRequest.RequestURI).To(ContainSubstring(sharedInstanceID))
				Expect(lastRequest.Method).To(ContainSubstring("PUT"))

				ctx.SMWithOAuth.GET(web.ServiceBindingsURL+"/"+bindingID).
					Expect().
					Status(http.StatusOK).
					JSON().
					Object().
					ContainsKey("service_instance_id").
					ValueEqual("service_instance_id", sharedInstanceID)

				// verify not communicating the service broker after the get request.
				lastRequest = instanceSharingBrokerServer.LastRequest
				Expect(lastRequest.RequestURI).To(ContainSubstring(sharedInstanceID))
				Expect(lastRequest.Method).To(ContainSubstring("PUT"))
			})
		})
		When("binding a reference instance in cf platform", func() {
			BeforeEach(func() {
				platformJSON = common.MakePlatform("cf-platform", "cf-platform", "cloudfoundry", "test-platform-cf")
			})
			It("creates bindings successfully", func() {
				sharedInstanceID, referenceInstanceID := createSharedInstanceAndReference(platform, false)
				bindingID := createBinding(referenceInstanceID, sharedInstanceID, http.StatusCreated, "false")

				lastRequest := instanceSharingBrokerServer.LastRequest
				Expect(lastRequest.RequestURI).To(ContainSubstring(sharedInstanceID))
				Expect(lastRequest.Method).To(ContainSubstring("PUT"))

				ctx.SMWithOAuth.GET(web.ServiceBindingsURL+"/"+bindingID).
					Expect().
					Status(http.StatusOK).
					JSON().
					Object().
					ContainsKey("service_instance_id").
					ValueEqual("service_instance_id", referenceInstanceID)

				// verify not communicating the service broker after the get request.
				lastRequest = instanceSharingBrokerServer.LastRequest
				Expect(lastRequest.RequestURI).To(ContainSubstring(sharedInstanceID))
				Expect(lastRequest.Method).To(ContainSubstring("PUT"))
			})
			When("a shared instance is consumed from a different platform", func() {
				var k8sPlatform *types.Platform
				var cfSharedInstanceID string
				var k8sReferenceInstanceID string
				BeforeEach(func() {
					k8SPlatformJSON := common.MakePlatform("k8s-platform", "k8s-platform", "kubernetes", "test-platform-k8s-test5")
					k8sPlatform = common.RegisterPlatformInSM(k8SPlatformJSON, ctx.SMWithOAuth, map[string]string{})
				})
				JustBeforeEach(func() {
					_, cfSharedInstanceID = createAndShareInstance(false)
					_, k8sReferenceInstanceID = createReferenceInstance(k8sPlatform.ID, instance_sharing.ReferencedInstanceIDKey, cfSharedInstanceID, false)
				})
				AfterEach(func() {
					ctx.SMWithOAuth.DELETE(web.PlatformsURL + "/" + k8sPlatform.ID).Expect().Status(http.StatusOK)
				})
				It("binds reference instance successfully from different platform", func() {
					bindingID := createBinding(k8sReferenceInstanceID, cfSharedInstanceID, http.StatusCreated, "false")
					Expect(instanceSharingBrokerServer.LastRequest.RequestURI).To(ContainSubstring(cfSharedInstanceID))
					ctx.SMWithOAuth.GET(web.ServiceBindingsURL+"/"+bindingID).
						Expect().
						Status(http.StatusOK).
						JSON().
						Object().ContainsKey("service_instance_id").
						ValueEqual("service_instance_id", k8sReferenceInstanceID)

					verifyOperationExists(operationExpectations{
						Type:         types.CREATE,
						State:        types.SUCCEEDED,
						ResourceID:   bindingID,
						ResourceType: "/v1/service_bindings",
						ExternalID:   "",
					})

					VerifyResourceExists(ctx.SMWithOAuthForTenant, ResourceExpectations{
						ID:    bindingID,
						Type:  types.ServiceBindingType,
						Ready: true,
					})

				})
			})
		})
		When("binding a reference instance in k8s platform", func() {
			BeforeEach(func() {
				UUID, _ := uuid.NewV4()
				platformJSON = common.MakePlatform(UUID.String(), UUID.String(), "kubernetes", "test-platform-k8s")
			})
			It("binds reference instance successfully", func() {
				instanceSharingBrokerServer.ShouldRecordRequests(true)

				provisionRequestBody = buildRequestBody(shareablePlanCatalogID, service2CatalogID, platform.ID, "shared-instance")
				sharedInstanceID, referenceInstanceID := createSharedInstanceAndReference(platform, false)
				provisionRequestBody = buildRequestBody(service1CatalogID, plan1CatalogID, platform.ID, "reference-instance")
				bindingID := createBinding(referenceInstanceID, sharedInstanceID, http.StatusCreated, "false")

				// The last broker request should be the "PUT" binding request:
				lastRequest := instanceSharingBrokerServer.LastRequest
				Expect(lastRequest.RequestURI).To(ContainSubstring(sharedInstanceID))
				Expect(lastRequest.Method).To(Equal("PUT"))

				// validate request body.context when creating the binding
				jsonBody := Object{}
				json.Unmarshal(instanceSharingBrokerServer.LastRequestBody, &jsonBody)
				sharedInstance, _ := GetInstanceObjectByID(ctx, sharedInstanceID)
				expectedContextToBroker := Object{
					"space_name":        "development",
					TenantIdentifier:    TenantValue,
					"instance_name":     sharedInstance.Name,
					"organization_guid": organizationGUID,
					"organization_name": "system",
					"platform":          sharedInstance.PlatformID,
					"space_guid":        instanceSharingSpaceGUID,
				}
				Expect(jsonBody["context"]).To(Equal(expectedContextToBroker))

				// The new binding should be registered in sm under the reference instance.
				object := ctx.SMWithOAuth.GET(web.ServiceBindingsURL + "/" + bindingID).
					Expect().
					Status(http.StatusOK).
					JSON().
					Object()
				object.ContainsKey("service_instance_id").
					ValueEqual("service_instance_id", referenceInstanceID)

				// validate the context of the binding is saved in the db with the reference data
				referenceInstance, _ := GetInstanceObjectByID(ctx, referenceInstanceID)
				expectedContextFromDB := Object{
					"space_name":        "development",
					TenantIdentifier:    TenantValue,
					"instance_name":     referenceInstance.Name,
					"organization_guid": organizationGUID,
					"organization_name": "system",
					"platform":          referenceInstance.PlatformID,
					"space_guid":        instanceSharingSpaceGUID,
				}
				object.ContainsKey("context").
					ValueEqual("context", expectedContextFromDB)

			})
		})
	})

	Describe("UNBIND", func() {
		When("unbinding a reference instance in cf platform", func() {
			var sharedInstanceID, referenceInstanceID, bindingID string
			BeforeEach(func() {
				platformJSON = common.MakePlatform("cf-platform", "cf-platform", "cloudfoundry", "test-platform-cf")
			})
			JustBeforeEach(func() {
				sharedInstanceID, referenceInstanceID = createSharedInstanceAndReference(platform, false)
				bindingID = createBinding(referenceInstanceID, sharedInstanceID, http.StatusCreated, "false")
				ctx.SMWithOAuth.GET(web.ServiceBindingsURL+"/"+bindingID).
					Expect().
					Status(http.StatusOK).
					JSON().
					Object().ContainsKey("service_instance_id").
					ValueEqual("service_instance_id", referenceInstanceID)
			})
			It("deletes reference binding successfully", func() {
				ctx.SMWithBasic.DELETE(instanceSharingBrokerURL+"/v2/service_instances/"+referenceInstanceID+"/service_bindings/"+bindingID).
					WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
					Expect().
					Status(http.StatusOK)

				ctx.SMWithOAuth.GET(web.ServiceBindingsURL + "/" + bindingID).
					Expect().Status(http.StatusNotFound)

				verifyOperationExists(operationExpectations{
					Type:         types.DELETE,
					State:        types.SUCCEEDED,
					ResourceID:   bindingID,
					ResourceType: "/v1/service_bindings",
					ExternalID:   "",
				})
			})
		})
	})

	Describe("FETCH SERVICE", func() {
		var sharedInstanceID, referenceInstanceID string
		BeforeEach(func() {
			platformJSON = common.MakePlatform("cf-platform", "cf-platform", "cloudfoundry", "test-platform-cf")
		})
		JustBeforeEach(func() {
			sharedInstanceID, referenceInstanceID = createSharedInstanceAndReference(platform, false)
		})
		It("smaap api returns the shared instance object with shared property", func() {
			_, sharedInstanceID := createAndShareInstance(false)
			VerifyResourceExists(ctx.SMWithOAuthForTenant, ResourceExpectations{
				ID:    sharedInstanceID,
				Type:  types.ServiceInstanceType,
				Ready: true,
			})
			// shared instance
			object := ctx.SMWithOAuthForTenant.GET(web.ServiceInstancesURL + "/" + sharedInstanceID).
				Expect().Status(http.StatusOK).
				JSON().Object()
			// should contain shared property
			object.ContainsKey("shared").
				ValueEqual("shared", true)
		})
		It("fetches reference instance successfully", func() {
			resp := ctx.SMWithBasic.GET(instanceSharingBrokerPath+"/v2/service_instances/"+referenceInstanceID).
				WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				Expect().
				Status(http.StatusOK)
			referencePlan := GetReferencePlanOfExistingPlan(ctx, "catalog_id", shareablePlanCatalogID)
			object := resp.JSON().Object()
			object.Value("plan_id").String().Contains(referencePlan.ID)
			object.Value("service_id").String().Contains(service2CatalogID)
			object.Value("parameters").Object().Value("referenced_instance_id").String().Contains(sharedInstanceID)
		})
		It("communicates the broker with the correct shared instance id", func() {
			ctx.SMWithBasic.GET(instanceSharingBrokerPath+"/v2/service_instances/"+sharedInstanceID).
				WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
				Expect().
				Status(http.StatusOK)
			Expect(instanceSharingBrokerServer.LastRequest.RequestURI).To(ContainSubstring(sharedInstanceID))
		})
	})
})

func createBinding(sourceInstanceID, targetInstanceID string, expectedStatus int, acceptsIncomplete string) string {
	UUID, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}
	bindingID := UUID.String()

	body := provisionRequestBodyMap()()
	resp := ctx.SMWithBasic.PUT(instanceSharingBrokerURL+"/v2/service_instances/"+sourceInstanceID+"/service_bindings/"+bindingID).
		WithQuery(acceptsIncompleteKey, acceptsIncomplete).
		WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
		WithJSON(body).
		Expect()

	resp.Status(expectedStatus)

	Expect(instanceSharingBrokerServer.LastRequest.RequestURI).To(ContainSubstring(targetInstanceID))
	Expect(instanceSharingBrokerServer.LastRequest.Method).To(ContainSubstring("PUT"))

	return bindingID
}

func deleteInstance(instanceID string, expectedStatus int) *httpexpect.Response {
	resp := ctx.SMWithBasic.DELETE(instanceSharingBrokerURL+"/v2/service_instances/"+instanceID).
		WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
		WithQuery(acceptsIncompleteKey, "false")
	return resp.Expect().Status(expectedStatus)
}

func updateInstance(instanceID string, json map[string]interface{}, expectedStatus int) *httpexpect.Response {
	resp := ctx.SMWithBasic.PATCH(instanceSharingBrokerURL+"/v2/service_instances/"+instanceID).
		WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
		WithJSON(json).
		WithQuery(acceptsIncompleteKey, "false")
	return resp.Expect().Status(expectedStatus)
}

func createSharedInstanceAndReference(platform *types.Platform, async bool) (string, string) {
	_, sharedInstanceID := createAndShareInstance(async)
	VerifyResourceExists(ctx.SMWithOAuthForTenant, ResourceExpectations{
		ID:    sharedInstanceID,
		Type:  types.ServiceInstanceType,
		Ready: true,
	})
	_, referenceInstanceID := createReferenceInstance(platform.ID, instance_sharing.ReferencedInstanceIDKey, sharedInstanceID, async)
	obj := VerifyResourceExists(ctx.SMWithOAuthForTenant, ResourceExpectations{
		ID:    referenceInstanceID,
		Type:  types.ServiceInstanceType,
		Ready: true,
	})
	obj.ContainsKey("platform_id").
		ValueEqual("platform_id", platform.ID)
	return sharedInstanceID, referenceInstanceID
}

func createReferenceInstance(platformID, selectorKey string, selectorValue interface{}, accepts_incomplete bool) (*httpexpect.Response, string) {
	UUID, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}
	instanceID := UUID.String()

	referencePlan := GetReferencePlanOfExistingPlan(ctx, "catalog_id", shareablePlanCatalogID)
	referenceProvisionBody := buildReferenceProvisionBody(referencePlan.CatalogID, platformID)
	referenceProvisionBody["parameters"] = Object{selectorKey: selectorValue}
	utils.SetAuthContext(ctx.SMWithOAuth).AddPlanVisibilityForPlatform(referencePlan.CatalogID, platformID, organizationGUID)
	resp := ctx.SMWithBasic.PUT(instanceSharingBrokerPath+"/v2/service_instances/"+instanceID).
		WithQuery(acceptsIncompleteKey, accepts_incomplete).
		WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
		WithJSON(referenceProvisionBody).
		Expect().Status(http.StatusCreated)
	resp.Body().Contains("{}")

	// validate broker communications:
	if selectorKey == instance_sharing.ReferencedInstanceIDKey && selectorValue != "*" {
		sprintf := fmt.Sprintf("%v", selectorValue)
		Expect(instanceSharingBrokerServer.LastRequest.RequestURI).To(ContainSubstring(sprintf))
	}
	Expect(instanceSharingBrokerServer.LastRequest.Method).To(ContainSubstring("PUT"))

	if selectorKey == instance_sharing.ReferencedInstanceIDKey {
		referenceInstance, _ := GetInstanceObjectByID(ctx, instanceID)
		Expect(referenceInstance.ReferencedInstanceID).To(Equal(selectorValue))
	}

	return resp, instanceID
}

func createAndShareInstance(accepts_incomplete bool) (*httpexpect.Response, string) {
	UUID, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}
	sharedInstanceID := UUID.String()

	json := provisionRequestBodyMapWith("plan_id", shareablePlanCatalogID)()
	json = provisionRequestBodyMapWith("service_id", service2CatalogID)()
	resp := ctx.SMWithBasic.PUT(instanceSharingBrokerPath+"/v2/service_instances/"+sharedInstanceID).
		WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
		WithQuery(acceptsIncompleteKey, accepts_incomplete).
		WithJSON(json).
		Expect().Status(http.StatusCreated)
	resp.Body().Contains("{}")
	err = ShareInstanceOnDB(ctx, sharedInstanceID)
	Expect(err).NotTo(HaveOccurred())
	return resp, sharedInstanceID
}

func buildReferenceProvisionBody(planID, platformID string) Object {
	return Object{
		"service_id":        service2CatalogID,
		"plan_id":           planID,
		"organization_guid": organizationGUID,
		"space_guid":        instanceSharingSpaceGUID,
		"context": Object{
			"platform":          platformID,
			"organization_guid": organizationGUID,
			"organization_name": "system",
			"space_guid":        instanceSharingSpaceGUID,
			"space_name":        "development",
			"instance_name":     "reference-instance",
			TenantIdentifier:    TenantValue,
		},
		"maintenance_info": Object{
			"version": "old",
		},
	}
}
