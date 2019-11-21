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
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/gofrs/uuid"
	"net/http"
	"testing"
	"time"

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
	TenantValue      = "tenant_value"
)

var _ = test.DescribeTestsFor(test.TestCase{
	API: web.ServiceInstancesURL,
	SupportedOps: []test.Op{
		test.Get, test.List,
	},
	MultitenancySettings: &test.MultitenancySettings{
		ClientID:           "tenancyClient",
		ClientIDTokenClaim: "cid",
		TenantTokenClaim:   "zid",
		LabelKey:           TenantIdentifier,
		TokenClaims: map[string]interface{}{
			"cid": "tenancyClient",
			"zid": "tenantID",
		},
	},
	ResourceType:                           types.ServiceInstanceType,
	DisableTenantResources:                 true,
	ResourceBlueprint:                      blueprint,
	ResourceWithoutNullableFieldsBlueprint: blueprint,
	PatchResource: func(ctx *common.TestContext, apiPath string, objID string, resourceType types.ObjectType, patchLabels []*query.LabelChange) {
		byID := query.ByField(query.EqualsOperator, "id", objID)
		si, err := ctx.SMRepository.Get(context.Background(), resourceType, byID)
		if err != nil {
			Fail(fmt.Sprintf("unable to retrieve resource %s: %s", resourceType, err))
		}

		_, err = ctx.SMRepository.Update(context.Background(), si, patchLabels)
		if err != nil {
			Fail(fmt.Sprintf("unable to update resource %s: %s", resourceType, err))
		}
	},
	AdditionalTests: func(ctx *common.TestContext) {
		Context("additional non-generic tests", func() {
			Describe("GET", func() {
				var serviceInstance *types.ServiceInstance

				AfterEach(func() {
					ctx.CleanupAdditionalResources()
				})

				When("service instance contains tenant identifier in OSB context", func() {
					BeforeEach(func() {
						serviceInstance = prepareServiceInstance(ctx, ctx.SMWithOAuth, fmt.Sprintf(`{"%s":"%s"}`, TenantIdentifier, TenantValue))
						ctx.SMRepository.Create(context.Background(), serviceInstance)
					})

					It("labels instance with tenant identifier", func() {
						ctx.SMWithOAuth.GET(web.ServiceInstancesURL + "/" + serviceInstance.ID).Expect().
							Status(http.StatusOK).
							JSON().
							Object().Path(fmt.Sprintf("$.labels[%s][*]", TenantIdentifier)).Array().Contains(TenantValue)
					})
				})
				When("service instance doesn't contain tenant identifier in OSB context", func() {
					BeforeEach(func() {
						serviceInstance = prepareServiceInstance(ctx, ctx.SMWithOAuth, "{}")
						ctx.SMRepository.Create(context.Background(), serviceInstance)
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
			})
		})
	},
})

func blueprint(ctx *common.TestContext, auth *common.SMExpect) common.Object {
	serviceInstance := prepareServiceInstance(ctx, auth, fmt.Sprintf(`{"%s":"%s"}`, TenantIdentifier, TenantValue))
	ctx.SMRepository.Create(context.Background(), serviceInstance)

	return auth.ListWithQuery(web.ServiceInstancesURL, fmt.Sprintf("fieldQuery=id eq '%s'", serviceInstance.ID)).First().Object().Raw()
}

func prepareServiceInstance(ctx *common.TestContext, auth *common.SMExpect, OSBContext string) *types.ServiceInstance {
	cService := common.GenerateTestServiceWithPlans(common.GenerateFreeTestPlan())
	catalog := common.NewEmptySBCatalog()
	catalog.AddService(cService)
	id, _, _ := ctx.RegisterBrokerWithCatalog(catalog)

	byBrokerID := query.ByField(query.EqualsOperator, "broker_id", id)
	obj, err := ctx.SMRepository.Get(context.Background(), types.ServiceOfferingType, byBrokerID)
	if err != nil {
		Fail(fmt.Sprintf("unable to fetch service offering: %s", err))
	}

	byServiceOfferingID := query.ByField(query.EqualsOperator, "service_offering_id", obj.GetID())
	obj, err = ctx.SMRepository.Get(context.Background(), types.ServicePlanType, byServiceOfferingID)
	if err != nil {
		Fail(fmt.Sprintf("unable to service plan: %s", err))
	}

	planID := obj.GetID()

	instanceID, err := uuid.NewV4()
	if err != nil {
		Fail(fmt.Sprintf("failed to generate instance GUID: %s", err))
	}

	return &types.ServiceInstance{
		Base: types.Base{
			ID:        instanceID.String(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Name:          "test-service-instance",
		ServicePlanID: planID,
		PlatformID:    ctx.TestPlatform.ID,
		Context:       []byte(OSBContext),
		Ready:         true,
		Usable:        true,
	}
}
