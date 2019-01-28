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
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
)

type deleteOpEntry struct {
	resourcesToExpectBeforeOp func() []common.Object

	queryTemplate               string
	queryArgs                   func() common.Object
	resourcesToExpectAfterOp    func() []common.Object
	resourcesNotToExpectAfterOp func() []common.Object
	expectedStatusCode          int
}

func DescribeDeleteListFor(ctx *common.TestContext, t TestCase) bool {
	var r []common.Object
	var rWithMandatoryFields common.Object

	entries := []TableEntry{
		Entry("returns 200 for operator =",
			deleteOpEntry{
				resourcesToExpectBeforeOp: func() []common.Object {
					return []common.Object{r[0]}
				},
				queryTemplate: "%s = %v",
				queryArgs: func() common.Object {
					return r[0]
				},
				resourcesNotToExpectAfterOp: func() []common.Object {
					return []common.Object{r[0]}
				},
				expectedStatusCode: http.StatusOK,
			},
		),
		Entry("returns 200 for operator !=",
			deleteOpEntry{
				resourcesToExpectBeforeOp: func() []common.Object {
					return []common.Object{r[0], r[1]}
				},
				queryTemplate: "%s != %v",
				queryArgs: func() common.Object {
					return r[0]
				},
				resourcesToExpectAfterOp: func() []common.Object {
					return []common.Object{r[0]}
				},
				expectedStatusCode: http.StatusOK,
			},
		),

		Entry("returns 200 for operator in with multiple right operands",
			deleteOpEntry{
				resourcesToExpectBeforeOp: func() []common.Object {
					return []common.Object{r[0], r[1]}
				},
				queryTemplate: "%[1]s in [%[2]v||%[2]v||%[2]v]",
				queryArgs: func() common.Object {
					return r[0]
				},
				resourcesNotToExpectAfterOp: func() []common.Object {
					return []common.Object{r[0]}
				},
				expectedStatusCode: http.StatusOK,
			},
		),

		Entry("returns 200 for operator in with single right operand",
			deleteOpEntry{
				resourcesToExpectBeforeOp: func() []common.Object {
					return []common.Object{r[0], r[1]}
				},
				queryTemplate: "%s in [%v]",
				queryArgs: func() common.Object {
					return r[0]
				},
				resourcesNotToExpectAfterOp: func() []common.Object {
					return []common.Object{r[0]}
				},
				expectedStatusCode: http.StatusOK,
			},
		),
		Entry("returns 200 for operator notin with multiple right operands",
			deleteOpEntry{
				resourcesToExpectBeforeOp: func() []common.Object {
					return []common.Object{r[0], r[1]}
				},
				queryTemplate: "%[1]s notin [%[2]v||%[2]v||%[2]v]",
				queryArgs: func() common.Object {
					return r[0]
				},
				resourcesToExpectAfterOp: func() []common.Object {
					return []common.Object{r[0]}
				},
				expectedStatusCode: http.StatusOK,
			},
		),
		Entry("returns 200 for operator notin with single right operand",
			deleteOpEntry{
				resourcesToExpectBeforeOp: func() []common.Object {
					return []common.Object{r[0], r[1]}
				},
				queryTemplate: "%s notin [%v]",
				queryArgs: func() common.Object {
					return r[0]
				},
				resourcesToExpectAfterOp: func() []common.Object {
					return []common.Object{r[0]}
				},
				expectedStatusCode: http.StatusOK,
			},
		),
		Entry("returns 200 for operator gt",
			deleteOpEntry{
				resourcesToExpectBeforeOp: func() []common.Object {
					return []common.Object{r[0], r[1]}
				},
				queryTemplate: "%s gt %v",
				queryArgs: func() common.Object {
					return common.RemoveNonNumericArgs(r[0])
				},
				resourcesToExpectAfterOp: func() []common.Object {
					return []common.Object{r[0]}
				},
				expectedStatusCode: http.StatusOK,
			},
		),
		Entry("returns 200 for operator lt",
			deleteOpEntry{
				resourcesToExpectBeforeOp: func() []common.Object {
					return []common.Object{r[0], r[1]}
				},
				queryTemplate: "%s lt %v",
				queryArgs: func() common.Object {
					return common.RemoveNonNumericArgs(r[0])
				},
				resourcesToExpectAfterOp: func() []common.Object {
					return []common.Object{r[0]}
				},
				expectedStatusCode: http.StatusOK,
			},
		),
		Entry("returns 200 for operator eqornil",
			deleteOpEntry{
				resourcesToExpectBeforeOp: func() []common.Object {
					return []common.Object{r[0], rWithMandatoryFields}
				},
				queryTemplate: "%s eqornil %v",
				queryArgs: func() common.Object {
					return common.RemoveNotNullableFieldAndLabels(r[0], rWithMandatoryFields)
				},
				resourcesNotToExpectAfterOp: func() []common.Object {
					return []common.Object{r[0], rWithMandatoryFields}
				},
				expectedStatusCode: http.StatusOK,
			},
		),
		Entry("returns 200 for JSON fields with stripped new lines",
			deleteOpEntry{
				resourcesToExpectBeforeOp: func() []common.Object {
					return []common.Object{r[0]}
				},
				queryTemplate: "%s = %v",
				queryArgs: func() common.Object {
					return common.RemoveNonJSONArgs(r[0])
				},
				resourcesNotToExpectAfterOp: func() []common.Object {
					return []common.Object{r[0]}
				},
				expectedStatusCode: http.StatusOK,
			},
		),

		Entry("returns 400 when query operator is invalid",
			deleteOpEntry{
				queryTemplate: "%s @@ %v",
				queryArgs: func() common.Object {
					return r[0]
				},
				expectedStatusCode: http.StatusBadRequest,
			},
		),
		Entry("returns 400 when query is duplicated",
			deleteOpEntry{
				queryTemplate: "%[1]s = %[2]v|%[1]s = %[2]v",
				queryArgs: func() common.Object {
					return r[0]
				},
				expectedStatusCode: http.StatusBadRequest,
			},
		),
		Entry("returns 400 when operator is not properly separated with right space from operands",
			deleteOpEntry{
				queryTemplate: "%s =%v",
				queryArgs: func() common.Object {
					return r[0]
				},
				expectedStatusCode: http.StatusBadRequest,
			},
		),
		Entry("returns 400 when operator is not properly separated with left space from operands",
			deleteOpEntry{
				queryTemplate: "%s= %v",
				queryArgs: func() common.Object {
					return r[0]
				},
				expectedStatusCode: http.StatusBadRequest,
			},
		),

		Entry("returns 400 when field query left operands are unknown",
			deleteOpEntry{
				queryTemplate: "%[1]s in [%[2]v||%[2]v]",
				queryArgs: func() common.Object {
					return common.Object{"unknownkey": "unknownvalue"}
				},
				expectedStatusCode: http.StatusBadRequest,
			},
		),
		Entry("returns 400 when single value operator is used with multiple right value arguments",
			deleteOpEntry{
				queryTemplate: "%[1]s != [%[2]v||%[2]v||%[2]v]",
				queryArgs: func() common.Object {
					return r[0]
				},
				expectedStatusCode: http.StatusBadRequest,
			},
		),

		Entry("returns 400 when numeric operator is used with non-numeric operands",
			deleteOpEntry{
				queryTemplate: "%s < %v",

				queryArgs: func() common.Object {
					return common.RemoveNumericArgs(r[0])
				},
				expectedStatusCode: http.StatusBadRequest,
			},
		),
	}

	beforeEachHelper := func() {
		By(fmt.Sprintf("[BEFOREEACH]: Preparing and creating test resources"))

		r = make([]common.Object, 0, 0)
		rWithMandatoryFields = t.ResourceWithoutNullableFieldsBlueprint(ctx)
		for i := 0; i < 2; i++ {
			gen := t.ResourceBlueprint(ctx)
			delete(gen, "created_at")
			delete(gen, "updated_at")
			r = append(r, gen)
		}
		By(fmt.Sprintf("[BEFOREEACH]: Successfully finished preparing and creating test resources"))
	}

	afterEachHelper := func() {
		By(fmt.Sprintf("[AFTEREACH]: Cleaning up test resources"))
		ctx.CleanupAdditionalResources()

		By(fmt.Sprintf("[AFTEREACH]: Sucessfully finished cleaning up test resources"))
	}

	verifyDeleteListOpHelper := func(deleteListOpEntry deleteOpEntry, query string) {
		jsonArrayKey := strings.Replace(t.API, "/v1/", "", 1)

		expectedAfterOpIDs := make([]string, 0)
		unexpectedAfterOpIDs := make([]string, 0)

		if deleteListOpEntry.resourcesToExpectAfterOp != nil {
			expectedAfterOpIDs = common.ExtractResourceIDs(deleteListOpEntry.resourcesToExpectAfterOp())
		}
		if deleteListOpEntry.resourcesNotToExpectAfterOp != nil {
			unexpectedAfterOpIDs = common.ExtractResourceIDs(deleteListOpEntry.resourcesNotToExpectAfterOp())
		}

		By("[TEST]: ======= Expectations Summary =======")

		By(fmt.Sprintf("[TEST]: Deleting %s at %s", t.API, query))
		By(fmt.Sprintf("[TEST]: Currently present resources: %v", r))
		By(fmt.Sprintf("[TEST]: Expected %s ids after operations: %s", t.API, expectedAfterOpIDs))
		By(fmt.Sprintf("[TEST]: Unexpected %s ids after operations: %s", t.API, unexpectedAfterOpIDs))
		By(fmt.Sprintf("[TEST]: Expected status code %d", deleteListOpEntry.expectedStatusCode))

		By("[TEST]: ====================================")

		if deleteListOpEntry.resourcesToExpectBeforeOp != nil {
			By(fmt.Sprintf("[TEST]: Verifying expected %s before operation are present", t.API))
			beforeOpArray := ctx.SMWithOAuth.GET(t.API).
				Expect().
				Status(http.StatusOK).JSON().Object().Value(jsonArrayKey).Array()

			for _, v := range beforeOpArray.Iter() {
				obj := v.Object().Raw()
				delete(obj, "created_at")
				delete(obj, "updated_at")
			}

			for _, entity := range deleteListOpEntry.resourcesToExpectBeforeOp() {
				delete(entity, "created_at")
				delete(entity, "updated_at")
				beforeOpArray.Contains(entity)
			}
		}

		req := ctx.SMWithOAuth.DELETE(t.API)
		if query != "" {
			req = req.WithQueryString(query)
		}

		By(fmt.Sprintf("[TEST]: Verifying expected status code %d is returned from operation", deleteListOpEntry.expectedStatusCode))
		resp := req.
			Expect().
			Status(deleteListOpEntry.expectedStatusCode)

		if deleteListOpEntry.expectedStatusCode != http.StatusOK {
			By(fmt.Sprintf("[TEST]: Verifying error and description fields are returned after operation"))
			resp.JSON().Object().Keys().Contains("error", "description")
		} else {
			afterOpArray := ctx.SMWithOAuth.GET(t.API).
				Expect().
				Status(http.StatusOK).JSON().Object().Value(jsonArrayKey).Array()

			for _, v := range afterOpArray.Iter() {
				obj := v.Object().Raw()
				delete(obj, "created_at")
				delete(obj, "updated_at")
			}

			if deleteListOpEntry.resourcesToExpectAfterOp != nil {
				By(fmt.Sprintf("[TEST]: Verifying expected %s are returned after operation", t.API))
				for _, entity := range deleteListOpEntry.resourcesToExpectAfterOp() {
					delete(entity, "created_at")
					delete(entity, "updated_at")
					afterOpArray.Contains(entity)
				}
			}

			if deleteListOpEntry.resourcesNotToExpectAfterOp != nil {
				By(fmt.Sprintf("[TEST]: Verifying unexpected %s are NOT returned after operation", t.API))
				for _, entity := range deleteListOpEntry.resourcesNotToExpectAfterOp() {
					delete(entity, "created_at")
					delete(entity, "updated_at")
					afterOpArray.NotContains(entity)
				}
			}
		}
	}

	verifyDeleteListOp := func(entry deleteOpEntry) {
		beforeEachHelper()
		if len(entry.queryTemplate) == 0 {
			Fail("Query template missing")
		}

		queryKeys := extractQueryKeys(entry.queryArgs)
		if len(queryKeys) > 1 {
			fquery := "fieldQuery=" + expandMultiFieldQuery(entry.queryTemplate, entry.queryArgs)
			verifyDeleteListOpHelper(entry, fquery)
		}
		afterEachHelper()

		for _, queryKey := range queryKeys {
			beforeEachHelper()
			queryValue := expandNextFieldQuery(entry.queryTemplate, entry.queryArgs, queryKey)
			query := "fieldQuery=" + queryValue
			verifyDeleteListOpHelper(entry, query)

			afterEachHelper()
		}

	}

	return Describe("DELETE LIST", func() {
		Context("with no filtering", func() {
			BeforeEach(beforeEachHelper)

			AfterEach(afterEachHelper)

			Context("with basic auth", func() {
				It("returns 200", func() {
					ctx.SMWithBasic.DELETE(t.API).
						Expect().
						Status(http.StatusUnauthorized)
				})
			})

			Context("with bearer auth", func() {
				Context("with no field query", func() {
					It("deletes all the resources", func() {
						verifyDeleteListOpHelper(deleteOpEntry{
							resourcesToExpectBeforeOp: func() []common.Object {
								return []common.Object{r[0], r[1]}
							},
							resourcesNotToExpectAfterOp: func() []common.Object {
								return []common.Object{r[0], r[1]}
							},
							expectedStatusCode: http.StatusOK,
						}, "")
					})
				})

				Context("with empty field query", func() {
					It("returns 200", func() {
						verifyDeleteListOpHelper(deleteOpEntry{
							resourcesToExpectBeforeOp: func() []common.Object {
								return []common.Object{r[0], r[1]}
							},
							resourcesNotToExpectAfterOp: func() []common.Object {
								return []common.Object{r[0], r[1]}
							},
							expectedStatusCode: http.StatusOK,
						}, "fieldQuery=")
					})
				})
			})
		})

		DescribeTable("with non-empty query", verifyDeleteListOp, entries...)
	})
}

