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

package platform_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/Peripli/service-manager/pkg/types"

	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo"
)

// TestPlatforms tests for platform API
func TestPlatforms(t *testing.T) {
	RunSpecs(t, "Platform API Tests Suite")
}

var _ = Describe("Service Manager Platform API", func() {
	var ctx *common.TestContext

	BeforeSuite(func() {
		ctx = common.NewTestContext(nil)
	})

	AfterSuite(func() {
		ctx.Cleanup()
	})

	BeforeEach(func() {
		common.RemoveAllPlatforms(ctx.SMWithOAuth)
	})

	Describe("GET", func() {
		Context("Missing platform", func() {
			It("returns 404", func() {
				ctx.SMWithOAuth.GET("/v1/platforms/999").
					Expect().
					Status(http.StatusNotFound).
					JSON().Object().Keys().Contains("error", "description")
			})
		})

		Context("Existing platform", func() {
			It("returns the platform with given id", func() {
				platform := common.MakePlatform("platform1", "cf-10", "cf", "descr")
				reply := ctx.SMWithOAuth.POST("/v1/platforms").WithJSON(platform).
					Expect().Status(http.StatusCreated).JSON().Object()
				id := reply.Value("id").String().Raw()

				reply = ctx.SMWithOAuth.GET("/v1/platforms/" + id).
					Expect().
					Status(http.StatusOK).
					JSON().Object()

				platform["id"] = id
				common.MapContains(reply.Raw(), platform)
			})
		})
	})

	Describe("List", func() {
		Context("With no existing platforms", func() {
			It("returns empty array", func() {
				ctx.SMWithOAuth.GET("/v1/platforms").
					Expect().
					Status(http.StatusOK).
					JSON().Object().Value("platforms").Array().Empty()
			})
		})

		Context("With some existing platforms", func() {
			platformObjects := []common.Object{common.MakeRandomizedPlatform(), common.MakeRandomizedPlatform(), common.MakeRandomizedPlatformWithNoDescription()}

			BeforeEach(func() {
				for _, platform := range platformObjects {
					_ = common.RegisterPlatformInSM(platform, ctx.SMWithOAuth)
				}
			})

			Context("with no field query", func() {
				It("returns all the platforms", func() {
					reply := ctx.SMWithOAuth.GET("/v1/platforms").
						Expect().
						Status(http.StatusOK).
						JSON().Object().Value("platforms").Array()

					for _, v := range reply.Iter() {
						obj := v.Object().Raw()
						delete(obj, "created_at")
						delete(obj, "updated_at")
					}
					reply.Contains(platformObjects[0], platformObjects[1], platformObjects[2])

				})
			})

			type testCase struct {
				fieldQueryTemplate        string
				expectedResourceObjects   []interface{}
				unexpectedResourceObjects []interface{}
				expectedStatusCode        int
			}

			tests := []testCase{
				{
					fieldQueryTemplate:        "%s+=+%s",
					expectedResourceObjects:   []interface{}{platformObjects[0]},
					unexpectedResourceObjects: []interface{}{platformObjects[1]},
					expectedStatusCode:        http.StatusOK,
				},
				//{
				//	fieldQueryTemplate:        "%s+!=+%s",
				//	expectedResourceObjects:   []interface{}{platformObjects[1]},
				//	unexpectedResourceObjects: []interface{}{platformObjects[0], platformObjects[2]},
				//	expectedStatusCode:        http.StatusOK,
				//},
				//{
				//	fieldQueryTemplate:        "%s+in+[%s,123,456]",
				//	expectedResourceObjects:   []interface{}{platformObjects[0]},
				//	unexpectedResourceObjects: []interface{}{platformObjects[1], platformObjects[2]},
				//	expectedStatusCode:        http.StatusOK,
				//},
				//{
				//	fieldQueryTemplate:        "%s+notin+[%s,123,456]",
				//	expectedResourceObjects:   []interface{}{platformObjects[1]},
				//	unexpectedResourceObjects: []interface{}{platformObjects[0]},
				//	expectedStatusCode:        http.StatusOK,
				//},
				//{
				//	fieldQueryTemplate: "%s+>+%s",
				//	expectedStatusCode: http.StatusBadRequest,
				//},
				//{
				//	fieldQueryTemplate: "%s+<+%s",
				//	expectedStatusCode: http.StatusBadRequest,
				//},
				//{
				//	fieldQueryTemplate:        "%s+eqornil+%s",
				//	expectedResourceObjects:   []interface{}{},
				//	unexpectedResourceObjects: []interface{}{platformObjects[0], platformObjects[1], platformObjects[2]},
				//	expectedStatusCode:        http.StatusOK,
				//},
			}

			Context("with field query", func() {
				for _, testArgs := range tests {
					t := testArgs
					Context("when multiple field queries are provided", func() {
						FIt("returns the expected platforms", func() {
							var queries []string

							for key, value := range platformObjects[0] {
								queries = append(queries, fmt.Sprintf(t.fieldQueryTemplate, key, value))
							}

							req := ctx.SMWithOAuth.GET("/v1/platforms")

							if len(queries) != 0 {
								req = req.WithQuery("fieldQuery", strings.Join(queries, ","))
							}

							reply := req.
								Expect().
								Status(t.expectedStatusCode)

							if t.expectedStatusCode == http.StatusOK {
								array := reply.JSON().Object().Value("platforms").Array()

								for _, v := range array.Iter() {
									obj := v.Object().Raw()
									delete(obj, "created_at")
									delete(obj, "updated_at")
								}

								if len(t.expectedResourceObjects) != 0 {
									array.Contains(t.expectedResourceObjects...)
								}

								if len(t.unexpectedResourceObjects) != 0 {
									array.NotContains(t.unexpectedResourceObjects...)
								}
							} else {
								reply.JSON().Object().Keys().Contains("error", "description")
							}
						})
					})

					for k, v := range platformObjects[0] {
						key, value := k, v
						t.fieldQueryTemplate = fmt.Sprintf(t.fieldQueryTemplate, key, value)

						Context(fmt.Sprintf("when field query is [%s] and has matching values", t.fieldQueryTemplate), func() {
							It("returns the correct platforms", func() {
								req := ctx.SMWithOAuth.GET("/v1/platforms")
								req = req.WithQuery("fieldQuery", t.fieldQueryTemplate)

								reply := req.
									Expect().
									Status(t.expectedStatusCode)
								if t.expectedStatusCode != http.StatusOK {

									array := reply.JSON().Object().Value("platforms").Array()

									for _, v := range array.Iter() {
										obj := v.Object().Raw()
										delete(obj, "created_at")
										delete(obj, "updated_at")
									}

									if len(t.expectedResourceObjects) != 0 {
										array.Contains(t.expectedResourceObjects...)
									}

									if len(t.unexpectedResourceObjects) != 0 {
										array.NotContains(t.unexpectedResourceObjects...)
									}
								} else {
									reply.JSON().Object().Keys().Contains("error", "description")
								}
							})
						})

						Context(fmt.Sprintf("when field query is [%s] and has a key with no matching values", t.fieldQueryTemplate), func() {
							It("returns 200 with empty platforms", func() {
								reply := ctx.SMWithOAuth.GET("/v1/platforms").WithQuery("fieldQuery", t.fieldQueryTemplate).
									Expect().
									Status(t.expectedStatusCode).JSON().Object()

								if t.expectedStatusCode != http.StatusOK {
									reply.Value("platforms").Array().Empty()
								} else {
									reply.Keys().Contains("error", "description")
								}

							})
						})
					}

					Context(fmt.Sprintf("when field query %s has a key that is non existing", fmt.Sprintf(t.fieldQueryTemplate, "non-existing-key", "non-existing-value")), func() {
						It("returns 400", func() {
							ctx.SMWithOAuth.GET("/v1/platforms").WithQuery("fieldQuery", fmt.Sprintf(t.fieldQueryTemplate, "non-existing-key", "random-value")).
								Expect().
								Status(http.StatusBadRequest).JSON().Object().Keys().Contains("error", "description")
						})
					})

				}
			})
		})
	})

	Describe("POST", func() {
		Context("With invalid content type", func() {
			It("returns 415", func() {
				ctx.SMWithOAuth.POST("/v1/platforms").
					WithText("text").
					Expect().Status(http.StatusUnsupportedMediaType)
			})
		})

		Context("With invalid content JSON", func() {
			It("returns 400 if input is not valid JSON", func() {
				ctx.SMWithOAuth.POST("/v1/platforms").
					WithText("invalid json").
					WithHeader("content-type", "application/json").
					Expect().Status(http.StatusBadRequest)
			})
		})

		Context("With missing mandatory fields", func() {
			It("returns 400", func() {
				newplatform := func() common.Object {
					return common.MakePlatform("platform1", "cf-10", "cf", "descr")
				}
				ctx.SMWithOAuth.POST("/v1/platforms").
					WithJSON(newplatform()).
					Expect().Status(http.StatusCreated)

				for _, prop := range []string{"name", "type"} {
					platform := newplatform()
					delete(platform, prop)

					ctx.SMWithOAuth.POST("/v1/platforms").
						WithJSON(platform).
						Expect().Status(http.StatusBadRequest)
				}
			})
		})

		Context("With conflicting fields", func() {
			It("returns 409", func() {
				platform := common.MakePlatform("platform1", "cf-10", "cf", "descr")
				ctx.SMWithOAuth.POST("/v1/platforms").
					WithJSON(platform).
					Expect().Status(http.StatusCreated)
				ctx.SMWithOAuth.POST("/v1/platforms").
					WithJSON(platform).
					Expect().Status(http.StatusConflict)
			})
		})

		Context("With optional fields skipped", func() {
			It("succeeds", func() {
				platform := common.MakePlatform("platform1", "cf-10", "cf", "descr")
				// delete optional fields
				delete(platform, "id")
				delete(platform, "description")

				reply := ctx.SMWithOAuth.POST("/v1/platforms").
					WithJSON(platform).
					Expect().Status(http.StatusCreated).JSON().Object()

				platform["id"] = reply.Value("id").String().Raw()
				// optional fields returned with default values
				platform["description"] = ""

				common.MapContains(reply.Raw(), platform)
			})
		})

		Context("With invalid id", func() {
			It("fails", func() {
				platform := common.MakePlatform("platform/1", "cf-10", "cf", "descr")

				reply := ctx.SMWithOAuth.POST("/v1/platforms").
					WithJSON(platform).
					Expect().Status(http.StatusBadRequest).JSON().Object()

				reply.Value("description").Equal("platform/1 contains invalid character(s)")
			})
		})

		Context("Without id", func() {
			It("returns the new platform with generated id and credentials", func() {
				platform := common.MakePlatform("", "cf-10", "cf", "descr")
				delete(platform, "id")

				By("POST returns the new platform")

				reply := ctx.SMWithOAuth.POST("/v1/platforms").
					WithJSON(platform).
					Expect().Status(http.StatusCreated).JSON().Object()

				id := reply.Value("id").String().NotEmpty().Raw()
				platform["id"] = id
				common.MapContains(reply.Raw(), platform)
				basic := reply.Value("credentials").Object().Value("basic").Object()
				basic.Value("username").String().NotEmpty()
				basic.Value("password").String().NotEmpty()

				By("GET returns the same platform")

				reply = ctx.SMWithOAuth.GET("/v1/platforms/" + id).
					Expect().Status(http.StatusOK).JSON().Object()

				common.MapContains(reply.Raw(), platform)
			})
		})
	})

	Describe("PATCH", func() {
		var platform common.Object
		const id = "p1"

		BeforeEach(func() {
			By("Create new platform")

			platform = common.MakePlatform(id, "cf-10", "cf", "descr")
			ctx.SMWithOAuth.POST("/v1/platforms").
				WithJSON(platform).
				Expect().Status(http.StatusCreated)
		})

		Context("With all properties updated", func() {
			It("returns 200", func() {
				By("Update platform")

				updatedPlatform := common.MakePlatform("", "cf-11", "cff", "descr2")
				delete(updatedPlatform, "id")

				reply := ctx.SMWithOAuth.PATCH("/v1/platforms/" + id).
					WithJSON(updatedPlatform).
					Expect().
					Status(http.StatusOK).JSON().Object()

				updatedPlatform["id"] = id
				common.MapContains(reply.Raw(), updatedPlatform)

				By("Update is persisted")

				reply = ctx.SMWithOAuth.GET("/v1/platforms/" + id).
					Expect().
					Status(http.StatusOK).JSON().Object()

				common.MapContains(reply.Raw(), updatedPlatform)
			})
		})

		Context("With created_at in body", func() {
			It("should not update created_at", func() {
				By("Update platform")

				createdAt := "2015-01-01T00:00:00Z"
				updatedPlatform := common.Object{
					"created_at": createdAt,
				}

				ctx.SMWithOAuth.PATCH("/v1/platforms/"+id).
					WithJSON(updatedPlatform).
					Expect().
					Status(http.StatusOK).JSON().Object().
					ContainsKey("created_at").
					ValueNotEqual("created_at", createdAt)

				By("Update is persisted")

				ctx.SMWithOAuth.GET("/v1/platforms/"+id).
					Expect().
					Status(http.StatusOK).JSON().Object().
					ContainsKey("created_at").
					ValueNotEqual("created_at", createdAt)
			})
		})

		Context("With properties updated separately", func() {
			It("returns 200", func() {
				updatedPlatform := common.MakePlatform("", "cf-11", "cff", "descr2")
				delete(updatedPlatform, "id")

				for prop, val := range updatedPlatform {
					update := common.Object{}
					update[prop] = val
					reply := ctx.SMWithOAuth.PATCH("/v1/platforms/" + id).
						WithJSON(update).
						Expect().
						Status(http.StatusOK).JSON().Object()

					platform[prop] = val
					common.MapContains(reply.Raw(), platform)

					reply = ctx.SMWithOAuth.GET("/v1/platforms/" + id).
						Expect().
						Status(http.StatusOK).JSON().Object()

					common.MapContains(reply.Raw(), platform)
				}
			})
		})

		Context("With provided id", func() {
			It("should not update platform id", func() {
				ctx.SMWithOAuth.PATCH("/v1/platforms/" + id).
					WithJSON(common.Object{"id": "123"}).
					Expect().
					Status(http.StatusOK)

				ctx.SMWithOAuth.GET("/v1/platforms/123").
					Expect().
					Status(http.StatusNotFound)
			})
		})

		Context("On missing platform", func() {
			It("returns 404", func() {
				ctx.SMWithOAuth.PATCH("/v1/platforms/123").
					WithJSON(common.Object{"name": "123"}).
					Expect().
					Status(http.StatusNotFound)
			})
		})

		Context("With conflicting fields", func() {
			It("should return 409", func() {
				platform2 := common.MakePlatform("p2", "cf-12", "cf2", "descr2")
				ctx.SMWithOAuth.POST("/v1/platforms").
					WithJSON(platform2).
					Expect().Status(http.StatusCreated)

				ctx.SMWithOAuth.PATCH("/v1/platforms/" + id).
					WithJSON(platform2).
					Expect().
					Status(http.StatusConflict)
			})
		})
	})

	Describe("DELETE", func() {
		Context("Non existing platform", func() {
			It("returns 404", func() {
				ctx.SMWithOAuth.DELETE("/v1/platforms/999").
					Expect().
					Status(http.StatusNotFound)
			})
		})

		Context("Existing platform", func() {
			var platform *types.Platform
			platformObject := common.MakeRandomizedPlatform()

			BeforeEach(func() {
				platform = common.RegisterPlatformInSM(platformObject, ctx.SMWithOAuth)
			})

			Context("with field query", func() {
				// loop over testcase that has platformobject, leftop, operator, rightops
				for k, v := range platformObject {
					key, value := k, v
					Context("when the field query matches some platforms", func() {
						It("returns the expected platforms", func() {
							ctx.SMWithOAuth.GET("/v1/platforms/" + platform.ID).
								Expect().
								Status(http.StatusOK)

							ctx.SMWithOAuth.DELETE("/v1/platforms").WithQuery("fieldQuery", fmt.Sprintf("%s%s%s", key, " = ", value)).
								Expect().
								Status(http.StatusOK)

							ctx.SMWithOAuth.GET("/v1/platforms/" + platform.ID).
								Expect().
								Status(http.StatusNotFound)
						})
					})

					Context("when the field query value does not match any platforms values", func() {
						It("returns 404", func() {
							ctx.SMWithOAuth.DELETE("/v1/platforms").WithQuery("fieldQuery", fmt.Sprintf("%s%s%s", key, " = ", "non-existing-val")).
								Expect().
								Status(http.StatusNotFound)
						})
					})
				}

				Context("when multiple field queries are provided", func() {
					It("returns the expected platforms", func() {
						ctx.SMWithOAuth.GET("/v1/platforms/" + platform.ID).
							Expect().
							Status(http.StatusOK)

						req := ctx.SMWithOAuth.DELETE("/v1/platforms")
						for key, value := range platformObject {
							req = req.WithQuery("fieldQuery", fmt.Sprintf("%s%s%s", key, " = ", value))
						}
						req.
							Expect().
							Status(http.StatusOK)

						ctx.SMWithOAuth.GET("/v1/platforms/" + platform.ID).
							Expect().
							Status(http.StatusNotFound)
					})
				})

				Context("when the field query key does not exist", func() {
					It("returns 400", func() {
						ctx.SMWithOAuth.GET("/v1/platforms/" + platform.ID).
							Expect().
							Status(http.StatusOK)

						ctx.SMWithOAuth.DELETE("/v1/platforms").WithQuery("fieldQuery", fmt.Sprintf("%s%s%s", "non-existing-key", " = ", "non-existing-val")).
							Expect().
							Status(http.StatusBadRequest).
							JSON().Object().Keys().Contains("error", "description")

						ctx.SMWithOAuth.GET("/v1/platforms/" + platform.ID).
							Expect().
							Status(http.StatusOK)
					})
				})
			})

			Context("with no field query", func() {
				It("succeeds", func() {
					ctx.SMWithOAuth.GET("/v1/platforms/" + platform.ID).
						Expect().
						Status(http.StatusOK)

					ctx.SMWithOAuth.DELETE("/v1/platforms/" + platform.ID).
						Expect().
						Status(http.StatusOK).JSON().Object().Empty()

					ctx.SMWithOAuth.GET("/v1/platforms/" + platform.ID).
						Expect().
						Status(http.StatusNotFound)
				})
			})
		})
	})

})
