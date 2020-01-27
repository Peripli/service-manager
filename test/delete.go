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
	"strconv"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/gavv/httpexpect"

	. "github.com/onsi/gomega"

	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo"
)

func DescribeDeleteTestsfor(ctx *common.TestContext, t TestCase, responseMode ResponseMode) bool {
	return Describe(fmt.Sprintf("DELETE %s", t.API), func() {
		const notFoundMsg = "could not find"

		var (
			testResource   common.Object
			testResourceID string

			successfulDeletionRequestResponseCode int
			failedDeletionRequestResponseCode     int

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
			createResourceFunc := func(auth *common.SMExpect) {
				By(fmt.Sprintf("[SETUP]: Creating test resource of type %s", t.API))
				testResource = t.ResourceBlueprint(ctx, auth, bool(responseMode))
				Expect(testResource).ToNot(BeEmpty())

				By(fmt.Sprintf("[SETUP]: Verifying that test resource %v has an non empty id of type string", testResource))
				testResourceID = testResource["id"].(string)
				Expect(testResourceID).ToNot(BeEmpty())
				stripObject(testResource, t.ResourcePropertiesToIgnore...)
			}

			verifyResourceDeletionWithErrorMsg := func(auth *common.SMExpect, deletionRequestResponseCode, getAfterDeletionRequestCode int, expectedOpState types.OperationState, expectedErrMsg string) {
				By("[TEST]: Verify resource of type %s exists before delete")
				ctx.SMWithOAuth.GET(fmt.Sprintf("%s/%s", t.API, testResourceID)).
					Expect().
					Status(http.StatusOK).JSON().Object().ContainsMap(testResource)

				By("[TEST]: Verify resource of type %s is deleted successfully")
				resp := auth.DELETE(fmt.Sprintf("%s/%s", t.API, testResourceID)).WithQuery("async", asyncParam).
					Expect().
					Status(deletionRequestResponseCode)

				if responseMode == Async {
					_, err := ExpectOperationWithError(auth, resp, expectedOpState, expectedErrMsg)
					Expect(err).To(BeNil())
				}

				By("[TEST]: Verify resource of type %s does not exist after delete")
				ctx.SMWithOAuth.GET(fmt.Sprintf("%s/%s", t.API, testResourceID)).
					Expect().
					Status(getAfterDeletionRequestCode)
			}

			verifyResourceDeletion := func(auth *common.SMExpect, deletionRequestResponseCode, getAfterDeletionRequestCode int, expectedOpState types.OperationState) {
				verifyResourceDeletionWithErrorMsg(auth, deletionRequestResponseCode, getAfterDeletionRequestCode, expectedOpState, "")
			}

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
						verifyResourceDeletion(ctx.SMWithOAuth, successfulDeletionRequestResponseCode, http.StatusNotFound, types.SUCCEEDED)
					})
				})

				if !t.DisableTenantResources {
					Context("when authenticating with tenant scoped token", func() {
						It("returns 404", func() {
							verifyResourceDeletionWithErrorMsg(ctx.SMWithOAuthForTenant, failedDeletionRequestResponseCode, http.StatusOK, types.FAILED, notFoundMsg)
						})
					})
				}
			})

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
						It("returns 200", func() {
							verifyResourceDeletion(ctx.SMWithOAuth, successfulDeletionRequestResponseCode, http.StatusNotFound, types.SUCCEEDED)
						})
					})

					Context("when authenticating with tenant scoped token", func() {
						It("returns 200", func() {
							verifyResourceDeletion(ctx.SMWithOAuthForTenant, successfulDeletionRequestResponseCode, http.StatusNotFound, types.SUCCEEDED)
						})
					})
				})
			}
		})

		Context("Not existing resource", func() {
			BeforeEach(func() {
				testResourceID = "non-existing-id"
			})

			verifyMissingResourceFailedDeletion := func(resp *httpexpect.Response, expectedErrMsg string) {
				switch responseMode {
				case Async:
					_, err := ExpectOperationWithError(ctx.SMWithOAuth, resp, types.FAILED, expectedErrMsg)
					Expect(err).To(BeNil())
				case Sync:
					resp.Status(http.StatusNotFound).JSON().Object().Keys().Contains("error", "description")
				}
			}

			Context("when authenticating with basic auth", func() {
				It("returns 404", func() {
					resp := ctx.SMWithOAuth.DELETE(fmt.Sprintf("%s/%s", t.API, testResourceID)).WithQuery("async", asyncParam).Expect()
					verifyMissingResourceFailedDeletion(resp, notFoundMsg)
				})
			})

			Context("when authenticating with global token", func() {
				It("returns 404", func() {
					resp := ctx.SMWithOAuth.DELETE(fmt.Sprintf("%s/%s", t.API, testResourceID)).WithQuery("async", asyncParam).Expect()
					verifyMissingResourceFailedDeletion(resp, notFoundMsg)
				})
			})
		})
	})
}
