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
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/Peripli/service-manager/pkg/web"

	"github.com/Peripli/service-manager/pkg/util"

	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/gofrs/uuid"

	. "github.com/onsi/gomega"

	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo"
)

func DescribeGetTestsfor(ctx *common.TestContext, t TestCase, responseMode ResponseMode) bool {
	return Describe("GET", func() {
		Context("Resource", func() {
			var testResource common.Object
			var testResourceID string

			Context(fmt.Sprintf("Existing resource of type %s", t.API), func() {
				createTestResourceWithAuth := func(auth *common.SMExpect) (common.Object, string) {
					testResource = t.ResourceBlueprint(ctx, auth, bool(responseMode))
					stripObject(testResource)

					By(fmt.Sprintf("[SETUP]: Verifying that test resource %v is not empty", testResource))
					Expect(testResource).ToNot(BeEmpty())

					By(fmt.Sprintf("[SETUP]: Verifying that test resource %v has an id of type string", testResource))
					testResourceID := testResource["id"].(string)
					Expect(testResourceID).ToNot(BeEmpty())

					return testResource, testResourceID
				}

				if !t.StrictlyTenantScoped {
					Context("when the resource is global", func() {
						BeforeEach(func() {
							testResource, testResourceID = createTestResourceWithAuth(ctx.SMWithOAuth)
							stripObject(testResource, t.ResourcePropertiesToIgnore...)
						})

						Context("when authenticating with global token", func() {
							It("returns 200", func() {
								ctx.SMWithOAuth.GET(fmt.Sprintf("%s/%s", t.API, testResourceID)).
									Expect().
									Status(http.StatusOK).JSON().Object().ContainsMap(testResource)
							})

							if t.SupportsAsyncOperations && responseMode == Async {
								Context("when resource is created async", func() {
									It("returns last operation with the resource", func() {
										response := ctx.SMWithOAuth.GET(fmt.Sprintf("%s/%s", t.API, testResourceID)).
											Expect().
											Status(http.StatusOK).JSON().Object()
										result := response.Raw()
										if _, found := result["last_operation"]; found {
											response.Value("last_operation").Object().ValueEqual("state", "succeeded")
										}
									})
								})
							}
						})

						if !t.DisableTenantResources {
							Context("when authenticating with tenant scoped token", func() {
								It("returns 404", func() {
									ctx.SMWithOAuthForTenant.GET(fmt.Sprintf("%s/%s", t.API, testResourceID)).
										Expect().
										Status(http.StatusNotFound).JSON().Object().Keys().Contains("error", "description")
								})

								if t.SupportsAsyncOperations && responseMode == Async {
									Context("when resource is created async", func() {
										It("returns 404", func() {
											ctx.SMWithOAuthForTenant.GET(fmt.Sprintf("%s/%s", t.API, testResourceID)).
												Expect().
												Status(http.StatusNotFound)
										})
									})
								}
							})
						}
					})
				}

				if !t.DisableTenantResources {
					Context("when the resource is tenant scoped", func() {
						BeforeEach(func() {
							testResource, testResourceID = createTestResourceWithAuth(ctx.SMWithOAuthForTenant)
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

							if t.SupportsAsyncOperations && responseMode == Async {
								Context("when resource is created async", func() {
									It("returns last operation with the resource", func() {
										response := ctx.SMWithOAuthForTenant.GET(fmt.Sprintf("%s/%s", t.API, testResourceID)).
											Expect().
											Status(http.StatusOK).JSON().Object()
										result := response.Raw()
										if _, found := result["last_operation"]; found {
											response.Value("last_operation").Object().ValueEqual("state", "succeeded")
										}
									})
								})
							}
						})

						Context("when authenticating with tenant scoped token", func() {
							It("returns 200", func() {
								ctx.SMWithOAuthForTenant.GET(fmt.Sprintf("%s/%s", t.API, testResourceID)).
									Expect().
									Status(http.StatusOK).JSON().Object().ContainsMap(testResource)
							})

							if t.SupportsAsyncOperations && responseMode == Async {
								Context("when resource is created async", func() {
									It("returns last operation with the resource", func() {
										response := ctx.SMWithOAuthForTenant.GET(fmt.Sprintf("%s/%s", t.API, testResourceID)).
											Expect().
											Status(http.StatusOK).JSON().Object()
										result := response.Raw()
										if _, found := result["last_operation"]; found {
											response.Value("last_operation").Object().ValueEqual("state", "succeeded")
										}
									})
								})
							}
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

		if t.SupportsAsyncOperations {
			Context("Operation", func() {
				const testResourceID = "test-resource-id"
				var testOperation types.Object
				var testOperationID string

				createTestOperation := func(resourceID string, tenantAccess bool) (types.Object, string) {
					id, err := uuid.NewV4()
					Expect(err).ToNot(HaveOccurred())
					labels := make(map[string][]string)
					if tenantAccess {
						labels[t.MultitenancySettings.LabelKey] = []string{t.MultitenancySettings.TokenClaims[t.MultitenancySettings.TenantTokenClaim].(string)}
					}
					testResource, err := ctx.SMRepository.Create(context.TODO(), &types.Operation{
						Base: types.Base{
							ID:        id.String(),
							CreatedAt: time.Now(),
							UpdatedAt: time.Now(),
							Labels:    labels,
							Ready:     true,
						},
						Description:   "test",
						Type:          types.CREATE,
						State:         types.IN_PROGRESS,
						ResourceID:    resourceID,
						ResourceType:  types.ObjectType(t.API),
						CorrelationID: id.String(),
					})
					Expect(err).ToNot(HaveOccurred())
					testRes := testResource.(*types.Operation)
					testRes.Context = nil
					return testResource, id.String()
				}

				Context(fmt.Sprintf("Existing operation for resource of type %s", t.API), func() {
					Context("when the operation is global", func() {
						BeforeEach(func() {
							testOperation, testOperationID = createTestOperation(testResourceID, false)
						})

						AfterEach(func() {
							byID := query.ByField(query.EqualsOperator, "id", testOperationID)
							err := ctx.SMRepository.Delete(context.TODO(), types.OperationType, byID)
							Expect(err).To(SatisfyAny(Equal(util.ErrNotFoundInStorage), BeNil()))
						})

						Context("when authenticating with global token", func() {
							It("returns 200", func() {
								ctx.SMWithOAuth.GET(fmt.Sprintf("%s/%s%s/%s", t.API, testResourceID, web.ResourceOperationsURL, testOperationID)).
									Expect().
									Status(http.StatusOK).JSON().Object().ContainsMap(testOperation)
							})
						})

						Context("when authenticating with basic auth", func() {
							It("returns 401", func() {
								ctx.SMWithBasic.GET(fmt.Sprintf("%s/%s%s/%s", t.API, testResourceID, web.ResourceOperationsURL, testOperationID)).
									Expect().
									Status(http.StatusUnauthorized).JSON().Object().Keys().Contains("error", "description")
							})
						})

						if !t.DisableTenantResources {
							Context("when authenticating with tenant scoped token", func() {
								It("returns 404", func() {
									ctx.SMWithOAuthForTenant.GET(fmt.Sprintf("%s/%s%s/%s", t.API, testResourceID, web.ResourceOperationsURL, testOperationID)).
										Expect().
										Status(http.StatusNotFound).JSON().Object().Keys().Contains("error", "description")
								})
							})
						}
					})

					if !t.DisableTenantResources {
						Context("when the operation is tenant scoped", func() {
							BeforeEach(func() {
								testOperation, testOperationID = createTestOperation(testResourceID, true)
							})

							Context("when authenticating with basic auth", func() {
								It("returns 401", func() {
									ctx.SMWithBasic.GET(fmt.Sprintf("%s/%s%s/%s", t.API, testResourceID, web.ResourceOperationsURL, testOperationID)).
										Expect().
										Status(http.StatusUnauthorized).JSON().Object().Keys().Contains("error", "description")
								})
							})

							Context("when authenticating with global token", func() {
								It("returns 200", func() {
									ctx.SMWithOAuth.GET(fmt.Sprintf("%s/%s%s/%s", t.API, testResourceID, web.ResourceOperationsURL, testOperationID)).
										Expect().
										Status(http.StatusOK).JSON().Object().ContainsMap(testOperation)
								})
							})

							Context("when authenticating with tenant scoped token", func() {
								It("returns 200", func() {
									ctx.SMWithOAuthForTenant.GET(fmt.Sprintf("%s/%s%s/%s", t.API, testResourceID, web.ResourceOperationsURL, testOperationID)).
										Expect().
										Status(http.StatusOK).JSON().Object().ContainsMap(testOperation)
								})
							})
						})
					}
				})

				Context(fmt.Sprintf("Not existing operation for resource of type %s", t.API), func() {
					BeforeEach(func() {
						testOperationID = "non-existing-id"
					})

					Context("when authenticating with basic auth", func() {
						It("returns 401", func() {
							ctx.SMWithBasic.GET(fmt.Sprintf("%s/%s%s/%s", t.API, testResourceID, web.ResourceOperationsURL, testOperationID)).
								Expect().
								Status(http.StatusUnauthorized).JSON().Object().Keys().Contains("error", "description")
						})
					})

					Context("when authenticating with global token", func() {
						It("returns 404", func() {
							ctx.SMWithOAuth.GET(fmt.Sprintf("%s/%s%s/%s", t.API, testResourceID, web.ResourceOperationsURL, testOperationID)).
								Expect().
								Status(http.StatusNotFound).JSON().Object().Keys().Contains("error", "description")
						})
					})
				})

			})
		}
	})
}
