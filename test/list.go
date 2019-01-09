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
	expectedResourcesBeforeOp []common.Object

	FieldQueryOp   query.Operator
	FieldQueryArgs common.Object

	expectedResourcesAfterOp   []common.Object
	unexpectedResourcesAfterOp []common.Object
	expectedStatusCode         int
}

//TODO bad requests in loop, valid queries without (entry per op)
//TOdO only operator and no templates ?
//TODO stariq variant samo 4e bez da se podava queryargs obekt? a da se pozvat ot blueprint.keyset (imame i optionalfields)
func DescribeListTestsFor(ctx *common.TestContext, t TestCase, r []common.Object, rWithMandatoryFields common.Object) bool {

	//	// nezavisimo ot operatora
	//	operatorIndependantInvalidQueryEntries := []TableEntry{
	//		Entry("returns 400 when field query operator is invalid",
	//			listOpEntry{
	//				FieldQueryOp:       "@@",
	//				expectedStatusCode: http.StatusBadRequest,
	//			},
	//		),
	//		Entry("returns 400 when field query operator is missing",
	//			listOpEntry{
	//				//TODO make it operator? make a list of operators in productive code?
	//				// use the methods defined on opeartor that determine its type
	//				FieldQueryOp:       "",
	//				expectedStatusCode: http.StatusBadRequest,
	//			},
	//		),
	//	}
	//
	//	// za vseki operator - tuk multivalue trqbva da si promenqt dqsnoto... rightop tr se smqta na bazata
	//	// na operatora
	//	commonInvalidQueryEntries := []TableEntry{
	//		Entry("returns 400 when field query is duplicated", listOpEntry{
	//			expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
	//			FieldQueryOp:              "%[1]s+%s+%[2]s|%[1]s+%s+%[2]s",
	//			OpFieldQueryArgs:          r[0],
	//			expectedStatusCode:        http.StatusBadRequest,
	//		}),
	//		Entry("returns 400 when operator is not properly separated with + from operands",
	//			listOpEntry{
	//				expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
	//				FieldQueryOp:              "%s+%s%s",
	//				OpFieldQueryArgs:          r[0],
	//				expectedStatusCode:        http.StatusBadRequest,
	//			},
	//		),
	//		Entry("returns 400 when operator is not properly separated with spaces from operands",
	//			listOpEntry{
	//				expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
	//				FieldQueryOp:              "%s%s %s",
	//				OpFieldQueryArgs:          r[0],
	//				expectedStatusCode:        http.StatusBadRequest,
	//			},
	//		),
	//		Entry("returns 400 when left operands are unknown",
	//			listOpEntry{
	//				expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
	//				FieldQueryOp:              "%[1]s+%s+[%[2]s,%[2]s]",
	//				OpFieldQueryArgs:          common.Object{"unknownkey": "unknownvalue"},
	//				expectedStatusCode:        http.StatusBadRequest,
	//			},
	//		),
	//	}
	//
	//	// za vseki op koito ne e multivalue
	//	singleValueOpInvalidQueryEntries := []TableEntry{
	//		Entry("returns 400 when single value operator is used with multiple right value arguments",
	//			listOpEntry{
	//				expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
	//				FieldQueryOp:              "%[1]s+%s+[%[2]s|%[2]s|%[2]s]",
	//				OpFieldQueryArgs:          r[0],
	//				expectedStatusCode:        http.StatusBadRequest,
	//			},
	//		),
	//	}
	//	// za vseki op koito e multivalue
	//	multiValueOpInvalidQueryEntries := []TableEntry{
	//		// in operator multiple right arguments without brackets
	//		Entry("returns 400 if brackets are missing",
	//			listOpEntry{
	//				expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
	//				FieldQueryOp:              "%[1]s+%s+%[2]s,%[2]s",
	//				OpFieldQueryArgs:          r[0],
	//				expectedResourcesAfterOp:  []common.Object{r[0]},
	//				expectedStatusCode:        http.StatusBadRequest,
	//			},
	//		),
	//	}
	//
	//	// za vseki koito e numeric
	//	numericOpInvalidQueryEntries := []TableEntry{
	//		Entry("returns 400 when numeric operator is used with non-numeric operands",
	//			listOpEntry{
	//				expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
	//				FieldQueryOp:              "%s+<+%s",
	//				//OpFieldQueryArgs:          common.RmNumbericFieldNames(common.CopyObject(r[0])),
	//				expectedStatusCode: http.StatusBadRequest,
	//			},
	//		),
	//	}
	//
	//	nonNumericOpInvalidQueryEntries := []TableEntry{
	//		Entry("returns 400 when right operand is json with empty lines",
	//			listOpEntry{
	//				expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
	//				OpFieldQueryArgs: common.Object{"unknownkey": `{
	//}`},
	//				expectedStatusCode: http.StatusBadRequest,
	//			},
	//		),
	//	}

	validQueryEntries := []TableEntry{
		Entry("returns 200",
			listOpEntry{
				expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
				FieldQueryOp:              query.EqualsOperator,
				FieldQueryArgs:            r[0],
				expectedResourcesAfterOp:  []common.Object{r[0]},
				expectedStatusCode:        http.StatusOK,
			},
		),
		Entry("returns 200",
			listOpEntry{
				expectedResourcesBeforeOp:  []common.Object{r[0], r[1], r[2], r[3]},
				FieldQueryOp:               query.NotEqualsOperator,
				FieldQueryArgs:             r[0],
				unexpectedResourcesAfterOp: []common.Object{r[0]},
				expectedStatusCode:         http.StatusOK,
			},
		),

		Entry("returns 200",
			listOpEntry{
				expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
				//FieldQueryOp:      "%[1]s+in+[%[2]s|%[2]s|%[2]s]",
				FieldQueryOp:             query.InOperator,
				FieldQueryArgs:           r[0],
				expectedResourcesAfterOp: []common.Object{r[0]},
				expectedStatusCode:       http.StatusOK,
			},
		),

		//Entry("returns 200",
		//	listOpEntry{
		//		expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
		//		FieldQueryOp:      "%s+in+[%s]",
		//		OpFieldQueryArgs:          r[0],
		//		expectedResourcesAfterOp:  []common.Object{r[0]},
		//		expectedStatusCode:        http.StatusOK,
		//	},
		//),
		Entry("returns 200",
			listOpEntry{
				expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
				FieldQueryOp:              query.NotInOperator,
				//FieldQueryOp:       "%[1]s+notin+[%[2]s|%[2]s|%[2]s]",
				FieldQueryArgs:             r[0],
				unexpectedResourcesAfterOp: []common.Object{r[0]},
				expectedStatusCode:         http.StatusOK,
			},
		),
		//
		//Entry("returns 200",
		//	listOpEntry{
		//		expectedResourcesBeforeOp:  []common.Object{r[0], r[1], r[2], r[3]},
		//		FieldQueryOp:       "%s+notin+[%s]",
		//		OpFieldQueryArgs:           r[0],
		//		unexpectedResourcesAfterOp: []common.Object{r[0]},
		//		expectedStatusCode:         http.StatusOK,
		//	},
		//),
		Entry("returns 200",
			listOpEntry{
				expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
				//FieldQueryOp:      "%s+>+%s",
				FieldQueryOp:             query.GreaterThanOperator,
				FieldQueryArgs:           common.RmNonNumbericFieldNames(common.CopyObject(r[0])),
				expectedResourcesAfterOp: []common.Object{r[0]},
				expectedStatusCode:       http.StatusOK,
			},
		),
		Entry("returns 200",
			listOpEntry{
				expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
				//FieldQueryOp:      "%s+<+%s",
				FieldQueryOp:             query.LessThanOperator,
				FieldQueryArgs:           common.RmNonNumbericFieldNames(common.CopyObject(r[0])),
				expectedResourcesAfterOp: []common.Object{r[0]},
				expectedStatusCode:       http.StatusOK,
			},
		),

		Entry("returns 200",
			listOpEntry{
				expectedResourcesBeforeOp: []common.Object{r[0], rWithMandatoryFields},
				//FieldQueryOp:      "%s+eqornil+%s",
				FieldQueryOp:             query.EqualsOrNilOperator,
				FieldQueryArgs:           common.RmNotNullableFieldNames(r[0], t.LIST.NullableFields),
				expectedResourcesAfterOp: []common.Object{r[0], rWithMandatoryFields},
				expectedStatusCode:       http.StatusOK,
			},
		),
	}
	//
	//entries := []TableEntry{
	//	// invalid operator
	//	Entry("returns 400 when field query operator is invalid",
	//		listOpEntry{
	//			expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
	//			FieldQueryOp:              "%s+@@@+%s",
	//			OpFieldQueryArgs:          r[0],
	//			expectedStatusCode:        http.StatusBadRequest,
	//		},
	//	),
	//
	//	// missing operator
	//	Entry("returns 400 when field query operator is missing",
	//		listOpEntry{
	//			expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
	//			//TODO make it operator? make a list of operators in productive code?
	//			// use the methods defined on opeartor that determine its type
	//			FieldQueryOp:       "%s++%s",
	//			OpFieldQueryArgs:   r[0],
	//			expectedStatusCode: http.StatusBadRequest,
	//		},
	//	),
	//
	//	// some created resources, valid operators and field query right operands that match some resources
	//
	//	// one time validate that spaces instead of + works
	//	Entry("returns 200 when spaces are used",
	//		listOpEntry{
	//			expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
	//			FieldQueryOp:              "%s = %s",
	//			OpFieldQueryArgs:          r[0],
	//			expectedResourcesAfterOp:  []common.Object{r[0]},
	//			expectedStatusCode:        http.StatusOK,
	//		},
	//	),
	//
	//	// one time duplicated query
	//	Entry("returns 400 when field query is duplicated", listOpEntry{
	//		expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
	//		FieldQueryOp:              "%[1]s+=+%[2]s|%[1]s+=+%[2]s",
	//		OpFieldQueryArgs:          r[0],
	//		expectedStatusCode:        http.StatusBadRequest,
	//	}),
	//	// one time single value operator with multiple right values
	//	Entry("returns 400 when single value operator is used with multiple right value arguments",
	//		listOpEntry{
	//			expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
	//			FieldQueryOp:              "%[1]s+=+[%[2]s|%[2]s|%[2]s]",
	//			OpFieldQueryArgs:          r[0],
	//			expectedStatusCode:        http.StatusBadRequest,
	//		},
	//	),
	//
	//	// one time invalid +
	//	Entry("returns 400 when operator is not properly separated with + from operands",
	//		listOpEntry{
	//			expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
	//			FieldQueryOp:              "%s+=%s",
	//			OpFieldQueryArgs:          r[0],
	//			expectedStatusCode:        http.StatusBadRequest,
	//		},
	//	),
	//
	//	// one time invalid spacing
	//	Entry("returns 400 when operator is not properly separated with space from operands",
	//		listOpEntry{
	//			expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
	//			FieldQueryOp:              "%s =%s",
	//			OpFieldQueryArgs:          r[0],
	//			expectedStatusCode:        http.StatusBadRequest,
	//		},
	//	),
	//
	//	Entry("returns 200",
	//		listOpEntry{
	//			expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
	//			FieldQueryOp:              "%s+=+%s",
	//			OpFieldQueryArgs:          r[0],
	//			expectedResourcesAfterOp:  []common.Object{r[0]},
	//			expectedStatusCode:        http.StatusOK,
	//		},
	//	),
	//
	//	Entry("returns 200",
	//		listOpEntry{
	//			expectedResourcesBeforeOp:  []common.Object{r[0], r[1], r[2], r[3]},
	//			FieldQueryOp:               "%s+!=+%s",
	//			OpFieldQueryArgs:           r[0],
	//			unexpectedResourcesAfterOp: []common.Object{r[0]},
	//			expectedStatusCode:         http.StatusOK,
	//		},
	//	),
	//
	//	// in operator multiple right arguments
	//	Entry("returns 200",
	//		listOpEntry{
	//			expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
	//			FieldQueryOp:              "%[1]s+in+[%[2]s|%[2]s|%[2]s]",
	//			OpFieldQueryArgs:          r[0],
	//			expectedResourcesAfterOp:  []common.Object{r[0]},
	//			expectedStatusCode:        http.StatusOK,
	//		},
	//	),
	//
	//	// in operator single right argument
	//	Entry("returns 200",
	//		listOpEntry{
	//			expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
	//			FieldQueryOp:              "%s+in+[%s]",
	//			OpFieldQueryArgs:          r[0],
	//			expectedResourcesAfterOp:  []common.Object{r[0]},
	//			expectedStatusCode:        http.StatusOK,
	//		},
	//	),
	//
	//	//TODO bug
	//	// in operator multiple right arguments without brackets
	//	Entry("returns 400",
	//		listOpEntry{
	//			expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
	//			FieldQueryOp:              "%[1]s+in+%[2]s,%[2]s",
	//			OpFieldQueryArgs:          r[0],
	//			expectedResourcesAfterOp:  []common.Object{r[0]},
	//			expectedStatusCode:        http.StatusBadRequest,
	//		},
	//	),
	//	// not in operator multiple right arguments
	//	Entry("returns 200",
	//		listOpEntry{
	//			expectedResourcesBeforeOp:  []common.Object{r[0], r[1], r[2], r[3]},
	//			FieldQueryOp:               "%[1]s+notin+[%[2]s|%[2]s|%[2]s]",
	//			OpFieldQueryArgs:           r[0],
	//			unexpectedResourcesAfterOp: []common.Object{r[0]},
	//			expectedStatusCode:         http.StatusOK,
	//		},
	//	),
	//
	//	// not in operator single right argument
	//	Entry("returns 200",
	//		listOpEntry{
	//			expectedResourcesBeforeOp:  []common.Object{r[0], r[1], r[2], r[3]},
	//			FieldQueryOp:               "%s+notin+[%s]",
	//			OpFieldQueryArgs:           r[0],
	//			unexpectedResourcesAfterOp: []common.Object{r[0]},
	//			expectedStatusCode:         http.StatusOK,
	//		},
	//	),
	//
	//	//TODO bug
	//	// not in operator multiple right arguments without brackets
	//	Entry("returns 400",
	//		listOpEntry{
	//			expectedResourcesBeforeOp:  []common.Object{r[0], r[1], r[2], r[3]},
	//			FieldQueryOp:               "%[1]s+notin+%[2]s,%[2]s,%[2]s",
	//			OpFieldQueryArgs:           r[0],
	//			unexpectedResourcesAfterOp: []common.Object{r[0]},
	//			expectedStatusCode:         http.StatusBadRequest,
	//		},
	//	),
	//
	//	Entry("returns 400 when numeric operator is used with non-numeric operands",
	//		listOpEntry{
	//			expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
	//			FieldQueryOp:              "%s+gt+%s",
	//			//OpFieldQueryArgs:          common.RmNumbericFieldNames(common.CopyObject(r[0])),
	//			expectedStatusCode: http.StatusBadRequest,
	//		},
	//	),
	//	Entry("returns 200",
	//		listOpEntry{
	//			expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
	//			FieldQueryOp:              "%s+gt+%s",
	//			//OpFieldQueryArgs:          common.RmNonNumbericFieldNames(common.CopyObject(r[0])),
	//			expectedResourcesAfterOp: []common.Object{r[0]},
	//			expectedStatusCode:       http.StatusOK,
	//		},
	//	),
	//	Entry("returns 400",
	//		listOpEntry{
	//			expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
	//			FieldQueryOp:              "%s+lt+%s",
	//			//OpFieldQueryArgs:          common.RmNumbericFieldNames(common.CopyObject(r[0])),
	//			expectedStatusCode: http.StatusBadRequest,
	//		},
	//	),
	//	Entry("returns 200",
	//		listOpEntry{
	//			expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
	//			FieldQueryOp:              "%s+lt+%s",
	//			//OpFieldQueryArgs:          common.RmNonNumbericFieldNames(common.CopyObject(r[0])),
	//			expectedResourcesAfterOp: []common.Object{r[0]},
	//			expectedStatusCode:       http.StatusOK,
	//		},
	//	),
	//
	//	//TODO mandatory fields resource
	//	//Entry("returns 200",
	//	//	listOpEntry{
	//	//		expectedResourcesBeforeOp: func() []common.Object {
	//	//			return []common.Object{r[0], mandatoryFieldsResource}
	//	//		},
	//	//		FieldQueryOp: "%s+eqornil+%s",
	//	//		OpFieldQueryArgs:     func() common.Object { return rmMandatoryFields(t.optionalFields, r[0]) },
	//	//		expectedResourcesAfterOp: func() []common.Object {
	//	//			return []common.Object{r[0], mandatoryFieldsResource}
	//	//		},
	//	//		expectedStatusCode: http.StatusOK,
	//	//	},
	//	//),
	//	// invalid field query left/right operand
	//	Entry("returns 400 when left/right operands are unknown",
	//		listOpEntry{
	//			expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
	//			FieldQueryOp:              "%s+=+%s",
	//			OpFieldQueryArgs:          common.Object{"unknownkey": "unknownvalue"},
	//			expectedStatusCode:        http.StatusBadRequest,
	//		},
	//	),
	//	Entry("returns 400 when left/right operands are unknown",
	//		listOpEntry{
	//			expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
	//			FieldQueryOp:              "%s+!=+%s",
	//			OpFieldQueryArgs:          common.Object{"unknownkey": "unknownvalue"},
	//			expectedStatusCode:        http.StatusBadRequest,
	//		},
	//	),
	//	Entry("returns 400 when left/right operands are unknown",
	//		listOpEntry{
	//			expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
	//			FieldQueryOp:              "%[1]s+in+[%[2]s|%[2]s|%[2]s]",
	//			OpFieldQueryArgs:          common.Object{"unknownkey": "unknownvalue"},
	//			expectedStatusCode:        http.StatusBadRequest,
	//		},
	//	),
	//	Entry("returns 400 when left/right operands are unknown",
	//		listOpEntry{
	//			expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
	//			FieldQueryOp:              "%s+in+[%s]",
	//			OpFieldQueryArgs:          common.Object{"unknownkey": "unknownvalue"},
	//			expectedStatusCode:        http.StatusBadRequest,
	//		},
	//	),
	//	Entry("returns 400 when left/right operands are unknown",
	//		listOpEntry{
	//			expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
	//			FieldQueryOp:              "%[1]s+notin+[%[2]s,%[2]s,%[2]s]",
	//			OpFieldQueryArgs:          common.Object{"unknownkey": "unknownvalue"},
	//			expectedStatusCode:        http.StatusBadRequest,
	//		},
	//	),
	//	Entry("returns 400 when left/right operands are unknown",
	//		listOpEntry{
	//			expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
	//			FieldQueryOp:              "%s+notin+[%s]",
	//			OpFieldQueryArgs:          common.Object{"unknownkey": "unknownvalue"},
	//			expectedStatusCode:        http.StatusBadRequest,
	//		},
	//	),
	//	Entry("returns 400 when left/right operands are unknown",
	//		listOpEntry{
	//			expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
	//			FieldQueryOp:              "%s+>+%s",
	//			OpFieldQueryArgs:          common.Object{"unknownkey": "unknownvalue"},
	//			expectedStatusCode:        http.StatusBadRequest,
	//		},
	//	),
	//	Entry("returns 400 when left/right operands are unknown",
	//		listOpEntry{
	//			expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
	//			FieldQueryOp:              "%s+<+%s",
	//			OpFieldQueryArgs:          common.Object{"unknownkey": "unknownvalue"},
	//			expectedStatusCode:        http.StatusBadRequest,
	//		},
	//	),
	//	Entry("returns 400 when left/right operands are unknown",
	//		listOpEntry{
	//			expectedResourcesBeforeOp: []common.Object{r[0], r[1], r[2], r[3]},
	//			FieldQueryOp:              "%s+eqornil+%s",
	//			OpFieldQueryArgs:          common.Object{"unknownkey": "unknownvalue"},
	//			expectedStatusCode:        http.StatusBadRequest,
	//		},
	//	),
	//}
	//
	////multiplerigtargs
	//singlerightarg

	//oneTimers := Entry("returns 200", listOpEntry{
	//	expectedResourcesBeforeOp: []common.Object{r[0], r[1]},
	//	FieldQueryOp:      "",
	//	OpFieldQueryArgs:          common.Object{},
	//	expectedResourcesAfterOp:  []common.Object{r[0], r[1]},
	//	expectedStatusCode:        http.StatusOK,
	//})

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
						FieldQueryOp:              "",
						expectedResourcesAfterOp:  []common.Object{r[0], r[1]},
						expectedStatusCode:        http.StatusOK,
					}, "", t.API)
				})
			})

			Describe("with empty field query", func() {
				It("returns 200", func() {
					verifyListOp(listOpEntry{
						expectedResourcesBeforeOp: []common.Object{r[0], r[1]},
						FieldQueryOp:              "",
						expectedResourcesAfterOp:  []common.Object{r[0], r[1]},
						expectedStatusCode:        http.StatusOK,
					}, "fieldQuery=", t.API)
				})
			})

			Context("with numeric operators", func() {

			})

			Context("with nullable operators", func() {

			})

			Context("with multi value operators", func() {

			})

			for k, operator := range query.Operators {
				switch {
				case operator.IsNullable():

				case operator.IsNumeric():

				case !operator.IsMultiVariate():

				case operator.IsMultiVariate():
				}
			}
			// za vseki operator
			// switch {
			// case nullable
			//fallthrough
			// case multivar

			// case numeric

			// }
			for i := 0; i < len(entries); i++ {
				params := entries[i].Parameters[0].(listOpEntry)
				if params.FieldQueryOp == "" {
					panic("field query template missing")
				}

				// create tests: multiply each testEntry by the number of fields in OpFieldQueryArgs
				queryForEntry := make([]string, 0, 0)
				for key, value := range t.LIST.Fields {
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
					queryValue := fmt.Sprintf(params.FieldQueryOp, key, value)
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
