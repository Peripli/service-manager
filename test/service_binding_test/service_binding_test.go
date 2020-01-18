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
	"fmt"
	"net/http"
	"strconv"

	"github.com/gofrs/uuid"

	"testing"

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
		test.Get, test.List, test.Delete, test.DeleteList,
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
	ResourcePropertiesToIgnore:             []string{"volume_mounts", "endpoints", "bind_resource", "credentials"},
	PatchResource:                          test.StorageResourcePatch,
	AdditionalTests:                        func(ctx *common.TestContext) {},
})

func blueprint(ctx *common.TestContext, auth *common.SMExpect, async bool) common.Object {
	ID, err := uuid.NewV4()
	if err != nil {
		Fail(fmt.Sprintf("failed to generate instance GUID: %s", err))
	}
	instanceID := "instance-" + ID.String()

	resp := ctx.SMWithOAuth.POST(web.ServiceInstancesURL).
		WithQuery("async", strconv.FormatBool(async)).
		WithJSON(common.Object{
			"id":               instanceID,
			"name":             instanceID + "name",
			"service_plan_id":  newServicePlan(ctx),
			"maintenance_info": "{}",
		}).Expect()

	if async {
		test.ExpectSuccessfulAsyncResourceCreation(resp, auth, instanceID, web.ServiceInstancesURL)
	} else {
		resp.Status(http.StatusCreated)
	}

	bindingID := "binding-" + ID.String()

	resp = ctx.SMWithOAuth.POST(web.ServiceBindingsURL).
		WithQuery("async", strconv.FormatBool(async)).
		WithJSON(common.Object{
			"id":                  bindingID,
			"name":                bindingID + "name",
			"service_instance_id": instanceID,
			"credentials":         `{"password": "secret"}`,
		}).Expect()

	var binding map[string]interface{}
	if async {
		binding = test.ExpectSuccessfulAsyncResourceCreation(resp, auth, bindingID, web.ServiceBindingsURL)
	} else {
		binding = resp.Status(http.StatusCreated).JSON().Object().Raw()
	}

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
