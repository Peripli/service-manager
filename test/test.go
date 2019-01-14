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
	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo"
)

type Prerequisite struct {
	FieldName string
	Required  func() bool
}

type POST struct {
	Prerequisites []Prerequisite
	AcceptsID     bool

	PostRequestBlueprint func() (requestBody common.Object, expectedResponse common.Object)
}

type GET struct {
	ResourceBlueprint func(ctx *common.TestContext) common.Object
}

type PATCH struct {
}

type LIST struct {
	ResourceBlueprint                      func(ctx *common.TestContext) common.Object
	ResourceWithoutNullableFieldsBlueprint func(ctx *common.TestContext) common.Object
}

type CascadeDeletion struct {
	Child          string
	ChildReference string
}

type DELETE struct {
	ResourceCreationBlueprint func(ctx *common.TestContext) common.Object

	CascadeDeletions []CascadeDeletion
}

type DELETELIST struct {
	ResourceBlueprint                      func(ctx *common.TestContext) common.Object
	ResourceWithoutNullableFieldsBlueprint func(ctx *common.TestContext) common.Object
}

type TestCase struct {
	API            string
	SupportsLabels bool

	POST       *POST
	GET        *GET
	LIST       *LIST
	PATCH      *PATCH
	DELETE     *DELETE
	DELETELIST *DELETELIST
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

		if t.POST != nil {

		}
		if t.GET != nil {
			DescribeGetTestsfor(ctx, t)
		}
		if t.LIST != nil {
			DescribeListTestsFor(ctx, t)
		}
		if t.PATCH != nil {

		}
		if t.DELETE != nil {
			DescribeDeleteTestsfor(ctx, t)
		}
		if t.DELETELIST != nil {
			DescribeDeleteListFor(ctx, t)
		}
	})
}
