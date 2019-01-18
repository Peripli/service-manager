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

package visibility_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"

	"github.com/Peripli/service-manager/test"

	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestVisibilities(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Visibilities API Tests Suite")
}

var _ = test.DescribeTestsFor(test.TestCase{
	API:            "/v1/visibilities",
	SupportsLabels: true,
	SupportedOps: []test.Op{
		test.Get, test.List, test.Delete, test.DeleteList,
	},
	ResourceBlueprint:                      blueprint(true),
	ResourceWithoutNullableFieldsBlueprint: blueprint(false),
	AdditionalTests: func(ctx *common.TestContext) {
		Context("non-generic tests", func() {
			var (
				existingPlatformID string
				existingBrokerID   string
				existingPlanIDs    []interface{}

				labels                          common.Object
				postVisibilityRequestNoLabels   common.Object
				postVisibilityRequestWithLabels labeledVisibility
			)

			BeforeEach(func() {
				existingBrokerID, _, _ = ctx.RegisterBroker()
				Expect(existingBrokerID).ToNot(BeEmpty())

				platform := ctx.TestPlatform
				existingPlatformID = platform.ID
				Expect(existingPlatformID).ToNot(BeEmpty())

				existingPlanIDs = ctx.SMWithOAuth.GET("/v1/service_plans").
					Expect().Status(http.StatusOK).
					JSON().Path("$.service_plans[*].id").Array().Raw()
				length := len(existingPlanIDs)
				Expect(length).Should(BeNumerically(">=", 2))

				postVisibilityRequestNoLabels = common.Object{
					"platform_id":     existingPlatformID,
					"service_plan_id": existingPlanIDs[0],
				}

				labels = common.Object{
					"cluster_id": common.Array{"cluster_id_value"},
					"org_id":     common.Array{"org_id_value1", "org_id_value2", "org_id_value3"},
				}

				registerPlatform := ctx.RegisterPlatform()
				postVisibilityRequestWithLabels = common.Object{
					"platform_id":     registerPlatform.ID,
					"service_plan_id": existingPlanIDs[1],
					"labels":          labels,
				}

				common.RemoveAllVisibilities(ctx.SMWithOAuth)

			})

			Describe("POST", func() {
				Context("With invalid content type", func() {
					It("returns 415", func() {
						ctx.SMWithOAuth.POST("/v1/visibilities").
							WithText("text").
							Expect().Status(http.StatusUnsupportedMediaType)
					})
				})

				Context("With invalid content JSON", func() {
					It("returns 400", func() {
						ctx.SMWithOAuth.POST("/v1/visibilities").
							WithText("invalid json").
							WithHeader("content-type", "application/json").
							Expect().Status(http.StatusBadRequest)
					})
				})

				Context("With missing mandatory fields", func() {
					It("returns 400", func() {
						ctx.SMWithOAuth.POST("/v1/visibilities").
							WithJSON(postVisibilityRequestNoLabels).
							Expect().Status(http.StatusCreated)

						for _, prop := range []string{"service_plan_id"} {
							delete(postVisibilityRequestNoLabels, prop)

							ctx.SMWithOAuth.POST("/v1/visibilities").
								WithJSON(postVisibilityRequestNoLabels).
								Expect().Status(http.StatusBadRequest).JSON().Object().Keys().Contains("error", "description")
						}
					})
				})

				Context("with not existing related platform", func() {
					It("returns 400", func() {
						platformId := "not-existing"
						ctx.SMWithOAuth.GET("/v1/platforms/"+platformId).
							WithJSON(postVisibilityRequestNoLabels).
							Expect().Status(http.StatusNotFound).JSON().Object().Keys().Contains("error", "description")

						ctx.SMWithOAuth.POST("/v1/visibilities").
							WithJSON(common.Object{
								"service_plan_id": existingPlanIDs[0],
								"platform_id":     platformId,
							}).
							Expect().Status(http.StatusBadRequest).JSON().Object().Keys().Contains("error", "description")
					})
				})

				Context("with missing platform id field", func() {
					It("returns 201 if no visibilities for the plan exist", func() {
						ctx.SMWithOAuth.GET("/v1/visibilities").
							Expect().Status(http.StatusOK).JSON().Path("$.visibilities[*].id").Array().NotContains(existingPlanIDs[1])

						ctx.SMWithOAuth.POST("/v1/visibilities").
							WithJSON(common.Object{
								"service_plan_id": existingPlanIDs[1],
							}).
							Expect().Status(http.StatusCreated).JSON().Object().ContainsMap(common.Object{
							"service_plan_id": existingPlanIDs[1],
						})
					})

					It("returns 400 if visibilities for the plan exist", func() {
						ctx.SMWithOAuth.POST("/v1/visibilities").
							WithJSON(common.Object{
								"service_plan_id": existingPlanIDs[0],
								"platform_id":     existingPlatformID,
							}).
							Expect().Status(http.StatusCreated)

						ctx.SMWithOAuth.GET("/v1/visibilities").
							Expect().Status(http.StatusOK).JSON().Path("$.visibilities[*].service_plan_id").Array().Contains(existingPlanIDs[0])

						ctx.SMWithOAuth.POST("/v1/visibilities").
							WithJSON(common.Object{
								"service_plan_id": existingPlanIDs[0],
							}).
							Expect().Status(http.StatusBadRequest).JSON().Object().Keys().Contains("error", "description")
					})
				})

				Context("with not existing related service plan", func() {
					It("returns 400", func() {
						planID := "not-existing"
						ctx.SMWithOAuth.GET("/v1/service_plans/"+planID).
							WithJSON(postVisibilityRequestNoLabels).
							Expect().Status(http.StatusNotFound).JSON().Object().Keys().Contains("error", "description")

						ctx.SMWithOAuth.POST("/v1/visibilities").
							WithJSON(common.Object{
								"platform_id":     existingPlatformID,
								"service_plan_id": planID,
							}).
							Expect().Status(http.StatusBadRequest).JSON().Object().Keys().Contains("error", "description")
					})
				})

				Context("with missing related service plan", func() {
					It("returns 400", func() {
						ctx.SMWithOAuth.POST("/v1/visibilities").
							WithJSON(common.Object{
								"platform_id": existingPlatformID,
							}).
							Expect().Status(http.StatusBadRequest).JSON().Object().Keys().Contains("error", "description")
					})
				})

				Context("with valid existing platform id and service plan id", func() {
					Context("when a record with the same platform id and service plan id already exists", func() {
						It("returns 409", func() {
							ctx.SMWithOAuth.POST("/v1/visibilities").
								WithJSON(postVisibilityRequestNoLabels).
								Expect().Status(http.StatusCreated)

							ctx.SMWithOAuth.POST("/v1/visibilities").
								WithJSON(postVisibilityRequestNoLabels).
								Expect().Status(http.StatusConflict).JSON().Object().Keys().Contains("error", "description")
						})
					})

					Context("when a record with null platform id and the same service plan id already exists", func() {
						It("returns 400", func() {
							ctx.SMWithOAuth.POST("/v1/visibilities").
								WithJSON(common.Object{
									"service_plan_id": existingPlanIDs[0],
								}).
								Expect().Status(http.StatusCreated)

							ctx.SMWithOAuth.POST("/v1/visibilities").
								WithJSON(common.Object{
									"service_plan_id": existingPlanIDs[0],
									"platform_id":     existingPlatformID,
								}).
								Expect().Status(http.StatusBadRequest).JSON().Object().Keys().Contains("error", "description")

						})
					})

					Context("when a record with the same or null platform id does not exist", func() {
						It("returns 201", func() {
							ctx.SMWithOAuth.POST("/v1/visibilities").
								WithJSON(postVisibilityRequestNoLabels).
								Expect().Status(http.StatusCreated).JSON().Object().ContainsMap(postVisibilityRequestNoLabels).Keys().Contains("id")
						})
					})
				})
				Context("Labelled", func() {
					Context("When labels are valid", func() {
						It("should return 201", func() {
							ctx.SMWithOAuth.POST("/v1/visibilities").
								WithJSON(postVisibilityRequestWithLabels).
								Expect().Status(http.StatusCreated).JSON().Object().Keys().Contains("id", "labels")
						})
					})

					Context("When creating labeled visibility for which a public one exists", func() {
						It("Should return 409", func() {
							ctx.SMWithOAuth.POST("/v1/visibilities").
								WithJSON(postVisibilityRequestNoLabels).
								Expect().Status(http.StatusCreated)

							oldVisibility := postVisibilityRequestNoLabels
							oldVisibility["labels"] = labels
							ctx.SMWithOAuth.POST("/v1/visibilities").
								WithJSON(oldVisibility).
								Expect().Status(http.StatusConflict)
						})
					})

					Context("When creating labeled visibility with key containing forbidden character", func() {
						It("Should return 400", func() {
							labels[fmt.Sprintf("containing%cseparator", query.Separator)] = common.Array{"val"}
							ctx.SMWithOAuth.POST("/v1/visibilities").
								WithJSON(postVisibilityRequestWithLabels).
								Expect().Status(http.StatusBadRequest).JSON().Object().Value("description").String().Contains("cannot contain whitespaces and special symbol")
						})
					})

					Context("When label key has new line", func() {
						It("Should return 400", func() {
							labels[`key with
	new line`] = common.Array{"label-value"}
							ctx.SMWithOAuth.POST("/v1/visibilities").
								WithJSON(postVisibilityRequestWithLabels).
								Expect().Status(http.StatusBadRequest).JSON().Object().Value("description").String().Contains("cannot contain whitespaces and special symbol")
						})
					})

					Context("When label value has new line", func() {
						It("Should return 400", func() {
							labels["cluster_id"] = common.Array{`{
	"key": "k1",
	"val": "val1"
	}`}
							ctx.SMWithOAuth.POST("/v1/visibilities").
								WithJSON(postVisibilityRequestWithLabels).
								Expect().Status(http.StatusBadRequest)
						})
					})
				})
			})

			Describe("PATCH", func() {
				var existingVisibilityID string
				var existingVisibilityReqBody common.Object
				var updatedVisibilityReqBody common.Object
				var expectedUpdatedVisibilityRespBody common.Object
				var anotherExistingPlatformID string

				BeforeEach(func() {
					anotherPlatform := ctx.RegisterPlatform()
					anotherExistingPlatformID = anotherPlatform.ID

					existingVisibilityReqBody = common.Object{
						"platform_id":     existingPlatformID,
						"service_plan_id": existingPlanIDs[0],
					}

					updatedVisibilityReqBody = common.Object{
						"platform_id":     anotherExistingPlatformID,
						"service_plan_id": existingPlanIDs[1],
					}

					existingVisibilityID = ctx.SMWithOAuth.POST("/v1/visibilities").
						WithJSON(existingVisibilityReqBody).
						Expect().Status(http.StatusCreated).JSON().Object().Value("id").String().Raw()

				})

				Context("when updating properties with valid values", func() {
					BeforeEach(func() {
						expectedUpdatedVisibilityRespBody = common.Object{
							"id":              existingVisibilityID,
							"platform_id":     anotherExistingPlatformID,
							"service_plan_id": existingPlanIDs[1],
						}
					})

					It("returns 200", func() {
						ctx.SMWithOAuth.PATCH("/v1/visibilities/" + existingVisibilityID).
							WithJSON(updatedVisibilityReqBody).
							Expect().
							Status(http.StatusOK).JSON().Object().ContainsMap(expectedUpdatedVisibilityRespBody)

						ctx.SMWithOAuth.GET("/v1/visibilities/" + existingVisibilityID).
							Expect().
							Status(http.StatusOK).JSON().Object().ContainsMap(expectedUpdatedVisibilityRespBody)
					})
				})

				Context("when update is partial", func() {
					BeforeEach(func() {
						expectedUpdatedVisibilityRespBody = common.Object{
							"id":              existingVisibilityID,
							"platform_id":     existingPlatformID,
							"service_plan_id": existingPlanIDs[0],
						}
					})

					It("returns 200 and patches the resource, keeping current values and overriding only provided values", func() {
						for prop, val := range updatedVisibilityReqBody {
							update := common.Object{}
							update[prop] = val
							expectedUpdatedVisibilityRespBody[prop] = val
							ctx.SMWithOAuth.PATCH("/v1/visibilities/" + existingVisibilityID).
								WithJSON(update).
								Expect().
								Status(http.StatusOK).JSON().Object().ContainsMap(expectedUpdatedVisibilityRespBody)

							ctx.SMWithOAuth.GET("/v1/visibilities/" + existingVisibilityID).
								Expect().
								Status(http.StatusOK).JSON().Object().ContainsMap(expectedUpdatedVisibilityRespBody)
						}
					})
				})

				Context("when created_at is in the body", func() {
					It("should not update created_at", func() {
						createdAt := "2015-01-01T00:00:00Z"

						ctx.SMWithOAuth.PATCH("/v1/visibilities/"+existingVisibilityID).
							WithJSON(common.Object{
								"created_at": createdAt,
							}).
							Expect().
							Status(http.StatusOK).JSON().Object().
							ContainsKey("created_at").
							ValueNotEqual("created_at", createdAt)

						ctx.SMWithOAuth.GET("/v1/visibilities/"+existingVisibilityID).
							Expect().
							Status(http.StatusOK).JSON().Object().
							ContainsKey("created_at").
							ValueNotEqual("created_at", createdAt)
					})
				})

				Context("when updated_at is in the body", func() {
					It("should not update updated_at", func() {
						updatedAt := "2015-01-01T00:00:00Z"

						ctx.SMWithOAuth.PATCH("/v1/visibilities/"+existingVisibilityID).
							WithJSON(common.Object{
								"updated_at": updatedAt,
							}).
							Expect().
							Status(http.StatusOK).JSON().Object().
							ContainsKey("updated_at").
							ValueNotEqual("updated_at", updatedAt)

						ctx.SMWithOAuth.GET("/v1/visibilities/"+existingVisibilityID).
							Expect().
							Status(http.StatusOK).JSON().Object().
							ContainsKey("updated_at").
							ValueNotEqual("updated_at", updatedAt)
					})
				})

				Context("when id is in the body", func() {
					It("should not update the id", func() {
						id := "123"
						ctx.SMWithOAuth.PATCH("/v1/visibilities/" + existingVisibilityID).
							WithJSON(common.Object{
								"id": id,
							}).
							Expect().Status(http.StatusOK)

						ctx.SMWithOAuth.GET("/v1/visibilities/"+id).
							Expect().Status(http.StatusNotFound).JSON().Object().Keys().Contains("error", "description")
					})
				})

				Context("when related service plan does not exist", func() {
					It("returns 400", func() {
						planID := "does-not-exist"
						ctx.SMWithOAuth.GET("/v1/service_plans/"+planID).
							Expect().Status(http.StatusNotFound).JSON().Object().Keys().Contains("error", "description")

						ctx.SMWithOAuth.PATCH("/v1/visibilities/"+existingVisibilityID).
							WithJSON(common.Object{
								"platform_id":     existingPlatformID,
								"service_plan_id": planID,
							}).
							Expect().Status(http.StatusBadRequest).JSON().Object().Keys().Contains("error", "description")
					})
				})

				Context("when related platform does not exist", func() {
					It("returns 400", func() {
						platformID := "does-not-exist"
						ctx.SMWithOAuth.GET("/v1/platforms/"+platformID).
							Expect().Status(http.StatusNotFound).JSON().Object().Keys().Contains("error", "description")

						ctx.SMWithOAuth.PATCH("/v1/visibilities/"+existingVisibilityID).
							WithJSON(common.Object{
								"platform_id":     platformID,
								"service_plan_id": existingPlanIDs[0],
							}).
							Expect().Status(http.StatusBadRequest).JSON().Object().Keys().Contains("error", "description")
					})
				})

				Context("when visibility does not exist", func() {
					It("returns 404", func() {
						id := "does-not-exist"
						ctx.SMWithOAuth.GET("/v1/visibilities/"+id).
							Expect().Status(http.StatusNotFound).JSON().Object().Keys().Contains("error", "description")

						ctx.SMWithOAuth.PATCH("/v1/visibilities/"+id).
							WithJSON(common.Object{}).
							Expect().
							Status(http.StatusNotFound).JSON().Object().Keys().Contains("error", "description")
					})
				})

				Describe("Labelled", func() {
					var id string
					var patchLabels []query.LabelChange
					var patchLabelsBody map[string]interface{}
					changedLabelKey := "label_key"
					changedLabelValues := []string{"label_value1", "label_value2"}
					operation := query.AddLabelOperation
					BeforeEach(func() {
						patchLabels = []query.LabelChange{}
					})
					JustBeforeEach(func() {
						patchLabelsBody = make(map[string]interface{})
						patchLabels = append(patchLabels, query.LabelChange{
							Operation: operation,
							Key:       changedLabelKey,
							Values:    changedLabelValues,
						})
						patchLabelsBody["labels"] = patchLabels

						id = ctx.SMWithOAuth.POST("/v1/visibilities").
							WithJSON(postVisibilityRequestWithLabels).
							Expect().Status(http.StatusCreated).JSON().Object().Value("id").String().Raw()
					})

					Context("Add new label", func() {
						It("Should return 200", func() {
							label := types.Labels{changedLabelKey: changedLabelValues}
							ctx.SMWithOAuth.PATCH("/v1/visibilities/" + id).
								WithJSON(patchLabelsBody).
								Expect().
								Status(http.StatusOK).JSON().Object().Value("labels").Object().ContainsMap(label)
						})
					})

					Context("Add label with existing key and value", func() {
						It("Should return 400", func() {
							ctx.SMWithOAuth.PATCH("/v1/visibilities/" + id).
								WithJSON(patchLabelsBody).
								Expect().
								Status(http.StatusOK)

							ctx.SMWithOAuth.PATCH("/v1/visibilities/" + id).
								WithJSON(patchLabelsBody).
								Expect().
								Status(http.StatusBadRequest)
						})
					})

					Context("Add new label value", func() {
						BeforeEach(func() {
							operation = query.AddLabelValuesOperation
							changedLabelKey = "cluster_id"
							changedLabelValues = []string{"new-label-value"}
						})
						It("Should return 200", func() {
							var labelValuesObj []interface{}
							for _, val := range changedLabelValues {
								labelValuesObj = append(labelValuesObj, val)
							}
							ctx.SMWithOAuth.PATCH("/v1/visibilities/" + id).
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

							ctx.SMWithOAuth.PATCH("/v1/visibilities/" + id).
								WithJSON(patchLabelsBody).
								Expect().
								Status(http.StatusOK).JSON().
								Path("$.labels").Object().Values().Path("$[*][*]").Array().Contains(labelValuesObj...)
						})
					})

					Context("Add duplicate label value", func() {
						BeforeEach(func() {
							operation = query.AddLabelValuesOperation
							changedLabelKey = "cluster_id"
							values := labels["cluster_id"].([]interface{})
							changedLabelValues = []string{values[0].(string)}
						})
						It("Should return 400", func() {
							ctx.SMWithOAuth.PATCH("/v1/visibilities/" + id).
								WithJSON(patchLabelsBody).
								Expect().
								Status(http.StatusBadRequest).JSON().Object().
								Value("description").String().Contains("already exists")
						})
					})

					Context("Remove a label", func() {
						BeforeEach(func() {
							operation = query.RemoveLabelOperation
							changedLabelKey = "cluster_id"
						})
						It("Should return 200", func() {
							ctx.SMWithOAuth.PATCH("/v1/visibilities/" + id).
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
							ctx.SMWithOAuth.PATCH("/v1/visibilities/" + id).
								WithJSON(patchLabelsBody).
								Expect().
								Status(http.StatusBadRequest)
						})
					})

					Context("Remove a label key which does not exist", func() {
						BeforeEach(func() {
							operation = query.RemoveLabelOperation
							changedLabelKey = "non-existing-ey"
						})
						It("Should return 400", func() {
							ctx.SMWithOAuth.PATCH("/v1/visibilities/" + id).
								WithJSON(patchLabelsBody).
								Expect().
								Status(http.StatusBadRequest)
						})
					})

					Context("Remove label values and providing a single value", func() {
						var valueToRemove string
						BeforeEach(func() {
							operation = query.RemoveLabelValuesOperation
							changedLabelKey = "cluster_id"
							valueToRemove = labels[changedLabelKey].([]interface{})[0].(string)
							changedLabelValues = []string{valueToRemove}
						})
						It("Should return 200", func() {
							ctx.SMWithOAuth.PATCH("/v1/visibilities/" + id).
								WithJSON(patchLabelsBody).
								Expect().
								Status(http.StatusOK).JSON().
								Path("$.labels[*].value[*]").Array().NotContains(valueToRemove)
						})
					})

					Context("Remove label values and providing multiple values", func() {
						var valuesToRemove []string
						BeforeEach(func() {
							operation = query.RemoveLabelValuesOperation
							changedLabelKey = "org_id"
							val1 := labels[changedLabelKey].([]interface{})[0].(string)
							val2 := labels[changedLabelKey].([]interface{})[1].(string)
							valuesToRemove = []string{val1, val2}
							changedLabelValues = valuesToRemove
						})
						It("Should return 200", func() {
							ctx.SMWithOAuth.PATCH("/v1/visibilities/" + id).
								WithJSON(patchLabelsBody).
								Expect().
								Status(http.StatusOK).JSON().
								Path("$.labels[*].value[*]").Array().NotContains(valuesToRemove)
						})
					})

					Context("Remove all label values for a key", func() {
						var valuesToRemove []string
						BeforeEach(func() {
							operation = query.RemoveLabelValuesOperation
							changedLabelKey = "cluster_id"
							labelValues := labels[changedLabelKey].([]interface{})
							for _, val := range labelValues {
								valuesToRemove = append(valuesToRemove, val.(string))
							}
							changedLabelValues = valuesToRemove
						})
						It("Should return 200 with this key gone", func() {
							ctx.SMWithOAuth.PATCH("/v1/visibilities/" + id).
								WithJSON(patchLabelsBody).
								Expect().
								Status(http.StatusOK).JSON().
								Path("$.labels[*].key[*]").Array().NotContains(changedLabelKey)
						})
					})

					Context("Remove label values and not providing value to remove", func() {
						BeforeEach(func() {
							operation = query.RemoveLabelValuesOperation
							changedLabelValues = []string{}
						})
						It("Should return 400", func() {
							ctx.SMWithOAuth.PATCH("/v1/visibilities/" + id).
								WithJSON(patchLabelsBody).
								Expect().
								Status(http.StatusBadRequest)
						})
					})

					Context("Remove label value which does not exist", func() {
						BeforeEach(func() {
							operation = query.RemoveLabelValuesOperation
							changedLabelKey = "cluster_id"
							changedLabelValues = []string{"non-existing-value"}
						})
						It("Should return 400", func() {
							ctx.SMWithOAuth.PATCH("/v1/visibilities/" + id).
								WithJSON(patchLabelsBody).
								Expect().
								Status(http.StatusBadRequest)
						})
					})
				})
			})
		})

	},
})