func extractQueryKeys(queryArgsFunc func() common.Object) []string {
	queryKeys := make([]string, 0)
	queryArgsObj := queryArgsFunc()
	for key := range queryArgsObj {
		queryKeys = append(queryKeys, key)
	}

	return queryKeys
}

func expandNextFieldQuery(queryTemplate string, queryArgsFunc func() common.Object, currentArgKey string) string {
	queryArgs := queryArgsFunc()
	currentArgValue, ok := queryArgs[currentArgKey]

	if !ok || currentArgValue == nil {
		panic("decide what to do")
	}

	if m, ok := currentArgValue.(map[string]interface{}); ok {
		bytes, err := json.Marshal(m)
		if err != nil {
			panic(err)
		}
		currentArgValue = string(bytes)
	}

	if a, ok := currentArgValue.([]interface{}); ok {
		bytes, err := json.Marshal(a)
		if err != nil {
			panic(err)
		}
		currentArgValue = string(bytes)

	}
	return fmt.Sprintf(queryTemplate, currentArgKey, currentArgValue)
}

func expandMultiFieldQuery(queryTemplate string, queryArgsFunc func() common.Object) string {
	expandedMultiQuerySegments := make([]string, 0)
	queryArgs := queryArgsFunc()

	for queryArgKey, queryArgValue := range queryArgs {
		expandedMultiQuerySegments = append(expandedMultiQuerySegments, fmt.Sprintf(queryTemplate, queryArgKey, queryArgValue))
	}
	return strings.Join(expandedMultiQuerySegments, "|")
}
