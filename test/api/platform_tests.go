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

package itest

import (
	"net/http"
	. "github.com/Peripli/service-manager/test/api/util"
	. "github.com/onsi/ginkgo"
)

func makePlatform(id string, name string, atype string, description string) Object {
	return Object{
		"id":          id,
		"name":        name,
		"type":        atype,
		"description": description,
	}
}

func testPlatforms() {
	BeforeEach(func() {
		By("remove all platforms")
		resp := SM.GET("/v1/platforms").
			Expect().Status(http.StatusOK).JSON().Object()
		for _, val := range resp.Value("platforms").Array().Iter() {
			id := val.Object().Value("id").String().Raw()
			SM.DELETE("/v1/platforms/" + id).
				Expect().Status(http.StatusOK)
		}
	})

	Describe("GET", func() {
		It("returns 404 if platform does not exist", func() {
			SM.GET("/v1/platforms/999").
				Expect().
				Status(http.StatusNotFound).
				JSON().Object().Keys().Contains("error", "description")
		})

		It("returns the platform with given id", func() {
			platform := makePlatform("platform1", "cf-10", "cf", "descr")
			reply := SM.POST("/v1/platforms").WithJSON(platform).
				Expect().Status(http.StatusCreated).JSON().Object()
			id := reply.Value("id").String().Raw()

			reply = SM.GET("/v1/platforms/" + id).
				Expect().
				Status(http.StatusOK).
				JSON().Object()

			platform["id"] = id
			MapContains(reply.Raw(), platform)
		})
	})
	Describe("GET All", func() {
		It("returns empty array if no platforms exist", func() {
			SM.GET("/v1/platforms").
				Expect().
				Status(http.StatusOK).
				JSON().Object().Value("platforms").Array().Empty()
		})

		It("returns all the platforms", func() {
			platforms := Array{}

			addPlatform := func(id string, name string, atype string, description string) {
				platform := makePlatform(id, name, atype, description)
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

	Describe("POST", func() {
		It("returns 415 if input is not valid JSON", func() {
			SM.POST("/v1/platforms").
				WithText("text").
				Expect().Status(http.StatusUnsupportedMediaType)
		})

		It("returns 400 if input is not valid JSON", func() {
			SM.POST("/v1/platforms").
				WithText("invalid json").
				WithHeader("content-type", "application/json").
				Expect().Status(http.StatusBadRequest)
		})

		It("returns 400 if mandatory field is missing", func() {
			newplatform := func() Object {
				return makePlatform("platform1", "cf-10", "cf", "descr")
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

		It("returns 409 if duplicate platform already exists", func() {
			platform := makePlatform("platform1", "cf-10", "cf", "descr")
			SM.POST("/v1/platforms").
				WithJSON(platform).
				Expect().Status(http.StatusCreated)
			SM.POST("/v1/platforms").
				WithJSON(platform).
				Expect().Status(http.StatusConflict)
		})

		It("succeeds if optional fields are skipped", func() {
			platform := makePlatform("platform1", "cf-10", "cf", "descr")
			// delete optional fields
			delete(platform, "id")
			delete(platform, "description")

			reply := SM.POST("/v1/platforms").
				WithJSON(platform).
				Expect().Status(http.StatusCreated).JSON().Object()

			platform["id"] = reply.Value("id").String().Raw()
			// optional fields returned with default values
			platform["description"] = ""

			MapContains(reply.Raw(), platform)
		})

		It("returns the new platform with generated id and credentials", func() {
			platform := makePlatform("", "cf-10", "cf", "descr")
			delete(platform, "id")

			By("POST returns the new platform")

			reply := SM.POST("/v1/platforms").
				WithJSON(platform).
				Expect().Status(http.StatusCreated).JSON().Object()

			id := reply.Value("id").String().NotEmpty().Raw()
			platform["id"] = id
			MapContains(reply.Raw(), platform)
			basic := reply.Value("credentials").Object().Value("basic").Object()
			basic.Value("username").String().NotEmpty()
			basic.Value("password").String().NotEmpty()

			By("GET returns the same platform")

			reply = SM.GET("/v1/platforms/" + id).
				Expect().Status(http.StatusOK).JSON().Object()

			MapContains(reply.Raw(), platform)
		})
	})

	Describe("PATCH", func() {
		var platform Object
		const id = "p1"

		BeforeEach(func() {
			By("Create new platform")

			platform = makePlatform(id, "cf-10", "cf", "descr")
			SM.POST("/v1/platforms").
				WithJSON(platform).
				Expect().Status(http.StatusCreated)
		})

		It("returns 200 if all properties are updated", func() {
			By("Update platform")

			updatedPlatform := makePlatform("", "cf-11", "cff", "descr2")
			delete(updatedPlatform, "id")

			reply := SM.PATCH("/v1/platforms/" + id).
				WithJSON(updatedPlatform).
				Expect().
				Status(http.StatusOK).JSON().Object()

			updatedPlatform["id"] = id
			MapContains(reply.Raw(), updatedPlatform)

			By("Update is persisted")

			reply = SM.GET("/v1/platforms/" + id).
				Expect().
				Status(http.StatusOK).JSON().Object()

			MapContains(reply.Raw(), updatedPlatform)
		})

		It("can update each property separately", func() {
			updatedPlatform := makePlatform("", "cf-11", "cff", "descr2")
			delete(updatedPlatform, "id")

			for prop, val := range updatedPlatform {
				update := Object{}
				update[prop] = val
				reply := SM.PATCH("/v1/platforms/" + id).
					WithJSON(update).
					Expect().
					Status(http.StatusOK).JSON().Object()

				platform[prop] = val
				MapContains(reply.Raw(), platform)

				reply = SM.GET("/v1/platforms/" + id).
					Expect().
					Status(http.StatusOK).JSON().Object()

				MapContains(reply.Raw(), platform)
			}
		})

		It("should not update platform id if provided", func() {
			SM.PATCH("/v1/platforms/" + id).
				WithJSON(Object{"id": "123"}).
				Expect().
				Status(http.StatusOK)

			SM.GET("/v1/platforms/123").
				Expect().
				Status(http.StatusNotFound)
		})

		It("should return 404 on missing platform", func() {
			SM.PATCH("/v1/platforms/123").
				WithJSON(Object{"name": "123"}).
				Expect().
				Status(http.StatusNotFound)
		})

		It("should return 409 on missing platform", func() {
			platform2 := makePlatform("p2", "cf-12", "cf2", "descr2")
			SM.POST("/v1/platforms").
				WithJSON(platform2).
				Expect().Status(http.StatusCreated)

			SM.PATCH("/v1/platforms/" + id).
				WithJSON(platform2).
				Expect().
				Status(http.StatusConflict)
		})
	})

	Describe("DELETE", func() {
		It("returns 404 when trying to delete non-existing platform", func() {
			SM.DELETE("/v1/platforms/999").
				Expect().
				Status(http.StatusNotFound)
		})

		It("deletes platform", func() {
			platform := makePlatform("p1", "cf-10", "cf", "descr")
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

}