func blueprint(setNullFieldsValues bool) func(ctx *common.TestContext) common.Object {
	return func(ctx *common.TestContext) common.Object {
		visReqBody := make(common.Object, 0)
		cPaidPlan := common.GeneratePaidTestPlan()
		cService := common.GenerateTestServiceWithPlans(cPaidPlan)
		catalog := common.NewEmptySBCatalog()
		catalog.AddService(cService)
		id, _, _ := ctx.RegisterBrokerWithCatalog(catalog)

		object := ctx.SMWithOAuth.GET("/v1/service_offerings").WithQuery("fieldQuery", "broker_id = "+id).
			Expect()

		so := object.Status(http.StatusOK).JSON().Object().Value("service_offerings").Array().First()

		servicePlanID := ctx.SMWithOAuth.GET("/v1/service_plans").WithQuery("fieldQuery", fmt.Sprintf("service_offering_id = %s", so.Object().Value("id").String().Raw())).
			Expect().
			Status(http.StatusOK).JSON().Object().Value("service_plans").Array().First().Object().Value("id").String().Raw()
		visReqBody["service_plan_id"] = servicePlanID
		if setNullFieldsValues {
			platformID := ctx.SMWithOAuth.POST("/v1/platforms").WithJSON(common.GenerateRandomPlatform()).
				Expect().
				Status(http.StatusCreated).JSON().Object().Value("id").String().Raw()
			visReqBody["platform_id"] = platformID
		}

		visibility := ctx.SMWithOAuth.POST("/v1/visibilities").WithJSON(visReqBody).Expect().
			Status(http.StatusCreated).JSON().Object().Raw()
		return visibility
	}
}

type labeledVisibility common.Object

func (vis labeledVisibility) AddLabel(label common.Object) {
	vis["labels"] = append(vis["labels"].(common.Array), label)
}
