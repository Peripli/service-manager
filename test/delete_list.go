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

	"github.com/gavv/httpexpect"

	"github.com/Peripli/service-manager/pkg/query"

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
				queryTemplate: "%s eq '%v'",
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
				queryTemplate: "%s ne '%v'",
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
				queryTemplate: "%[1]s in ('%[2]v','%[2]v','%[2]v')",
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
				queryTemplate: "%s in ('%v')",
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
				queryTemplate: "%[1]s notin ('%[2]v','%[2]v','%[2]v')",
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
				queryTemplate: "%s notin ('%v')",
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
				queryTemplate: "%s gt '%v'",
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
				queryTemplate: "%s lt '%v'",
				queryArgs: func() common.Object {
					return common.RemoveNonNumericArgs(r[1])
				},
				resourcesToExpectAfterOp: func() []common.Object {
					return []common.Object{r[1]}
				},
				expectedStatusCode: http.StatusOK,
			},
		),
		Entry("returns 200 for greater than or equal queries",
			deleteOpEntry{
				resourcesToExpectBeforeOp: func() []common.Object {
					return []common.Object{r[0], r[1]}
				},
				queryTemplate: "%s ge %v",
				queryArgs: func() common.Object {
					return common.RemoveNonNumericArgs(r[1])
				},
				resourcesToExpectAfterOp: func() []common.Object {
					return []common.Object{r[0]}
				},
				expectedStatusCode: http.StatusOK,
			},
		),
		Entry("returns 400 for greater than or equal queries when query args are non numeric",
			deleteOpEntry{
				resourcesToExpectBeforeOp: func() []common.Object {
					return []common.Object{r[0], r[1]}
				},
				queryTemplate: "%s ge %v",
				queryArgs: func() common.Object {
					return common.RemoveNumericArgs(r[0])
				},
				resourcesToExpectAfterOp: func() []common.Object {
					return []common.Object{r[0], r[1]}
				},
				expectedStatusCode: http.StatusBadRequest,
			},
		),
		Entry("returns 200 for greater than or equal queries",
			deleteOpEntry{
				resourcesToExpectBeforeOp: func() []common.Object {
					return []common.Object{r[0], r[1]}
				},
				queryTemplate: "%s le %v",
				queryArgs: func() common.Object {
					return common.RemoveNonNumericArgs(r[1])
				},
				resourcesNotToExpectAfterOp: func() []common.Object {
					return []common.Object{r[0], r[1]}
				},
				expectedStatusCode: http.StatusOK,
			}),
		Entry("returns 400 for less than or equal queries when query args are non numeric",
			deleteOpEntry{
				resourcesToExpectBeforeOp: func() []common.Object {
					return []common.Object{r[0], r[1]}
				},
				queryTemplate: "%s le %v",
				queryArgs: func() common.Object {
					return common.RemoveNumericArgs(r[0])
				},
				resourcesToExpectAfterOp: func() []common.Object {
					return []common.Object{r[0], r[1]}
				},
				expectedStatusCode: http.StatusBadRequest,
			},
		),
		Entry("returns 200 for operator en",
			deleteOpEntry{
				resourcesToExpectBeforeOp: func() []common.Object {
					return []common.Object{r[0], rWithMandatoryFields}
				},
				queryTemplate: "%s en '%v'",
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
				queryTemplate: "%s eq '%v'",
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
				queryTemplate: "%s @@ '%v'",
				queryArgs: func() common.Object {
					return r[0]
				},
				expectedStatusCode: http.StatusBadRequest,
			},
		),
		Entry("returns 400 when query is duplicated",
			deleteOpEntry{
				queryTemplate: "%[1]s = '%[2]v' and %[1]s = '%[2]v'",
				queryArgs: func() common.Object {
					return r[0]
				},
				expectedStatusCode: http.StatusBadRequest,
			},
		),
		Entry("returns 400 when operator is not properly separated with right space from operands",
			deleteOpEntry{
				queryTemplate: "%s ='%v'",
				queryArgs: func() common.Object {
					return r[0]
				},
				expectedStatusCode: http.StatusBadRequest,
			},
		),
		Entry("returns 400 when operator is not properly separated with left space from operands",
			deleteOpEntry{
				queryTemplate: "%seq '%v'",
				queryArgs: func() common.Object {
					return r[0]
				},
				expectedStatusCode: http.StatusBadRequest,
			},
		),

		Entry("returns 400 when field query left operands are unknown",
			deleteOpEntry{
				queryTemplate: "%[1]s in ('%[2]v','%[2]v')",
				queryArgs: func() common.Object {
					return common.Object{"unknownkey": "unknownvalue"}
				},
				expectedStatusCode: http.StatusBadRequest,
			},
		),
		Entry("returns 200 when label query left operands are unknown",
			deleteOpEntry{
				resourcesToExpectBeforeOp: func() []common.Object {
					return []common.Object{r[0], r[1]}
				},
				queryTemplate: "%[1]s in ('%[2]v','%[2]v')",
				queryArgs: func() common.Object {
					return common.Object{
						"labels": map[string]interface{}{
							"unknown": []interface{}{
								"unknown",
							},
						}}
				},
				resourcesToExpectAfterOp: func() []common.Object {
					return []common.Object{r[0], r[1]}
				},
				expectedStatusCode: http.StatusNotFound,
			},
		),
		Entry("returns 400 when single value operator is used with multiple right value arguments",
			deleteOpEntry{
				queryTemplate: "%[1]s ne ('%[2]v','%[2]v','%[2]v')",
				queryArgs: func() common.Object {
					return r[0]
				},
				expectedStatusCode: http.StatusBadRequest,
			},
		),
		Entry("returns 400 when numeric operator is used with non-numeric operands",
			deleteOpEntry{
				queryTemplate: "%s < '%v'",

				queryArgs: func() common.Object {
					return common.RemoveNumericArgs(r[0])
				},
				expectedStatusCode: http.StatusBadRequest,
			},
		),
	}

	attachLabel := func(obj common.Object, i int) common.Object {
		patchLabelsBody := make(map[string]interface{})
		patchLabels := []query.LabelChange{
			{
				Operation: query.AddLabelOperation,
				Key:       "labelKey1",
				Values:    []string{fmt.Sprintf("%d", i)},
			},
			{
				Operation: query.AddLabelOperation,
				Key:       "labelKey2",
				Values:    []string{fmt.Sprintf("str%d", i)},
			},
			{
				Operation: query.AddLabelOperation,
				Key:       "labelKey3",
				Values:    []string{fmt.Sprintf(`{"key%d": "val%d"}`, i, i)},
			},
		}
		patchLabelsBody["labels"] = patchLabels

		By(fmt.Sprintf("Attempting to patch resource of %s with labels as labels are declared supported", t.API))
		ctx.SMWithOAuth.PATCH(t.API + "/" + obj["id"].(string)).WithJSON(patchLabelsBody).
			Expect().
			Status(http.StatusOK)

		result := ctx.SMWithOAuth.GET(t.API + "/" + obj["id"].(string)).
			Expect().
			Status(http.StatusOK).JSON().Object()
		result.ContainsKey("labels")
		r := result.Raw()
		return r
	}

	beforeEachHelper := func() {
		By(fmt.Sprintf("[BEFOREEACH]: Preparing and creating test resources"))

		r = make([]common.Object, 0, 0)
		rWithMandatoryFields = t.ResourceWithoutNullableFieldsBlueprint(ctx, ctx.SMWithOAuth)
		for i := 0; i < 2; i++ {
			gen := t.ResourceBlueprint(ctx, ctx.SMWithOAuth)
			gen = attachLabel(gen, i)
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

	verifyDeleteListOpHelperWithAuth := func(deleteListOpEntry deleteOpEntry, query string, auth *httpexpect.Expect) {
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

		req := auth.DELETE(t.API)
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
	verifyDeleteListOpHelper := func(deleteListOpEntry deleteOpEntry, query string) {
		verifyDeleteListOpHelperWithAuth(deleteListOpEntry, query, ctx.SMWithOAuth)
	}

	verifyDeleteListOp := func(entry deleteOpEntry) {
		if len(entry.queryTemplate) == 0 {
			Fail("Query template missing")
		}

		// test with multi field query - meaning all fields in one query
		beforeEachHelper()
		args := entry.queryArgs()
		labels := args["labels"]
		fields := common.CopyObject(args)
		delete(fields, "labels")

		queryKeys := extractQueryKeys(fields)
		if len(queryKeys) > 1 {
			fquery := "fieldQuery=" + expandMultiFieldQuery(entry.queryTemplate, fields)
			verifyDeleteListOpHelper(entry, fquery)
		}
		afterEachHelper()

		for _, queryKey := range queryKeys {
			// test with each field as separate field query
			beforeEachHelper()
			args := entry.queryArgs()
			fields := common.CopyObject(args)
			delete(fields, "labels")
			queryValue := expandNextFieldQuery(entry.queryTemplate, fields, queryKey)
			query := "fieldQuery=" + queryValue
			verifyDeleteListOpHelper(entry, query)
			afterEachHelper()
		}

		if labels != nil {
			// test with all labels as one label query
			beforeEachHelper()
			multiLabelQuery, labelQueries := expandLabelQuery(labels.(map[string]interface{}), entry.queryTemplate)
			verifyDeleteListOpHelper(entry, "labelQuery="+multiLabelQuery)
			afterEachHelper()

			for _, labelQuery := range labelQueries {
				// test with each label as separate label query
				beforeEachHelper()
				verifyDeleteListOpHelper(entry, "labelQuery="+labelQuery)
				afterEachHelper()
			}

			// test with all fields and all labels as one query
			beforeEachHelper()
			// recalculate the multi field query as each beforeEach creates new resources and field values may differ
			fields := common.CopyObject(entry.queryArgs())
			delete(fields, "labels")
			multiFieldQuery := expandMultiFieldQuery(entry.queryTemplate, fields)
			verifyDeleteListOpHelper(entry, "fieldQuery="+multiFieldQuery+"&"+"labelQuery="+multiLabelQuery)
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
				if !t.DisableTenantResources {
					Context("when authenticating with tenant scoped token", func() {
						var rForTenant common.Object

						BeforeEach(func() {
							rForTenant = t.ResourceBlueprint(ctx, ctx.SMWithOAuthForTenant)
						})

						It("deletes only tenant specific resources", func() {
							verifyDeleteListOpHelperWithAuth(deleteOpEntry{
								resourcesToExpectBeforeOp: func() []common.Object {
									return []common.Object{r[0], r[1], rForTenant}
								},
								resourcesNotToExpectAfterOp: func() []common.Object {
									return []common.Object{rForTenant}
								},
								resourcesToExpectAfterOp: func() []common.Object {
									return []common.Object{r[0], r[1]}
								},
								expectedStatusCode: http.StatusOK,
							}, "", ctx.SMWithOAuthForTenant)
						})

						Context("when authenticating with global token", func() {
							It("deletes all resources", func() {
								verifyDeleteListOpHelperWithAuth(deleteOpEntry{
									resourcesToExpectBeforeOp: func() []common.Object {
										return []common.Object{r[0], r[1], rForTenant}
									},
									resourcesNotToExpectAfterOp: func() []common.Object {
										return []common.Object{r[0], r[1], rForTenant}
									},
									expectedStatusCode: http.StatusOK,
								}, "", ctx.SMWithOAuth)
							})
						})
					})
				}

				Context("with no query", func() {
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

				Context("with empty label query", func() {
					It("returns 200", func() {
						verifyDeleteListOpHelper(deleteOpEntry{
							resourcesToExpectBeforeOp: func() []common.Object {
								return []common.Object{r[0], r[1]}
							},
							resourcesNotToExpectAfterOp: func() []common.Object {
								return []common.Object{r[0], r[1]}
							},
							expectedStatusCode: http.StatusOK,
						}, "labelQuery=")
					})
				})

				Context("with empty field and label query", func() {
					It("returns 200", func() {
						verifyDeleteListOpHelper(deleteOpEntry{
							resourcesToExpectBeforeOp: func() []common.Object {
								return []common.Object{r[0], r[1]}
							},
							resourcesNotToExpectAfterOp: func() []common.Object {
								return []common.Object{r[0], r[1]}
							},
							expectedStatusCode: http.StatusOK,
						}, "fieldQuery=&labelQuery=")
					})
				})
			})
		})

		DescribeTable("with non-empty query", verifyDeleteListOp, entries...)
	})
}

func extractQueryKeys(queryArgsObj common.Object) []string {
	queryKeys := make([]string, 0)
	for key := range queryArgsObj {
		queryKeys = append(queryKeys, key)
	}

	return queryKeys
}

func expandNextFieldQuery(queryTemplate string, queryArgs common.Object, currentArgKey string) string {
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

func expandMultiFieldQuery(queryTemplate string, queryArgs common.Object) string {
	expandedMultiQuerySegments := make([]string, 0)
	for queryArgKey, queryArgValue := range queryArgs {
		expandedMultiQuerySegments = append(expandedMultiQuerySegments, fmt.Sprintf(queryTemplate, queryArgKey, queryArgValue))
	}
	return strings.Join(expandedMultiQuerySegments, " and ")
}
