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
	"github.com/Peripli/service-manager/pkg/util/slice"
	"github.com/Peripli/service-manager/storage"
	"github.com/Peripli/service-manager/storage/service_plans"
	"github.com/gavv/httpexpect"
	"math/rand"
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

	findVisibilitiesForServicePlanID := func(servicePlanID string) *httpexpect.Array {
		return ctx.SMWithOAuth.ListWithQuery(web.VisibilitiesURL, fmt.Sprintf("fieldQuery=service_plan_id eq '%s'", servicePlanID))
	}

	findVisibilitiesForServicePlanIDAndPlatformID := func(servicePlanID, platformID string) *httpexpect.Array {
		return ctx.SMWithOAuth.ListWithQuery(web.VisibilitiesURL,
			fmt.Sprintf("fieldQuery=service_plan_id eq '%s' and platform_id eq '%s'", servicePlanID, platformID))
	}

	findOneVisibilityForServicePlanID := func(servicePlanID string) map[string]interface{} {
		vs := findVisibilitiesForServicePlanID(servicePlanID)

		vs.Length().Equal(1)
		return vs.First().Object().Raw()
	}

	verifyZeroVisibilityForServicePlanID := func(servicePlanID string) {
		vs := findVisibilitiesForServicePlanID(servicePlanID)
		vs.Length().Equal(0)
	}

	verifyZeroVisibilityForServicePlanIDAndPlatformID := func(servicePlanID, platformID string) {
		vs := findVisibilitiesForServicePlanIDAndPlatformID(servicePlanID, platformID)
		vs.Length().Equal(0)
	}

	findDatabaseIDForServicePlanByCatalogName := func(catalogServicePlanName string) string {
		planID := ctx.SMWithOAuth.ListWithQuery(web.ServicePlansURL, fmt.Sprintf("fieldQuery=catalog_name eq '%s'", catalogServicePlanName)).
			First().Object().Value("id").String().Raw()

		Expect(planID).ToNot(BeEmpty())
		return planID
	}

	BeforeEach(func() {
		ctx = common.NewTestContextBuilderWithSecurity().WithSMExtensions(func(ctx context.Context, smb *sm.ServiceManagerBuilder, e env.Environment) error {
			smb.WithCreateInterceptorProvider(types.ServiceBrokerType, &interceptors.PublicPlanCreateInterceptorProvider{
				IsCatalogPlanPublicFunc: func(broker *types.ServiceBroker, catalogService *types.ServiceOffering, catalogPlan *types.ServicePlan) (b bool, e error) {
					return *catalogPlan.Free, nil
				},
				SupportedPlatformsFunc: func(ctx context.Context, plan *types.ServicePlan, repository storage.Repository) (map[string]*types.Platform, error) {
					return service_plans.ResolveSupportedPlatformsForPlans(ctx, []*types.ServicePlan{plan}, repository)
				},
				TenantKey: "tenant",
			}).OnTxBefore(interceptors.BrokerCreateCatalogInterceptorName).Register()
			_, err := smb.EnableMultitenancy("tenant", common.ExtractTenantFunc)
			Expect(err).ToNot(HaveOccurred())
			smb.WithUpdateInterceptorProvider(types.ServiceBrokerType, &interceptors.PublicPlanUpdateInterceptorProvider{
				IsCatalogPlanPublicFunc: func(broker *types.ServiceBroker, catalogService *types.ServiceOffering, catalogPlan *types.ServicePlan) (b bool, e error) {
					return *catalogPlan.Free, nil
				},
				SupportedPlatformsFunc: func(ctx context.Context, plan *types.ServicePlan, repository storage.Repository) (map[string]*types.Platform, error) {
					return service_plans.ResolveSupportedPlatformsForPlans(ctx, []*types.ServicePlan{plan}, repository)
				},
				TenantKey: "tenant",
			}).OnTxBefore(interceptors.BrokerUpdateCatalogInterceptorName).Register()
			return nil
		}).WithTenantTokenClaims(map[string]interface{}{
			"cid": "tenancyClient",
			"zid": "tenant",
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

		existingBrokerID, _, existingBrokerServer = ctx.RegisterBrokerWithCatalog(c).GetBrokerAsParams()
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
			id, _, _ = ctx.RegisterBrokerWithCatalog(common.NewEmptySBCatalog()).GetBrokerAsParams()
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
			catalog, err := sjson.Set(testCatalog, "services.0.plans.0.free", false)
			Expect(err).ToNot(HaveOccurred())

			catalog, err = sjson.Set(catalog, "services.0.plans.1.free", false)
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

	Context("when a plan has specified supported platforms", func() {
		It("creates a single public visibility with empty platform_id", func() {
			plan := ctx.SMWithOAuth.ListWithQuery(web.ServicePlansURL, fmt.Sprintf("fieldQuery=catalog_name eq '%s'", oldPublicPlanCatalogName))

			plan.Path("$[*].metadata.supportedPlatforms").NotNull().Array().Empty()

			planID := plan.First().Object().Value("id").String().Raw()
			Expect(planID).ToNot(BeEmpty())

			visibilities := findOneVisibilityForServicePlanID(planID)
			Expect(visibilities["platform_id"]).To(Equal(""))

			ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + existingBrokerID).
				WithJSON(common.Object{}).
				Expect().
				Status(http.StatusOK)

			vis := findOneVisibilityForServicePlanID(planID)
			Expect(vis["platform_id"]).To(Equal(""))
		})
	})

	Context("when a plan has specified supported platforms", func() {
		var supportedPlatforms []string
		var planID string

		JustBeforeEach(func() {
			catalog, err := sjson.Set(testCatalog, "services.0.plans.0.metadata.supportedPlatforms", supportedPlatforms)
			Expect(err).ToNot(HaveOccurred())

			existingBrokerServer.Catalog = common.SBCatalog(catalog)

			plan := ctx.SMWithOAuth.ListWithQuery(web.ServicePlansURL, fmt.Sprintf("fieldQuery=catalog_name eq '%s'", oldPublicPlanCatalogName))

			plan.Path("$[*].metadata.supportedPlatforms").NotNull().Array().Empty()

			planID = plan.First().Object().Value("id").String().Raw()
			Expect(planID).ToNot(BeEmpty())

			visibilities := findOneVisibilityForServicePlanID(planID)
			Expect(visibilities["platform_id"]).To(Equal(""))
		})

		Context("when a plan has no supported platforms", func() {
			BeforeEach(func() {
				supportedPlatforms = []string{}
			})

			It("creates a public visibility associated with the plan", func() {
				ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + existingBrokerID).
					WithJSON(common.Object{}).
					Expect().
					Status(http.StatusOK)

				vis := findOneVisibilityForServicePlanID(planID)
				Expect(vis["platform_id"]).To(Equal(""))
			})
		})

		Context("when a plan supports only one platform type", func() {

			BeforeEach(func() {
				supportedPlatforms = []string{"cloudfoundry"}
			})

			It("creates a single public visibility for that platform", func() {
				platform := ctx.RegisterPlatformWithType(supportedPlatforms[0])

				ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + existingBrokerID).
					WithJSON(common.Object{}).
					Expect().
					Status(http.StatusOK)

				vis := findOneVisibilityForServicePlanID(planID)
				Expect(vis["platform_id"]).To(Equal(platform.ID))
			})
		})

		Context("when a plan supports multiple platform types", func() {

			BeforeEach(func() {
				supportedPlatforms = []string{"cloudfoundry", "kubernetes", "abap"}
			})

			It("creates a single visibility for each supported platform", func() {
				var platformCount int

				platformIDMap := make(map[string]bool)
				for _, platformType := range supportedPlatforms {
					count := rand.Intn(10-1) + 1
					for i := 0; i < count; i++ {
						plt := ctx.RegisterPlatformWithType(platformType)
						platformIDMap[plt.ID] = true
					}
					platformCount += count
				}

				ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + existingBrokerID).
					WithJSON(common.Object{}).
					Expect().
					Status(http.StatusOK)

				vis := findVisibilitiesForServicePlanID(planID)
				vis.Length().Equal(platformCount)

				for i := 0; i < platformCount; i++ {
					plt := vis.Element(i).Object()
					plt.NotContainsKey("labels")

					pltID := plt.Value("platform_id").String().Raw()
					_, ok := platformIDMap[pltID]
					if ok {
						delete(platformIDMap, pltID)
					} else {
						Fail(fmt.Sprintf("unexpected platform_id with id (%s) was set to visibility", pltID))
					}
				}

				Expect(len(platformIDMap)).To(Equal(0))
			})
		})
	})

	Context("when a plan has specified supported platform names", func() {
		var supportedPlatformsByID map[string]*types.Platform
		var planID string

		var getSupportedPlatformNames = func() []string {
			result := make([]string, 0)
			for _, platform := range supportedPlatformsByID {
				result = append(result, platform.Name)
			}

			return result
		}

		var getSupportedPlatformIDs = func() []string {
			result := make([]string, 0)
			for id := range supportedPlatformsByID {
				result = append(result, id)
			}

			return result
		}

		JustBeforeEach(func() {
			catalog, err := sjson.Set(testCatalog, "services.0.plans.0.metadata.supportedPlatformNames", getSupportedPlatformNames())
			Expect(err).ToNot(HaveOccurred())

			existingBrokerServer.Catalog = common.SBCatalog(catalog)

			plan := ctx.SMWithOAuth.ListWithQuery(web.ServicePlansURL, fmt.Sprintf("fieldQuery=catalog_name eq '%s'", oldPublicPlanCatalogName))

			plan.Path("$[*].metadata.supportedPlatformNames").NotNull().Array().Empty()

			planID = plan.First().Object().Value("id").String().Raw()
			Expect(planID).ToNot(BeEmpty())

			visibilities := findOneVisibilityForServicePlanID(planID)
			Expect(visibilities["platform_id"]).To(Equal(""))
		})

		Context("when a plan supports only one platform name", func() {

			BeforeEach(func() {
				platform := ctx.RegisterPlatform()
				supportedPlatformsByID = map[string]*types.Platform{platform.ID: platform}
			})

			It("creates a single public visibility for that platform", func() {
				ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + existingBrokerID).
					WithJSON(common.Object{}).
					Expect().
					Status(http.StatusOK)

				vis := findOneVisibilityForServicePlanID(planID)
				Expect(vis["platform_id"]).To(Equal(getSupportedPlatformIDs()[0]))
			})
		})

		Context("when a plan supports multiple platform names", func() {

			BeforeEach(func() {
				firstPlatform := ctx.RegisterPlatform()
				secondPlatform := ctx.RegisterPlatform()
				supportedPlatformsByID = map[string]*types.Platform{
					firstPlatform.ID:  firstPlatform,
					secondPlatform.ID: secondPlatform,
				}
			})

			It("creates a single visibility for each supported platform", func() {
				ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + existingBrokerID).
					WithJSON(common.Object{}).
					Expect().
					Status(http.StatusOK)

				vis := findVisibilitiesForServicePlanID(planID)
				vis.Length().Equal(len(supportedPlatformsByID))

				visPlatformIDs := make([]string, 0)
				for i := 0; i < len(supportedPlatformsByID); i++ {
					plt := vis.Element(i).Object()
					plt.NotContainsKey("labels")

					pltID := plt.Value("platform_id").String().Raw()
					visPlatformIDs = append(visPlatformIDs, pltID)
				}

				Expect(len(slice.StringsDistinct(visPlatformIDs, getSupportedPlatformIDs()))).To(BeEquivalentTo(0))
			})
		})

		Context("when a broker has a plan supporting platform names and another supporting platform types", func() {
			var newPlanCatalogName, k8sPlatformID, cfPlatformID string

			BeforeEach(func() {
				cfPlatform := ctx.RegisterPlatformWithType(types.CFPlatformType)
				cfPlatformID = cfPlatform.ID
				supportedPlatformsByID = map[string]*types.Platform{
					cfPlatform.ID: cfPlatform,
				}
			})

			JustBeforeEach(func() {
				k8sPlatform := ctx.RegisterPlatformWithType(types.K8sPlatformType)
				k8sPlatformID = k8sPlatform.ID

				newPlan := common.GenerateFreeTestPlan()
				newPlanCatalogName = gjson.Get(newPlan, "name").String()
				Expect(newPlanCatalogName).ToNot(BeEmpty())
				additionalPublicPlan, err := sjson.Set(newPlan, "metadata.supportedPlatforms", []string{k8sPlatform.Type})
				Expect(err).ToNot(HaveOccurred())
				var catalog string
				catalog, err = sjson.Set(string(existingBrokerServer.Catalog), "services.0.plans.-1", common.JSONToMap(additionalPublicPlan))
				Expect(err).ShouldNot(HaveOccurred())

				existingBrokerServer.Catalog = common.SBCatalog(catalog)
			})

			It("creates visibilities according to both platform names and types", func() {
				ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + existingBrokerID).
					WithJSON(common.Object{}).
					Expect().
					Status(http.StatusOK)

				By("visibility by platform name", func() {
					cfVis := findVisibilitiesForServicePlanID(planID)
					cfVis.Length().Equal(1)
					Expect(cfVis.Element(0).Object().Value("platform_id").String().Raw()).To(BeEquivalentTo(cfPlatformID))
				})

				By("visibility by platform type", func() {
					newPlanID := findDatabaseIDForServicePlanByCatalogName(newPlanCatalogName)
					k8sVis := findVisibilitiesForServicePlanID(newPlanID)
					k8sVis.Length().Equal(1)
					Expect(k8sVis.Element(0).Object().Value("platform_id").String().Raw()).To(BeEquivalentTo(k8sPlatformID))
				})

			})
		})

		When("when a non public plan visibility exist for a tenant scoped platform", func() {

			BeforeEach(func() {
				platform := ctx.RegisterTenantPlatform()
				supportedPlatformsByID = map[string]*types.Platform{platform.ID: platform}
			})

			JustBeforeEach(func() {
				ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + existingBrokerID).
					WithJSON(common.Object{}).
					Expect().
					Status(http.StatusOK)
				catalog, err := sjson.Set(string(existingBrokerServer.Catalog), "services.0.plans.0.free", false)
				Expect(err).ToNot(HaveOccurred())
				existingBrokerServer.Catalog = common.SBCatalog(catalog)
			})

			It("should keep the existing plan visibility", func() {
				ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + existingBrokerID).
					WithJSON(common.Object{}).
					Expect().
					Status(http.StatusOK)

				vis := findOneVisibilityForServicePlanID(planID)
				Expect(vis["platform_id"]).To(Equal(getSupportedPlatformIDs()[0]))
			})
		})

		When("when a public plan visibility exist for a tenant scoped platform", func() {

			BeforeEach(func() {
				platform := ctx.RegisterTenantPlatform()
				supportedPlatformsByID = map[string]*types.Platform{platform.ID: platform}
			})

			JustBeforeEach(func() {
				ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + existingBrokerID).
					WithJSON(common.Object{}).
					Expect().
					Status(http.StatusOK)
			})

			It("should keep the existing plan visibility", func() {
				ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + existingBrokerID).
					WithJSON(common.Object{}).
					Expect().
					Status(http.StatusOK)

				vis := findOneVisibilityForServicePlanID(planID)
				Expect(vis["platform_id"]).To(Equal(getSupportedPlatformIDs()[0]))
			})
		})
	})

	Context("when a plan has specified excluded platform names", func() {
		var excludedPlatformsByID map[string]*types.Platform
		var planID string
		var platformsCount float64

		var getExcludedPlatformNames = func() []string {
			result := make([]string, 0)
			for _, platform := range excludedPlatformsByID {
				result = append(result, platform.Name)
			}

			return result
		}

		BeforeEach(func() {
			platformsCount = ctx.SMWithOAuth.GET(web.PlatformsURL).Expect().Status(http.StatusOK).JSON().Path("$.num_items").Number().Raw()
			Expect(platformsCount).ToNot((BeZero()))
		})

		JustBeforeEach(func() {
			catalog, err := sjson.Set(testCatalog, "services.0.plans.0.metadata.excludedPlatformNames", getExcludedPlatformNames())
			Expect(err).ToNot(HaveOccurred())

			existingBrokerServer.Catalog = common.SBCatalog(catalog)

			plan := ctx.SMWithOAuth.ListWithQuery(web.ServicePlansURL, fmt.Sprintf("fieldQuery=catalog_name eq '%s'", oldPublicPlanCatalogName))

			plan.Path("$[*].metadata.excludedPlatformNames").NotNull().Array().Empty()

			planID = plan.First().Object().Value("id").String().Raw()
			Expect(planID).ToNot(BeEmpty())

			visibilities := findOneVisibilityForServicePlanID(planID)
			Expect(visibilities["platform_id"]).To(Equal(""))
		})

		Context("when a plan excludes only one platform name", func() {
			BeforeEach(func() {
				platform := ctx.RegisterPlatform()
				excludedPlatformsByID = map[string]*types.Platform{platform.ID: platform}
			})

			It("creates visibilities for all other platforms", func() {
				ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + existingBrokerID).
					WithJSON(common.Object{}).
					Expect().
					Status(http.StatusOK)

				By("does not create visibility for excluded platform")
				for excludedPlatformID := range excludedPlatformsByID {
					verifyZeroVisibilityForServicePlanIDAndPlatformID(planID, excludedPlatformID)
				}

				By("creates visibilities for all non-excluded platform")
				vis := findVisibilitiesForServicePlanID(planID)
				Expect(vis.Length().Raw()).To(Equal(platformsCount))

			})
		})

		Context("when a plan excludes multiple platform names", func() {

			BeforeEach(func() {
				firstPlatform := ctx.RegisterPlatform()
				secondPlatform := ctx.RegisterPlatform()
				excludedPlatformsByID = map[string]*types.Platform{
					firstPlatform.ID:  firstPlatform,
					secondPlatform.ID: secondPlatform,
				}
			})

			It("creates a single visibility for each supported platform", func() {
				ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + existingBrokerID).
					WithJSON(common.Object{}).
					Expect().
					Status(http.StatusOK)

				By("does not create visibility for excluded platforms")
				for excludedPlatformID := range excludedPlatformsByID {
					verifyZeroVisibilityForServicePlanIDAndPlatformID(planID, excludedPlatformID)
				}

				By("creates visibilities for all non-excluded platform")
				vis := findVisibilitiesForServicePlanID(planID)
				Expect(vis.Length().Raw()).To(Equal(platformsCount))
			})
		})

		Context("when a broker has a plan excluding a platforms and another supporting all platforms", func() {
			var newPlanCatalogName string

			BeforeEach(func() {
				cfPlatform := ctx.RegisterPlatformWithType(types.CFPlatformType)
				excludedPlatformsByID = map[string]*types.Platform{
					cfPlatform.ID: cfPlatform,
				}
			})

			JustBeforeEach(func() {
				newPlan := common.GenerateFreeTestPlan()
				newPlanCatalogName = gjson.Get(newPlan, "name").String()
				Expect(newPlanCatalogName).ToNot(BeEmpty())

				catalog, err := sjson.Set(string(existingBrokerServer.Catalog), "services.0.plans.-1", common.JSONToMap(newPlan))
				Expect(err).ShouldNot(HaveOccurred())

				existingBrokerServer.Catalog = common.SBCatalog(catalog)
			})

			It("creates visibilities according to both platform names and types", func() {
				ctx.SMWithOAuth.PATCH(web.ServiceBrokersURL + "/" + existingBrokerID).
					WithJSON(common.Object{}).
					Expect().
					Status(http.StatusOK)

				By("no visibility created for plan with excluded platform on that platform")
				for excludedPlatformID := range excludedPlatformsByID {
					verifyZeroVisibilityForServicePlanIDAndPlatformID(planID, excludedPlatformID)
				}

				By("visibilities created for plan with excluded platform on all non-excluded platform")
				vis := findVisibilitiesForServicePlanID(planID)
				Expect(vis.Length().Raw()).To(Equal(platformsCount))

				By("visibility created for plan supporting all platforms on all platform")
				newPlanID := findDatabaseIDForServicePlanByCatalogName(newPlanCatalogName)
				allPlatformsVisibility := findOneVisibilityForServicePlanID(newPlanID)
				Expect(allPlatformsVisibility["platform_id"]).To(BeEmpty())
			})
		})
	})
})
