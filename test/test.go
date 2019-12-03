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
	"encoding/json"
	"fmt"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/tidwall/gjson"
	"net/http"

	"github.com/Peripli/service-manager/pkg/multitenancy"

	"github.com/Peripli/service-manager/pkg/web"

	"github.com/Peripli/service-manager/pkg/env"
	"github.com/Peripli/service-manager/pkg/sm"

	. "github.com/onsi/gomega"

	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo"
)

type Op string

const (
	Get        Op = "get"
	List       Op = "list"
	Delete     Op = "delete"
	DeleteList Op = "deletelist"
	Patch      Op = "patch"
)

type MultitenancySettings struct {
	ClientID           string
	ClientIDTokenClaim string
	TenantTokenClaim   string
	LabelKey           string

	TokenClaims map[string]interface{}
}

type TestCase struct {
	API          string
	SupportedOps []Op
	ResourceType types.ObjectType

	MultitenancySettings                   *MultitenancySettings
	DisableTenantResources                 bool
	ResourceBlueprint                      func(ctx *common.TestContext, smClient *common.SMExpect) common.Object
	ResourceWithoutNullableFieldsBlueprint func(ctx *common.TestContext, smClient *common.SMExpect) common.Object
	PatchResource                          func(ctx *common.TestContext, apiPath string, objID string, resourceType types.ObjectType, patchLabels []*query.LabelChange)
	AdditionalTests                        func(ctx *common.TestContext)
}

func DefaultResourcePatch(ctx *common.TestContext, apiPath string, objID string, _ types.ObjectType, patchLabels []*query.LabelChange) {
	patchLabelsBody := make(map[string]interface{})
	patchLabelsBody["labels"] = patchLabels

	By(fmt.Sprintf("Attempting to patch resource of %s with labels as labels are declared supported", apiPath))
	ctx.SMWithOAuth.PATCH(apiPath + "/" + objID).WithJSON(patchLabelsBody).
		Expect().
		Status(http.StatusOK)
}

func DescribeTestsFor(t TestCase) bool {
	return Describe(t.API, func() {
		var ctx *common.TestContext

		AfterSuite(func() {
			ctx.Cleanup()
		})

		func() {
			By("==== Preparation for SM tests... ====")

			defer GinkgoRecover()
			ctxBuilder := common.NewTestContextBuilder()

			if t.MultitenancySettings != nil {
				ctxBuilder.
					WithTenantTokenClaims(t.MultitenancySettings.TokenClaims).
					WithSMExtensions(func(ctx context.Context, smb *sm.ServiceManagerBuilder, e env.Environment) error {
						smb.EnableMultitenancy(t.MultitenancySettings.LabelKey, func(request *web.Request) (string, error) {
							extractTenantFromToken := multitenancy.ExtractTenantFromTokenWrapperFunc(t.MultitenancySettings.TenantTokenClaim)
							user, ok := web.UserFromContext(request.Context())
							if !ok {
								return "", nil
							}
							var userData json.RawMessage
							if err := user.Data(&userData); err != nil {
								return "", fmt.Errorf("could not unmarshal claims from token: %s", err)
							}
							clientIDFromToken := gjson.GetBytes([]byte(userData), t.MultitenancySettings.ClientIDTokenClaim).String()
							if t.MultitenancySettings.ClientID != clientIDFromToken {
								return "", nil
							}
							user.AccessLevel = web.TenantAccess
							request.Request = request.WithContext(web.ContextWithUser(request.Context(), user))
							return extractTenantFromToken(request)
						})
						return nil
					})
			}
			ctx = ctxBuilder.Build()

			// A panic outside of Ginkgo's primitives (during test setup) would be recovered
			// by the deferred GinkgoRecover() and the error will be associated with the first
			// It to be ran in the suite. There, we add a dummy It to reduce confusion.
			It("sets up all test prerequisites that are ran outside of Ginkgo primitives properly", func() {
				Expect(true).To(BeTrue())
			})

			for _, op := range t.SupportedOps {
				switch op {
				case Get:
					DescribeGetTestsfor(ctx, t)
				case List:
					DescribeListTestsFor(ctx, t)
				case Delete:
					DescribeDeleteTestsfor(ctx, t)
				case DeleteList:
					DescribeDeleteListFor(ctx, t)
				case Patch:
					DescribePatchTestsFor(ctx, t)
				default:
					_, err := fmt.Fprintf(GinkgoWriter, "Generic test cases for op %s are not implemented\n", op)
					if err != nil {
						panic(err)
					}
				}
			}

			if t.AdditionalTests != nil {
				t.AdditionalTests(ctx)
			}

			By("==== Successfully finished preparation for SM tests. Running API tests suite... ====")
		}()
	})
}
