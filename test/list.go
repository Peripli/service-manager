package test

import (
	"encoding/json"
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo/extensions/table"

	"net/http"

	. "github.com/onsi/ginkgo"

	"github.com/Peripli/service-manager/test/common"
)

type listOpEntry struct {
	expectedResourcesBeforeOp []common.Object

	fieldQueryTemplate string
	fieldQueryArgs     common.Object

	expectedResourcesAfterOp   []common.Object
	unexpectedResourcesAfterOp []common.Object
	expectedStatusCode         int
}

func DescribeListTestsFor(ctx *common.TestContext, t TestCase, r []common.Object, rWithMandatoryFields common.Object) bool {
	validQueryEntries := []TableEntry{
		Entry("returns 200",
			listOpEntry{
				expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
				fieldQueryTemplate:        "%s+=+%s",
				fieldQueryArgs:            r[0],
				expectedResourcesAfterOp:  []common.Object{r[0]},
				expectedStatusCode:        http.StatusOK,
			},
		),
		Entry("returns 200",
			listOpEntry{
				expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
				fieldQueryTemplate:        "%s = %s",
				fieldQueryArgs:            r[0],
				expectedResourcesAfterOp:  []common.Object{r[0]},
				expectedStatusCode:        http.StatusOK,
			},
		),
		Entry("returns 200",
			listOpEntry{
				expectedResourcesBeforeOp:  []common.Object{r[0], r[1], r[2], r[3]},
				fieldQueryTemplate:         "%s+!=+%s",
				fieldQueryArgs:             r[0],
				unexpectedResourcesAfterOp: []common.Object{r[0]},
				expectedStatusCode:         http.StatusOK,
			},
		),

		Entry("returns 200",
			listOpEntry{
				expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
				fieldQueryTemplate:        "%[1]s+in+[%[2]s|%[2]s|%[2]s]",
				fieldQueryArgs:            r[0],
				expectedResourcesAfterOp:  []common.Object{r[0]},
				expectedStatusCode:        http.StatusOK,
			},
		),

		Entry("returns 200",
			listOpEntry{
				expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
				fieldQueryTemplate:        "%s+in+[%s]",
				fieldQueryArgs:            r[0],
				expectedResourcesAfterOp:  []common.Object{r[0]},
				expectedStatusCode:        http.StatusOK,
			},
		),
		Entry("returns 200",
			listOpEntry{
				expectedResourcesBeforeOp:  []common.Object{r[0], r[1], r[2], r[3]},
				fieldQueryTemplate:         "%[1]s+notin+[%[2]s|%[2]s|%[2]s]",
				fieldQueryArgs:             r[0],
				unexpectedResourcesAfterOp: []common.Object{r[0]},
				expectedStatusCode:         http.StatusOK,
			},
		),
		Entry("returns 200",
			listOpEntry{
				expectedResourcesBeforeOp:  []common.Object{r[0], r[1], r[2], r[3]},
				fieldQueryTemplate:         "%s+notin+[%s]",
				fieldQueryArgs:             r[0],
				unexpectedResourcesAfterOp: []common.Object{r[0]},
				expectedStatusCode:         http.StatusOK,
			},
		),
		Entry("returns 200",
			listOpEntry{
				expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
				fieldQueryTemplate:        "%s+>+%s",
				fieldQueryArgs:            common.RmNonNumbericFieldNames(common.CopyObject(r[0])),
				expectedResourcesAfterOp:  []common.Object{r[0]},
				expectedStatusCode:        http.StatusOK,
			},
		),
		Entry("returns 200",
			listOpEntry{
				expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
				fieldQueryTemplate:        "%s+<+%s",
				fieldQueryArgs:            common.RmNonNumbericFieldNames(common.CopyObject(r[0])),
				expectedResourcesAfterOp:  []common.Object{r[0]},
				expectedStatusCode:        http.StatusOK,
			},
		),
		Entry("returns 200",
			listOpEntry{
				expectedResourcesBeforeOp: []common.Object{r[0], rWithMandatoryFields},
				fieldQueryTemplate:        "%s+eqornil+%s",
				fieldQueryArgs:            common.RmNotNullableFieldNames(r[0], t.LIST.NullableFields),
				expectedResourcesAfterOp:  []common.Object{r[0], rWithMandatoryFields},
				expectedStatusCode:        http.StatusOK,
			},
		),
		Entry("returns 200 for JSON fields with stripped new lines",
			listOpEntry{
				expectedResourcesBeforeOp: []common.Object{r[0]},
				fieldQueryTemplate:        "%s+=+%s",
				fieldQueryArgs:            common.RmNonJSONFieldNames(common.CopyObject(r[0])),
				expectedResourcesAfterOp:  []common.Object{r[0]},
				expectedStatusCode:        http.StatusOK,
			},
		),

		Entry("returns 400 when field query operator is invalid",
			listOpEntry{
				fieldQueryTemplate: "@@",
				expectedStatusCode: http.StatusBadRequest,
			},
		),
		//Entry("returns 400 when field query operator is missing",
		//	listOpEntry{
		//		fieldQueryTemplate: "",
		//		fieldQueryArgs:     nil,
		//		expectedStatusCode: http.StatusBadRequest,
		//	},
		//),
		Entry("returns 400 when field query is duplicated",
			listOpEntry{
				fieldQueryTemplate: "%[1]s+=+%[2]s|%[1]s+=+%[2]s",
				fieldQueryArgs:     r[0],
				expectedStatusCode: http.StatusBadRequest,
			}),
		Entry("returns 400 when operator is not properly separated with + from operands",
			listOpEntry{
				fieldQueryTemplate: "%s+=%s",
				fieldQueryArgs:     r[0],
				expectedStatusCode: http.StatusBadRequest,
			},
		),
		Entry("returns 400 when operator is not properly separated with spaces from operands",
			listOpEntry{
				fieldQueryTemplate: "%s= %s",
				fieldQueryArgs:     r[0],
				expectedStatusCode: http.StatusBadRequest,
			},
		),
		Entry("returns 400 when left operands are unknown",
			listOpEntry{
				fieldQueryTemplate: "%[1]s+in+[%[2]s,%[2]s]",
				fieldQueryArgs:     common.Object{"unknownkey": "unknownvalue"},
				expectedStatusCode: http.StatusBadRequest,
			},
		),

		Entry("returns 400 when single value operator is used with multiple right value arguments",
			listOpEntry{
				fieldQueryTemplate: "%[1]s+!=+[%[2]s|%[2]s|%[2]s]",
				fieldQueryArgs:     r[0],
				expectedStatusCode: http.StatusBadRequest,
			},
		),

		Entry("returns 400 if brackets are missing",
			listOpEntry{
				fieldQueryTemplate: "%[1]s+in+%[2]s,%[2]s",
				fieldQueryArgs:     r[0],
				expectedStatusCode: http.StatusBadRequest,
			},
		),

		Entry("returns 400 when numeric operator is used with non-numeric operands",
			listOpEntry{
				fieldQueryTemplate: "%s+<+%s",
				fieldQueryArgs:     common.RmNumbericFieldNames(common.CopyObject(r[0])),
				expectedStatusCode: http.StatusBadRequest,
			},
		),

		Entry("returns 400 when right operand is json with empty lines",
			listOpEntry{
				fieldQueryTemplate: "%[1]s+in+[%[2]s,%[2]s]",
				fieldQueryArgs:     common.RmNonJSONFieldNames(common.CopyObject(r[0])),
				expectedStatusCode: http.StatusBadRequest,
			},
		),
	}

	verifyValidListOpWithValidQuery := func(listOpEntry listOpEntry, query string) {
		var expectedAfterOpIDs []string
		var unexpectedAfterOpIDs []string
		expectedAfterOpIDs = common.ExtractResourceIDs(listOpEntry.expectedResourcesAfterOp)
		unexpectedAfterOpIDs = common.ExtractResourceIDs(listOpEntry.unexpectedResourcesAfterOp)

		By(fmt.Sprintf("[SETUP]: Verifying expected [%s] before operation after present", t.API))
		beforeOpArray := ctx.SMWithOAuth.GET("/v1/" + t.API).
			Expect().
			Status(http.StatusOK).JSON().Object().Value(t.API).Array()

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

		By(fmt.Sprintf("[TEST]: Listing %s with fieldquery [%s]", t.API, query))
		By(fmt.Sprintf("[TEST]: Currently present resources: [%v]", r))
		By(fmt.Sprintf("[TEST]: Expected %s ids after operations: [%s]", t.API, expectedAfterOpIDs))
		By(fmt.Sprintf("[TEST]: Unexpected %s ids after operations: [%s]", t.API, unexpectedAfterOpIDs))
		By(fmt.Sprintf("[TEST]: Expected status code %d", listOpEntry.expectedStatusCode))

		By("[TEST]: ====================================")

		req := ctx.SMWithOAuth.GET("/v1/" + t.API)
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

			array := resp.JSON().Object().Value(t.API).Array()
			for _, v := range array.Iter() {
				obj := v.Object().Raw()
				delete(obj, "created_at")
				delete(obj, "updated_at")
			}

			if listOpEntry.expectedResourcesAfterOp != nil {
				By(fmt.Sprintf("[TEST]: Verifying expected %s are returned after list operation", t.API))
				for _, entity := range listOpEntry.expectedResourcesAfterOp {
					delete(entity, "created_at")
					delete(entity, "updated_at")
					array.Contains(entity)
				}
			}

			if listOpEntry.unexpectedResourcesAfterOp != nil {
				By(fmt.Sprintf("[TEST]: Verifying unexpected %s are NOT returned after list operation", t.API))

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
					verifyValidListOpWithValidQuery(listOpEntry{
						expectedResourcesBeforeOp: []common.Object{r[0], r[1]},
						fieldQueryTemplate:        "",
						expectedResourcesAfterOp:  []common.Object{r[0], r[1]},
						expectedStatusCode:        http.StatusOK,
					}, "")
				})
			})

			Describe("with empty field query", func() {
				It("returns 200", func() {
					verifyValidListOpWithValidQuery(listOpEntry{
						expectedResourcesBeforeOp: []common.Object{r[0], r[1]},
						fieldQueryTemplate:        "",
						expectedResourcesAfterOp:  []common.Object{r[0], r[1]},
						expectedStatusCode:        http.StatusOK,
					}, "fieldQuery=")
				})
			})

			for i := 0; i < len(validQueryEntries); i++ {
				params := validQueryEntries[i].Parameters[0].(listOpEntry)
				if params.fieldQueryTemplate == "" {
					panic("field query template missing")
				}

				// create tests: multiply each testEntry by the number of fields in OpFieldQueryArgs
				queryForEntry := make([]string, 0, 0)
				for key, value := range params.fieldQueryArgs {
					key, value := key, value
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
					queryValue := fmt.Sprintf(params.fieldQueryTemplate, key, value)
					query := "fieldQuery=" + queryValue
					// add len(OpFieldQueryArgs) tests for this entry
					DescribeTable(fmt.Sprintf("with %s", query), func(test listOpEntry) {
						verifyValidListOpWithValidQuery(test, query)
					}, validQueryEntries[i])
					queryForEntry = append(queryForEntry, queryValue)
				}
				queryValue := strings.Join(queryForEntry, ",")
				query := "fieldQuery=" + queryValue
				// add one more test, merging all fields in OpFieldQueryArgs into one field query
				DescribeTable(fmt.Sprintf("with multi %s", query), func(test listOpEntry) {
					verifyValidListOpWithValidQuery(test, query)
				}, validQueryEntries[i])
			}
		})
	})
}
