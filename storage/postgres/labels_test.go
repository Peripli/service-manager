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
	BeforeEach(func() {
		fakePgDB := &postgresfakes.FakePgDB{}
		fakePgDB.RebindStub = func(s string) string {
			return s
		}
		extContext = fakePgDB
	})

	Describe("translate list", func() {

		labelableEntity := dummyLabelableEntity{}
		baseTableName := "testTable"
		labelsTableName, _, _ := labelableEntity.Label()
		baseQuery := constructBaseQueryForLabelable(labelableEntity, baseTableName)
		var criteria []query.Criterion

		Context("No query", func() {
			It("Should return base query", func() {
				actualQuery, actualQueryParams, err := buildQueryWithParams(extContext, baseQuery, baseTableName, labelableEntity, criteria)
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
					actualQuery, actualQueryParams, err := buildQueryWithParams(extContext, baseQuery, baseTableName, labelableEntity, criteria)
					Expect(err).ToNot(HaveOccurred())
					Expect(actualQuery).To(ContainSubstring(fmt.Sprintf(" WHERE (%[1]s.key = ? AND %[1]s.val IN (?, ?, ?)) OR (%[1]s.key = ? AND %[1]s.val IN (?, ?))", labelsTableName)))

					expectedQueryParams := buildExpectedQueryParams(criteria)
					Expect(actualQueryParams).To(Equal(expectedQueryParams))
				})
			})

			Context("Called with multivalue operator and single value", func() {
				It("Should return proper result surrounded in brackets", func() {
					criteria = []query.Criterion{
						query.ByLabel(query.InOperator, "orgId", "o1"),
					}
					actualQuery, actualQueryParams, err := buildQueryWithParams(extContext, baseQuery, baseTableName, labelableEntity, criteria)
					Expect(err).ToNot(HaveOccurred())
					Expect(actualQuery).To(ContainSubstring(fmt.Sprintf(" WHERE (%[1]s.key = ? AND %[1]s.val IN (?))", labelsTableName)))

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
					actualQuery, actualQueryParams, err := buildQueryWithParams(extContext, baseQuery, baseTableName, labelableEntity, criteria)
					Expect(err).ToNot(HaveOccurred())
					Expect(actualQuery).To(ContainSubstring(fmt.Sprintf("WHERE %s.%s::text %s ?;", baseTableName, criteria[0].LeftOp, strings.ToUpper(string(criteria[0].Operator)))))

					expectedQueryParams := buildExpectedQueryParams(criteria)
					Expect(actualQueryParams).To(Equal(expectedQueryParams))
				})
			})

			Context("Called with multivalue operator and single value", func() {
				It("Should return proper result surrounded in brackets", func() {
					criteria = []query.Criterion{
						query.ByField(query.InOperator, "platformId", "1"),
					}
					actualQuery, actualQueryParams, err := buildQueryWithParams(extContext, baseQuery, baseTableName, labelableEntity, criteria)
					Expect(err).ToNot(HaveOccurred())
					Expect(actualQuery).To(ContainSubstring(fmt.Sprintf(" WHERE %s.%s::text %s (?);", baseTableName, criteria[0].LeftOp, strings.ToUpper(string(criteria[0].Operator)))))

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
