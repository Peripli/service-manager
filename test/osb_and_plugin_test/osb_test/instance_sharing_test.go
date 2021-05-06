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
	"github.com/Peripli/service-manager/pkg/instance_sharing"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/util"
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
				_, referenceInstanceID = createReferenceInstance(platform.ID, sharedInstanceID, false)
			})
			It("returns 202", func() {
				_, referenceInstanceID = createReferenceInstance(platform.ID, sharedInstanceID, true)
			})
		})
		When("provisioning a reference instance in K8S platform", func() {
			BeforeEach(func() {
				platformJSON = common.MakePlatform("k8s-platform", "k8s-platform", "kubernetes", "test-platform-k8s")
			})
			It("creates reference instance successfully", func() {
				createSharedInstanceAndReference(platform, false)
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
					_, referenceInstanceID = createReferenceInstance(platform.ID, sharedInstanceID, false)
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
			It("should fail updating the shared instance plan", func() {
				referencePlan := GetReferencePlanOfExistingPlan(ctx, "catalog_id", shareablePlanCatalogID)
				utils.SetAuthContext(ctx.SMWithOAuth).AddPlanVisibilityForPlatform(referencePlan.CatalogID, platformID, organizationGUID)

				body := generateUpdateRequestBody(service2CatalogID, shareablePlanCatalogID, referencePlan.CatalogID, "renamed-shared-instance")()
				resp := updateInstance(sharedInstanceID, body, http.StatusBadRequest)
				resp.JSON().Object().Equal(util.HandleInstanceSharingError(util.ErrChangingPlanOfSharedInstance, sharedInstanceID))
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
					_, k8sReferenceInstanceID = createReferenceInstance(k8sPlatform.ID, cfSharedInstanceID, false)
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
			When("a reference instance exists in k8s platform", func() {
				BeforeEach(func() {
					UUID, _ := uuid.NewV4()
					platformJSON = common.MakePlatform(UUID.String(), UUID.String(), "kubernetes", "test-platform-k8s")
				})
				It("binds reference instance successfully", func() {
					sharedInstanceID, referenceInstanceID := createSharedInstanceAndReference(platform, false)
					bindingID := createBinding(referenceInstanceID, sharedInstanceID, http.StatusCreated, "false")

					// The last broker request should be the "PUT" binding request:
					lastRequest := instanceSharingBrokerServer.LastRequest
					Expect(lastRequest.RequestURI).To(ContainSubstring(sharedInstanceID))
					Expect(lastRequest.Method).To(Equal("PUT"))

					// The new binding should be registered in sm under the reference instance.
					ctx.SMWithOAuth.GET(web.ServiceBindingsURL+"/"+bindingID).
						Expect().
						Status(http.StatusOK).
						JSON().
						Object().ContainsKey("service_instance_id").
						ValueEqual("service_instance_id", referenceInstanceID)

				})
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
	_, referenceInstanceID := createReferenceInstance(platform.ID, sharedInstanceID, async)
	obj := VerifyResourceExists(ctx.SMWithOAuthForTenant, ResourceExpectations{
		ID:    referenceInstanceID,
		Type:  types.ServiceInstanceType,
		Ready: true,
	})
	obj.ContainsKey("platform_id").
		ValueEqual("platform_id", platform.ID)
	return sharedInstanceID, referenceInstanceID
}

func createReferenceInstance(platformID, sharedInstanceID string, accepts_incomplete bool) (*httpexpect.Response, string) {
	UUID, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}
	instanceID := UUID.String()

	referencePlan := GetReferencePlanOfExistingPlan(ctx, "catalog_id", shareablePlanCatalogID)
	referenceProvisionBody := buildReferenceProvisionBody(referencePlan.CatalogID, sharedInstanceID)
	utils.SetAuthContext(ctx.SMWithOAuth).AddPlanVisibilityForPlatform(referencePlan.CatalogID, platformID, organizationGUID)
	resp := ctx.SMWithBasic.PUT(instanceSharingBrokerPath+"/v2/service_instances/"+instanceID).
		WithQuery(acceptsIncompleteKey, accepts_incomplete).
		WithHeader(brokerAPIVersionHeaderKey, brokerAPIVersionHeaderValue).
		WithJSON(referenceProvisionBody).
		Expect().Status(http.StatusCreated)
	resp.Body().Contains("{}")
	Expect(instanceSharingBrokerServer.LastRequest.RequestURI).To(ContainSubstring(sharedInstanceID))
	Expect(instanceSharingBrokerServer.LastRequest.Method).To(ContainSubstring("PUT"))
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

func buildReferenceProvisionBody(planID, sharedInstanceID string) Object {
	return Object{
		"service_id":        service2CatalogID,
		"plan_id":           planID,
		"organization_guid": organizationGUID,
		"space_guid":        instanceSharingSpaceGUID,
		"parameters": Object{
			instance_sharing.ReferencedInstanceIDKey: sharedInstanceID,
		},
		"context": Object{
			"platform":          "cloudfoundry",
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
