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
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/Peripli/service-manager/test/common"
	"github.com/gavv/httpexpect"
	. "github.com/onsi/ginkgo"
)

type object = common.Object
type array = common.Array

// TestPlatforms tests for platform API
func TestPlatforms(t *testing.T) {
	os.Chdir("../..")
	RunSpecs(t, "Platform API Tests Suite")
}

var _ = Describe("Service Manager Platform API", func() {
	var SM *httpexpect.Expect
	var testServer *httptest.Server

	BeforeSuite(func() {
		testServer = httptest.NewServer(common.GetServerRouter(nil))
		SM = httpexpect.New(GinkgoT(), testServer.URL)
	})

	AfterSuite(func() {
		if testServer != nil {
			testServer.Close()
		}
	})

	BeforeEach(func() {
		common.RemoveAllPlatforms(SM)
	})

	Describe("GET", func() {
		Context("Missing platform", func() {
			It("returns 404", func() {
				SM.GET("/v1/platforms/999").
					Expect().
					Status(http.StatusNotFound).
					JSON().Object().Keys().Contains("error", "description")
			})
		})

		Context("Existing platform", func() {
			It("returns the platform with given id", func() {
				platform := common.MakePlatform("platform1", "cf-10", "cf", "descr")
				reply := SM.POST("/v1/platforms").WithJSON(platform).
					Expect().Status(http.StatusCreated).JSON().Object()
				id := reply.Value("id").String().Raw()

				reply = SM.GET("/v1/platforms/" + id).
					Expect().
					Status(http.StatusOK).
					JSON().Object()

				platform["id"] = id
				common.MapContains(reply.Raw(), platform)
			})
		})
	})
	Describe("GET All", func() {
		Context("With no platforms", func() {
			It("returns empty array", func() {
				SM.GET("/v1/platforms").
					Expect().
					Status(http.StatusOK).
					JSON().Object().Value("platforms").Array().Empty()
			})
		})

		Context("With some platforms", func() {
			It("returns all the platforms", func() {
				platforms := array{}

				addPlatform := func(id string, name string, atype string, description string) {
					platform := common.MakePlatform(id, name, atype, description)
					SM.POST("/v1/platforms").WithJSON(platform).
						Expect().Status(http.StatusCreated)
					platforms = append(platforms, platform)

					replyArray := SM.GET("/v1/platforms").
						Expect().
						Status(http.StatusOK).
						JSON().Object().Value("platforms").Array()
					for _, v := range replyArray.Iter() {
						obj := v.Object().Raw()
						delete(obj, "created_at")
						delete(obj, "updated_at")
					}
					replyArray.ContainsOnly(platforms...)
				}

				addPlatform("id1", "platform1", "cf", "platform one")
				addPlatform("id2", "platform2", "k8s", "platform two")
			})
		})
	})

	Describe("POST", func() {
		Context("With invalid content type", func() {
			It("returns 415", func() {
				SM.POST("/v1/platforms").
					WithText("text").
					Expect().Status(http.StatusUnsupportedMediaType)
			})
		})

		Context("With invalid content JSON", func() {
			It("returns 400 if input is not valid JSON", func() {
				SM.POST("/v1/platforms").
					WithText("invalid json").
					WithHeader("content-type", "application/json").
					Expect().Status(http.StatusBadRequest)
			})
		})

		Context("With missing mandatory fields", func() {
			It("returns 400", func() {
				newplatform := func() object {
					return common.MakePlatform("platform1", "cf-10", "cf", "descr")
				}
				SM.POST("/v1/platforms").
					WithJSON(newplatform()).
					Expect().Status(http.StatusCreated)

				for _, prop := range []string{"name", "type"} {
					platform := newplatform()
					delete(platform, prop)

					SM.POST("/v1/platforms").
						WithJSON(platform).
						Expect().Status(http.StatusBadRequest)
				}
			})
		})

		Context("With conflicting fields", func() {
			It("returns 409", func() {
				platform := common.MakePlatform("platform1", "cf-10", "cf", "descr")
				SM.POST("/v1/platforms").
					WithJSON(platform).
					Expect().Status(http.StatusCreated)
				SM.POST("/v1/platforms").
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

				reply := SM.POST("/v1/platforms").
					WithJSON(platform).
					Expect().Status(http.StatusCreated).JSON().Object()

				platform["id"] = reply.Value("id").String().Raw()
				// optional fields returned with default values
				platform["description"] = ""

				common.MapContains(reply.Raw(), platform)
			})
		})

		Context("Without id", func() {
			It("returns the new platform with generated id and credentials", func() {
				platform := common.MakePlatform("", "cf-10", "cf", "descr")
				delete(platform, "id")

				By("POST returns the new platform")

				reply := SM.POST("/v1/platforms").
					WithJSON(platform).
					Expect().Status(http.StatusCreated).JSON().Object()

				id := reply.Value("id").String().NotEmpty().Raw()
				platform["id"] = id
				common.MapContains(reply.Raw(), platform)
				basic := reply.Value("credentials").Object().Value("basic").Object()
				basic.Value("username").String().NotEmpty()
				basic.Value("password").String().NotEmpty()

				By("GET returns the same platform")

				reply = SM.GET("/v1/platforms/" + id).
					Expect().Status(http.StatusOK).JSON().Object()

				common.MapContains(reply.Raw(), platform)
			})
		})
	})

	Describe("PATCH", func() {
		var platform object
		const id = "p1"

		BeforeEach(func() {
			By("Create new platform")

			platform = common.MakePlatform(id, "cf-10", "cf", "descr")
			SM.POST("/v1/platforms").
				WithJSON(platform).
				Expect().Status(http.StatusCreated)
		})

		Context("With all properties updated", func() {
			It("returns 200", func() {
				By("Update platform")

				updatedPlatform := common.MakePlatform("", "cf-11", "cff", "descr2")
				delete(updatedPlatform, "id")

				reply := SM.PATCH("/v1/platforms/" + id).
					WithJSON(updatedPlatform).
					Expect().
					Status(http.StatusOK).JSON().Object()

				updatedPlatform["id"] = id
				common.MapContains(reply.Raw(), updatedPlatform)

				By("Update is persisted")

				reply = SM.GET("/v1/platforms/" + id).
					Expect().
					Status(http.StatusOK).JSON().Object()

				common.MapContains(reply.Raw(), updatedPlatform)
			})
		})

		Context("With properties updated separately", func() {
			It("returns 200", func() {
				updatedPlatform := common.MakePlatform("", "cf-11", "cff", "descr2")
				delete(updatedPlatform, "id")

				for prop, val := range updatedPlatform {
					update := object{}
					update[prop] = val
					reply := SM.PATCH("/v1/platforms/" + id).
						WithJSON(update).
						Expect().
						Status(http.StatusOK).JSON().Object()

					platform[prop] = val
					common.MapContains(reply.Raw(), platform)

					reply = SM.GET("/v1/platforms/" + id).
						Expect().
						Status(http.StatusOK).JSON().Object()

					common.MapContains(reply.Raw(), platform)
				}
			})
		})

		Context("With provided id", func() {
			It("should not update platform id", func() {
				SM.PATCH("/v1/platforms/" + id).
					WithJSON(object{"id": "123"}).
					Expect().
					Status(http.StatusOK)

				SM.GET("/v1/platforms/123").
					Expect().
					Status(http.StatusNotFound)
			})
		})

		Context("On missing platform", func() {
			It("returns 404", func() {
				SM.PATCH("/v1/platforms/123").
					WithJSON(object{"name": "123"}).
					Expect().
					Status(http.StatusNotFound)
			})
		})

		Context("With conflicting fields", func() {
			It("should return 409", func() {
				platform2 := common.MakePlatform("p2", "cf-12", "cf2", "descr2")
				SM.POST("/v1/platforms").
					WithJSON(platform2).
					Expect().Status(http.StatusCreated)

				SM.PATCH("/v1/platforms/" + id).
					WithJSON(platform2).
					Expect().
					Status(http.StatusConflict)
			})
		})
	})

	Describe("DELETE", func() {
		Context("Non existing platform", func() {
			It("returns 404", func() {
				SM.DELETE("/v1/platforms/999").
					Expect().
					Status(http.StatusNotFound)
			})
		})

		Context("Existing platform", func() {
			It("succeeds", func() {
				platform := common.MakePlatform("p1", "cf-10", "cf", "descr")
				SM.POST("/v1/platforms").
					WithJSON(platform).
					Expect().Status(http.StatusCreated)

				SM.GET("/v1/platforms/p1").
					Expect().
					Status(http.StatusOK)

				SM.DELETE("/v1/platforms/p1").
					Expect().
					Status(http.StatusOK).JSON().Object().Empty()

				SM.GET("/v1/platforms/p1").
					Expect().
					Status(http.StatusNotFound)
			})
		})
	})

})
