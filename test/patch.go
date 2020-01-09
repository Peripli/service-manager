package test

import (
	"fmt"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/gavv/httpexpect"
	"net/http"
	"strconv"

	. "github.com/onsi/gomega"

	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo"
)

func DescribePatchTestsFor(ctx *common.TestContext, t TestCase, responseMode ResponseMode) bool {
	return Describe("Patch", func() {
		var testResource common.Object
		var testResourceID string

		var asyncParam = strconv.FormatBool(bool(responseMode))

		createTestResourceWithAuth := func(auth *common.SMExpect) {
			testResource = t.ResourceBlueprint(ctx, auth, bool(responseMode))
			By(fmt.Sprintf("[SETUP]: Verifying that test resource %v is not empty", testResource))
			Expect(testResource).ToNot(BeEmpty())

			By(fmt.Sprintf("[SETUP]: Verifying that test resource %v has an id of type string", testResource))
			testResourceID = testResource["id"].(string)
			Expect(testResourceID).ToNot(BeEmpty())
		}

		verifyPatchedResource := func(resp *httpexpect.Response, httpStatus int) {
			switch responseMode {
			case Async:
				resp.Status(http.StatusAccepted)
				asyncState := types.SUCCEEDED
				if httpStatus != http.StatusOK {
					asyncState = types.FAILED
				}

				err := ExpectOperation(ctx.SMWithOAuth, resp, asyncState)
				Expect(err).To(BeNil())
			case Sync:
				resp.Status(httpStatus)
			}
		}

		Context(fmt.Sprintf("Existing resource of type %s", t.API), func() {
			Context("with bearer auth", func() {
				Context("when the resource is global", func() {
					BeforeEach(func() {
						createTestResourceWithAuth(ctx.SMWithOAuth)
					})

					Context("when authenticating with basic auth", func() {
						It("returns 401", func() {
							ctx.SMWithBasic.PATCH(t.API+"/"+testResourceID).WithQuery("async", asyncParam).WithJSON(common.Object{}).
								Expect().
								Status(http.StatusUnauthorized)
						})
					})

					Context("when authenticating with global token", func() {
						It("returns 200", func() {
							resp := ctx.SMWithOAuth.PATCH(t.API+"/"+testResourceID).WithQuery("async", asyncParam).WithJSON(common.Object{}).Expect()
							verifyPatchedResource(resp, http.StatusOK)
						})
					})

					if !t.DisableTenantResources {
						Context("when authenticating with tenant scoped token", func() {
							It("returns 404", func() {
								resp := ctx.SMWithOAuthForTenant.PATCH(t.API+"/"+testResourceID).WithQuery("async", asyncParam).WithJSON(common.Object{}).Expect()
								verifyPatchedResource(resp, http.StatusNotFound)
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
								ctx.SMWithBasic.PATCH(t.API+"/"+testResourceID).WithQuery("async", asyncParam).WithJSON(common.Object{}).
									Expect().
									Status(http.StatusUnauthorized)
							})
						})

						Context("when authenticating with global token", func() {
							It("returns 200", func() {
								resp := ctx.SMWithOAuth.PATCH(t.API+"/"+testResourceID).WithQuery("async", asyncParam).WithJSON(common.Object{}).Expect()
								verifyPatchedResource(resp, http.StatusOK)
							})
						})

						Context("when authenticating with tenant scoped token", func() {
							It("returns 200", func() {
								resp := ctx.SMWithOAuthForTenant.PATCH(t.API+"/"+testResourceID).WithQuery("async", asyncParam).WithJSON(common.Object{}).Expect()
								verifyPatchedResource(resp, http.StatusOK)
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
					ctx.SMWithBasic.PATCH(t.API+"/"+testResourceID).WithQuery("async", asyncParam).WithJSON(common.Object{}).
						Expect().
						Status(http.StatusUnauthorized)
				})
			})

			Context("when authenticating with global token", func() {
				It("returns 404", func() {
					resp := ctx.SMWithOAuth.PATCH(t.API+"/"+testResourceID).WithQuery("async", asyncParam).WithJSON(common.Object{}).Expect()
					verifyPatchedResource(resp, http.StatusNotFound)
				})
			})
		})
	})
}
