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
		platform1 := common.GenerateRandomPlatform()
		platform2 := common.GenerateRandomPlatform()

		platformWithNilDescription := common.MakeRandomizedPlatformWithNoDescription()
		platformWithUnknownKeys := common.Object{
			"unknownkey": "unknownvalue",
		}

		nonExistingPlatform := common.GenerateRandomPlatform()

		type testCase struct {
			expectedPlatformsBeforeOp []interface{}
			fieldQueryTemplate        string
			fieldQueryArgs            common.Object

			expectedPlatformsAfterOp   []interface{}
			unexpectedPlatformsAfterOp []interface{}
			expectedStatusCode         int
		}

		testCases := []testCase{
			// no field query and some created resources
			{
				expectedPlatformsBeforeOp:  []interface{}{platform1, platform2},
				fieldQueryTemplate:         "",
				fieldQueryArgs:             common.Object{},
				expectedPlatformsAfterOp:   []interface{}{platform1, platform2},
				unexpectedPlatformsAfterOp: []interface{}{},
				expectedStatusCode:         http.StatusOK,
			},

			// no field query and no created resources
			{
				expectedPlatformsBeforeOp:  []interface{}{},
				fieldQueryTemplate:         "",
				fieldQueryArgs:             common.Object{},
				expectedPlatformsAfterOp:   []interface{}{},
				unexpectedPlatformsAfterOp: []interface{}{platform1, platform2},
				expectedStatusCode:         http.StatusOK,
			},

			// invalid operator
			{
				expectedPlatformsBeforeOp: []interface{}{platform1, platform2},
				fieldQueryTemplate:        "%s+@@@+%s",
				fieldQueryArgs:            common.Object{},
				expectedStatusCode:        http.StatusBadRequest,
			},

			// missing operator
			{
				expectedPlatformsBeforeOp: []interface{}{platform1, platform2},
				fieldQueryTemplate:        "%s++%s",
				fieldQueryArgs:            common.Object{},
				expectedStatusCode:        http.StatusBadRequest,
			},

			// some created resources, valid operators and field query right operands that match some resources
			{
				expectedPlatformsBeforeOp:  []interface{}{platform1, platform2},
				fieldQueryTemplate:         "%s+=+%s",
				fieldQueryArgs:             platform1,
				expectedPlatformsAfterOp:   []interface{}{platform1},
				unexpectedPlatformsAfterOp: []interface{}{platform2},
				expectedStatusCode:         http.StatusOK,
			},
			{
				expectedPlatformsBeforeOp:  []interface{}{platform1, platform2},
				fieldQueryTemplate:         "%s+!=+%s",
				fieldQueryArgs:             platform1,
				expectedPlatformsAfterOp:   []interface{}{platform2},
				unexpectedPlatformsAfterOp: []interface{}{platform1},
				expectedStatusCode:         http.StatusOK,
			},
			{
				expectedPlatformsBeforeOp:  []interface{}{platform1, platform2},
				fieldQueryTemplate:         "%s+in+[%s,123,456]",
				fieldQueryArgs:             platform1,
				expectedPlatformsAfterOp:   []interface{}{platform1},
				unexpectedPlatformsAfterOp: []interface{}{platform2},
				expectedStatusCode:         http.StatusOK,
			},
			{
				expectedPlatformsBeforeOp:  []interface{}{platform1, platform2},
				fieldQueryTemplate:         "%s+notin+[%s,123,456]",
				fieldQueryArgs:             platform1,
				expectedPlatformsAfterOp:   []interface{}{platform2},
				unexpectedPlatformsAfterOp: []interface{}{platform1},
				expectedStatusCode:         http.StatusOK,
			},
			{
				expectedPlatformsBeforeOp: []interface{}{platform1, platform2},
				fieldQueryTemplate:        "%s+>+%s",
				fieldQueryArgs:            platform1,
				expectedStatusCode:        http.StatusBadRequest,
			},
			{
				expectedPlatformsBeforeOp: []interface{}{platform1, platform2},
				fieldQueryTemplate:        "%s+<+%s",
				fieldQueryArgs:            platform1,
				expectedStatusCode:        http.StatusBadRequest,
			},
			{
				expectedPlatformsBeforeOp:  []interface{}{platform1, platform2},
				fieldQueryTemplate:         "%s+eqornil+%s",
				fieldQueryArgs:             platform1,
				expectedPlatformsAfterOp:   []interface{}{platform1},
				unexpectedPlatformsAfterOp: []interface{}{platform2},
				expectedStatusCode:         http.StatusOK,
			},
			{
				expectedPlatformsBeforeOp:  []interface{}{platform1, platformWithNilDescription},
				fieldQueryTemplate:         "%s+eqornil+%s",
				fieldQueryArgs:             common.Object{"description": platform1["description"]},
				expectedPlatformsAfterOp:   []interface{}{platform1, platformWithNilDescription},
				unexpectedPlatformsAfterOp: []interface{}{platform2},
				expectedStatusCode:         http.StatusOK,
			},

			// some created platforms, valid operators and field query right operands that do not match any resources
			{
				expectedPlatformsBeforeOp:  []interface{}{platform1, platform2},
				fieldQueryTemplate:         "%s+=+%s",
				fieldQueryArgs:             nonExistingPlatform,
				expectedPlatformsAfterOp:   []interface{}{},
				unexpectedPlatformsAfterOp: []interface{}{platform1, platform2},
				expectedStatusCode:         http.StatusOK,
			},
			{
				expectedPlatformsBeforeOp:  []interface{}{platform1, platform2},
				fieldQueryTemplate:         "%s+!=+%s",
				fieldQueryArgs:             nonExistingPlatform,
				expectedPlatformsAfterOp:   []interface{}{platform1, platform2},
				unexpectedPlatformsAfterOp: []interface{}{},
				expectedStatusCode:         http.StatusOK,
			},
			{
				expectedPlatformsBeforeOp:  []interface{}{platform1, platform2},
				fieldQueryTemplate:         "%s+in+[%s,123,456]",
				fieldQueryArgs:             nonExistingPlatform,
				expectedPlatformsAfterOp:   []interface{}{},
				unexpectedPlatformsAfterOp: []interface{}{platform1, platform2},
				expectedStatusCode:         http.StatusOK,
			},
			{
				expectedPlatformsBeforeOp:  []interface{}{platform1, platform2},
				fieldQueryTemplate:         "%s+notin+[%s,123,456]",
				fieldQueryArgs:             nonExistingPlatform,
				expectedPlatformsAfterOp:   []interface{}{platform1, platform2},
				unexpectedPlatformsAfterOp: []interface{}{},
				expectedStatusCode:         http.StatusOK,
			},
			{
				expectedPlatformsBeforeOp: []interface{}{platform1, platform2},
				fieldQueryTemplate:        "%s+>+%s",
				fieldQueryArgs:            nonExistingPlatform,
				expectedStatusCode:        http.StatusBadRequest,
			},
			{
				expectedPlatformsBeforeOp: []interface{}{platform1, platform2},
				fieldQueryTemplate:        "%s+<+%s",
				fieldQueryArgs:            nonExistingPlatform,
				expectedStatusCode:        http.StatusBadRequest,
			},
			{
				expectedPlatformsBeforeOp:  []interface{}{platform1, platform2},
				fieldQueryTemplate:         "%s+eqornil+%s",
				fieldQueryArgs:             nonExistingPlatform,
				expectedPlatformsAfterOp:   []interface{}{},
				unexpectedPlatformsAfterOp: []interface{}{platform1, platform2},
				expectedStatusCode:         http.StatusOK,
			},

			// invalid field query left/right operand

			{
				expectedPlatformsBeforeOp: []interface{}{platform1, platform2},
				fieldQueryTemplate:        "%s+=+%s",
				fieldQueryArgs:            platformWithUnknownKeys,
				expectedStatusCode:        http.StatusBadRequest,
			},
			{
				expectedPlatformsBeforeOp: []interface{}{platform1, platform2},
				fieldQueryTemplate:        "%s+!=+%s",
				fieldQueryArgs:            platformWithUnknownKeys,
				expectedStatusCode:        http.StatusBadRequest,
			},
			{
				expectedPlatformsBeforeOp: []interface{}{platform1, platform2},
				fieldQueryTemplate:        "%s+in+[%s,123,456]",
				fieldQueryArgs:            platformWithUnknownKeys,
				expectedStatusCode:        http.StatusBadRequest,
			},
			{
				expectedPlatformsBeforeOp: []interface{}{platform1, platform2},
				fieldQueryTemplate:        "%s+notin+[%s,123,456]",
				fieldQueryArgs:            platformWithUnknownKeys,
				expectedStatusCode:        http.StatusBadRequest,
			},
			{
				expectedPlatformsBeforeOp: []interface{}{platform1, platform2},
				fieldQueryTemplate:        "%s+>+%s",
				fieldQueryArgs:            platformWithUnknownKeys,
				expectedStatusCode:        http.StatusBadRequest,
			},
			{
				expectedPlatformsBeforeOp: []interface{}{platform1, platform2},
				fieldQueryTemplate:        "%s+<+%s",
				fieldQueryArgs:            platformWithUnknownKeys,
				expectedStatusCode:        http.StatusBadRequest,
			},
			{
				expectedPlatformsBeforeOp: []interface{}{platform1, platform2},
				fieldQueryTemplate:        "%s+eqornil+%s",
				fieldQueryArgs:            platformWithUnknownKeys,
				expectedStatusCode:        http.StatusBadRequest,
			},
		}

		verifyListOp := func(t *testCase, query string) func() {
			return func() {
				beforeOpIDs := common.ExtractResourceIDs(t.expectedPlatformsBeforeOp)
				expectedAfterOpIDs := common.ExtractResourceIDs(t.expectedPlatformsAfterOp)
				unexpectedAfterOpIDs := common.ExtractResourceIDs(t.unexpectedPlatformsAfterOp)

				BeforeEach(func() {
					q := fmt.Sprintf("id+in+[%s]", strings.Join(beforeOpIDs, ","))
					ctx.SMWithOAuth.DELETE("/v1/platforms").WithQuery("fieldQuery", q).
						Expect()

					for _, p := range t.expectedPlatformsBeforeOp {
						ctx.SMWithOAuth.POST("/v1/platforms").WithJSON(p).
							Expect().Status(http.StatusCreated)
					}
				})

				It(fmt.Sprintf("before op are found: %s; after op are expected %s; after op are unexpected: %s", beforeOpIDs, expectedAfterOpIDs, unexpectedAfterOpIDs), func() {
					beforeOpArray := ctx.SMWithOAuth.GET("/v1/platforms").
						Expect().
						Status(http.StatusOK).JSON().Object().Value("platforms").Array()

					for _, v := range beforeOpArray.Iter() {
						obj := v.Object().Raw()
						delete(obj, "created_at")
						delete(obj, "updated_at")
					}

					beforeOpArray.Contains(t.expectedPlatformsBeforeOp...)

					req := ctx.SMWithOAuth.GET("/v1/platforms")
					if query != "" {
						req = req.WithQuery("fieldQuery", query)
					}
					resp := req.
						Expect().
						Status(t.expectedStatusCode)

					if t.expectedStatusCode == http.StatusOK {
						array := resp.JSON().Object().Value("platforms").Array()
						for _, v := range array.Iter() {
							obj := v.Object().Raw()
							delete(obj, "created_at")
							delete(obj, "updated_at")
						}
						array.Contains(t.expectedPlatformsAfterOp...)
						array.NotContains(t.unexpectedPlatformsAfterOp...)
					} else {
						resp.JSON().Object().Keys().Contains("error", "description")
					}

				})
			}
		}

		for _, t := range testCases {
			t := t
			if len(t.fieldQueryArgs) == 0 && t.fieldQueryTemplate != "" {
				panic("Invalid test input")
			}

			var queries []string
			for key, value := range t.fieldQueryArgs {
				queries = append(queries, fmt.Sprintf(t.fieldQueryTemplate, key, value))
			}
			query := strings.Join(queries, ",")

			Context(fmt.Sprintf("with multi field query=%s", query), verifyListOp(&t, query))

			for _, query := range queries {
				query := query
				Context(fmt.Sprintf("with field query=%s", query), verifyListOp(&t, query))
			}
		}
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

	Describe("DELETE Multiple", func() {
		platformObject := common.GenerateRandomPlatform()
		anotherPlatformObject := common.GenerateRandomPlatform()
		platformWithUnknownKeys := common.Object{
			"unknownkey": "unknownvalue",
		}

		nonExistingPlatform := common.GenerateRandomPlatform()
		platformWithNilDescription := common.MakeRandomizedPlatformWithNoDescription()

		type testCase struct {
			expectedPlatformsBeforeOp []interface{}
			fieldQueryTemplate        string
			fieldQueryArgs            common.Object

			expectedPlatformsAfterOp   []interface{}
			unexpectedPlatformsAfterOp []interface{}
			expectedStatusCode         int
		}

		testCases := []testCase{
			{
				expectedPlatformsBeforeOp:  []interface{}{platformObject, anotherPlatformObject},
				fieldQueryTemplate:         "",
				fieldQueryArgs:             common.Object{},
				expectedPlatformsAfterOp:   []interface{}{},
				unexpectedPlatformsAfterOp: []interface{}{platformObject, anotherPlatformObject},
				expectedStatusCode:         http.StatusOK,
			},
			{
				expectedPlatformsBeforeOp:  []interface{}{},
				fieldQueryTemplate:         "",
				fieldQueryArgs:             common.Object{},
				expectedPlatformsAfterOp:   []interface{}{},
				unexpectedPlatformsAfterOp: []interface{}{platformObject, anotherPlatformObject},
				expectedStatusCode:         http.StatusNotFound,
			},
			{
				expectedPlatformsBeforeOp:  []interface{}{},
				fieldQueryTemplate:         "%s+=+%s",
				fieldQueryArgs:             platformObject,
				expectedPlatformsAfterOp:   []interface{}{},
				unexpectedPlatformsAfterOp: []interface{}{platformObject, anotherPlatformObject},
				expectedStatusCode:         http.StatusNotFound,
			},
			// known existing valid platform for the field query
			{
				expectedPlatformsBeforeOp:  []interface{}{platformObject, anotherPlatformObject},
				fieldQueryTemplate:         "%s+=+%s",
				fieldQueryArgs:             platformObject,
				expectedPlatformsAfterOp:   []interface{}{anotherPlatformObject},
				unexpectedPlatformsAfterOp: []interface{}{platformObject},
				expectedStatusCode:         http.StatusOK,
			},
			{
				expectedPlatformsBeforeOp:  []interface{}{platformObject, anotherPlatformObject},
				fieldQueryTemplate:         "%s+!=+%s",
				fieldQueryArgs:             platformObject,
				expectedPlatformsAfterOp:   []interface{}{platformObject},
				unexpectedPlatformsAfterOp: []interface{}{anotherPlatformObject},
				expectedStatusCode:         http.StatusOK,
			},
			{
				expectedPlatformsBeforeOp:  []interface{}{platformObject, anotherPlatformObject},
				fieldQueryTemplate:         "%s+in+[%s,123,456]",
				fieldQueryArgs:             platformObject,
				expectedPlatformsAfterOp:   []interface{}{anotherPlatformObject},
				unexpectedPlatformsAfterOp: []interface{}{platformObject},
				expectedStatusCode:         http.StatusOK,
			},
			{
				expectedPlatformsBeforeOp:  []interface{}{platformObject, anotherPlatformObject},
				fieldQueryTemplate:         "%s+notin+[%s,123,456]",
				fieldQueryArgs:             platformObject,
				expectedPlatformsAfterOp:   []interface{}{platformObject},
				unexpectedPlatformsAfterOp: []interface{}{anotherPlatformObject},
				expectedStatusCode:         http.StatusOK,
			},
			{
				expectedPlatformsBeforeOp:  []interface{}{platformObject, anotherPlatformObject},
				fieldQueryTemplate:         "%s+>+%s",
				fieldQueryArgs:             platformObject,
				expectedPlatformsAfterOp:   []interface{}{platformObject, anotherPlatformObject},
				unexpectedPlatformsAfterOp: []interface{}{},
				expectedStatusCode:         http.StatusBadRequest,
			},
			{
				expectedPlatformsBeforeOp:  []interface{}{platformObject, anotherPlatformObject},
				fieldQueryTemplate:         "%s+<+%s",
				fieldQueryArgs:             platformObject,
				expectedPlatformsAfterOp:   []interface{}{platformObject, anotherPlatformObject},
				unexpectedPlatformsAfterOp: []interface{}{},
				expectedStatusCode:         http.StatusBadRequest,
			},
			{
				expectedPlatformsBeforeOp:  []interface{}{platformObject, anotherPlatformObject},
				fieldQueryTemplate:         "%s+eqornil+%s",
				fieldQueryArgs:             platformObject,
				expectedPlatformsAfterOp:   []interface{}{anotherPlatformObject},
				unexpectedPlatformsAfterOp: []interface{}{platformObject},
				expectedStatusCode:         http.StatusOK,
			},

			// with non existing valid platform for field query
			{
				expectedPlatformsBeforeOp:  []interface{}{platformObject, anotherPlatformObject},
				fieldQueryTemplate:         "%s+=+%s",
				fieldQueryArgs:             nonExistingPlatform,
				expectedPlatformsAfterOp:   []interface{}{platformObject, anotherPlatformObject},
				unexpectedPlatformsAfterOp: []interface{}{},
				expectedStatusCode:         http.StatusNotFound,
			},
			{
				expectedPlatformsBeforeOp:  []interface{}{platformObject, anotherPlatformObject},
				fieldQueryTemplate:         "%s+!=+%s",
				fieldQueryArgs:             nonExistingPlatform,
				expectedPlatformsAfterOp:   []interface{}{},
				unexpectedPlatformsAfterOp: []interface{}{platformObject, anotherPlatformObject},
				expectedStatusCode:         http.StatusOK,
			},
			{
				expectedPlatformsBeforeOp:  []interface{}{platformObject, anotherPlatformObject},
				fieldQueryTemplate:         "%s+in+[%s,123,456]",
				fieldQueryArgs:             nonExistingPlatform,
				expectedPlatformsAfterOp:   []interface{}{platformObject, anotherPlatformObject},
				unexpectedPlatformsAfterOp: []interface{}{},
				expectedStatusCode:         http.StatusNotFound,
			},
			{
				expectedPlatformsBeforeOp:  []interface{}{platformObject, anotherPlatformObject},
				fieldQueryTemplate:         "%s+notin+[%s,123,456]",
				fieldQueryArgs:             nonExistingPlatform,
				expectedPlatformsAfterOp:   []interface{}{},
				unexpectedPlatformsAfterOp: []interface{}{platformObject, anotherPlatformObject},
				expectedStatusCode:         http.StatusOK,
			},
			{
				expectedPlatformsBeforeOp:  []interface{}{platformObject, anotherPlatformObject},
				fieldQueryTemplate:         "%s+>+%s",
				fieldQueryArgs:             nonExistingPlatform,
				expectedPlatformsAfterOp:   []interface{}{platformObject, anotherPlatformObject},
				unexpectedPlatformsAfterOp: []interface{}{},
				expectedStatusCode:         http.StatusBadRequest,
			},
			{
				expectedPlatformsBeforeOp:  []interface{}{platformObject, anotherPlatformObject},
				fieldQueryTemplate:         "%s+<+%s",
				fieldQueryArgs:             nonExistingPlatform,
				expectedPlatformsAfterOp:   []interface{}{platformObject, anotherPlatformObject},
				unexpectedPlatformsAfterOp: []interface{}{},
				expectedStatusCode:         http.StatusBadRequest,
			},
			{
				expectedPlatformsBeforeOp:  []interface{}{platformObject, anotherPlatformObject},
				fieldQueryTemplate:         "%s+eqornil+%s",
				fieldQueryArgs:             nonExistingPlatform,
				expectedPlatformsAfterOp:   []interface{}{platformObject, anotherPlatformObject},
				unexpectedPlatformsAfterOp: []interface{}{},
				expectedStatusCode:         http.StatusNotFound,
			},

			// with invalid platform for field query

			{
				expectedPlatformsBeforeOp:  []interface{}{platformObject, anotherPlatformObject},
				fieldQueryTemplate:         "%s+=+%s",
				fieldQueryArgs:             platformWithUnknownKeys,
				expectedPlatformsAfterOp:   []interface{}{platformObject, anotherPlatformObject},
				unexpectedPlatformsAfterOp: []interface{}{},
				expectedStatusCode:         http.StatusBadRequest,
			},
			{
				expectedPlatformsBeforeOp:  []interface{}{platformObject, anotherPlatformObject},
				fieldQueryTemplate:         "%s+!=+%s",
				fieldQueryArgs:             platformWithUnknownKeys,
				expectedPlatformsAfterOp:   []interface{}{platformObject, anotherPlatformObject},
				unexpectedPlatformsAfterOp: []interface{}{},
				expectedStatusCode:         http.StatusBadRequest,
			},
			{
				expectedPlatformsBeforeOp:  []interface{}{platformObject, anotherPlatformObject},
				fieldQueryTemplate:         "%s+in+[%s,123,456]",
				fieldQueryArgs:             platformWithUnknownKeys,
				expectedPlatformsAfterOp:   []interface{}{platformObject, anotherPlatformObject},
				unexpectedPlatformsAfterOp: []interface{}{},
				expectedStatusCode:         http.StatusBadRequest,
			},
			{
				expectedPlatformsBeforeOp:  []interface{}{platformObject, anotherPlatformObject},
				fieldQueryTemplate:         "%s+notin+[%s,123,456]",
				fieldQueryArgs:             platformWithUnknownKeys,
				expectedPlatformsAfterOp:   []interface{}{platformObject, anotherPlatformObject},
				unexpectedPlatformsAfterOp: []interface{}{},
				expectedStatusCode:         http.StatusBadRequest,
			},
			{
				expectedPlatformsBeforeOp:  []interface{}{platformObject, anotherPlatformObject},
				fieldQueryTemplate:         "%s+>+%s",
				fieldQueryArgs:             platformWithUnknownKeys,
				expectedPlatformsAfterOp:   []interface{}{platformObject, anotherPlatformObject},
				unexpectedPlatformsAfterOp: []interface{}{},
				expectedStatusCode:         http.StatusBadRequest,
			},
			{
				expectedPlatformsBeforeOp:  []interface{}{platformObject, anotherPlatformObject},
				fieldQueryTemplate:         "%s+<+%s",
				fieldQueryArgs:             platformWithUnknownKeys,
				expectedPlatformsAfterOp:   []interface{}{platformObject, anotherPlatformObject},
				unexpectedPlatformsAfterOp: []interface{}{},
				expectedStatusCode:         http.StatusBadRequest,
			},
			{
				expectedPlatformsBeforeOp:  []interface{}{platformObject, anotherPlatformObject},
				fieldQueryTemplate:         "%s+eqornil+%s",
				fieldQueryArgs:             platformWithUnknownKeys,
				expectedPlatformsAfterOp:   []interface{}{platformObject, anotherPlatformObject},
				unexpectedPlatformsAfterOp: []interface{}{},
				expectedStatusCode:         http.StatusBadRequest,
			},
			{
				expectedPlatformsBeforeOp:  []interface{}{platformObject, anotherPlatformObject, platformWithNilDescription},
				fieldQueryTemplate:         "%s+eqornil+%s",
				fieldQueryArgs:             common.Object{"description": platformObject["description"]},
				expectedPlatformsAfterOp:   []interface{}{anotherPlatformObject},
				unexpectedPlatformsAfterOp: []interface{}{platformObject, platformWithNilDescription},
				expectedStatusCode:         http.StatusOK,
			},
		}

		verifyDeleteOp := func(t *testCase, query string) func() {
			return func() {
				beforeOpIDs := common.ExtractResourceIDs(t.expectedPlatformsBeforeOp)
				expectedAfterOpIDs := common.ExtractResourceIDs(t.expectedPlatformsAfterOp)
				unexpectedAfterOpIDs := common.ExtractResourceIDs(t.unexpectedPlatformsAfterOp)

				BeforeEach(func() {
					q := fmt.Sprintf("id+in+[%s]", strings.Join(beforeOpIDs, ","))
					ctx.SMWithOAuth.DELETE("/v1/platforms").WithQuery("fieldQuery", q).
						Expect()

					for _, p := range t.expectedPlatformsBeforeOp {
						ctx.SMWithOAuth.POST("/v1/platforms").WithJSON(p).
							Expect().Status(http.StatusCreated)
					}
				})

				It(fmt.Sprintf("before op are found: %s; after op are expected %s; after op are unexpected: %s", beforeOpIDs, expectedAfterOpIDs, unexpectedAfterOpIDs), func() {
					beforeOpArray := ctx.SMWithOAuth.GET("/v1/platforms/").
						Expect().
						Status(http.StatusOK).JSON().Object().Value("platforms").Array()

					for _, v := range beforeOpArray.Iter() {
						obj := v.Object().Raw()
						delete(obj, "created_at")
						delete(obj, "updated_at")
					}

					beforeOpArray.Contains(t.expectedPlatformsBeforeOp...)

					req := ctx.SMWithOAuth.DELETE("/v1/platforms")
					if query != "" {
						req.WithQuery("fieldQuery", query)
					}
					req.
						Expect().
						Status(t.expectedStatusCode)

					afterOpArray := ctx.SMWithOAuth.GET("/v1/platforms/").
						Expect().
						Status(http.StatusOK).JSON().Object().Value("platforms").Array()

					for _, v := range afterOpArray.Iter() {
						obj := v.Object().Raw()
						delete(obj, "created_at")
						delete(obj, "updated_at")
					}

					afterOpArray.Contains(t.expectedPlatformsAfterOp...)
					afterOpArray.NotContains(t.unexpectedPlatformsAfterOp...)
				})

			}

		}

		for _, t := range testCases {
			t := t
			if len(t.fieldQueryArgs) == 0 && t.fieldQueryTemplate != "" {
				panic("Invalid test input")
			}

			var queries []string
			for key, value := range t.fieldQueryArgs {
				queries = append(queries, fmt.Sprintf(t.fieldQueryTemplate, key, value))
			}
			query := strings.Join(queries, ",")

			Context(fmt.Sprintf("with multi field query=%s", query), verifyDeleteOp(&t, query))

			for _, query := range queries {
				query := query
				Context(fmt.Sprintf("with field query=%s", query), verifyDeleteOp(&t, query))
			}
		}

		Describe("Delete Single", func() {
			Context("when non existing", func() {
				It("returns 404 when non existing", func() {
					ctx.SMWithOAuth.DELETE("/v1/platforms/999").
						Expect().
						Status(http.StatusNotFound)
				})
			})

			Context("when existing", func() {
				It("succeeds when existing", func() {
					common.RegisterPlatformInSM(platformObject, ctx.SMWithOAuth)

					ctx.SMWithOAuth.GET("/v1/platforms/" + platformObject["id"].(string)).
						Expect().
						Status(http.StatusOK)

					ctx.SMWithOAuth.DELETE("/v1/platforms/" + platformObject["id"].(string)).
						Expect().
						Status(http.StatusOK)

					ctx.SMWithOAuth.GET("/v1/platforms/" + platformObject["id"].(string)).
						Expect().
						Status(http.StatusNotFound)
				})
			})
		})
	})
})
