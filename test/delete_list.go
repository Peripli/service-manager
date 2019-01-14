package test

import (
	"fmt"
	"net/http"

	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo/extensions/table"
)
import . "github.com/onsi/ginkgo"

type deleteOpEntry struct {
	expectedResourcesBeforeOp func() []common.Object

	queryTemplate              string
	queryArgs                  func() common.Object
	expectedResourcesAfterOp   func() []common.Object
	unexpectedResourcesAfterOp func() []common.Object
	expectedStatusCode         int
}

func DescribeDeleteListFor(ctx *common.TestContext, t TestCase) bool {
	var r []common.Object
	rWithMandatoryFields := t.DELETELIST.ResourceWithoutNullableFieldsBlueprint(ctx)
	for i := 0; i < 2; i++ {
		gen := t.DELETELIST.ResourceBlueprint(ctx)
		delete(gen, "created_at")
		delete(gen, "updated_at")
		r = append(r, gen)
	}

	entries := []TableEntry{
		Entry("returns 200",
			deleteOpEntry{
				expectedResourcesBeforeOp:  []common.Object{r[0]},
				queryTemplate:              "%s = %v",
				unexpectedResourcesAfterOp: []common.Object{r[0]},
				expectedStatusCode:         http.StatusOK,
			},
		),
		FEntry("returns 200",
			deleteOpEntry{
				expectedResourcesBeforeOp: []common.Object{r[0], r[1]},
				queryTemplate:             "%s != %v",
				queryArgs:                 r[0],
				expectedResourcesAfterOp:  []common.Object{r[0]},
				expectedStatusCode:        http.StatusOK,
			},
		),

		Entry("returns 200",
			deleteOpEntry{
				expectedResourcesBeforeOp:  []common.Object{r[0], r[1]},
				queryTemplate:              "%[1]s in [%[2]v||%[2]v||%[2]v]",
				queryArgs:                  r[0],
				unexpectedResourcesAfterOp: []common.Object{r[0]},
				expectedStatusCode:         http.StatusOK,
			},
		),

		Entry("returns 200",
			deleteOpEntry{
				expectedResourcesBeforeOp:  []common.Object{r[0], r[1]},
				queryTemplate:              "%s in [%v]",
				queryArgs:                  r[0],
				unexpectedResourcesAfterOp: []common.Object{r[0]},
				expectedStatusCode:         http.StatusOK,
			},
		),
		Entry("returns 200",
			deleteOpEntry{
				expectedResourcesBeforeOp: []common.Object{r[0], r[1]},
				queryTemplate:             "%[1]s notin [%[2]v||%[2]v||%[2]v]",
				queryArgs:                 r[0],
				expectedResourcesAfterOp:  []common.Object{r[0]},
				expectedStatusCode:        http.StatusOK,
			},
		),
		Entry("returns 200",
			deleteOpEntry{
				expectedResourcesBeforeOp: []common.Object{r[0], r[1]},
				queryTemplate:             "%s notin [%v]",
				queryArgs:                 r[0],
				expectedResourcesAfterOp:  []common.Object{r[0]},
				expectedStatusCode:        http.StatusOK,
			},
		),
		Entry("returns 200",
			deleteOpEntry{
				expectedResourcesBeforeOp: []common.Object{r[0], r[1]},
				queryTemplate:             "%s gt %v",
				queryArgs:                 common.RmNonNumericArgs(r[0]),
				expectedResourcesAfterOp:  []common.Object{r[0]},
				expectedStatusCode:        http.StatusOK,
			},
		),
		Entry("returns 200",
			deleteOpEntry{
				expectedResourcesBeforeOp: []common.Object{r[0], r[1]},
				queryTemplate:             "%s lt %v",
				queryArgs:                 common.RmNonNumericArgs(r[0]),
				expectedResourcesAfterOp:  []common.Object{r[0]},
				expectedStatusCode:        http.StatusOK,
			},
		),
		Entry("returns 200 for field queries",
			deleteOpEntry{
				expectedResourcesBeforeOp:  []common.Object{r[0], rWithMandatoryFields},
				queryTemplate:              "%s eqornil %v",
				queryArgs:                  common.RmNotNullableFieldAndLabels(r[0], rWithMandatoryFields),
				unexpectedResourcesAfterOp: []common.Object{r[0], rWithMandatoryFields},
				expectedStatusCode:         http.StatusOK,
			},
		),
		Entry("returns 200 for JSON fields with stripped new lines",
			deleteOpEntry{
				expectedResourcesBeforeOp:  []common.Object{r[0]},
				queryTemplate:              "%s = %v",
				queryArgs:                  common.RmNonJSONArgs(r[0]),
				unexpectedResourcesAfterOp: []common.Object{r[0]},
				expectedStatusCode:         http.StatusOK,
			},
		),

		Entry("returns 400 when query operator is invalid",
			deleteOpEntry{
				queryTemplate:      "%s @@ %v",
				queryArgs:          r[0],
				expectedStatusCode: http.StatusBadRequest,
			},
		),
		Entry("returns 400 when query is duplicated",
			deleteOpEntry{
				queryTemplate:      "%[1]s = %[2]v|%[1]s = %[2]v",
				queryArgs:          r[0],
				expectedStatusCode: http.StatusBadRequest,
			},
		),
		Entry("returns 400 when operator is not properly separated with right space from operands",
			deleteOpEntry{
				queryTemplate:      "%s =%v",
				queryArgs:          r[0],
				expectedStatusCode: http.StatusBadRequest,
			},
		),
		Entry("returns 400 when operator is not properly separated with left space from operands",
			deleteOpEntry{
				queryTemplate:      "%s= %v",
				queryArgs:          r[0],
				expectedStatusCode: http.StatusBadRequest,
			},
		),

		Entry("returns 400 when field query left operands are unknown",
			deleteOpEntry{
				queryTemplate:      "%[1]s in [%[2]v||%[2]v]",
				queryArgs:          common.Object{"unknownkey": "unknownvalue"},
				expectedStatusCode: http.StatusBadRequest,
			},
		),
		Entry("returns 400 when single value operator is used with multiple right value arguments",
			deleteOpEntry{
				queryTemplate:      "%[1]s != [%[2]v||%[2]v||%[2]v]",
				queryArgs:          r[0],
				expectedStatusCode: http.StatusBadRequest,
			},
		),

		Entry("returns 400 when numeric operator is used with non-numeric operands",
			deleteOpEntry{
				queryTemplate:      "%s < %v",
				queryArgs:          common.RmNumericArgs(r[0]),
				expectedStatusCode: http.StatusBadRequest,
			},
		),
	}

	verifyDeleteListOp := func(entry deleteOpEntry) {
		// expand
		// build queries and for each run helper
	}
	verifyDeleteListOpHelper := func(listOpEntry deleteOpEntry, query string) {
		var expectedAfterOpIDs []string
		var unexpectedAfterOpIDs []string
		expectedAfterOpIDs = common.ExtractResourceIDs(listOpEntry.expectedResourcesAfterOp)
		unexpectedAfterOpIDs = common.ExtractResourceIDs(listOpEntry.unexpectedResourcesAfterOp)

		By("[TEST]: ======= Expectations Summary =======")

		By(fmt.Sprintf("[TEST]: Deleting %s with %s", t.API, query))
		By(fmt.Sprintf("[TEST]: Currently present resources: %v", r))
		By(fmt.Sprintf("[TEST]: Expected %s ids after operations: %s", t.API, expectedAfterOpIDs))
		By(fmt.Sprintf("[TEST]: Unexpected %s ids after operations: %s", t.API, unexpectedAfterOpIDs))
		By(fmt.Sprintf("[TEST]: Expected status code %d", listOpEntry.expectedStatusCode))

		By("[TEST]: ====================================")

		By(fmt.Sprintf("[TEST]: Verifying expected %s before operation after present", t.API))
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

		req := ctx.SMWithOAuth.DELETE("/v1/" + t.API)
		if query != "" {
			req = req.WithQueryString(query)
		}

		By(fmt.Sprintf("[TEST]: Verifying expected status code %d is returned from operation", listOpEntry.expectedStatusCode))
		resp := req.
			Expect().
			Status(listOpEntry.expectedStatusCode)

		if listOpEntry.expectedStatusCode != http.StatusOK {
			By(fmt.Sprintf("[TEST]: Verifying error and description fields are returned after operation"))

			resp.JSON().Object().Keys().Contains("error", "description")
		} else {

			afterOpArray := ctx.SMWithOAuth.GET("/v1/" + t.API).
				Expect().
				Status(http.StatusOK).JSON().Object().Value(t.API).Array()

			for _, v := range afterOpArray.Iter() {
				obj := v.Object().Raw()
				delete(obj, "created_at")
				delete(obj, "updated_at")
			}

			if listOpEntry.expectedResourcesAfterOp != nil {
				By(fmt.Sprintf("[TEST]: Verifying expected %s are returned after operation", t.API))
				for _, entity := range listOpEntry.expectedResourcesAfterOp {
					delete(entity, "created_at")
					delete(entity, "updated_at")
					afterOpArray.Contains(entity)
				}
			}

			if listOpEntry.unexpectedResourcesAfterOp != nil {
				By(fmt.Sprintf("[TEST]: Verifying unexpected %s are NOT returned after operation", t.API))

				for _, entity := range listOpEntry.unexpectedResourcesAfterOp {
					delete(entity, "created_at")
					delete(entity, "updated_at")
					afterOpArray.NotContains(entity)
				}
			}
		}
	}

	return Describe("DELETE LIST", func() {
		BeforeEach(func() {
			By(fmt.Sprintf("[BEFOREEACH]: Preparing and creating test resources"))

			r = make([]common.Object, 0, 0)
			ctx = common.NewTestContext(nil)
			rWithMandatoryFields = t.DELETELIST.ResourceWithoutNullableFieldsBlueprint(ctx)
			for i := 0; i < 2; i++ {
				gen := t.DELETELIST.ResourceBlueprint(ctx)
				delete(gen, "created_at")
				delete(gen, "updated_at")
				r = append(r, gen)
			}
			By(fmt.Sprintf("[BEFOREEACH]: Successfully finished preparing and creating test resources"))

		})

		AfterEach(func() {
			By(fmt.Sprintf("[AFTEREACH]: Cleaning up test resources"))
			ctx.Cleanup()
			By(fmt.Sprintf("[AFTEREACH]: Sucessfully finished cleaning up test resources"))

		})

		Context("with basic auth", func() {
			It("returns 200", func() {
				ctx.SMWithBasic.DELETE("/v1/" + t.API).
					Expect().
					Status(http.StatusUnauthorized)
			})
		})

		Context("with bearer auth", func() {
			Context("with no query", func() {
				It("deletes all the resources", func() {
					verifyDeleteListOp(deleteOpEntry{
						expectedResourcesBeforeOp:  []common.Object{r[0], r[1]},
						unexpectedResourcesAfterOp: []common.Object{r[0], r[1]},
						expectedStatusCode:         http.StatusOK,
					}, "")
				})
			})

			Context("with empty field query", func() {
				It("returns 200", func() {
					verifyDeleteListOp(deleteOpEntry{
						expectedResourcesBeforeOp:  []common.Object{r[0], r[1]},
						unexpectedResourcesAfterOp: []common.Object{r[0], r[1]},
						expectedStatusCode:         http.StatusOK,
					}, "fieldQuery=")
				})
			})

			DescribeTable("with field query", verifyDeleteListOp, entries...)
			//Context("with field query=", func() {

			//for i := 0; i < len(entries); i++ {
			//	params := entries[i].Parameters[0].(deleteOpEntry)
			//	if len(params.queryTemplate) == 0 {
			//		panic("query templates missing")
			//	}
			//	var multiQueryValue string
			//	var queryValues []string

			//fields := common.CopyObject(params.queryArgs)
			//expand in IT and move creation in beforeeach so that static describes dont depend on query
			//multiQueryValue, queryValues = expandFieldQuery(fields, params.queryTemplate)
			//fquery := "fieldQuery" + "=" + multiQueryValue

			//for _, queryValue := range queryValues {
			//	query := "fieldQuery" + "=" + queryValue
			//	DescribeTable(fmt.Sprintf("%s", queryValue), verifyDeleteListOp, entries[i])
			//}

			//if len(queryValues) > 1 {
			//	DescribeTable(fmt.Sprintf("%s", multiQueryValue),
			//		verifyDeleteListOp, entries[i])
			//}
			//}
			//})
		})
	})
}
