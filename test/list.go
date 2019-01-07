package test

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"

	. "github.com/onsi/ginkgo/extensions/table"

	"net/http"

	. "github.com/onsi/ginkgo"

	"github.com/Peripli/service-manager/test/common"
)

type listOpEntry struct {
	expectedResourcesBeforeOp []common.Object

	OpFieldQueryTemplate string
	OpFieldQueryArgs     common.Object

	expectedResourcesAfterOp   []common.Object
	unexpectedResourcesAfterOp []common.Object
	expectedStatusCode         int
}

//TODO duplicated field query - maybe offieldqueryargs can become an array when we can put r[0], r[1] and expect bad request
//TODO singleval operator and multiple right values
// separate entries into groups
// consider splitting entries and therefore tables
//todo this entries wont work if (esp > < bad request if the fieldqueryargs object has key: 5

func DescribeListTestsFor(ctx *common.TestContext, t TestCase, r []common.Object) bool {
	entries := []TableEntry{
		// invalid operator
		Entry("returns 400 when field query operator is invalid",
			listOpEntry{
				expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
				OpFieldQueryTemplate:      "%s+@@@+%s",
				OpFieldQueryArgs:          r[0],
				expectedStatusCode:        http.StatusBadRequest,
			},
		),

		// missing operator
		Entry("returns 400 when field query operator is missing",
			listOpEntry{
				expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
				OpFieldQueryTemplate:      "%s++%s",
				OpFieldQueryArgs:          r[0],
				expectedStatusCode:        http.StatusBadRequest,
			},
		),

		// some created resources, valid operators and field query right operands that match some resources

		// one time validate that spaces instead of + works
		Entry("returns 200 when spaces are used instead of tabs",
			listOpEntry{
				expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
				OpFieldQueryTemplate:      "%s = %s",
				OpFieldQueryArgs:          r[0],
				expectedResourcesAfterOp:  []common.Object{r[0]},
				expectedStatusCode:        http.StatusOK,
			},
		),

		//TODO arrays for field template and field op args

		// one time duplicated query
		//Entry("returns 400",
		//	listOpEntry{
		//		expectedResourcesBeforeOp: func() []common.Object {
		//			return []common.Object{r[0], r[1], r[2], r[3]}
		//		},
		//		OpFieldQueryTemplate: []string{"%s+=+%s"},
		//		OpFieldQueryArgs:     func() []common.Object { return []common.Object{r[0],r[0] },
		//		expectedStatusCode: http.StatusOK,
		//	},
		//),

		// one time duplicated query
		Entry("returns 400 when field query is duplicated", listOpEntry{
			expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
			OpFieldQueryTemplate:      "%[1]s+=+%[2]s|%[1]s+=+%[2]s",
			OpFieldQueryArgs:          r[0],
			expectedStatusCode:        http.StatusBadRequest,
		}),
		// one time single value operator with multiple right values
		Entry("returns 400 when single value operator is used with multiple right value arguments",
			listOpEntry{
				expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
				OpFieldQueryTemplate:      "%[1]s+=+[%[2]s|%[2]s|%[2]s]",
				OpFieldQueryArgs:          r[0],
				expectedStatusCode:        http.StatusBadRequest,
			},
		),

		// one time invalid +
		Entry("returns 400 when operator is not properly separated with + from operands",
			listOpEntry{
				expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
				OpFieldQueryTemplate:      "%s+=%s",
				OpFieldQueryArgs:          r[0],
				expectedStatusCode:        http.StatusBadRequest,
			},
		),

		// one time invalid spacing
		Entry("returns 400 when operator is not properly separated with space from operands",
			listOpEntry{
				expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
				OpFieldQueryTemplate:      "%s =%s",
				OpFieldQueryArgs:          r[0],
				expectedStatusCode:        http.StatusBadRequest,
			},
		),

		Entry("returns 200",
			listOpEntry{
				expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
				OpFieldQueryTemplate:      "%s+=+%s",
				OpFieldQueryArgs:          r[0],
				expectedResourcesAfterOp:  []common.Object{r[0]},
				expectedStatusCode:        http.StatusOK,
			},
		),

		Entry("returns 200",
			listOpEntry{
				expectedResourcesBeforeOp:  []common.Object{r[0], r[1], r[2], r[3]},
				OpFieldQueryTemplate:       "%s+!=+%s",
				OpFieldQueryArgs:           r[0],
				unexpectedResourcesAfterOp: []common.Object{r[0]},
				expectedStatusCode:         http.StatusOK,
			},
		),

		// in operator multiple right arguments
		Entry("returns 200",
			listOpEntry{
				expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
				OpFieldQueryTemplate:      "%[1]s+in+[%[2]s|%[2]s|%[2]s]",
				OpFieldQueryArgs:          r[0],
				expectedResourcesAfterOp:  []common.Object{r[0]},
				expectedStatusCode:        http.StatusOK,
			},
		),

		// in operator single right argument
		Entry("returns 200",
			listOpEntry{
				expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
				OpFieldQueryTemplate:      "%s+in+[%s]",
				OpFieldQueryArgs:          r[0],
				expectedResourcesAfterOp:  []common.Object{r[0]},
				expectedStatusCode:        http.StatusOK,
			},
		),

		//TODO bug
		// in operator multiple right arguments without brackets
		//FEntry("returns 400",
		//	listOpEntry{
		//		expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
		//		OpFieldQueryTemplate:      "%[1]s+in+%[2]s,%[2]s",
		//		OpFieldQueryArgs:          r[0],
		//		expectedResourcesAfterOp:  []common.Object{r[0]},
		//		expectedStatusCode:        http.StatusBadRequest,
		//	},
		//),
		// not in operator multiple right arguments
		Entry("returns 200",
			listOpEntry{
				expectedResourcesBeforeOp:  []common.Object{r[0], r[1], r[2], r[3]},
				OpFieldQueryTemplate:       "%[1]s+notin+[%[2]s|%[2]s|%[2]s]",
				OpFieldQueryArgs:           r[0],
				unexpectedResourcesAfterOp: []common.Object{r[0]},
				expectedStatusCode:         http.StatusOK,
			},
		),

		// not in operator single right argument
		Entry("returns 200",
			listOpEntry{
				expectedResourcesBeforeOp:  []common.Object{r[0], r[1], r[2], r[3]},
				OpFieldQueryTemplate:       "%s+notin+[%s]",
				OpFieldQueryArgs:           r[0],
				unexpectedResourcesAfterOp: []common.Object{r[0]},
				expectedStatusCode:         http.StatusOK,
			},
		),

		//TODO bug
		// not in operator multiple right arguments without brackets
		//FEntry("returns 400",
		//	listOpEntry{
		//		expectedResourcesBeforeOp:  []common.Object{r[0], r[1], r[2], r[3]},
		//		OpFieldQueryTemplate:       "%[1]s+notin+%[2]s,%[2]s,%[2]s",
		//		OpFieldQueryArgs:           r[0],
		//		unexpectedResourcesAfterOp: []common.Object{r[0]},
		//		expectedStatusCode:         http.StatusBadRequest,
		//	},
		//),

		Entry("returns 400 when numeric operator is used with non-numeric operands",
			listOpEntry{
				expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
				OpFieldQueryTemplate:      "%s+>+%s",
				OpFieldQueryArgs:          common.RmNumbericFieldNames(common.CopyObject(r[0])),
				expectedStatusCode:        http.StatusBadRequest,
			},
		),
		Entry("returns 200",
			listOpEntry{
				expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
				OpFieldQueryTemplate:      "%s+>+%s",
				OpFieldQueryArgs:          common.RmNonNumbericFieldNames(common.CopyObject(r[0])),
				expectedResourcesAfterOp:  []common.Object{r[0]},
				expectedStatusCode:        http.StatusOK,
			},
		),
		Entry("returns 400",
			listOpEntry{
				expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
				OpFieldQueryTemplate:      "%s+<+%s",
				OpFieldQueryArgs:          common.RmNumbericFieldNames(common.CopyObject(r[0])),
				expectedStatusCode:        http.StatusBadRequest,
			},
		),
		Entry("returns 200",
			listOpEntry{
				expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
				OpFieldQueryTemplate:      "%s+<+%s",
				OpFieldQueryArgs:          common.RmNonNumbericFieldNames(common.CopyObject(r[0])),
				expectedResourcesAfterOp:  []common.Object{r[0]},
				expectedStatusCode:        http.StatusOK,
			},
		),

		//TODO mandatory fields resource
		//Entry("returns 200",
		//	listOpEntry{
		//		expectedResourcesBeforeOp: func() []common.Object {
		//			return []common.Object{r[0], mandatoryFieldsResource}
		//		},
		//		OpFieldQueryTemplate: "%s+eqornil+%s",
		//		OpFieldQueryArgs:     func() common.Object { return rmMandatoryFields(t.optionalFields, r[0]) },
		//		expectedResourcesAfterOp: func() []common.Object {
		//			return []common.Object{r[0], mandatoryFieldsResource}
		//		},
		//		expectedStatusCode: http.StatusOK,
		//	},
		//),

		// invalid field query left/right operand
		Entry("returns 400 when left/right operands are unknown",
			listOpEntry{
				expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
				OpFieldQueryTemplate:      "%s+=+%s",
				OpFieldQueryArgs:          common.Object{"unknownkey": "unknownvalue"},
				expectedStatusCode:        http.StatusBadRequest,
			},
		),
		Entry("returns 400 when left/right operands are unknown",
			listOpEntry{
				expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
				OpFieldQueryTemplate:      "%s+!=+%s",
				OpFieldQueryArgs:          common.Object{"unknownkey": "unknownvalue"},
				expectedStatusCode:        http.StatusBadRequest,
			},
		),
		Entry("returns 400 when left/right operands are unknown",
			listOpEntry{
				expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
				OpFieldQueryTemplate:      "%[1]s+in+[%[2]s|%[2]s|%[2]s]",
				OpFieldQueryArgs:          common.Object{"unknownkey": "unknownvalue"},
				expectedStatusCode:        http.StatusBadRequest,
			},
		),
		Entry("returns 400 when left/right operands are unknown",
			listOpEntry{
				expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
				OpFieldQueryTemplate:      "%s+in+[%s]",
				OpFieldQueryArgs:          common.Object{"unknownkey": "unknownvalue"},
				expectedStatusCode:        http.StatusBadRequest,
			},
		),
		Entry("returns 400 when left/right operands are unknown",
			listOpEntry{
				expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
				OpFieldQueryTemplate:      "%[1]s+notin+[%[2]s,%[2]s,%[2]s]",
				OpFieldQueryArgs:          common.Object{"unknownkey": "unknownvalue"},
				expectedStatusCode:        http.StatusBadRequest,
			},
		),
		Entry("returns 400 when left/right operands are unknown",
			listOpEntry{
				expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
				OpFieldQueryTemplate:      "%s+notin+[%s]",
				OpFieldQueryArgs:          common.Object{"unknownkey": "unknownvalue"},
				expectedStatusCode:        http.StatusBadRequest,
			},
		),
		Entry("returns 400 when left/right operands are unknown",
			listOpEntry{
				expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
				OpFieldQueryTemplate:      "%s+>+%s",
				OpFieldQueryArgs:          common.Object{"unknownkey": "unknownvalue"},
				expectedStatusCode:        http.StatusBadRequest,
			},
		),
		Entry("returns 400 when left/right operands are unknown",
			listOpEntry{
				expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
				OpFieldQueryTemplate:      "%s+<+%s",
				OpFieldQueryArgs:          common.Object{"unknownkey": "unknownvalue"},
				expectedStatusCode:        http.StatusBadRequest,
			},
		),
		Entry("returns 400 when left/right operands are unknown",
			listOpEntry{
				expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
				OpFieldQueryTemplate:      "%s+eqornil+%s",
				OpFieldQueryArgs:          common.Object{"unknownkey": "unknownvalue"},
				expectedStatusCode:        http.StatusBadRequest,
			},
		),
	}

	//multiplerigtargs
	//singlerightarg

	//oneTimers := Entry("returns 200", listOpEntry{
	//	expectedResourcesBeforeOp: []common.Object{r[0], r[1]},
	//	OpFieldQueryTemplate:      "",
	//	OpFieldQueryArgs:          common.Object{},
	//	expectedResourcesAfterOp:  []common.Object{r[0], r[1]},
	//	expectedStatusCode:        http.StatusOK,
	//})

	//for _, fieldName := range t.optionalFields {
	//	queryArgs := opiotnalFieldsCopyOf()
	//	// create records with nulled optional fields
	//	Entry("", listOpEntry{
	//
	//		expectedResourcesBeforeOp: func() map[string][]common.Object {
	//			return map[string][]common.Object{t.API: {r[t.API][1], r[t.API][2], r[t.API][3]}}
	//		},
	//		OpFieldQueryTemplate: "%s+eqornil+%s",
	//		OpFieldQueryArgs:     optionalsOf(r[t.API][0]),
	//		expectedResourcesAfterOp: func() map[string][]common.Object {
	//			return map[string][]common.Object{t.API: {r[t.API][0]}, nulledResource}
	//		},
	//		unexpectedResourcesAfterOp: func() map[string][]common.Object {
	//			return map[string][]common.Object{t.API: {r[t.API][1], r[t.API][2], r[t.API][3]}}
	//		},
	//		expectedStatusCode: http.StatusOK,
	//	},
	//	),
	//}
	verifyListOp := func(listOpEntry listOpEntry, query string, api string) {
		var expectedAfterOpIDs []string
		var unexpectedAfterOpIDs []string
		expectedAfterOpIDs = common.ExtractResourceIDs(listOpEntry.expectedResourcesAfterOp)
		unexpectedAfterOpIDs = common.ExtractResourceIDs(listOpEntry.unexpectedResourcesAfterOp)

		By(fmt.Sprintf("[SETUP]: Verifying expected [%s] before operation after present", api))
		beforeOpArray := ctx.SMWithOAuth.GET("/v1/" + api).
			Expect().
			Status(http.StatusOK).JSON().Object().Value(api).Array()

		for _, v := range beforeOpArray.Iter() {
			obj := v.Object().Raw()
			delete(obj, "created_at")
			delete(obj, "updated_at")
		}

		for _, entity := range listOpEntry.expectedResourcesBeforeOp {
			delete(entity, "created_at")
			delete(entity, "updated_at")
			beforeOpArray.Contains(entity)
		}

		By("[TEST]: ======= Expectations Summary =======")

		By(fmt.Sprintf("[TEST]: Listing %s with fieldquery [%s]", api, query))
		By(fmt.Sprintf("[TEST]: Currently present resources: [%v]", r))
		By(fmt.Sprintf("[TEST]: Expected %s ids after operations: [%s]", api, expectedAfterOpIDs))
		By(fmt.Sprintf("[TEST]: Unexpected %s ids after operations: [%s]", api, unexpectedAfterOpIDs))
		By(fmt.Sprintf("[TEST]: Expected status code %d", listOpEntry.expectedStatusCode))

		By("[TEST]: ====================================")

		req := ctx.SMWithOAuth.GET("/v1/" + api)
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

			array := resp.JSON().Object().Value(api).Array()
			for _, v := range array.Iter() {
				obj := v.Object().Raw()
				delete(obj, "created_at")
				delete(obj, "updated_at")
			}

			if listOpEntry.expectedResourcesAfterOp != nil {
				By(fmt.Sprintf("[TEST]: Verifying expected %s are returned after list operation", api))
				for _, entity := range listOpEntry.expectedResourcesAfterOp {
					delete(entity, "created_at")
					delete(entity, "updated_at")
					array.Contains(entity)
				}
			}

			if listOpEntry.unexpectedResourcesAfterOp != nil {
				By(fmt.Sprintf("[TEST]: Verifying unexpected %s are NOT returned after list operation", api))

				for _, entity := range listOpEntry.unexpectedResourcesAfterOp {
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
				ctx.SMWithBasic.GET("/v1/" + t.API).
					Expect().
					Status(http.StatusOK)
			})
		})

		Context("with bearer auth", func() {

			// For each entry generate len(OpFieldQueryArgs) + 1  test cases
			// - one per field of the field query args plus one with multiple field queries using all the field query args plus one with no field query at all

			Describe("with no field query", func() {
				It("returns all the resources", func() {
					verifyListOp(listOpEntry{
						expectedResourcesBeforeOp: []common.Object{r[0], r[1]},
						OpFieldQueryTemplate:      "",
						OpFieldQueryArgs:          common.Object{},
						expectedResourcesAfterOp:  []common.Object{r[0], r[1]},
						expectedStatusCode:        http.StatusOK,
					}, "", t.API)
				})
			})

			Describe("with empty field query", func() {
				It("returns 200", func() {
					verifyListOp(listOpEntry{
						expectedResourcesBeforeOp: []common.Object{r[0], r[1]},
						OpFieldQueryTemplate:      "",
						OpFieldQueryArgs:          common.Object{},
						expectedResourcesAfterOp:  []common.Object{r[0], r[1]},
						expectedStatusCode:        http.StatusOK,
					}, "fieldQuery=", t.API)
				})
			})

			for i := 0; i < len(entries); i++ {
				params := entries[i].Parameters[0].(listOpEntry)
				if params.OpFieldQueryTemplate == "" {
					panic("field query template missing")
				}

				// create tests: multiply each testEntry by the number of fields in OpFieldQueryArgs
				queryForEntry := make([]string, 0, 0)
				for key, value := range params.OpFieldQueryArgs {
					if key == "plans" {
						logrus.Error("break")
					}
					key, value := key, value
					//TODO remove when json rightval works
					if _, ok := value.(map[string]interface{}); ok {
						continue
					}
					//TODO remove when json rightval works
					if _, ok := value.([]interface{}); ok {
						continue
					}

					// TODO plans+=+null -what to do here?
					if value == nil {
						continue
					}

					if _, ok := value.(string); !ok {
						var err error
						value, err = json.Marshal(value)
						if err != nil {
							panic(err)
						}
					}
					queryValue := fmt.Sprintf(params.OpFieldQueryTemplate, key, value)
					query := "fieldQuery=" + queryValue
					// add len(OpFieldQueryArgs) tests for this entry
					DescribeTable(fmt.Sprintf("with %s", query), func(test listOpEntry) {
						verifyListOp(test, query, t.API)
					}, entries[i])
					queryForEntry = append(queryForEntry, queryValue)
				}
				queryValue := strings.Join(queryForEntry, ",")
				query := "fieldQuery=" + queryValue
				// add one more test, merging all fields in OpFieldQueryArgs into one field query
				DescribeTable(fmt.Sprintf("with multi %s", query), func(test listOpEntry) {
					verifyListOp(test, query, t.API)
				}, entries[i])
			}
		})
	})
}
