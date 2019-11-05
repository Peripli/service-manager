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

package interceptor_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/Peripli/service-manager/pkg/web"

	"github.com/Peripli/service-manager/pkg/env"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage/interceptors"

	"github.com/Peripli/service-manager/pkg/sm"

	"github.com/tidwall/gjson"

	"github.com/tidwall/sjson"

	"github.com/Peripli/service-manager/test/common"

	. "github.com/onsi/ginkgo"

	. "github.com/onsi/gomega"
)

func TestPublicPlansInterceptor(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Public plans Interceptor Tests Suite")
}

var _ = Describe("Service Manager Public Plans Interceptor", func() {
	var ctx *common.TestContext
	var existingBrokerID string
	var existingBrokerServer *common.BrokerServer

	var oldPublicPlanCatalogID string
	var oldPublicPlanCatalogName string

	var oldPaidPlanCatalogID string
	var oldPaidPlanCatalogName string

	var newPublicPlanCatalogID string
	var newPublicPlanCatalogName string

	var newPaidPlanCatalogID string
	var newPaidPlanCatalogName string

	var serviceCatalogID string
	var serviceCatalogName string

	var testCatalog string
	var newPaidPlan string
	var newPublicPlan string
	var oldPaidPlan string

	findOneVisibilityForServicePlanID := func(servicePlanID string) map[string]interface{} {
		vs := ctx.SMWithOAuth.ListWithQuery(web.VisibilitiesURL, fmt.Sprintf("fieldQuery=service_plan_id eq '%s'", servicePlanID))

		vs.Length().Equal(1)
		return vs.First().Object().Raw()
	}

	verifyZeroVisibilityForServicePlanID := func(servicePlanID string) {
		vs := ctx.SMWithOAuth.ListWithQuery(web.VisibilitiesURL, fmt.Sprintf("fieldQuery=service_plan_id eq '%s'", servicePlanID))
		vs.Length().Equal(0)
	}

	findDatabaseIDForServicePlanByCatalogName := func(catalogServicePlanName string) string {
		planID := ctx.SMWithOAuth.ListWithQuery(web.ServicePlansURL, fmt.Sprintf("fieldQuery=catalog_name eq '%s'", catalogServicePlanName)).
			First().Object().Value("id").String().Raw()

		Expect(planID).ToNot(BeEmpty())
		return planID
	}

	BeforeEach(func() {
		ctx = common.NewTestContextBuilder().WithSMExtensions(func(ctx context.Context, smb *sm.ServiceManagerBuilder, e env.Environment) error {
			smb.WithCreateInterceptorProvider(types.ServiceBrokerType, &interceptors.PublicPlanCreateInterceptorProvider{
				IsCatalogPlanPublicFunc: func(broker *types.ServiceBroker, catalogService *types.ServiceOffering, catalogPlan *types.ServicePlan) (b bool, e error) {
					return catalogPlan.Free, nil
				},
			}).OnTxBefore(interceptors.BrokerCreateCatalogInterceptorName).Register()

			smb.WithUpdateInterceptorProvider(types.ServiceBrokerType, &interceptors.PublicPlanUpdateInterceptorProvider{
				IsCatalogPlanPublicFunc: func(broker *types.ServiceBroker, catalogService *types.ServiceOffering, catalogPlan *types.ServicePlan) (b bool, e error) {
					return catalogPlan.Free, nil
				},
			}).OnTxBefore(interceptors.BrokerUpdateCatalogInterceptorName).Register()
			return nil
		}).Build()

		ctx.SMWithOAuth.List(web.ServicePlansURL).
			Path("$[*].catalog_id").Array().NotContains(newPublicPlanCatalogID, newPaidPlanCatalogID)
		c := common.NewEmptySBCatalog()

		oldPublicPlan := common.GenerateFreeTestPlan()
		oldPaidPlan = common.GeneratePaidTestPlan()
		newPublicPlan = common.GenerateFreeTestPlan()
		newPaidPlan = common.GeneratePaidTestPlan()
		oldService := common.GenerateTestServiceWithPlans(oldPublicPlan, oldPaidPlan)
		c.AddService(oldService)
		testCatalog = string(c)

		existingBrokerID, _, existingBrokerServer = ctx.RegisterBrokerWithCatalog(c)
		Expect(existingBrokerID).ToNot(BeEmpty())

		serviceCatalogID = gjson.Get(oldService, "id").Str
		Expect(serviceCatalogID).ToNot(BeEmpty())

		serviceCatalogName = gjson.Get(oldService, "name").Str
		Expect(serviceCatalogName).ToNot(BeEmpty())

		oldPublicPlanCatalogID = gjson.Get(oldPublicPlan, "id").Str
		Expect(oldPublicPlanCatalogID).ToNot(BeEmpty())

		oldPublicPlanCatalogName = gjson.Get(oldPublicPlan, "name").Str
		Expect(oldPublicPlanCatalogName).ToNot(BeEmpty())

		oldPaidPlanCatalogID = gjson.Get(oldPaidPlan, "id").Str
		Expect(oldPaidPlanCatalogID).ToNot(BeEmpty())

		oldPaidPlanCatalogName = gjson.Get(oldPaidPlan, "name").Str
		Expect(oldPaidPlanCatalogName).ToNot(BeEmpty())

		newPublicPlanCatalogID = gjson.Get(newPublicPlan, "id").Str
		Expect(newPublicPlanCatalogID).ToNot(BeEmpty())

		newPublicPlanCatalogName = gjson.Get(newPublicPlan, "name").Str
		Expect(newPublicPlanCatalogName).ToNot(BeEmpty())

		newPaidPlanCatalogID = gjson.Get(newPaidPlan, "id").Str
		Expect(newPaidPlanCatalogID).ToNot(BeEmpty())

		newPaidPlanCatalogName = gjson.Get(newPaidPlan, "name").Str
		Expect(newPaidPlanCatalogName).ToNot(BeEmpty())

		existingBrokerServer.Catalog = common.SBCatalog(testCatalog)

	})

	AfterEach(func() {
		ctx.CleanupBroker(existingBrokerID)
		ctx.Cleanup()
	})

	Specify("plans and visibilities for the registered brokers are known to SM", func() {
		publicPlanID := findDatabaseIDForServicePlanByCatalogName(oldPublicPlanCatalogName)

		visibility := findOneVisibilityForServicePlanID(publicPlanID)
		Expect(visibility["platform_id"]).To(Equal(""))

		paidPlanID := findDatabaseIDForServicePlanByCatalogName(oldPaidPlanCatalogName)
		Expect(paidPlanID).ToNot(BeEmpty())

		verifyZeroVisibilityForServicePlanID(paidPlanID)
	})

	Context("when the catalog is empty", func() {
		var id string

		BeforeEach(func() {
			id, _, _ = ctx.RegisterBrokerWithCatalog(common.NewEmptySBCatalog())
			Expect(id).ToNot(BeEmpty())
		})

		Specify("request succeeds", func() {
			ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + id).
				WithJSON(common.Object{}).
				Expect().
				Status(http.StatusOK)
		})
	})

	Context("when no modifications to the plans occurs", func() {
		It("does not change the state of the visibilities for the existing plans", func() {
			oldPublicPlanDatabaseID := findDatabaseIDForServicePlanByCatalogName(oldPublicPlanCatalogName)
			visibilitiesForPublicPlan := findOneVisibilityForServicePlanID(oldPublicPlanDatabaseID)
			Expect(visibilitiesForPublicPlan["platform_id"]).To(Equal(""))

			oldPaidPlanDatabaseID := findDatabaseIDForServicePlanByCatalogName(oldPaidPlanCatalogName)
			verifyZeroVisibilityForServicePlanID(oldPaidPlanDatabaseID)

			ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + existingBrokerID).
				WithJSON(common.Object{}).
				Expect().
				Status(http.StatusOK)

			visibilitiesForPublicPlan = findOneVisibilityForServicePlanID(oldPublicPlanDatabaseID)
			Expect(visibilitiesForPublicPlan["platform_id"]).To(Equal(""))

			verifyZeroVisibilityForServicePlanID(oldPaidPlanDatabaseID)
		})
	})

	Context("when a new public plan is added", func() {
		BeforeEach(func() {
			s, err := sjson.Set(testCatalog, "services.0.plans.-1", common.JSONToMap(newPublicPlan))
			Expect(err).ShouldNot(HaveOccurred())
			existingBrokerServer.Catalog = common.SBCatalog(s)
		})

		It("creates the plan and creates a public visibility for it", func() {
			ctx.SMWithOAuth.List(web.ServicePlansURL).
				Path("$[*].catalog_id").Array().NotContains(newPublicPlanCatalogID)

			ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + existingBrokerID).
				WithJSON(common.Object{}).
				Expect().
				Status(http.StatusOK)

			planID := findDatabaseIDForServicePlanByCatalogName(newPublicPlanCatalogName)
			Expect(planID).ToNot(BeEmpty())

			visibilities := findOneVisibilityForServicePlanID(planID)
			Expect(visibilities["platform_id"]).To(Equal(""))
		})
	})

	Context("when a new paid plan is added", func() {
		BeforeEach(func() {
			s, err := sjson.Set(testCatalog, "services.0.plans.-1", common.JSONToMap(newPaidPlan))
			Expect(err).ShouldNot(HaveOccurred())
			existingBrokerServer.Catalog = common.SBCatalog(s)
		})

		It("creates the plan and does not create a new public visibility for it", func() {
			ctx.SMWithOAuth.List(web.ServicePlansURL).
				Path("$[*].catalog_id").Array().NotContains(newPaidPlanCatalogID)

			ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + existingBrokerID).
				WithJSON(common.Object{}).
				Expect().
				Status(http.StatusOK)

			planID := findDatabaseIDForServicePlanByCatalogName(newPaidPlanCatalogName)

			verifyZeroVisibilityForServicePlanID(planID)
		})
	})

	Context("when an existing public plan is made paid", func() {
		BeforeEach(func() {
			tempCatalog, err := sjson.Set(testCatalog, "services.0.plans.0.free", false)
			Expect(err).ToNot(HaveOccurred())

			catalog, err := sjson.Set(tempCatalog, "services.0.plans.1.free", false)
			Expect(err).ToNot(HaveOccurred())

			existingBrokerServer.Catalog = common.SBCatalog(catalog)
		})

		It("deletes the public visibility associated with the plan", func() {
			plan := ctx.SMWithOAuth.ListWithQuery(web.ServicePlansURL, fmt.Sprintf("fieldQuery=catalog_name eq '%s'", oldPublicPlanCatalogName))

			plan.Path("$[*].free").Array().Contains(true)
			plan.Length().Equal(1)
			planID := plan.First().Object().Value("id").String().Raw()
			Expect(planID).ToNot(BeEmpty())

			visibilities := findOneVisibilityForServicePlanID(planID)
			Expect(visibilities["platform_id"]).To(Equal(""))

			ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + existingBrokerID).
				WithJSON(common.Object{}).
				Expect().
				Status(http.StatusOK)

			verifyZeroVisibilityForServicePlanID(planID)
		})
	})

	Context("when fetching with query parameter", func() {
		It("returns correct result", func() {
			isPlanFree := gjson.Get(oldPaidPlan, "free").Raw
			ctx.SMWithOAuth.ListWithQuery(web.ServicePlansURL, fmt.Sprintf("fieldQuery=free eq %s", isPlanFree)).
				Element(0).Object().Value("catalog_id").Equal(oldPaidPlanCatalogID)
		})
	})

	Context("when an existing paid plan is made public", func() {
		var planID string
		var platformID string

		BeforeEach(func() {
			tempCatalog, err := sjson.Set(testCatalog, "services.0.plans.0.free", true)
			Expect(err).ToNot(HaveOccurred())

			catalog, err := sjson.Set(tempCatalog, "services.0.plans.1.free", true)
			Expect(err).ToNot(HaveOccurred())

			existingBrokerServer.Catalog = common.SBCatalog(catalog)
			planID = findDatabaseIDForServicePlanByCatalogName(oldPaidPlanCatalogName)

			platform := ctx.RegisterPlatform()
			platformID = platform.ID

			// register a non-public visiblity for the paid plan
			ctx.SMWithOAuth.POST(web.VisibilitiesURL).
				WithJSON(common.Object{
					"service_plan_id": planID,
					"platform_id":     platformID,
				}).
				Expect().Status(http.StatusCreated).JSON().Object().ContainsMap(common.Object{
				"service_plan_id": planID,
				"platform_id":     platformID,
			})

			plan := ctx.SMWithOAuth.ListWithQuery(web.ServicePlansURL, fmt.Sprintf("fieldQuery=catalog_name eq '%s'", oldPaidPlanCatalogName))

			plan.Path("$[*].free").Array().Contains(false)
			plan.Length().Equal(1)

			visibilities := findOneVisibilityForServicePlanID(planID)
			Expect(visibilities["platform_id"]).To(Equal(platformID))
		})

		It("deletes all non-public visibilities that were associated with the plan", func() {
			ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + existingBrokerID).
				WithJSON(common.Object{}).
				Expect().
				Status(http.StatusOK)

			visibilities := findOneVisibilityForServicePlanID(planID)
			Expect(visibilities["platform_id"]).To(Equal(""))
		})

		It("creates a public visibility associated with the plan", func() {
			ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + existingBrokerID).
				WithJSON(common.Object{}).
				Expect().
				Status(http.StatusOK)

			visibilities := findOneVisibilityForServicePlanID(planID)
			Expect(visibilities["platform_id"]).To(Equal(""))
		})
	})
})
