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
	"net/http"
	"testing"

	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/gavv/httpexpect"

	"github.com/Peripli/service-manager/test/common"

	"github.com/Peripli/service-manager/test"

	. "github.com/onsi/ginkgo"

	. "github.com/onsi/gomega"
)

func TestServiceOfferings(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Offerings Tests Suite")
}

var _ = test.DescribeTestsFor(test.TestCase{
	API: web.ServiceOfferingsURL,
	SupportedOps: []test.Op{
		test.Get, test.List, test.Patch,
	},
	DisableTenantResources:                 true,
	ResourceBlueprint:                      blueprint,
	ResourceWithoutNullableFieldsBlueprint: blueprint,
	AdditionalTests: func(ctx *common.TestContext) {
		Context("additional non-generic tests", func() {
			Describe("PATCH", func() {
				var id string

				var patchLabels []query.LabelChange
				var patchLabelsBody map[string]interface{}
				changedLabelKey := "label_key"
				changedLabelValues := []string{"label_value1", "label_value2"}
				operation := query.AddLabelOperation

				BeforeEach(func() {
					patchLabelsBody = make(map[string]interface{})
					patchLabels = append(patchLabels, query.LabelChange{
						Operation: operation,
						Key:       changedLabelKey,
						Values:    changedLabelValues,
					})
					patchLabelsBody["labels"] = patchLabels

					offering := blueprint(ctx, ctx.SMWithOAuth)
					id = offering["id"].(string)
				})

				Context("When not only labels updated", func() {
					It("should return bad request", func() {
						patchLabelsBody["description"] = "new-description"

						ctx.SMWithOAuth.PATCH(web.ServiceOfferingsURL + "/" + id).
							WithJSON(patchLabelsBody).
							Expect().
							Status(http.StatusBadRequest)

					})
				})

				Context("When labels not updated", func() {
					It("should return bad request", func() {
						body := make(map[string]interface{})
						body["description"] = "new-description"

						ctx.SMWithOAuth.PATCH(web.ServiceOfferingsURL + "/" + id).
							WithJSON(body).
							Expect().
							Status(http.StatusBadRequest)

					})
				})
			})

			Describe("GET", func() {
				getPlansByOffering := func(offeringID string) *types.ServicePlan {
					plans, err := ctx.SMRepository.List(context.Background(), types.ServicePlanType, query.ByField(query.EqualsOperator, "service_offering_id", offeringID))
					Expect(err).ShouldNot(HaveOccurred())
					Expect(plans.Len()).To(BeNumerically(">", 0))
					return (plans.(*types.ServicePlans)).ServicePlans[0]
				}

				var k8sPlatform *types.Platform
				var k8sAgent *httpexpect.Expect

				BeforeEach(func() {
					k8sPlatformJSON := common.MakePlatform("k8s-platform", "k8s-platform", "kubernetes", "test-platform-k8s")
					k8sPlatform = common.RegisterPlatformInSM(k8sPlatformJSON, ctx.SMWithOAuth, map[string]string{})
					k8sAgent = ctx.SM.Builder(func(req *httpexpect.Request) {
						username, password := k8sPlatform.Credentials.Basic.Username, k8sPlatform.Credentials.Basic.Password
						req.WithBasicAuth(username, password)
					})
				})

				AfterEach(func() {
					ctx.CleanupAdditionalResources()
				})

				It("With no visibilities for offering's plans", func() {
					offering := blueprint(ctx, ctx.SMWithOAuth)
					k8sAgent.GET(web.ServiceOfferingsURL).
						Expect().
						Status(http.StatusOK).JSON().Object().Value("service_offerings").Array().Length().Equal(0)
					k8sAgent.GET(fmt.Sprintf("%s/%s", web.ServiceOfferingsURL, offering["id"].(string))).
						Expect().
						Status(http.StatusNotFound)
				})

				It("With visibility for offering's plan and empty platform", func() {
					offering := blueprint(ctx, ctx.SMWithOAuth)
					plan := getPlansByOffering(offering["id"].(string))

					ctx.SMWithOAuth.POST(web.VisibilitiesURL).WithJSON(common.Object{
						"service_plan_id": plan.ID,
					}).Expect().Status(http.StatusCreated)

					k8sAgent.GET(web.ServiceOfferingsURL).
						Expect().
						Status(http.StatusOK).JSON().Object().Value("service_offerings").Array().Length().Equal(1)
					k8sAgent.GET(fmt.Sprintf("%s/%s", web.ServiceOfferingsURL, offering["id"].(string))).
						Expect().
						Status(http.StatusOK).JSON().Object().ContainsMap(offering)
				})

				It("With visibility for offering's plan and the calling platform's id", func() {
					offering := blueprint(ctx, ctx.SMWithOAuth)
					plan := getPlansByOffering(offering["id"].(string))

					ctx.SMWithOAuth.POST(web.VisibilitiesURL).WithJSON(common.Object{
						"service_plan_id": plan.ID,
						"platform_id":     k8sPlatform.ID,
					}).Expect().Status(http.StatusCreated)

					k8sAgent.GET(web.ServiceOfferingsURL).
						Expect().
						Status(http.StatusOK).JSON().Object().Value("service_offerings").Array().Length().Equal(1)
					k8sAgent.GET(fmt.Sprintf("%s/%s", web.ServiceOfferingsURL, offering["id"].(string))).
						Expect().
						Status(http.StatusOK).JSON().Object().ContainsMap(offering)
				})

				It("With visibility for offering's plan and NOT calling platform's id", func() {
					offering := blueprint(ctx, ctx.SMWithOAuth)
					plan := getPlansByOffering(offering["id"].(string))

					otherPlatform := ctx.RegisterPlatform()
					ctx.SMWithOAuth.POST(web.VisibilitiesURL).WithJSON(common.Object{
						"service_plan_id": plan.ID,
						"platform_id":     otherPlatform.ID,
					}).Expect().Status(http.StatusCreated)

					k8sAgent.GET(web.ServiceOfferingsURL).
						Expect().
						Status(http.StatusOK).JSON().Object().Value("service_offerings").Array().Length().Equal(0)
					k8sAgent.GET(fmt.Sprintf("%s/%s", web.ServiceOfferingsURL, offering["id"].(string))).
						Expect().
						Status(http.StatusNotFound)
				})

				Context("with 2 offerings", func() {
					var offering1, offering2 common.Object
					BeforeEach(func() {
						offering1 = blueprint(ctx, ctx.SMWithOAuth)
						offering2 = blueprint(ctx, ctx.SMWithOAuth)
					})

					It("With 2 offerings, but no visibilities, should not get either of them", func() {
						k8sAgent.GET(fmt.Sprintf("%s/%s", web.ServiceOfferingsURL, offering1["id"].(string))).
							Expect().
							Status(http.StatusNotFound)
						k8sAgent.GET(fmt.Sprintf("%s/%s", web.ServiceOfferingsURL, offering2["id"].(string))).
							Expect().
							Status(http.StatusNotFound)
						k8sAgent.GET(web.ServiceOfferingsURL).
							Expect().
							Status(http.StatusOK).JSON().Object().Value("service_offerings").Array().Length().Equal(0)
					})

					It("With 2 offerings, but visibility for one", func() {
						plan := getPlansByOffering(offering1["id"].(string))
						ctx.SMWithOAuth.POST(web.VisibilitiesURL).WithJSON(common.Object{
							"service_plan_id": plan.ID,
						}).Expect().Status(http.StatusCreated)
						k8sAgent.GET(fmt.Sprintf("%s/%s", web.ServiceOfferingsURL, offering1["id"].(string))).
							Expect().
							Status(http.StatusOK)
						k8sAgent.GET(fmt.Sprintf("%s/%s", web.ServiceOfferingsURL, offering2["id"].(string))).
							Expect().
							Status(http.StatusNotFound)
						k8sAgent.GET(web.ServiceOfferingsURL).
							Expect().
							Status(http.StatusOK).JSON().Object().Value("service_offerings").Array().Length().Equal(1)
					})
				})
			})

			Describe("Labelled", func() {
				var id string

				var initialLabels []query.LabelChange
				var initialLabelsBody map[string]interface{}
				initialLabelsKeys := []string{"initial_key", "initial_key2"}
				initialLabelValues := []string{"initial_value", "initial_value2"}

				var patchLabels []query.LabelChange
				var patchLabelsBody map[string]interface{}
				changedLabelKey := "label_key"
				changedLabelValues := []string{"label_value1", "label_value2"}
				operation := query.AddLabelOperation

				BeforeEach(func() {
					patchLabels = []query.LabelChange{}
					initialLabelsBody = make(map[string]interface{})
					initialLabels = []query.LabelChange{
						{
							Operation: query.AddLabelOperation,
							Key:       initialLabelsKeys[0],
							Values:    initialLabelValues[:1],
						},
						{
							Operation: query.AddLabelOperation,
							Key:       initialLabelsKeys[1],
							Values:    initialLabelValues,
						},
					}
					initialLabelsBody["labels"] = initialLabels
				})

				JustBeforeEach(func() {
					patchLabelsBody = make(map[string]interface{})
					patchLabels = append(patchLabels, query.LabelChange{
						Operation: operation,
						Key:       changedLabelKey,
						Values:    changedLabelValues,
					})
					patchLabelsBody["labels"] = patchLabels

					offering := blueprint(ctx, ctx.SMWithOAuth)
					id = offering["id"].(string)

					ctx.SMWithOAuth.PATCH(web.ServiceOfferingsURL + "/" + id).
						WithJSON(initialLabelsBody).
						Expect().
						Status(http.StatusOK)

				})

				Context("Add new label", func() {
					It("Should return 200", func() {
						label := types.Labels{changedLabelKey: changedLabelValues}
						ctx.SMWithOAuth.PATCH(web.ServiceOfferingsURL + "/" + id).
							WithJSON(patchLabelsBody).
							Expect().
							Status(http.StatusOK).JSON().Object().Value("labels").Object().ContainsMap(label)
					})
				})

				Context("Add label with existing key and value", func() {
					It("Should return 200", func() {
						ctx.SMWithOAuth.PATCH(web.ServiceOfferingsURL + "/" + id).
							WithJSON(patchLabelsBody).
							Expect().
							Status(http.StatusOK)

						ctx.SMWithOAuth.PATCH(web.ServiceOfferingsURL + "/" + id).
							WithJSON(patchLabelsBody).
							Expect().
							Status(http.StatusOK)
					})
				})

				Context("Add new label value", func() {
					BeforeEach(func() {
						operation = query.AddLabelValuesOperation
						changedLabelKey = initialLabelsKeys[0]
						changedLabelValues = []string{"new-label-value"}
					})
					It("Should return 200", func() {
						var labelValuesObj []interface{}
						for _, val := range changedLabelValues {
							labelValuesObj = append(labelValuesObj, val)
						}
						ctx.SMWithOAuth.PATCH(web.ServiceOfferingsURL + "/" + id).
							WithJSON(patchLabelsBody).
							Expect().
							Status(http.StatusOK).JSON().
							Path("$.labels").Object().Values().Path("$[*][*]").Array().Contains(labelValuesObj...)
					})
				})

				Context("Add new label value to a non-existing label", func() {
					BeforeEach(func() {
						operation = query.AddLabelValuesOperation
						changedLabelKey = "cluster_id_new"
						changedLabelValues = []string{"new-label-value"}
					})
					It("Should return 200", func() {
						var labelValuesObj []interface{}
						for _, val := range changedLabelValues {
							labelValuesObj = append(labelValuesObj, val)
						}

						ctx.SMWithOAuth.PATCH(web.ServiceOfferingsURL + "/" + id).
							WithJSON(patchLabelsBody).
							Expect().
							Status(http.StatusOK).JSON().
							Path("$.labels").Object().Values().Path("$[*][*]").Array().Contains(labelValuesObj...)
					})
				})

				Context("Add duplicate label value", func() {
					BeforeEach(func() {
						operation = query.AddLabelValuesOperation
						changedLabelKey = initialLabelsKeys[0]
						changedLabelValues = initialLabelValues[:1]
					})
					It("Should return 200", func() {
						ctx.SMWithOAuth.PATCH(web.ServiceOfferingsURL + "/" + id).
							WithJSON(patchLabelsBody).
							Expect().
							Status(http.StatusOK)
					})
				})

				Context("Remove a label", func() {
					BeforeEach(func() {
						operation = query.RemoveLabelOperation
						changedLabelKey = initialLabelsKeys[0]
					})
					It("Should return 200", func() {
						ctx.SMWithOAuth.PATCH(web.ServiceOfferingsURL + "/" + id).
							WithJSON(patchLabelsBody).
							Expect().
							Status(http.StatusOK).JSON().
							Path("$.labels").Object().Keys().NotContains(changedLabelKey)
					})
				})

				Context("Remove a label and providing no key", func() {
					BeforeEach(func() {
						operation = query.RemoveLabelOperation
						changedLabelKey = ""
					})
					It("Should return 400", func() {
						ctx.SMWithOAuth.PATCH(web.ServiceOfferingsURL + "/" + id).
							WithJSON(patchLabelsBody).
							Expect().
							Status(http.StatusBadRequest)
					})
				})

				Context("Remove a label key which does not exist", func() {
					BeforeEach(func() {
						operation = query.RemoveLabelOperation
						changedLabelKey = "non-existing-key"
					})
					It("Should return 200", func() {
						ctx.SMWithOAuth.PATCH(web.ServiceOfferingsURL + "/" + id).
							WithJSON(patchLabelsBody).
							Expect().
							Status(http.StatusOK)
					})
				})

				Context("Remove label values and providing a single value", func() {
					BeforeEach(func() {
						operation = query.RemoveLabelValuesOperation
						changedLabelKey = initialLabelsKeys[0]
						changedLabelValues = initialLabelValues[:1]
					})
					It("Should return 200", func() {
						ctx.SMWithOAuth.PATCH(web.ServiceOfferingsURL + "/" + id).
							WithJSON(patchLabelsBody).
							Expect().
							Status(http.StatusOK).JSON().
							Path("$.labels[*]").Array().NotContains(changedLabelValues)
					})
				})

				Context("Remove label values and providing multiple values", func() {
					BeforeEach(func() {
						operation = query.RemoveLabelValuesOperation
						changedLabelKey = initialLabelsKeys[1]
						changedLabelValues = initialLabelValues
					})
					It("Should return 200", func() {
						ctx.SMWithOAuth.PATCH(web.ServiceOfferingsURL + "/" + id).
							WithJSON(patchLabelsBody).
							Expect().
							Status(http.StatusOK).JSON().
							Path("$.labels[*]").Array().NotContains(changedLabelValues)
					})
				})

				Context("Remove all label values for a key", func() {
					BeforeEach(func() {
						operation = query.RemoveLabelValuesOperation
						changedLabelKey = initialLabelsKeys[0]
						changedLabelValues = initialLabelValues[:1]
					})
					It("Should return 200 with this key gone", func() {
						ctx.SMWithOAuth.PATCH(web.ServiceOfferingsURL + "/" + id).
							WithJSON(patchLabelsBody).
							Expect().
							Status(http.StatusOK).JSON().
							Path("$.labels").Object().Keys().NotContains(changedLabelKey)
					})
				})

				Context("Remove label values and not providing value to remove", func() {
					BeforeEach(func() {
						operation = query.RemoveLabelValuesOperation
						changedLabelValues = []string{}
					})
					It("Should return 400", func() {
						ctx.SMWithOAuth.PATCH(web.ServiceOfferingsURL + "/" + id).
							WithJSON(patchLabelsBody).
							Expect().
							Status(http.StatusBadRequest)
					})
				})

				Context("Remove label value which does not exist", func() {
					BeforeEach(func() {
						operation = query.RemoveLabelValuesOperation
						changedLabelKey = initialLabelsKeys[0]
						changedLabelValues = []string{"non-existing-value"}
					})
					It("Should return 200", func() {
						ctx.SMWithOAuth.PATCH(web.ServiceOfferingsURL + "/" + id).
							WithJSON(patchLabelsBody).
							Expect().
							Status(http.StatusOK)
					})
				})
			})
		})
	},
})

func blueprint(ctx *common.TestContext, auth *httpexpect.Expect) common.Object {
	cService := common.GenerateTestServiceWithPlans(common.GenerateFreeTestPlan())
	catalog := common.NewEmptySBCatalog()
	catalog.AddService(cService)
	id, _, _ := ctx.RegisterBrokerWithCatalog(catalog)

	so := auth.GET(web.ServiceOfferingsURL).WithQuery("fieldQuery", "broker_id = "+id).
		Expect().
		Status(http.StatusOK).JSON().Object().Value("service_offerings").Array().First()

	return so.Object().Raw()
}
