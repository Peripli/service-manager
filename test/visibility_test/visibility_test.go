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
	"net/http"
	"testing"

	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestVisibilities(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Platform API Tests Suite")
}

var _ = Describe("Service Manager Platform API", func() {
	var (
		ctx                *common.TestContext
		existingPlatformID string
		existingBrokerID   string
		existingPlanIDs    []interface{}

		postVisibilityRequest common.Object
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

		postVisibilityRequest = common.Object{
			"platform_id":     existingPlatformID,
			"service_plan_id": existingPlanIDs[0],
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
					WithJSON(postVisibilityRequest).
					Expect().Status(http.StatusCreated).JSON().Object().ContainsMap(postVisibilityRequest).
					Value("id").String().Raw()

			})

			It("returns the platform with given id", func() {
				ctx.SMWithOAuth.GET("/v1/visibilities/" + visibilityID).
					Expect().
					Status(http.StatusOK).
					JSON().Object().ContainsMap(postVisibilityRequest)
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
					WithJSON(postVisibilityRequest).
					Expect().Status(http.StatusCreated).JSON().Object()

				visibilityID := json.Value("id").String().Raw()
				postVisibilityRequest["id"] = visibilityID

				json.ContainsMap(postVisibilityRequest)

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
					expectedVisibilities = common.Array{postVisibilityRequest, postVisibilityRequestForAnotherPlatform, postVisibilityForAllPlatforms}
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
					expectedVisibilities = common.Array{postVisibilityRequest, postVisibilityForAllPlatforms}
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
					WithJSON(postVisibilityRequest).
					Expect().Status(http.StatusCreated)

				for _, prop := range []string{"service_plan_id"} {
					delete(postVisibilityRequest, prop)

					ctx.SMWithOAuth.POST("/v1/visibilities").
						WithJSON(postVisibilityRequest).
						Expect().Status(http.StatusBadRequest).JSON().Object().Keys().Contains("error", "description")
				}
			})
		})

		Context("with not existing related platform", func() {
			It("returns 400", func() {
				platformId := "not-existing"
				ctx.SMWithOAuth.GET("/v1/platforms/"+platformId).
					WithJSON(postVisibilityRequest).
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
					WithJSON(postVisibilityRequest).
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
						WithJSON(postVisibilityRequest).
						Expect().Status(http.StatusCreated)

					ctx.SMWithOAuth.POST("/v1/visibilities").
						WithJSON(postVisibilityRequest).
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
						WithJSON(postVisibilityRequest).
						Expect().Status(http.StatusCreated).JSON().Object().ContainsMap(postVisibilityRequest).Keys().Contains("id")
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
					WithJSON(postVisibilityRequest).
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
