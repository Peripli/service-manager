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

	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/gavv/httpexpect"

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
	API: web.ServicePlansURL,
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

					plan := blueprint(ctx, ctx.SMWithOAuth)
					id = plan["id"].(string)
				})

				Context("When not only labels updated", func() {
					It("should return bad request", func() {
						patchLabelsBody["description"] = "new-description"

						ctx.SMWithOAuth.PATCH(web.ServicePlansURL + "/" + id).
							WithJSON(patchLabelsBody).
							Expect().
							Status(http.StatusBadRequest)

					})
				})

				Context("When labels not updated", func() {
					It("should return bad request", func() {
						body := make(map[string]interface{})
						body["description"] = "new-description"

						ctx.SMWithOAuth.PATCH(web.ServicePlansURL + "/" + id).
							WithJSON(body).
							Expect().
							Status(http.StatusBadRequest)

					})
				})
			})

			Describe("GET", func() {
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

				Context("for plan", func() {
					var plan common.Object
					BeforeEach(func() {
						plan = blueprint(ctx, ctx.SMWithOAuth)
					})

					It("with no visibilities", func() {
						k8sAgent.GET(web.ServicePlansURL).
							Expect().
							Status(http.StatusOK).JSON().Object().Value("service_plans").Array().Length().Equal(0)
					})

					It("with visibility for plan and empty platform", func() {
						ctx.SMWithOAuth.POST(web.VisibilitiesURL).WithJSON(common.Object{
							"service_plan_id": plan["id"].(string),
						}).Expect().Status(http.StatusCreated)
						k8sAgent.GET(web.ServicePlansURL).
							Expect().
							Status(http.StatusOK).JSON().Object().Value("service_plans").Array().Length().Equal(1)
					})

					It("list with field query catalog_name for not visible plan", func() {
						planCatalogName := plan["catalog_name"].(string)
						Expect(planCatalogName).To(Not(BeEmpty()))
						k8sAgent.GET(web.ServicePlansURL).WithQuery("fieldQuery", "catalog_name = "+planCatalogName).
							Expect().
							Status(http.StatusOK).JSON().Object().Value("service_plans").Array().Length().Equal(0)
					})

					It("list with field query plan id for not visible plan", func() {
						planID := plan["id"].(string)
						Expect(planID).To(Not(BeEmpty()))
						k8sAgent.GET(web.ServicePlansURL).WithQuery("fieldQuery", "id = "+planID).
							Expect().
							Status(http.StatusOK).JSON().Object().Value("service_plans").Array().Length().Equal(0)
					})

					Context("with additional plan", func() {
						var plan2 common.Object
						BeforeEach(func() {
							plan2 = blueprint(ctx, ctx.SMWithOAuth)
						})

						It("should not get either of them when no visibilities are present", func() {
							k8sAgent.GET(fmt.Sprintf("%s/%s", web.ServicePlansURL, plan["id"].(string))).
								Expect().
								Status(http.StatusNotFound)
							k8sAgent.GET(fmt.Sprintf("%s/%s", web.ServicePlansURL, plan2["id"].(string))).
								Expect().
								Status(http.StatusNotFound)
						})

						Context("with visibility for one plan", func() {
							var planID1, planID2 string
							BeforeEach(func() {
								planID1 = plan["id"].(string)
								Expect(planID1).To(Not(BeEmpty()))
								planID2 = plan2["id"].(string)
								Expect(planID2).To(Not(BeEmpty()))
								ctx.SMWithOAuth.POST(web.VisibilitiesURL).WithJSON(common.Object{
									"service_plan_id": plan["id"].(string),
								}).Expect().Status(http.StatusCreated)
							})

							It("should return only one plan for get operation", func() {
								k8sAgent.GET(fmt.Sprintf("%s/%s", web.ServicePlansURL, planID1)).
									Expect().
									Status(http.StatusOK)
								k8sAgent.GET(fmt.Sprintf("%s/%s", web.ServicePlansURL, planID2)).
									Expect().
									Status(http.StatusNotFound)
								k8sAgent.GET(web.ServicePlansURL).
									Expect().
									Status(http.StatusOK).JSON().Object().Value("service_plans").Array().Length().Equal(1)
							})

							It("should return only one plan with id in field query", func() {
								result := k8sAgent.GET(web.ServicePlansURL).WithQuery("fieldQuery", "id in ["+planID1+"||"+planID2+"]").
									Expect().
									Status(http.StatusOK).JSON().Object().Value("service_plans").Array()
								result.Length().Equal(1)
								result.First().Object().ValueEqual("id", planID1)
							})

							It("should return only empty plan list with id not in field query", func() {
								result := k8sAgent.GET(web.ServicePlansURL).WithQuery("fieldQuery", "id notin ["+planID1+"]").
									Expect().
									Status(http.StatusOK).JSON().Object().Value("service_plans").Array()
								result.Length().Equal(0)
							})

							It("should return only empty plan list with id in not visible id field query", func() {
								result := k8sAgent.GET(web.ServicePlansURL).WithQuery("fieldQuery", "id in ["+planID2+"]").
									Expect().
									Status(http.StatusOK).JSON().Object().Value("service_plans").Array()
								result.Length().Equal(0)
							})

							It("should return only one plan with catalog_name in query", func() {
								plan1CatalogName := plan["catalog_name"].(string)
								plan2CatalogName := plan2["catalog_name"].(string)
								result := k8sAgent.GET(web.ServicePlansURL).WithQuery("fieldQuery", "catalog_name in ["+plan1CatalogName+"||"+plan2CatalogName+"]").
									Expect().
									Status(http.StatusOK).JSON().Object().Value("service_plans").Array()
								result.Length().Equal(1)
								result.First().Object().ValueEqual("id", planID1)
							})

						})

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

					plan := blueprint(ctx, ctx.SMWithOAuth)
					id = plan["id"].(string)

					ctx.SMWithOAuth.PATCH(web.ServicePlansURL + "/" + id).
						WithJSON(initialLabelsBody).
						Expect().
						Status(http.StatusOK)

				})

				Context("Add new label", func() {
					It("Should return 200", func() {
						label := types.Labels{changedLabelKey: changedLabelValues}
						ctx.SMWithOAuth.PATCH(web.ServicePlansURL + "/" + id).
							WithJSON(patchLabelsBody).
							Expect().
							Status(http.StatusOK).JSON().Object().Value("labels").Object().ContainsMap(label)
					})
				})

				Context("Add label with existing key and value", func() {
					It("Should return 200", func() {
						ctx.SMWithOAuth.PATCH(web.ServicePlansURL + "/" + id).
							WithJSON(patchLabelsBody).
							Expect().
							Status(http.StatusOK)

						ctx.SMWithOAuth.PATCH(web.ServicePlansURL + "/" + id).
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
						ctx.SMWithOAuth.PATCH(web.ServicePlansURL + "/" + id).
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

						ctx.SMWithOAuth.PATCH(web.ServicePlansURL + "/" + id).
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
						ctx.SMWithOAuth.PATCH(web.ServicePlansURL + "/" + id).
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
						ctx.SMWithOAuth.PATCH(web.ServicePlansURL + "/" + id).
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
						ctx.SMWithOAuth.PATCH(web.ServicePlansURL + "/" + id).
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
						ctx.SMWithOAuth.PATCH(web.ServicePlansURL + "/" + id).
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
						ctx.SMWithOAuth.PATCH(web.ServicePlansURL + "/" + id).
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
						ctx.SMWithOAuth.PATCH(web.ServicePlansURL + "/" + id).
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
						ctx.SMWithOAuth.PATCH(web.ServicePlansURL + "/" + id).
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
						ctx.SMWithOAuth.PATCH(web.ServicePlansURL + "/" + id).
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
						ctx.SMWithOAuth.PATCH(web.ServicePlansURL + "/" + id).
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
	cPaidPlan := common.GeneratePaidTestPlan()
	cService := common.GenerateTestServiceWithPlans(cPaidPlan)
	catalog := common.NewEmptySBCatalog()
	catalog.AddService(cService)
	id, _, _ := ctx.RegisterBrokerWithCatalog(catalog)

	so := auth.GET(web.ServiceOfferingsURL).WithQuery("fieldQuery", "broker_id = "+id).
		Expect().
		Status(http.StatusOK).JSON().Object().Value("service_offerings").Array().First()

	sp := auth.GET(web.ServicePlansURL).WithQuery("fieldQuery", fmt.Sprintf("service_offering_id = %s", so.Object().Value("id").String().Raw())).
		Expect().
		Status(http.StatusOK).JSON().Object().Value("service_plans").Array().First()

	return sp.Object().Raw()
}
