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
	"strings"

	"github.com/Peripli/service-manager/pkg/query"

	. "github.com/onsi/ginkgo/extensions/table"

	"net/http"

	. "github.com/onsi/ginkgo"

	"github.com/Peripli/service-manager/test/common"
)

type listOpEntry struct {
	resourcesToExpectBeforeOp []common.Object

	queryTemplate               string
	queryArgs                   common.Object
	resourcesToExpectAfterOp    []common.Object
	resourcesNotToExpectAfterOp []common.Object
	expectedStatusCode          int
}

func DescribeListTestsFor(ctx *common.TestContext, t TestCase) bool {
	var r []common.Object
	var rWithMandatoryFields common.Object

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

	By(fmt.Sprintf("Attempting to create a random resource of %s with mandatory fields only", t.API))
	rWithMandatoryFields = t.ResourceWithoutNullableFieldsBlueprint(ctx)
	for i := 0; i < 4; i++ {
		By(fmt.Sprintf("Attempting to create a random resource of %s", t.API))

		gen := t.ResourceBlueprint(ctx)
		if t.SupportsLabels {
			gen = attachLabel(gen)
		}
		delete(gen, "created_at")
		delete(gen, "updated_at")
		r = append(r, gen)
	}

	entries := []TableEntry{
		Entry("returns 200",
			listOpEntry{
				resourcesToExpectBeforeOp: []common.Object{r[0]},
				queryTemplate:             "%s = %v",
				queryArgs:                 r[0],
				resourcesToExpectAfterOp:  []common.Object{r[0]},
				expectedStatusCode:        http.StatusOK,
			},
		),
		Entry("returns 200",
			listOpEntry{
				resourcesToExpectBeforeOp:   []common.Object{r[0], r[1], r[2], r[3]},
				queryTemplate:               "%s != %v",
				queryArgs:                   r[0],
				resourcesNotToExpectAfterOp: []common.Object{r[0]},
				expectedStatusCode:          http.StatusOK,
			},
		),

		Entry("returns 200",
			listOpEntry{
				resourcesToExpectBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
				queryTemplate:             "%[1]s in [%[2]v||%[2]v||%[2]v]",
				queryArgs:                 r[0],
				resourcesToExpectAfterOp:  []common.Object{r[0]},
				expectedStatusCode:        http.StatusOK,
			},
		),

		Entry("returns 200",
			listOpEntry{
				resourcesToExpectBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
				queryTemplate:             "%s in [%v]",
				queryArgs:                 r[0],
				resourcesToExpectAfterOp:  []common.Object{r[0]},
				expectedStatusCode:        http.StatusOK,
			},
		),
		Entry("returns 200",
			listOpEntry{
				resourcesToExpectBeforeOp:   []common.Object{r[0], r[1], r[2], r[3]},
				queryTemplate:               "%[1]s notin [%[2]v||%[2]v||%[2]v]",
				queryArgs:                   r[0],
				resourcesNotToExpectAfterOp: []common.Object{r[0]},
				expectedStatusCode:          http.StatusOK,
			},
		),
		Entry("returns 200",
			listOpEntry{
				resourcesToExpectBeforeOp:   []common.Object{r[0], r[1], r[2], r[3]},
				queryTemplate:               "%s notin [%v]",
				queryArgs:                   r[0],
				resourcesNotToExpectAfterOp: []common.Object{r[0]},
				expectedStatusCode:          http.StatusOK,
			},
		),
		Entry("returns 200",
			listOpEntry{
				resourcesToExpectBeforeOp:   []common.Object{r[0], r[1], r[2], r[3]},
				queryTemplate:               "%s gt %v",
				queryArgs:                   common.RemoveNonNumericArgs(r[0]),
				resourcesNotToExpectAfterOp: []common.Object{r[0]},
				expectedStatusCode:          http.StatusOK,
			},
		),
		Entry("returns 200",
			listOpEntry{
				resourcesToExpectBeforeOp:   []common.Object{r[0], r[1], r[2], r[3]},
				queryTemplate:               "%s lt %v",
				queryArgs:                   common.RemoveNonNumericArgs(r[0]),
				resourcesNotToExpectAfterOp: []common.Object{r[0]},
				expectedStatusCode:          http.StatusOK,
			},
		),
		Entry("returns 200 for field queries",
			listOpEntry{
				resourcesToExpectBeforeOp: []common.Object{r[0], rWithMandatoryFields},
				queryTemplate:             "%s eqornil %v",
				queryArgs:                 common.RemoveNotNullableFieldAndLabels(r[0], rWithMandatoryFields),
				resourcesToExpectAfterOp:  []common.Object{r[0], rWithMandatoryFields},
				expectedStatusCode:        http.StatusOK,
			},
		),
		Entry("returns 400 for label queries with operator eqornil",
			listOpEntry{
				queryTemplate: "%s eqornil %v",
				queryArgs: common.Object{
					"labels": map[string]interface{}{
						"labelKey1": []interface{}{
							"str",
						},
					}},
				expectedStatusCode: http.StatusBadRequest,
			},
		),
		Entry("returns 200 for JSON fields with stripped new lines",
			listOpEntry{
				resourcesToExpectBeforeOp: []common.Object{r[0]},
				queryTemplate:             "%s = %v",
				queryArgs:                 common.RemoveNonJSONArgs(r[0]),
				resourcesToExpectAfterOp:  []common.Object{r[0]},
				expectedStatusCode:        http.StatusOK,
			},
		),

		Entry("returns 400 when query operator is invalid",
			listOpEntry{
				queryTemplate:      "%s @@ %v",
				queryArgs:          r[0],
				expectedStatusCode: http.StatusBadRequest,
			},
		),
		Entry("returns 400 when query is duplicated",
			listOpEntry{
				queryTemplate:      "%[1]s = %[2]v|%[1]s = %[2]v",
				queryArgs:          r[0],
				expectedStatusCode: http.StatusBadRequest,
			},
		),
		Entry("returns 400 when operator is not properly separated with right space from operands",
			listOpEntry{
				queryTemplate:      "%s =%v",
				queryArgs:          r[0],
				expectedStatusCode: http.StatusBadRequest,
			},
		),
		Entry("returns 400 when operator is not properly separated with left space from operands",
			listOpEntry{
				queryTemplate:      "%s= %v",
				queryArgs:          r[0],
				expectedStatusCode: http.StatusBadRequest,
			},
		),

		Entry("returns 400 when field query left operands are unknown",
			listOpEntry{
				queryTemplate:      "%[1]s in [%[2]v||%[2]v]",
				queryArgs:          common.Object{"unknownkey": "unknownvalue"},
				expectedStatusCode: http.StatusBadRequest,
			},
		),
		Entry("returns 200 when label query left operands are unknown",
			listOpEntry{
				resourcesToExpectBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
				queryTemplate:             "%[1]s in [%[2]v||%[2]v]",
				queryArgs: common.Object{
					"labels": map[string]interface{}{
						"unknown": []interface{}{
							"unknown",
						},
					}},
				resourcesNotToExpectAfterOp: []common.Object{r[0], r[1], r[2], r[3]},
				expectedStatusCode:          http.StatusOK,
			},
		),
		Entry("returns 400 when single value operator is used with multiple right value arguments",
			listOpEntry{
				queryTemplate:      "%[1]s != [%[2]v||%[2]v||%[2]v]",
				queryArgs:          r[0],
				expectedStatusCode: http.StatusBadRequest,
			},
		),

		Entry("returns 400 when numeric operator is used with non-numeric operands",
			listOpEntry{
				queryTemplate:      "%s < %v",
				queryArgs:          common.RemoveNumericArgs(r[0]),
				expectedStatusCode: http.StatusBadRequest,
			},
		),
	}

	verifyListOp := func(listOpEntry listOpEntry, query string) {
		var expectedAfterOpIDs []string
		var unexpectedAfterOpIDs []string
		expectedAfterOpIDs = common.ExtractResourceIDs(listOpEntry.resourcesToExpectAfterOp)
		unexpectedAfterOpIDs = common.ExtractResourceIDs(listOpEntry.resourcesNotToExpectAfterOp)

		// workaround for brokers api
		jsonArrayKey := strings.Replace(t.API, "/v1/", "", 1)
		if jsonArrayKey == "service_brokers" {
			jsonArrayKey = "brokers"
		}

		By(fmt.Sprintf("[TEST]: Verifying expected %s before operation after present", t.API))
		beforeOpArray := ctx.SMWithOAuth.GET(t.API).
			Expect().
			Status(http.StatusOK).JSON().Object().Value(jsonArrayKey).Array()

		for _, v := range beforeOpArray.Iter() {
			obj := v.Object().Raw()
			delete(obj, "created_at")
			delete(obj, "updated_at")
		}

		for _, entity := range listOpEntry.resourcesToExpectBeforeOp {
			delete(entity, "created_at")
			delete(entity, "updated_at")
			beforeOpArray.Contains(entity)
		}

		By("[TEST]: ======= Expectations Summary =======")

		By(fmt.Sprintf("[TEST]: Listing %s with %s", t.API, query))
		By(fmt.Sprintf("[TEST]: Currently present resources: %v", r))
		By(fmt.Sprintf("[TEST]: Expected %s ids after operations: %s", t.API, expectedAfterOpIDs))
		By(fmt.Sprintf("[TEST]: Unexpected %s ids after operations: %s", t.API, unexpectedAfterOpIDs))
		By(fmt.Sprintf("[TEST]: Expected status code %d", listOpEntry.expectedStatusCode))

		By("[TEST]: ====================================")

		req := ctx.SMWithOAuth.GET(t.API)
		if query != "" {
			req = req.WithQueryString(query)
		}

		By(fmt.Sprintf("[TEST]: Verifying expected status code %d is returned from list operation", listOpEntry.expectedStatusCode))
		resp := req.
			Expect().
			Status(listOpEntry.expectedStatusCode)

		if listOpEntry.expectedStatusCode != http.StatusOK {
			By(fmt.Sprintf("[TEST]: Verifying error and description fields are returned after list operation"))

			resp.JSON().Object().Keys().Contains("error", "description")
		} else {
			array := resp.JSON().Object().Value(jsonArrayKey).Array()
			for _, v := range array.Iter() {
				obj := v.Object().Raw()
				delete(obj, "created_at")
				delete(obj, "updated_at")
			}

			if listOpEntry.resourcesToExpectAfterOp != nil {
				By(fmt.Sprintf("[TEST]: Verifying expected %s are returned after list operation", t.API))
				for _, entity := range listOpEntry.resourcesToExpectAfterOp {
					delete(entity, "created_at")
					delete(entity, "updated_at")
					array.Contains(entity)
				}
			}

			if listOpEntry.resourcesNotToExpectAfterOp != nil {
				By(fmt.Sprintf("[TEST]: Verifying unexpected %s are NOT returned after list operation", t.API))

				for _, entity := range listOpEntry.resourcesNotToExpectAfterOp {
					delete(entity, "created_at")
					delete(entity, "updated_at")
					array.NotContains(entity)
				}
			}
		}
	}

	return Describe("List", func() {
		Context("with basic auth", func() {
			It("returns 200", func() {
				ctx.SMWithBasic.GET(t.API).
					Expect().
					Status(http.StatusOK)
			})
		})

		Context("with bearer auth", func() {
			Context("with no query", func() {
				It("returns all the resources", func() {
					verifyListOp(listOpEntry{
						resourcesToExpectBeforeOp: []common.Object{r[0], r[1]},
						resourcesToExpectAfterOp:  []common.Object{r[0], r[1]},
						expectedStatusCode:        http.StatusOK,
					}, "")
				})
			})

			Context("with empty field query", func() {
				It("returns 200", func() {
					verifyListOp(listOpEntry{
						resourcesToExpectBeforeOp: []common.Object{r[0], r[1]},
						resourcesToExpectAfterOp:  []common.Object{r[0], r[1]},
						expectedStatusCode:        http.StatusOK,
					}, "fieldQuery=")
				})
			})

			Context("with empty label query", func() {
				It("returns 200", func() {
					verifyListOp(listOpEntry{
						resourcesToExpectBeforeOp: []common.Object{r[0], r[1]},
						resourcesToExpectAfterOp:  []common.Object{r[0], r[1]},
						expectedStatusCode:        http.StatusOK,
					}, "labelQuery=")
				})
			})

			Context("with empty label query and field query", func() {
				It("returns 200", func() {
					verifyListOp(listOpEntry{
						resourcesToExpectBeforeOp: []common.Object{r[0], r[1]},
						resourcesToExpectAfterOp:  []common.Object{r[0], r[1]},
						expectedStatusCode:        http.StatusOK,
					}, "labelQuery=&fieldQuery=")
				})
			})

			for i := 0; i < len(entries); i++ {
				params := entries[i].Parameters[0].(listOpEntry)
				if len(params.queryTemplate) == 0 {
					panic("query templates missing")
				}
				var multiQueryValue string
				var queryValues []string

				fields := common.CopyObject(params.queryArgs)
				delete(fields, "labels")
				multiQueryValue, queryValues = expandFieldQuery(fields, params.queryTemplate)
				fquery := "fieldQuery" + "=" + multiQueryValue

				Context("with field query=", func() {
					for _, queryValue := range queryValues {
						query := "fieldQuery" + "=" + queryValue
						DescribeTable(fmt.Sprintf("%s", queryValue), func(test listOpEntry) {
							verifyListOp(test, query)
						}, entries[i])
					}

					if len(queryValues) > 1 {
						DescribeTable(fmt.Sprintf("%s", multiQueryValue), func(test listOpEntry) {
							verifyListOp(test, fquery)
						}, entries[i])
					}
				})

				labels := params.queryArgs["labels"]
				if t.SupportsLabels && labels != nil {

					multiQueryValue, queryValues = expandLabelQuery(labels.(map[string]interface{}), params.queryTemplate)
					lquery := "labelQuery" + "=" + multiQueryValue

					Context("with label query=", func() {
						for _, queryValue := range queryValues {
							query := "labelQuery" + "=" + queryValue
							DescribeTable(fmt.Sprintf("%s", queryValue), func(test listOpEntry) {
								verifyListOp(test, query)
							}, entries[i])
						}

						if len(queryValues) > 1 {
							DescribeTable(fmt.Sprintf("%s", multiQueryValue), func(test listOpEntry) {
								verifyListOp(test, lquery)
							}, entries[i])
						}
					})

					Context("with multiple field and label queries", func() {
						DescribeTable(fmt.Sprintf("%s", fquery+"&"+lquery), func(test listOpEntry) {
							verifyListOp(test, fquery+"&"+lquery)
						}, entries[i])
					})
				}
			}
		})
	})
}

