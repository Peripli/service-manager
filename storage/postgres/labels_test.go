/*
 * Copyright 2018 The Service Manager Authors
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */

package postgres

import (
	"fmt"
	"strings"

	"github.com/Peripli/service-manager/storage/postgres/postgresfakes"

	"github.com/jmoiron/sqlx"

	"github.com/Peripli/service-manager/pkg/query"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type dummyLabelableEntity struct {
}

func (dummyLabelableEntity) Label() (labelTableName string, referenceColumnName string, primaryColumnName string) {
	return "testLabelTable", "base_table_id", "id"
}

var _ = Describe("Postgres Translator", func() {

	var extContext sqlx.ExtContext
	var criteria []query.Criterion
	var baseQuery string

	labelableEntity := dummyLabelableEntity{}
	baseTableName := "testTable"
	labelsTableName, referenceColumnName, primaryColumnName := labelableEntity.Label()

	BeforeEach(func() {
		fakePgDB := &postgresfakes.FakePgDB{}
		fakePgDB.RebindStub = func(s string) string {
			return s
		}
		extContext = fakePgDB
		criteria = []query.Criterion{}
	})

	Describe("delete action", func() {
		BeforeEach(func() {
			baseQuery = "DELETE FROM " + baseTableName
		})
		Context("When passing no criteria", func() {
			It("Should return base query", func() {
				actualQuery, actualQueryParams, err := buildQueryWithParams(extContext, baseQuery, baseTableName, labelableEntity, deleteAction, nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(actualQuery).To(Equal(baseQuery + ";"))
				Expect(actualQueryParams).To(BeEmpty())
			})
		})

		Context("With label query", func() {
			It("Should construct correct SQL query", func() {
				leftOp := "leftOp"
				rightOp := "rightOp"
				expectedQuery := baseQuery + fmt.Sprintf(" WHERE %[1]s.%[2]s IN (SELECT %[4]s FROM %[3]s WHERE %[3]s.key = ? AND %[3]s.val = ?);", baseTableName, primaryColumnName, labelsTableName, referenceColumnName)
				criteria := []query.Criterion{query.ByLabel(query.EqualsOperator, leftOp, rightOp)}
				actualQuery, actualQueryParams, err := buildQueryWithParams(extContext, baseQuery, baseTableName, labelableEntity, deleteAction, criteria)
				Expect(err).ToNot(HaveOccurred())
				Expect(actualQuery).To(Equal(expectedQuery))
				Expect(actualQueryParams).To(ConsistOf(leftOp, rightOp))
			})
		})

		Context("With field query", func() {
			It("Should construct correct SQL query", func() {
				fieldLeftOp := "fieldLeftOp"
				fieldRightOp := "5"
				expectedQuery := baseQuery + fmt.Sprintf(" WHERE %s.%s < ?;", baseTableName, fieldLeftOp)
				criteria := []query.Criterion{
					query.ByField(query.LessThanOperator, fieldLeftOp, fieldRightOp),
				}
				actualQuery, actualQueryParams, err := buildQueryWithParams(extContext, baseQuery, baseTableName, labelableEntity, deleteAction, criteria)
				Expect(err).ToNot(HaveOccurred())
				Expect(actualQuery).To(Equal(expectedQuery))
				Expect(actualQueryParams).To(ConsistOf(fieldRightOp))
			})
		})

		Context("With both label and field query", func() {
			It("Should construct correct SQL query", func() {
				labelLeftOp := "labelLeftOp"
				labelRightOp := "labelRightOp"
				fieldLeftOp := "fieldLeftOp"
				fieldRightOp := []interface{}{"fieldRightOp1", "fieldRightOp2"}
				expectedQuery := baseQuery + fmt.Sprintf(" WHERE %[1]s.%[2]s IN (SELECT %[4]s FROM %[3]s WHERE %[3]s.key = ? AND %[3]s.val = ?)"+
					" AND %[1]s.%[5]s IN (?, ?);", baseTableName, primaryColumnName, labelsTableName, referenceColumnName, fieldLeftOp)
				criteria := []query.Criterion{
					query.ByLabel(query.EqualsOperator, labelLeftOp, labelRightOp),
					query.ByField(query.InOperator, fieldLeftOp, fieldRightOp[0].(string), fieldRightOp[1].(string)),
				}
				actualQuery, actualQueryParams, err := buildQueryWithParams(extContext, baseQuery, baseTableName, labelableEntity, deleteAction, criteria)
				Expect(err).ToNot(HaveOccurred())
				Expect(actualQuery).To(Equal(expectedQuery))
				Expect(actualQueryParams).To(ConsistOf(labelLeftOp, labelRightOp, fieldRightOp[0], fieldRightOp[1]))
			})
		})
	})

	Describe("list action", func() {
		BeforeEach(func() {
			baseQuery = constructBaseListQueryForLabelable(labelableEntity, baseTableName)
		})

		Context("No query", func() {
			It("Should return base query", func() {
				actualQuery, actualQueryParams, err := buildQueryWithParams(extContext, baseQuery, baseTableName, labelableEntity, listAction, criteria)
				Expect(err).ToNot(HaveOccurred())
				Expect(actualQuery).To(Equal(baseQuery + ";"))
				Expect(actualQueryParams).To(BeEmpty())
			})
		})

		Context("Label query", func() {
			Context("Called with valid input", func() {
				It("Should return proper result", func() {
					criteria = []query.Criterion{
						query.ByLabel(query.InOperator, "orgId", "o1", "o2", "o3"),
						query.ByLabel(query.InOperator, "clusterId", "c1", "c2"),
					}
					actualQuery, actualQueryParams, err := buildQueryWithParams(extContext, baseQuery, baseTableName, labelableEntity, listAction, criteria)
					Expect(err).ToNot(HaveOccurred())
					Expect(actualQuery).To(ContainSubstring(fmt.Sprintf(" WHERE %[1]s.key = ? AND %[1]s.val IN (?, ?, ?) AND %[1]s.key = ? AND %[1]s.val IN (?, ?)", labelsTableName)))

					expectedQueryParams := buildExpectedQueryParams(criteria)
					Expect(actualQueryParams).To(Equal(expectedQueryParams))
				})
			})

			Context("Called with multivalue operator and single value", func() {
				It("Should return proper result surrounded in brackets", func() {
					criteria = []query.Criterion{
						query.ByLabel(query.InOperator, "orgId", "o1"),
					}
					actualQuery, actualQueryParams, err := buildQueryWithParams(extContext, baseQuery, baseTableName, labelableEntity, listAction, criteria)
					Expect(err).ToNot(HaveOccurred())
					Expect(actualQuery).To(ContainSubstring(fmt.Sprintf(" WHERE %[1]s.key = ? AND %[1]s.val IN (?)", labelsTableName)))

					expectedQueryParams := buildExpectedQueryParams(criteria)
					Expect(actualQueryParams).To(Equal(expectedQueryParams))
				})
			})
		})
		Context("Field query", func() {
			Context("Called with valid input", func() {
				It("Should return proper result", func() {
					criteria = []query.Criterion{
						query.ByField(query.EqualsOperator, "platformId", "5"),
					}
					actualQuery, actualQueryParams, err := buildQueryWithParams(extContext, baseQuery, baseTableName, labelableEntity, listAction, criteria)
					Expect(err).ToNot(HaveOccurred())
					Expect(actualQuery).To(ContainSubstring(fmt.Sprintf("WHERE %s.%s %s ?;", baseTableName, criteria[0].LeftOp, strings.ToUpper(string(criteria[0].Operator)))))

					expectedQueryParams := buildExpectedQueryParams(criteria)
					Expect(actualQueryParams).To(Equal(expectedQueryParams))
				})
			})

			Context("Called with multivalue operator and single value", func() {
				It("Should return proper result surrounded in brackets", func() {
					criteria = []query.Criterion{
						query.ByField(query.InOperator, "platformId", "1"),
					}
					actualQuery, actualQueryParams, err := buildQueryWithParams(extContext, baseQuery, baseTableName, labelableEntity, listAction, criteria)
					Expect(err).ToNot(HaveOccurred())
					Expect(actualQuery).To(ContainSubstring(fmt.Sprintf(" WHERE %s.%s %s (?);", baseTableName, criteria[0].LeftOp, strings.ToUpper(string(criteria[0].Operator)))))

					expectedQueryParams := buildExpectedQueryParams(criteria)
					Expect(actualQueryParams).To(Equal(expectedQueryParams))
				})
			})
		})
	})
})

func buildExpectedQueryParams(criteria []query.Criterion) interface{} {
	var expectedQueryParams []interface{}
	for _, criterion := range criteria {
		if criterion.Type == query.LabelQuery {
			expectedQueryParams = append(expectedQueryParams, criterion.LeftOp)
		}
		for _, param := range criterion.RightOp {
			expectedQueryParams = append(expectedQueryParams, param)
		}
	}
	return expectedQueryParams
}
