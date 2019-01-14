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

	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo"
)

type Op string

const (
	Get        Op = "get"
	List       Op = "list"
	Delete     Op = "delete"
	DeleteList Op = "deletelist"
)

type TestCase struct {
	API            string
	SupportsLabels bool
	SupportedOps   []Op

	RndResourceBlueprint                      func(ctx *common.TestContext) common.Object
	RndResourceWithoutNullableFieldsBlueprint func(ctx *common.TestContext) common.Object
}

func DescribeTestsFor(t TestCase) bool {
	return Describe(t.API, func() {
		var ctx *common.TestContext

		func() {
			defer GinkgoRecover()
			ctx = common.NewTestContext(nil)
		}()

		AfterSuite(func() {
			ctx.Cleanup()
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
			default:
				Fail(fmt.Sprintf("Generic test cases for op %s are not implemented", op))
			}
		}
	})
}