func expandFieldQuery(fieldQueryArgs common.Object, queryTemplate string) (string, []string) {
	var expandedMultiQuery string
	var expandedQueries []string
	for k, v := range fieldQueryArgs {
		if v == nil {
			continue
		}

		if m, ok := v.(map[string]interface{}); ok {
			bytes, err := json.Marshal(m)
			if err != nil {
				panic(err)
			}
			v = string(bytes)
		}
		if a, ok := v.([]interface{}); ok {
			bytes, err := json.Marshal(a)
			if err != nil {
				panic(err)
			}
			v = string(bytes)

		}
		expandedQueries = append(expandedQueries, fmt.Sprintf(queryTemplate, k, v))
	}

	expandedMultiQuery = strings.Join(expandedQueries, "|")
	return expandedMultiQuery, expandedQueries
}

func expandLabelQuery(labelQueryArgs map[string]interface{}, queryTemplate string) (string, []string) {
	var expandedMultiQuery string
	var expandedQueries []string

	for key, values := range labelQueryArgs {
		for _, value := range values.([]interface{}) {
			expandedQueries = append(expandedQueries, fmt.Sprintf(queryTemplate, key, value))
		}
	}

	expandedMultiQuery = strings.Join(expandedQueries, "|")
	return expandedMultiQuery, expandedQueries
}
