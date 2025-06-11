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
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/gavv/httpexpect/v2"
	. "github.com/onsi/gomega"
	"net/http"
	"strconv"

	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo"
)

func DescribeDeleteTestsfor(ctx *common.TestContext, t TestCase, responseMode ResponseMode, supportedCascadeDelete bool) bool {
	return Describe(fmt.Sprintf("DELETE %s", t.API), func() {
		const notFoundMsg = "could not find"

		var (
			testResource   common.Object
			testResourceID string

			successfulDeletionRequestResponseCode        int
			successfulCascadeDeletionRequestResponseCode = http.StatusAccepted
			failedDeletionRequestResponseCode            int

			asyncParam = strconv.FormatBool(bool(responseMode))
		)

		BeforeEach(func() {
			switch responseMode {
			case Async:
				successfulDeletionRequestResponseCode = http.StatusAccepted
				failedDeletionRequestResponseCode = http.StatusAccepted
			case Sync:
				successfulDeletionRequestResponseCode = http.StatusOK
				failedDeletionRequestResponseCode = http.StatusNotFound
			}
		})

		Context("Existing resource", func() {
			createResourceFunc := func(auth *common.SMExpect) common.Object {
				By(fmt.Sprintf("[SETUP]: Creating test resource of type %s", t.API))
				testResource = t.ResourceBlueprint(ctx, auth, bool(responseMode))
				Expect(testResource).ToNot(BeEmpty())

				By(fmt.Sprintf("[SETUP]: Verifying that test resource %v has an non empty id of type string", testResource))
				testResourceID = testResource["id"].(string)
				Expect(testResourceID).ToNot(BeEmpty())
				stripObject(testResource, t.ResourcePropertiesToIgnore...)
				return testResource
			}

			createSubResourcesFunc := func(auth *common.SMExpect, testResource common.Object) {
				By(fmt.Sprintf("[SETUP]: Creating test sub resources for type %s", t.API))
				t.SubResourcesBlueprint(ctx, auth, bool(responseMode), testResource["id"].(string), types.ObjectType(t.API), testResource)
			}

			verifyResourceDeletionWithErrorMsg := func(auth *common.SMExpect, deletionRequestResponseCode, resourceCountAfterDeletion int, expectedOpState types.OperationState, expectedErrMsg string, cascade bool) {
				By("[TEST]: Verify resource of type %s exists before delete")
				ctx.SMWithOAuth.ListWithQuery(t.API, fmt.Sprintf("fieldQuery=id eq '%s'", testResourceID)).First().Object().ContainsMap(testResource)

				By("[TEST]: Verify resource of type %s is deleted successfully")
				var resp *httpexpect.Response
				var req = auth.DELETE(fmt.Sprintf("%s/%s", t.API, testResourceID))
				if cascade {
					req = req.WithQuery("cascade", cascade)
				} else {
					req = req.WithQuery("async", asyncParam)
				}

				resp = req.Expect().Status(deletionRequestResponseCode)

				common.VerifyOperationExists(ctx, resp.Header("Location").Raw(), common.OperationExpectations{
					Category:          types.DELETE,
					State:             expectedOpState,
					ResourceType:      types.ObjectType(t.API),
					Reschedulable:     false,
					DeletionScheduled: false,
					Error:             expectedErrMsg,
				})

				By("[TEST]: Verify resource of type %s does not exist after delete")
				ctx.SMWithOAuth.GET(t.API).WithQuery("fieldQuery", fmt.Sprintf("id eq '%s'", testResourceID)).
					Expect().
					Status(http.StatusOK).JSON().Path("$.items[*]").Array().Length().Equal(resourceCountAfterDeletion)
			}

			verifyResourceDeletion := func(auth *common.SMExpect, deletionRequestResponseCode, resourceCountAfterDeletion int, expectedOpState types.OperationState) {
				verifyResourceDeletionWithErrorMsg(auth, deletionRequestResponseCode, resourceCountAfterDeletion, expectedOpState, "", false)
			}

			verifyCascadeResourceDeletion := func(auth *common.SMExpect, deletionRequestResponseCode, resourceCountAfterDeletion int, expectedOpState types.OperationState) {
				verifyResourceDeletionWithErrorMsg(auth, deletionRequestResponseCode, resourceCountAfterDeletion, expectedOpState, "", true)
			}

			if !t.StrictlyTenantScoped {
				Context("when the resource is global", func() {
					BeforeEach(func() {
						createResourceFunc(ctx.SMWithOAuth)
					})

					Context("when authenticating with basic auth", func() {
						It("returns 401", func() {
							ctx.SMWithBasic.DELETE(fmt.Sprintf("%s/%s", t.API, testResourceID)).WithQuery("async", asyncParam).
								Expect().
								Status(http.StatusUnauthorized).JSON().Object().Keys().Contains("error", "description")
						})
					})

					Context("when authenticating with global token", func() {
						It("returns 200", func() {
							verifyResourceDeletion(ctx.SMWithOAuth, successfulDeletionRequestResponseCode, 0, types.SUCCEEDED)
						})
					})

					if !t.DisableTenantResources {
						Context("when authenticating with tenant scoped token", func() {
							It("returns 404", func() {
								verifyResourceDeletionWithErrorMsg(ctx.SMWithOAuthForTenant, failedDeletionRequestResponseCode, 1, types.FAILED, notFoundMsg, false)
							})
						})
					}
				})
			}

			if !t.DisableTenantResources {
				Context("when the resource is tenant scoped", func() {
					BeforeEach(func() {
						createResourceFunc(ctx.SMWithOAuthForTenant)
					})

					Context("when authenticating with basic auth", func() {
						It("returns 401", func() {
							ctx.SMWithBasic.DELETE(fmt.Sprintf("%s/%s", t.API, testResourceID)).WithQuery("async", asyncParam).
								Expect().
								Status(http.StatusUnauthorized).JSON().Object().Keys().Contains("error", "description")
						})
					})

					Context("when authenticating with global token", func() {
						if !t.StrictlyTenantScoped {
							It("returns 200", func() {
								verifyResourceDeletion(ctx.SMWithOAuth, successfulDeletionRequestResponseCode, 0, types.SUCCEEDED)
							})
						} else {
							It("returns 400", func() {
								ctx.SMWithOAuth.DELETE(fmt.Sprintf("%s/%s", t.API, testResourceID)).WithQuery("async", asyncParam).Expect().
									Status(http.StatusBadRequest)
							})
						}
					})

					Context("when authenticating with tenant scoped token", func() {
						It("returns 200", func() {
							verifyResourceDeletion(ctx.SMWithOAuthForTenant, successfulDeletionRequestResponseCode, 0, types.SUCCEEDED)
						})
					})
				})
			}

			Context("Not existing resource", func() {
				BeforeEach(func() {
					testResourceID = "non-existing-id"
				})

				Context("when authenticating with basic auth", func() {
					if t.StrictlyTenantScoped {
						It("returns 401", func() {
							resp := ctx.SMWithBasic.DELETE(fmt.Sprintf("%s/%s", t.API, testResourceID)).WithQuery("async", asyncParam).Expect()
							resp.Status(http.StatusUnauthorized)
						})
					} else {
						It("returns 401", func() {
							resp := ctx.SMWithBasic.DELETE(fmt.Sprintf("%s/%s", t.API, testResourceID)).WithQuery("async", asyncParam).Expect()
							resp.Status(http.StatusUnauthorized)
						})
					}
				})

				Context("when authenticating with global token", func() {
					It("returns error", func() {
						resp := ctx.SMWithOAuth.DELETE(fmt.Sprintf("%s/%s", t.API, testResourceID)).WithQuery("async", asyncParam).
							Expect()
						statusCode := resp.Raw().StatusCode
						if statusCode == http.StatusAccepted {
							common.VerifyOperationExists(ctx, resp.Header("Location").Raw(), common.OperationExpectations{
								Category:          types.DELETE,
								State:             types.FAILED,
								ResourceType:      types.ObjectType(t.API),
								Reschedulable:     false,
								DeletionScheduled: false,
								Error:             notFoundMsg,
							})
						} else {
							if !t.StrictlyTenantScoped {
								Expect(statusCode).To(Equal(http.StatusNotFound))
							} else {
								Expect(statusCode).To(Equal(http.StatusBadRequest))
							}
						}
					})
				})
			})

			if t.SupportsCascadeDeleteOperations {
				Context("Cascade delete supported and nested resources exists", func() {
					if !t.StrictlyTenantScoped {
						Context("when the resource is global", func() {
							BeforeEach(func() {
								resource := createResourceFunc(ctx.SMWithOAuth)
								createSubResourcesFunc(ctx.SMWithOAuth, resource)
							})

							Context("when authenticating with global token", func() {
								It("by id returns 202", func() {
									verifyCascadeResourceDeletion(ctx.SMWithOAuth, successfulCascadeDeletionRequestResponseCode, 0, types.SUCCEEDED)
								})
							})

							if !t.DisableTenantResources {
								Context("when authenticating with tenant scoped token", func() {
									It("by id returns 404", func() {
										verifyResourceDeletionWithErrorMsg(ctx.SMWithOAuthForTenant, failedDeletionRequestResponseCode, 1, types.FAILED, notFoundMsg, true)
									})
								})
							}
						})
					}

					if !t.DisableTenantResources {
						Context("when the resource is tenant scoped", func() {
							BeforeEach(func() {
								resource := createResourceFunc(ctx.SMWithOAuthForTenant)
								createSubResourcesFunc(ctx.SMWithOAuthForTenant, resource)
							})

							Context("when authenticating with global token", func() {
								if !t.StrictlyTenantScoped {
									It("by id returns 202", func() {
										verifyCascadeResourceDeletion(ctx.SMWithOAuth, successfulCascadeDeletionRequestResponseCode, 0, types.SUCCEEDED)
									})

								} else {
									It("by id returns 400", func() {
										ctx.SMWithOAuth.DELETE(fmt.Sprintf("%s/%s", t.API, testResourceID)).WithQuery("cascade", true).Expect().
											Status(http.StatusBadRequest)
									})
								}
							})

							if !t.StrictlyTenantScoped {
								Context("when doing parallel delete", func() {
									It("parallel request return pending operation", func() {

										deleteReq1 := ctx.SMWithOAuth.DELETE(fmt.Sprintf("%s/%s", t.API, testResourceID)).
											WithQuery("cascade", true)

										deleteReq2 := ctx.SMWithOAuth.DELETE(fmt.Sprintf("%s/%s", t.API, testResourceID)).
											WithQuery("cascade", true)

										deleteLocation1 := deleteReq1.Expect().Status(http.StatusAccepted).Header("Location")
										deleteLocation2 := deleteReq2.Expect().Status(http.StatusAccepted).Header("Location")
										Expect(deleteLocation1).To(Equal(deleteLocation2), "Location should be the same for delete operations")
									})
								})
							}

							Context("when authenticating with tenant scoped token", func() {
								It("by id returns 202", func() {
									verifyCascadeResourceDeletion(ctx.SMWithOAuthForTenant, successfulCascadeDeletionRequestResponseCode, 0, types.SUCCEEDED)
								})
							})
						})
					}
				})

				Context("Not existing resource", func() {
					BeforeEach(func() {
						testResourceID = "non-existing-id"
					})

					Context("when authenticating with basic auth", func() {
						if t.StrictlyTenantScoped {
							It("by id returns 401", func() {
								resp := ctx.SMWithBasic.DELETE(fmt.Sprintf("%s/%s", t.API, testResourceID)).WithQuery("cascade", true).Expect()
								resp.Status(http.StatusUnauthorized)
							})
						} else {
							It("returns 401", func() {
								resp := ctx.SMWithBasic.DELETE(fmt.Sprintf("%s/%s", t.API, testResourceID)).WithQuery("cascade", true).Expect()
								resp.Status(http.StatusUnauthorized)
							})
						}
					})

					Context("when authenticating with global token", func() {
						It("by id returns error", func() {
							resp := ctx.SMWithOAuth.DELETE(fmt.Sprintf("%s/%s", t.API, testResourceID)).
								WithQuery("cascade", true).
								Expect()
							statusCode := resp.Raw().StatusCode
							if statusCode == http.StatusAccepted {
								common.VerifyOperationExists(ctx, resp.Header("Location").Raw(), common.OperationExpectations{
									Category:          types.DELETE,
									State:             types.FAILED,
									ResourceType:      types.ObjectType(t.API),
									Reschedulable:     false,
									DeletionScheduled: false,
									Error:             notFoundMsg,
								})
							} else {
								if !t.StrictlyTenantScoped {
									Expect(statusCode).To(Equal(http.StatusNotFound))
								} else {
									Expect(statusCode).To(Equal(http.StatusBadRequest))
								}
							}
						})
					})
				})

			} else {
				Context("Cascade delete is not supported", func() {
					It("by id returns 400", func() {
						ctx.SMWithOAuth.DELETE(fmt.Sprintf("%s/%s", t.API, testResourceID)).WithQuery("cascade", true).Expect().
							Status(http.StatusBadRequest)
					})
				})
			}

		})
	})
}
