package test

import (
	"fmt"
	"net/http"

	. "github.com/onsi/gomega"

	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo"
)

func DescribePatchTestsFor(ctx *common.TestContext, t TestCase) bool {
	return Describe("Patch", func() {
		var testResource common.Object
		var testResourceID string

		createTestResourceWithAuth := func(auth *common.SMExpect) {
			testResource = t.ResourceBlueprint(ctx, auth)
			By(fmt.Sprintf("[SETUP]: Verifying that test resource %v is not empty", testResource))
			Expect(testResource).ToNot(BeEmpty())

			By(fmt.Sprintf("[SETUP]: Verifying that test resource %v has an id of type string", testResource))
			testResourceID = testResource["id"].(string)
			Expect(testResourceID).ToNot(BeEmpty())
		}

		Context(fmt.Sprintf("Existing resource of type %s", t.API), func() {
			Context("with bearer auth", func() {
				Context("when the resource is global", func() {
					BeforeEach(func() {
						createTestResourceWithAuth(ctx.SMWithOAuth)
					})

					Context("when authenticating with basic auth", func() {
						It("returns 401", func() {
							ctx.SMWithBasic.PATCH(t.API + "/" + testResourceID).WithJSON(common.Object{}).
								Expect().
								Status(http.StatusUnauthorized)
						})
					})

					Context("when authenticating with global token", func() {
						It("returns 200", func() {
							ctx.SMWithOAuth.PATCH(t.API + "/" + testResourceID).WithJSON(common.Object{}).
								Expect().
								Status(http.StatusOK)
						})
					})

					if !t.DisableTenantResources {
						Context("when authenticating with tenant scoped token", func() {
							It("returns 404", func() {
								ctx.SMWithOAuthForTenant.PATCH(t.API + "/" + testResourceID).WithJSON(common.Object{}).
									Expect().
									Status(http.StatusNotFound)
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
							It("returns 401", func() {
								ctx.SMWithBasic.PATCH(t.API + "/" + testResourceID).WithJSON(common.Object{}).
									Expect().
									Status(http.StatusUnauthorized)
							})
						})

						Context("when authenticating with global token", func() {
							It("returns 200", func() {
								ctx.SMWithOAuth.PATCH(t.API + "/" + testResourceID).WithJSON(common.Object{}).
									Expect().
									Status(http.StatusOK)
							})
						})

						Context("when authenticating with tenant scoped token", func() {
							It("returns 200", func() {
								ctx.SMWithOAuthForTenant.PATCH(t.API + "/" + testResourceID).WithJSON(common.Object{}).
									Expect().
									Status(http.StatusOK)
							})
						})
					})
				}
			})
		})

		Context(fmt.Sprintf("Not existing resource of type %s", t.API), func() {
			BeforeEach(func() {
				testResourceID = "non-existing-id"
			})

			Context("when authenticating with basic auth", func() {
				It("returns 401", func() {
					ctx.SMWithBasic.PATCH(t.API + "/" + testResourceID).WithJSON(common.Object{}).
						Expect().
						Status(http.StatusUnauthorized)
				})
			})

			Context("when authenticating with global token", func() {
				It("returns 404", func() {
					ctx.SMWithOAuth.PATCH(t.API + "/" + testResourceID).WithJSON(common.Object{}).
						Expect().
						Status(http.StatusNotFound)
				})
			})
		})
	})
}
