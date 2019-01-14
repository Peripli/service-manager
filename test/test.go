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
	"net/http"

	"github.com/Peripli/service-manager/pkg/query"
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

type Query struct {
	Type     string
	Template string
	Args     interface{}
}

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
		var ctx *common.TestContext
		var r []common.Object
		var rWithMandatoryFields common.Object

		AfterSuite(func() {
			ctx.Cleanup()
		})

		func() {
			By("==== Preparation for SM component tests... ====")
			defer GinkgoRecover()
			attachLabel := func(obj common.Object) common.Object {
				patchLabelsBody := make(map[string]interface{})
				patchLabels := []query.LabelChange{
					{
						Operation: query.AddLabelOperation,
						Key:       "labelKey1",
						Values:    []string{"1"},
					},
					{
						Operation: query.AddLabelOperation,
						Key:       "labelKey2",
						Values:    []string{"str"},
					},
					{
						Operation: query.AddLabelOperation,
						Key:       "labelKey3",
						Values:    []string{`{"key1": "val1", "key2": "val2"}`},
					},
				}
				patchLabelsBody["labels"] = patchLabels

				ctx.SMWithOAuth.PATCH("/v1/" + t.API + "/" + obj["id"].(string)).WithJSON(patchLabelsBody).
					Expect().
					Status(http.StatusOK)

				result := ctx.SMWithOAuth.GET("/v1/" + t.API + "/" + obj["id"].(string)).
					Expect().
					Status(http.StatusOK).JSON().Object()
				result.ContainsKey("labels")
				r := result.Raw()
				return r
			}

			ctx = common.NewTestContext(nil)
			rWithMandatoryFields = t.DELETELIST.ResourceWithoutNullableFieldsBlueprint(ctx)
			for i := 0; i < 4; i++ {
				gen := t.DELETELIST.ResourceBlueprint(ctx)
				if t.SupportsLabels {
					gen = attachLabel(gen)
				}
				delete(gen, "created_at")
				delete(gen, "updated_at")
				r = append(r, gen)
			}

			if t.POST != nil {
				DescribePostTestsFor(ctx, t)
			}
			if t.GET != nil {
				DescribeGetTestsfor(ctx, t, r)
			}
			if t.LIST != nil {
				DescribeListTestsFor(ctx, t, r, rWithMandatoryFields)
			}
			if t.PATCH != nil {

			}
			if t.DELETE != nil {
				DescribeDeleteTestsfor(ctx, t)
			}
			if t.DELETELIST != nil {
				DescribeDeleteListFor(ctx, t)
			}
			By("==== Successfully finished preparation for SM component tests. Running API tests suites... ====")

		}()
	})
}
