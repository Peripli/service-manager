/*
 * Copyright 2018 The Service Manager Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package test

import (
	"fmt"
	"net/http"

	. "github.com/onsi/gomega"

	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo"
)

func DescribeGetTestsfor(ctx *common.TestContext, t TestCase) bool {
	return Describe("GET", func() {
		var testResource common.Object
		var testResourceID string

		Context(fmt.Sprintf("Existing resource of type %s", t.API), func() {
			createTestResourceWithAuth := func(auth *common.SMExpect) {
				testResource = t.ResourceBlueprint(ctx, auth)
				By(fmt.Sprintf("[SETUP]: Verifying that test resource %v is not empty", testResource))
				Expect(testResource).ToNot(BeEmpty())

				By(fmt.Sprintf("[SETUP]: Verifying that test resource %v has an id of type string", testResource))
				testResourceID = testResource["id"].(string)
				Expect(testResourceID).ToNot(BeEmpty())
			}

			Context("when the resource is global", func() {
				BeforeEach(func() {
					createTestResourceWithAuth(ctx.SMWithOAuth)
				})

				Context("when authenticating with global token", func() {
					It("returns 200", func() {
						ctx.SMWithOAuth.GET(fmt.Sprintf("%s/%s", t.API, testResourceID)).
							Expect().
							Status(http.StatusOK).JSON().Object().ContainsMap(testResource)
					})
				})

				if !t.DisableTenantResources {
					Context("when authenticating with tenant scoped token", func() {
						It("returns 404", func() {
							ctx.SMWithOAuthForTenant.GET(fmt.Sprintf("%s/%s", t.API, testResourceID)).
								Expect().
								Status(http.StatusNotFound).JSON().Object().Keys().Contains("error", "description")
						})
					})
				}
			})

			if !t.DisableTenantResources {
				Context("when the resource is tenant scoped", func() {
					BeforeEach(func() {
						createTestResourceWithAuth(ctx.SMWithOAuthForTenant)
					})

					Context("when authenticating with basic auth", func() {
						It("returns 200", func() {
							ctx.SMWithBasic.GET(fmt.Sprintf("%s/%s", t.API, testResourceID)).
								Expect().
								Status(http.StatusOK).JSON().Object().ContainsMap(testResource)
						})
					})

					Context("when authenticating with global token", func() {
						It("returns 200", func() {
							ctx.SMWithOAuth.GET(fmt.Sprintf("%s/%s", t.API, testResourceID)).
								Expect().
								Status(http.StatusOK).JSON().Object().ContainsMap(testResource)
						})
					})

					Context("when authenticating with tenant scoped token", func() {
						It("returns 200", func() {
							ctx.SMWithOAuthForTenant.GET(fmt.Sprintf("%s/%s", t.API, testResourceID)).
								Expect().
								Status(http.StatusOK).JSON().Object().ContainsMap(testResource)
						})
					})
				})
			}
		})

		Context(fmt.Sprintf("Not existing resource of type %s", t.API), func() {
			BeforeEach(func() {
				testResourceID = "non-existing-id"
			})

			Context("when authenticating with basic auth", func() {
				It("returns 404", func() {
					ctx.SMWithBasic.GET(fmt.Sprintf("%s/%s", t.API, testResourceID)).
						Expect().
						Status(http.StatusNotFound).JSON().Object().Keys().Contains("error", "description")
				})
			})

			Context("when authenticating with global token", func() {
				It("returns 404", func() {
					ctx.SMWithOAuth.GET(fmt.Sprintf("%s/%s", t.API, testResourceID)).
						Expect().
						Status(http.StatusNotFound).JSON().Object().Keys().Contains("error", "description")
				})
			})
		})
	})
}
