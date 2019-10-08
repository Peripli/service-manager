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

	"github.com/Peripli/service-manager/pkg/multitenancy"

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

	MultitenancySettings                   *MultitenancySettings
	DisableTenantResources                 bool
	ResourceBlueprint                      func(ctx *common.TestContext, smClient *common.SMExpect) common.Object
	ResourceWithoutNullableFieldsBlueprint func(ctx *common.TestContext, smClient *common.SMExpect) common.Object
	AdditionalTests                        func(ctx *common.TestContext)
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
						smb.EnableMultitenancy(t.MultitenancySettings.LabelKey, multitenancy.ExtractTenatFromTokenWrapperFunc(t.MultitenancySettings.ClientID, t.MultitenancySettings.ClientIDTokenClaim, t.MultitenancySettings.TenantTokenClaim))
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
