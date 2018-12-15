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

	"github.com/Peripli/service-manager/pkg/types"

	"github.com/Peripli/service-manager/pkg/query"

	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestVisibilities(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Platform API Tests Suite")
}

type labeledVisibility common.Object

func (vis labeledVisibility) AddLabel(label common.Object) {
	vis["labels"] = append(vis["labels"].(common.Array), label)
}

var _ = Describe("Service Manager Platform API", func() {
	var (
		ctx                *common.TestContext
		existingPlatformID string
		existingBrokerID   string
		existingPlanIDs    []interface{}

		labels                          common.Array
		postVisibilityRequestNoLabels   common.Object
		postVisibilityRequestWithLabels labeledVisibility
	)

	BeforeSuite(func() {
		ctx = common.NewTestContext(nil)
	})

	BeforeEach(func() {

		existingBrokerID, _ = ctx.RegisterBroker()
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

		labels = common.Array{
			common.Object{
				"key":   "org_id",
				"value": common.Array{"org_id_value1", "org_id_value2", "org_id_value3"},
			},
			common.Object{
				"key":   "cluster_id",
				"value": common.Array{"cluster_id_value"},
			},
		}

		registerPlatform := ctx.RegisterPlatform()
		postVisibilityRequestWithLabels = common.Object{
			"platform_id":     registerPlatform.ID,
			"service_plan_id": existingPlanIDs[1],
			"labels":          labels,
		}

		common.RemoveAllVisibilities(ctx.SMWithOAuth)

	})

	AfterSuite(func() {
		ctx.Cleanup()
	})

	Describe("GET", func() {
		Context("Missing visibility", func() {
			It("returns 404", func() {
				ctx.SMWithOAuth.GET("/v1/visibilities/999").
					Expect().
					Status(http.StatusNotFound).
					JSON().Object().Keys().Contains("error", "description")
			})
		})

		Context("Existing visibility", func() {
			var visibilityID string

			BeforeEach(func() {
				visibilityID = ctx.SMWithOAuth.POST("/v1/visibilities").
					WithJSON(postVisibilityRequestNoLabels).
					Expect().Status(http.StatusCreated).JSON().Object().ContainsMap(postVisibilityRequestNoLabels).
					Value("id").String().Raw()

			})

			It("returns the platform with given id", func() {
				ctx.SMWithOAuth.GET("/v1/visibilities/" + visibilityID).
					Expect().
					Status(http.StatusOK).
					JSON().Object().ContainsMap(postVisibilityRequestNoLabels)
			})
		})
	})

	Describe("List", func() {
		Context("With no visibilities", func() {
			It("returns empty array", func() {
				ctx.SMWithOAuth.GET("/v1/visibilities").
					Expect().
					Status(http.StatusOK).
					JSON().Object().Value("visibilities").Array().Empty()
			})
		})

		Context("With some visibilities", func() {

			var anotherPlatformID string
			var postVisibilityRequestForAnotherPlatform common.Object
			var postVisibilityForAllPlatforms common.Object
			var expectedVisibilities common.Array

			BeforeEach(func() {
				// register a visibility for the existing platform
				json := ctx.SMWithOAuth.POST("/v1/visibilities").
					WithJSON(postVisibilityRequestNoLabels).
					Expect().Status(http.StatusCreated).JSON().Object()

				visibilityID := json.Value("id").String().Raw()
				postVisibilityRequestNoLabels["id"] = visibilityID

				json.ContainsMap(postVisibilityRequestNoLabels)

				// register another platform
				anotherPlatform := ctx.RegisterPlatform()
				anotherPlatformID = anotherPlatform.ID

				postVisibilityRequestForAnotherPlatform = common.Object{
					"platform_id":     anotherPlatformID,
					"service_plan_id": existingPlanIDs[0],
				}

				// add a visibility related to the new platform
				json = ctx.SMWithOAuth.POST("/v1/visibilities").
					WithJSON(postVisibilityRequestForAnotherPlatform).
					Expect().Status(http.StatusCreated).JSON().Object()

				anotherPlatformVisibilityID := json.Value("id").String().Raw()
				postVisibilityRequestForAnotherPlatform["id"] = anotherPlatformVisibilityID

				json.ContainsMap(postVisibilityRequestForAnotherPlatform)

				// add a visibility related to no platform
				postVisibilityForAllPlatforms = common.Object{
					"service_plan_id": existingPlanIDs[1],
					"platform_id":     "",
				}

				json = ctx.SMWithOAuth.POST("/v1/visibilities").
					WithJSON(postVisibilityForAllPlatforms).
					Expect().Status(http.StatusCreated).JSON().Object()

				noPlatformVisibilityID := json.Value("id").String().Raw()
				postVisibilityForAllPlatforms["id"] = noPlatformVisibilityID

				json.ContainsMap(postVisibilityForAllPlatforms)
			})

			Context("when authentication is oauth", func() {
				BeforeEach(func() {
					expectedVisibilities = common.Array{postVisibilityRequestNoLabels, postVisibilityRequestForAnotherPlatform, postVisibilityForAllPlatforms}
				})

				It("returns all the visibilities if authn is oauth", func() {
					array := ctx.SMWithOAuth.GET("/v1/visibilities").
						Expect().Status(http.StatusOK).JSON().Object().Value("visibilities").Array()

					for _, v := range array.Iter() {
						obj := v.Object().Raw()
						delete(obj, "created_at")
						delete(obj, "updated_at")
					}
					array.Contains(expectedVisibilities...)
				})
			})

			Context("when authentication is basic", func() {
				BeforeEach(func() {
					expectedVisibilities = common.Array{postVisibilityRequestNoLabels, postVisibilityForAllPlatforms}
				})

				It("returns the visibilities with the credentials' platform id and the visibilities with null platform id if authn is basic", func() {
					array := ctx.SMWithBasic.GET("/v1/visibilities").
						Expect().Status(http.StatusOK).JSON().Object().Value("visibilities").Array()

					for _, v := range array.Iter() {
						obj := v.Object().Raw()
						delete(obj, "created_at")
						delete(obj, "updated_at")
					}
					array.Contains(expectedVisibilities...)

				})
			})

		})
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
		})

		Describe("DELETE", func() {
			Context("Non existing visibility", func() {
				It("returns 404", func() {
					ctx.SMWithOAuth.DELETE("/v1/visibilities/999").
						Expect().
						Status(http.StatusNotFound).JSON().Object().Keys().Contains("error", "description")
				})
			})

			Context("Existing visibility", func() {
				It("returns 200", func() {
					id := ctx.SMWithOAuth.POST("/v1/visibilities").
						WithJSON(postVisibilityRequestNoLabels).
						Expect().Status(http.StatusCreated).JSON().Object().Value("id").String().Raw()

					ctx.SMWithOAuth.GET("/v1/visibilities/" + id).
						Expect().
						Status(http.StatusOK)

					ctx.SMWithOAuth.DELETE("/v1/visibilities/" + id).
						Expect().
						Status(http.StatusOK).JSON().Object().Empty()

					ctx.SMWithOAuth.GET("/v1/visibilities/"+id).
						Expect().
						Status(http.StatusNotFound).JSON().Object().Keys().Contains("error", "description")
				})
			})
		})
	})

	Describe("Labelled", func() {
		Describe("POST", func() {
			Context("When labels are valid", func() {
				It("should return 201", func() {
					ctx.SMWithOAuth.POST("/v1/visibilities").
						WithJSON(postVisibilityRequestWithLabels).
						Expect().Status(http.StatusCreated).JSON().Object().Keys().Contains("id", "labels")
				})
			})

			Context("When labels have duplicates", func() {
				It("should return 400", func() {
					visibility := postVisibilityRequestWithLabels
					visibility.AddLabel(labels[0].(common.Object))
					description := ctx.SMWithOAuth.POST("/v1/visibilities").
						WithJSON(visibility).
						Expect().Status(http.StatusBadRequest).JSON().Object().Value("description").Raw().(string)
					Expect(description).To(ContainSubstring("duplicate"))
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
		})

		Describe("DELETE", func() {
			Context("When field query uses missing field", func() {
				It("Should return 400", func() {
					description := ctx.SMWithOAuth.DELETE("/v1/visibilities").
						WithQuery(string(query.FieldQuery), "missing_field = some_value").
						Expect().Status(http.StatusBadRequest).JSON().Object().Value("description").Raw().(string)
					Expect(description).To(ContainSubstring("unsupported"))
				})
			})

			Context("When query operator is missing", func() {
				It("Should return 400", func() {
					description := ctx.SMWithOAuth.DELETE("/v1/visibilities").
						WithQuery(string(query.FieldQuery), "missing_fieldsome_value").
						Expect().Status(http.StatusBadRequest).JSON().Object().Value("description").Raw().(string)
					Expect(description).To(ContainSubstring("missing"))
				})
			})

			Context("When deleting by label query", func() {
				It("Should return 400", func() {
					description := ctx.SMWithOAuth.DELETE("/v1/visibilities").
						WithQuery(string(query.LabelQuery), "platform_id = some_value").
						Expect().Status(http.StatusBadRequest).JSON().Object().Value("description").Raw().(string)
					Expect(description).To(ContainSubstring("conditional delete "))
				})
			})

			Context("When deleting by field for which no records exist", func() {
				It("Should return 404", func() {
					ctx.SMWithOAuth.DELETE("/v1/visibilities").
						WithQuery(string(query.FieldQuery), "platform_id = missing_value").
						Expect().Status(http.StatusNotFound).JSON().Object().Keys().Contains("error", "description")
				})
			})

			Context("When deleting by field for which a record exists", func() {
				It("Should return 200", func() {
					id := ctx.SMWithOAuth.POST("/v1/visibilities").
						WithJSON(postVisibilityRequestNoLabels).
						Expect().Status(http.StatusCreated).JSON().Object().Value("id").String().Raw()

					ctx.SMWithOAuth.GET("/v1/visibilities/" + id).
						Expect().
						Status(http.StatusOK)

					ctx.SMWithOAuth.DELETE("/v1/visibilities").
						WithQuery(string(query.FieldQuery), fmt.Sprintf("platform_id = %s", postVisibilityRequestNoLabels["platform_id"])).
						Expect().
						Status(http.StatusOK).JSON().Object().Empty()

					ctx.SMWithOAuth.GET("/v1/visibilities/"+id).
						Expect().
						Status(http.StatusNotFound).JSON().Object().Keys().Contains("error", "description")
				})
			})
		})

		Describe("LIST", func() {
			var id string
			var platformID string
			BeforeEach(func() {
				id = ctx.SMWithOAuth.POST("/v1/visibilities").
					WithJSON(postVisibilityRequestWithLabels).
					Expect().Status(http.StatusCreated).JSON().Object().Value("id").String().Raw()
				platformID = postVisibilityRequestWithLabels["platform_id"].(string)
			})
			Context("With id field query", func() {
				It("Should return the same result as get by id", func() {
					visibilityJSON := ctx.SMWithOAuth.GET("/v1/visibilities/" + id).
						Expect().
						Status(http.StatusOK).JSON().Raw()

					ctx.SMWithOAuth.GET("/v1/visibilities").
						WithQuery(string(query.FieldQuery), fmt.Sprintf("id = %s", id)).
						Expect().
						Status(http.StatusOK).JSON().Object().Value("visibilities").Array().Element(0).Equal(visibilityJSON)
				})
			})

			Context("With field query for which no entries exist", func() {
				It("Should return 200 with empty array", func() {
					ctx.SMWithOAuth.GET("/v1/visibilities").
						WithQuery(string(query.FieldQuery), "platform_id = non-existing-platform-id").
						Expect().
						Status(http.StatusOK).JSON().Object().Value("visibilities").Array().Empty()
				})
			})

			Context("With label query for which no entries exist", func() {
				It("Should return 200 with empty array", func() {
					ctx.SMWithOAuth.GET("/v1/visibilities").
						WithQuery(string(query.LabelQuery), "some_key = some_value").
						Expect().
						Status(http.StatusOK).JSON().Object().Value("visibilities").Array().Empty()
				})
			})

			Context("With field query for entries exist, but label query for which one does not", func() {
				It("Should return 200 with empty array", func() {
					ctx.SMWithOAuth.GET("/v1/visibilities").
						WithQuery(string(query.FieldQuery), fmt.Sprintf("platform_id = %s", platformID)).
						WithQuery(string(query.LabelQuery), "some_key = some_value").
						Expect().
						Status(http.StatusOK).JSON().Object().Value("visibilities").Array().Empty()
				})
			})

			Context("With label query for entries exist, but field query for which one does not", func() {
				It("Should return 200 with empty array", func() {
					labelKey := labels[0].(common.Object)["key"].(string)
					labelValue := labels[0].(common.Object)["value"].([]interface{})[0].(string)

					ctx.SMWithOAuth.GET("/v1/visibilities").
						WithQuery(string(query.LabelQuery), fmt.Sprintf("%s = %s", labelKey, labelValue)).
						WithQuery(string(query.FieldQuery), "platform_id = non-existing-platform-id").
						Expect().
						Status(http.StatusOK).JSON().Object().Value("visibilities").Array().Empty()
				})
			})

			Context("With only field query for which entries exists", func() {
				It("Should return 200 with all entries", func() {
					newVisibilityID := ctx.SMWithOAuth.POST("/v1/visibilities").
						WithJSON(postVisibilityRequestNoLabels).
						Expect().Status(http.StatusCreated).JSON().Object().Value("id").String().Raw()

					oldVisibilityJSON := ctx.SMWithOAuth.GET("/v1/visibilities/" + id).
						Expect().
						Status(http.StatusOK).JSON().Raw()

					newVisibilityJSON := ctx.SMWithOAuth.GET("/v1/visibilities/" + newVisibilityID).
						Expect().
						Status(http.StatusOK).JSON().Raw()

					ctx.SMWithOAuth.GET("/v1/visibilities").
						WithQuery(string(query.FieldQuery), fmt.Sprintf("platform_id in [%s,%s]", existingPlatformID, platformID)).
						Expect().
						Status(http.StatusOK).JSON().Object().Value("visibilities").Array().ContainsOnly(oldVisibilityJSON, newVisibilityJSON)
				})
			})

			Context("With only label query for which entry exists", func() {
				It("Should return 200 with this entry", func() {
					// TODO: list by label query does not return all labels for each visibility, but currently it returns only the label that matched from the query
					Skip("TODO: SQL needs rework")
					labelKey := labels[0].(common.Object)["key"].(string)
					labelValue := labels[0].(common.Object)["value"].([]interface{})[0].(string)

					visibilityJSON := ctx.SMWithOAuth.GET("/v1/visibilities/" + id).
						Expect().
						Status(http.StatusOK).JSON().Raw()

					ctx.SMWithOAuth.GET("/v1/visibilities").
						WithQuery(string(query.LabelQuery), fmt.Sprintf("%s = %s", labelKey, labelValue)).
						Expect().
						Status(http.StatusOK).JSON().Object().Value("visibilities").Array().ContainsOnly(visibilityJSON)
				})
			})

			Context("With both label and field query", func() {
				It("Should return 200", func() {
					// TODO: list by label query does not return all labels for each visibility, but currently it returns only the label that matched from the query
					Skip("TODO: SQL needs rework")

					labelKey := labels[0].(common.Object)["key"].(string)
					labelValue := labels[0].(common.Object)["value"].([]interface{})[0].(string)

					visibilityJSON := ctx.SMWithOAuth.GET("/v1/visibilities/" + id).
						Expect().
						Status(http.StatusOK).JSON().Raw()

					ctx.SMWithOAuth.GET("/v1/visibilities").
						WithQuery(string(query.LabelQuery), fmt.Sprintf("%s = %s", labelKey, labelValue)).
						WithQuery(string(query.FieldQuery), fmt.Sprintf("platform_id = %s", platformID)).
						Expect().
						Status(http.StatusOK).JSON().Object().Value("visibilities").Array().ContainsOnly(visibilityJSON)
				})
			})

			Context("With numeric operator applied to non-numeric operands", func() {
				It("Should return 400", func() {
					ctx.SMWithOAuth.GET("/v1/visibilities").
						WithQuery(string(query.FieldQuery), fmt.Sprintf("platform_id lt %s", platformID)).
						Expect().
						Status(http.StatusBadRequest)
				})
			})

			Context("With multivariate operator applied to empty right operand", func() {
				It("Should return 400", func() {
					ctx.SMWithOAuth.GET("/v1/visibilities").
						WithQuery(string(query.FieldQuery), "platform_id notin []").
						Expect().
						Status(http.StatusBadRequest)
				})
			})

			Context("With univariate operator applied to multiple right operands", func() {
				It("Should return 400", func() {
					ctx.SMWithOAuth.GET("/v1/visibilities").
						WithQuery(string(query.FieldQuery), "platform_id gt [5,6,7]").
						Expect().
						Status(http.StatusBadRequest)
				})
			})
		})

		Describe("PATCH", func() {
			var id string
			var patchLabels []query.LabelChange
			var patchLabelsBody map[string]interface{}
			newLabelKey := "label_key"
			changedLabelValues := []string{"label_value1", "label_value2"}
			operation := query.AddLabelOperation
			BeforeEach(func() {
				patchLabels = []query.LabelChange{}
			})
			JustBeforeEach(func() {
				patchLabelsBody = make(map[string]interface{})
				patchLabels = append(patchLabels, query.LabelChange{
					Operation: operation,
					Key:       newLabelKey,
					Values:    changedLabelValues,
				})
				patchLabelsBody["labels"] = patchLabels

				id = ctx.SMWithOAuth.POST("/v1/visibilities").
					WithJSON(postVisibilityRequestWithLabels).
					Expect().Status(http.StatusCreated).JSON().Object().Value("id").String().Raw()
			})

			Context("Add new label", func() {
				It("Should return 200", func() {
					label := types.Label{Key: newLabelKey, Value: changedLabelValues}
					ctx.SMWithOAuth.PATCH("/v1/visibilities/" + id).
						WithJSON(patchLabelsBody).
						Expect().
						Status(http.StatusOK).JSON().Object().Value("labels").Array().Contains(label)
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
					newLabelKey = labels[0].(common.Object)["key"].(string)
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
						Path("$.labels[*].value[*]").Array().Contains(labelValuesObj...)
				})
			})

			Context("Remove a label", func() {
				BeforeEach(func() {
					operation = query.RemoveLabelOperation
					newLabelKey = labels[0].(common.Object)["key"].(string)
				})
				It("Should return 200", func() {
					ctx.SMWithOAuth.PATCH("/v1/visibilities/" + id).
						WithJSON(patchLabelsBody).
						Expect().
						Status(http.StatusOK).JSON().
						Path("$.labels[*].key").Array().NotContains(labels[0].(common.Object)["key"].(string))
				})
			})

			Context("Remove a label and providing no key", func() {
				BeforeEach(func() {
					operation = query.RemoveLabelValuesOperation
					newLabelKey = ""
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
					newLabelKey = labels[0].(common.Object)["key"].(string)
					valueToRemove = labels[0].(common.Object)["value"].([]interface{})[0].(string)
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
					newLabelKey = labels[0].(common.Object)["key"].(string)
					val1 := labels[0].(common.Object)["value"].([]interface{})[0].(string)
					val2 := labels[0].(common.Object)["value"].([]interface{})[1].(string)
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
					newLabelKey = labels[0].(common.Object)["key"].(string)
					labelValues := labels[0].(common.Object)["value"].([]interface{})
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
						Path("$.labels[*].key[*]").Array().NotContains(newLabelKey)
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
		})
	})
})
