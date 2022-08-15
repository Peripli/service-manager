package test

import (
	"fmt"
	"net/http"
	"strconv"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types"

	"github.com/gavv/httpexpect"

	. "github.com/onsi/gomega"

	. "github.com/onsi/ginkgo"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/test/common"
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

		verifyPatchedResource := func(resp *httpexpect.Response) {
			switch responseMode {
			case Async:
				resp.Status(http.StatusAccepted)
			case Sync:
				resp.Status(http.StatusOK)
			}

			common.VerifyOperationExists(ctx, resp.Header("Location").Raw(), common.OperationExpectations{
				Category:          types.UPDATE,
				State:             types.SUCCEEDED,
				ResourceType:      types.ObjectType(t.API),
				Reschedulable:     false,
				DeletionScheduled: false,
			})
		}

		Context(fmt.Sprintf("Existing resource of type %s", t.API), func() {
			Context("with bearer auth", func() {
				if !t.StrictlyTenantScoped {
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
								verifyPatchedResource(resp)
							})
						})

						if !t.DisableTenantResources {
							Context("when authenticating with tenant scoped token", func() {
								It("returns 404", func() {
									ctx.SMWithOAuthForTenant.PATCH(t.API+"/"+testResourceID).WithQuery("async", asyncParam).WithJSON(common.Object{}).
										Expect().Status(http.StatusNotFound)
								})
							})
						}
					})
				}

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

						if !t.StrictlyTenantScoped {
							Context("when authenticating with global token", func() {
								It("returns 200", func() {
									resp := ctx.SMWithOAuth.PATCH(t.API+"/"+testResourceID).WithQuery("async", asyncParam).WithJSON(common.Object{}).Expect()
									verifyPatchedResource(resp)
								})
							})
						}

						Context("when authenticating with tenant scoped token", func() {
							It("returns 200", func() {
								resp := ctx.SMWithOAuthForTenant.PATCH(t.API+"/"+testResourceID).WithQuery("async", asyncParam).WithJSON(common.Object{}).Expect()
								verifyPatchedResource(resp)
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
				if t.StrictlyTenantScoped {
					It("returns 400", func() {
						ctx.SMWithOAuth.PATCH(t.API+"/"+testResourceID).WithQuery("async", asyncParam).WithJSON(common.Object{}).
							Expect().Status(http.StatusBadRequest)
					})
				} else {
					It("returns 404", func() {
						ctx.SMWithOAuth.PATCH(t.API+"/"+testResourceID).WithQuery("async", asyncParam).WithJSON(common.Object{}).
							Expect().Status(http.StatusNotFound)
					})
				}
			})
		})
	})
}
