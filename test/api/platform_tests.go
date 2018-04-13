package api

import (
	"net/http"

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
		resp := sm.GET("/v1/platforms").
			Expect().Status(http.StatusOK).JSON().Object()
		for _, val := range resp.Value("platforms").Array().Iter() {
			id := val.Object().Value("id").String().Raw()
			sm.DELETE("/v1/platforms/" + id).
				Expect().Status(http.StatusOK)
		}
	})

	Describe("GET", func() {
		It("returns 404 if platform does not exist", func() {
			sm.GET("/v1/platforms/999").
				Expect().
				Status(http.StatusNotFound).
				JSON().Object().Keys().Contains("error", "description")
		})

		It("returns the platform with given id", func() {
			platform := makePlatform("platform1", "cf-10", "cf", "descr")
			reply := sm.POST("/v1/platforms").WithJSON(platform).
				Expect().Status(http.StatusCreated).JSON().Object()
			id := reply.Value("id").String().Raw()

			reply = sm.GET("/v1/platforms/" + id).
				Expect().
				Status(http.StatusOK).
				JSON().Object()

			platform["id"] = id
			delete(platform, "credentials")
			mapContains(reply.Raw(), platform)
		})
	})
	Describe("GET All", func() {
		It("returns empty array if no platforms exist", func() {
			sm.GET("/v1/platforms").
				Expect().
				Status(http.StatusOK).
				JSON().Object().Value("platforms").Array().Empty()
		})

		It("returns all the platforms", func() {
			platforms := Array{}

			addPlatform := func(id string, name string, atype string, description string) {
				platform := makePlatform(id, name, atype, description)
				sm.POST("/v1/platforms").WithJSON(platform).
					Expect().Status(http.StatusCreated)
				platforms = append(platforms, platform)

				replyArray := sm.GET("/v1/platforms").
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
			sm.POST("/v1/platforms").
				WithText("text").
				Expect().Status(http.StatusUnsupportedMediaType)
		})

		It("returns 400 if input is not valid JSON", func() {
			sm.POST("/v1/platforms").
				WithText("invalid json").
				WithHeader("content-type", "application/json").
				Expect().Status(http.StatusBadRequest)
		})

		It("returns 400 if mandatory field is missing", func() {
			newplatform := func() Object {
				return makePlatform("platform1", "cf-10", "cf", "descr")
			}
			sm.POST("/v1/platforms").
				WithJSON(newplatform()).
				Expect().Status(http.StatusCreated)

			for _, prop := range []string{"name", "type"} {
				platform := newplatform()
				delete(platform, prop)

				sm.POST("/v1/platforms").
					WithJSON(platform).
					Expect().Status(http.StatusBadRequest)
			}
		})

		It("succeeds if optional fields are skipped", func() {
			platform := makePlatform("platform1", "cf-10", "cf", "descr")
			// delete optional fields
			delete(platform, "id")
			delete(platform, "description")

			reply := sm.POST("/v1/platforms").
				WithJSON(platform).
				Expect().Status(http.StatusCreated).JSON().Object()

			delete(platform, "credentials")
			platform["id"] = reply.Value("id").String().Raw()
			// optional fields returned with default values
			platform["description"] = ""

			mapContains(reply.Raw(), platform)
		})
		/*
			It("returns the new platform", func() {
				platform := makePlatform("platform1", "http://domain.com/platform", "platform one")

				By("POST returns the new platform")

				reply := sm.POST("/v1/platforms").
					WithJSON(platform).
					Expect().Status(http.StatusCreated).JSON().Object()
				delete(platform, "credentials")

				id := reply.Value("id").String().NotEmpty().Raw()
				platform["id"] = id
				mapContains(reply.Raw(), platform)

				By("GET returns the same platform")

				reply = sm.GET("/v1/platforms/" + id).
					Expect().Status(http.StatusOK).JSON().Object()

				mapContains(reply.Raw(), platform)
			})

			It("returns 409 if duplicate name is provided", func() {
				platform := makePlatform("platform1", "http://domain.com/platform", "")

				sm.POST("/v1/platforms").
					WithJSON(platform).
					Expect().Status(http.StatusCreated)
				sm.POST("/v1/platforms").
					WithJSON(platform).
					Expect().Status(http.StatusConflict)
			})
		*/
	})
	/*

		Describe("PATCH", func() {
			var platform Object
			var id string

			BeforeEach(func() {
				By("Create new platform")
				platform = makePlatform("platform1", "http://domain.com/platform", "desc1")
				reply := sm.POST("/v1/platforms").
					WithJSON(platform).
					Expect().Status(http.StatusCreated).JSON().Object()
				delete(platform, "credentials")
				id = reply.Value("id").String().Raw()
				platform["id"] = id
			})

			It("returns 200 if all properties are updated", func() {
				By("Update platform")

				updatedplatform := makePlatform("platform2", "http://domain.com/platform2", "desc2")
				updatedplatform["credentials"] = Object{
					"basic": Object{
						"username": "auser",
						"password": "apass",
					},
				}

				reply := sm.PATCH("/v1/platforms/" + id).
					WithJSON(updatedplatform).
					Expect().
					Status(http.StatusOK).JSON().Object()
				delete(updatedplatform, "credentials")

				mapContains(reply.Raw(), updatedplatform)

				By("Update is persisted")

				reply = sm.GET("/v1/platforms/" + id).
					Expect().
					Status(http.StatusOK).JSON().Object()

				mapContains(reply.Raw(), updatedplatform)
			})

			It("can update each property separately", func() {
				newplatform := makePlatform("bb8", "http://lucas.arts", "a robot")
				newplatform["credentials"] = Object{
					"basic": Object{
						"username": "auser",
						"password": "apass",
					},
				}

				for prop, val := range newplatform {
					update := Object{}
					update[prop] = val

					reply := sm.PATCH("/v1/platforms/" + id).
						WithJSON(update).
						Expect().
						Status(http.StatusOK).JSON().Object()

					if prop != "credentials" { // credentials are not returned
						platform[prop] = val
					}
					mapContains(reply.Raw(), platform)

					reply = sm.GET("/v1/platforms/" + id).
						Expect().
						Status(http.StatusOK).JSON().Object()

					mapContains(reply.Raw(), platform)
				}
			})

			It("should not update platform id if provided", func() {
				sm.PATCH("/v1/platforms/" + id).
					WithJSON(Object{"id": "123"}).
					Expect().
					Status(http.StatusOK)

				sm.GET("/v1/platforms/123").
					Expect().
					Status(http.StatusNotFound)
			})
		})

		Describe("DELETE", func() {
			It("returns 404 when trying to delete non-existing platform", func() {
				sm.DELETE("/v1/platforms/999").
					Expect().
					Status(http.StatusNotFound)
			})

			It("deletes platform", func() {
				platform := makePlatform("platform1", "http://domain.com/platform", "desc1")
				reply := sm.POST("/v1/platforms").
					WithJSON(platform).
					Expect().Status(http.StatusCreated).JSON().Object()
				id := reply.Value("id").String().Raw()

				sm.GET("/v1/platforms/" + id).
					Expect().
					Status(http.StatusOK)

				sm.DELETE("/v1/platforms/" + id).
					Expect().
					Status(http.StatusOK).JSON().Object().Empty()

				sm.GET("/v1/platforms/" + id).
					Expect().
					Status(http.StatusNotFound)
			})
		})
	*/
}
