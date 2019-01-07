// api - "service_plans"
// ops - (list, initializerFunc() map[string][]common.object), get post, put, delete

// generatorFunc() common.Object
//createResourcesBeforeOp: map[string][]interface{}{
//"service_brokers": {r["service_brokers"][0], r["service_brokers"][1]},
//},
//expectedResourcesBeforeOp: map[string][]interface{}{
//"service_brokers":   {r["service_brokers"][0], r["service_brokers"][1]},
//"service_offerings": {r["service_offerings"][0], r["service_offerings"][1]},
//"service_plans":     {r["service_plans"][0], r["service_plans"][1], r["service_plans"][2], r["service_plans"][3]},
//},

// optionalFields []string
// auth contexts - ctx.withbasic, ctx.withoauth

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

//func TestAPI(t *testing.T) {
//	RunSpecs(t, "Test API Suite")
//}

//var _ = SynchronizedBeforeSuite(func() []byte {
//	ctx := common.NewTestContext(nil)
//	return []byte{}
//}, func(data []byte) {
//
//})
//
//var _ = SynchronizedAfterSuite(func() []byte {
//	return []byte{}
//}, func(data []byte) {
//
//})

//TODO reporters? pretty output? decent logs? separate sm logs from test logs?
type Prerequisite struct {
	FieldName string
	Required  func() bool
}

// obsht interface za vsi4ki operacii
type POST struct {
	Prerequisites []Prerequisite
	AcceptsID     bool

	PostRequestBlueprint func() (requestBody common.Object, expectedResponse common.Object)
}

type GET struct {
	ResourceCreationBlueprint func(ctx *common.TestContext) common.Object
}

type PATCH struct {
}

type LIST struct {
	ResourceCreationBlueprint func(ctx *common.TestContext) common.Object
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
}

// plan needs service and service needs broker
// plan does not have create api
// some rnd resources creation  depend on other rnd resources being created
// creation (api)calls for some resoruces do not use their own apis
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
		//TODO any common stuff for all operations of should we just use the separate describes and not the merged one

		//TODO can we somehow suite these and not leave them hanging around?
		var ctx *common.TestContext
		ctx = common.NewTestContext(nil)
		var r []common.Object

		for i := 0; i < 4; i++ {
			//TODO move in get and list

			gen := t.GET.ResourceCreationBlueprint(ctx)
			delete(gen, "created_at")
			delete(gen, "updated_at")
			r = append(r, gen)
		}

		AfterSuite(func() {
			ctx.Cleanup()
		})

		//TODO can we make these describes focusable and skippable and more ginkgo like
		if t.POST != nil {

		}
		if t.GET != nil {
			DescribeGetTestsfor(ctx, t, r)
		}
		if t.LIST != nil {
			DescribeListTestsFor(ctx, t, r)
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
