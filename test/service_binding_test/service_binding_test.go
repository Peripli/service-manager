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

	"github.com/gofrs/uuid"

	"testing"

	"github.com/Peripli/service-manager/test/testutil/service_binding"

	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"

	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/test/common"

	"github.com/Peripli/service-manager/test"

	. "github.com/onsi/ginkgo"

	. "github.com/onsi/gomega"
)

func TestServiceBindings(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Bindings Tests Suite")
}

const (
	TenantIdentifier = "tenant"
	TenantValue      = "tenant_value"
)

var _ = test.DescribeTestsFor(test.TestCase{
	API: web.ServiceBindingsURL,
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
	ResourceType:                           types.ServiceBindingType,
	SupportsAsyncOperations:                true,
	DisableTenantResources:                 true,
	ResourceBlueprint:                      blueprint,
	ResourceWithoutNullableFieldsBlueprint: blueprint,
	ResourcePropertiesToIgnore:             []string{"credentials"},
	PatchResource: func(ctx *common.TestContext, apiPath string, objID string, resourceType types.ObjectType, patchLabels []*query.LabelChange, _ bool) {
		byID := query.ByField(query.EqualsOperator, "id", objID)
		sb, err := ctx.SMRepository.Get(context.Background(), resourceType, byID)
		if err != nil {
			Fail(fmt.Sprintf("unable to retrieve resource %s: %s", resourceType, err))
		}

		_, err = ctx.SMRepository.Update(context.Background(), sb, patchLabels)
		if err != nil {
			Fail(fmt.Sprintf("unable to update resource %s: %s", resourceType, err))
		}
	},
	AdditionalTests: func(ctx *common.TestContext) {},
})

func blueprint(ctx *common.TestContext, auth *common.SMExpect, _ bool) common.Object {
	instanceIDObj, err := uuid.NewV4()
	if err != nil {
		Fail(fmt.Sprintf("failed to generate instance GUID: %s", err))
	}
	instanceID := instanceIDObj.String()

	ctx.SMWithOAuth.POST(web.ServiceInstancesURL).WithJSON(common.Object{
		"id":               instanceID,
		"name":             instanceID + "name",
		"service_plan_id":  newServicePlan(ctx),
		"maintenance_info": "{}",
	}).Expect().Status(http.StatusCreated)

	serviceBindingObj := service_binding.Prepare(instanceID, fmt.Sprintf(`{"%s":"%s"}`, TenantIdentifier, TenantValue), `{"password": "secret"}`)
	_, err = ctx.SMRepository.Create(context.Background(), serviceBindingObj)
	if err != nil {
		Fail(fmt.Sprintf("could not create service binding: %s", err))
	}

	binding := auth.ListWithQuery(web.ServiceBindingsURL, fmt.Sprintf("fieldQuery=id eq '%s'", serviceBindingObj.ID)).First().Object().Raw()
	delete(binding, "credentials")
	return binding
}

func newServicePlan(ctx *common.TestContext) string {
	brokerID, _, _ := ctx.RegisterBrokerWithCatalog(common.NewRandomSBCatalog())
	so := ctx.SMWithOAuth.ListWithQuery(web.ServiceOfferingsURL, fmt.Sprintf("fieldQuery=broker_id eq '%s'", brokerID)).First()
	servicePlanID := ctx.SMWithOAuth.ListWithQuery(web.ServicePlansURL, "fieldQuery="+fmt.Sprintf("service_offering_id eq '%s'", so.Object().Value("id").String().Raw())).
		First().Object().Value("id").String().Raw()
	return servicePlanID
}
