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
	"fmt"
	"net/http"
	"testing"

	"github.com/Peripli/service-manager/test"
	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo"

	. "github.com/onsi/gomega"
)

func TestServicePlans(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Plans Tests Suite")
}

var _ = test.DescribeTestsFor(test.TestCase{
	API:            "/v1/service_plans",
	SupportsLabels: false,
	SupportedOps: []test.Op{
		test.Get, test.List,
	},
	ResourceBlueprint:                      blueprint,
	ResourceWithoutNullableFieldsBlueprint: blueprint,
})

func blueprint(ctx *common.TestContext) common.Object {
	cPaidPlan := common.GeneratePaidTestPlan()
	cService := common.GenerateTestServiceWithPlans(cPaidPlan)
	catalog := common.NewEmptySBCatalog()
	catalog.AddService(cService)
	id, _, _ := ctx.RegisterBrokerWithCatalog(catalog)

	so := ctx.SMWithOAuth.GET("/v1/service_offerings").WithQuery("fieldQuery", "broker_id = "+id).
		Expect().
		Status(http.StatusOK).JSON().Object().Value("service_offerings").Array().First()

	sp := ctx.SMWithOAuth.GET("/v1/service_plans").WithQuery("fieldQuery", fmt.Sprintf("service_offering_id = %s", so.Object().Value("id").String().Raw())).
		Expect().
		Status(http.StatusOK).JSON().Object().Value("service_plans").Array().First()

	return sp.Object().Raw()
}
