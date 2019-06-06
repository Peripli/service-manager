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

package query_test

import (
	"context"
	"fmt"
	"strings"

	. "github.com/Peripli/service-manager/pkg/query"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Selection", func() {

	var ctx context.Context
	var validCriterion Criterion

	BeforeEach(func() {
		ctx = context.TODO()
		validCriterion = ByField(EqualsOperator, "left", "right")
	})

	Describe("Add criteria to context", func() {
		addInvalidCriterion := func(criterion Criterion) {
			ctx, err := AddCriteria(ctx, criterion)
			Expect(err).To(HaveOccurred())
			Expect(ctx).To(BeNil())
		}
		Context("Invalid", func() {
			Specify("Univariate operator with multiple right operands", func() {
				addInvalidCriterion(ByField(EqualsOperator, "leftOp", "1", "2"))
			})
			Specify("IsNullable operator applied to label query", func() {
				addInvalidCriterion(ByLabel(EqualsOrNilOperator, "leftOp", "1"))
			})
			Specify("Numeric operator to non-numeric right operand", func() {
				addInvalidCriterion(ByField(GreaterThanOperator, "leftOp", "non-numeric"))
				addInvalidCriterion(ByField(GreaterThanOrEqualOperator, "leftOp", "non-numeric"))
				addInvalidCriterion(ByField(LessThanOperator, "leftOp", "non-numeric"))
				addInvalidCriterion(ByField(LessThanOrEqualOperator, "leftOp", "non-numeric"))
			})
			Specify("Right operand containing new line", func() {
				addInvalidCriterion(ByField(EqualsOperator, "leftOp", `value with 
new line`))
			})
			Specify("Left operand with query separator", func() {
				addInvalidCriterion(ByField(EqualsOperator, "leftop and more", "value"))
			})
			Specify("Field query with duplicate key", func() {
				var err error
				ctx, err = AddCriteria(ctx, validCriterion)
				Expect(err).ToNot(HaveOccurred())
				addInvalidCriterion(ByField(EqualsOrNilOperator, validCriterion.LeftOp, "right op"))
			})
		})

		Context("Valid", func() {
			Specify("Multivariate operator with single right operand", func() {
				_, err := AddCriteria(ctx, ByField(InOperator, "leftOp", "1"))
				Expect(err).ToNot(HaveOccurred())
			})
			Specify("With numeric right operand", func() {
				_, err := AddCriteria(ctx, ByField(LessThanOperator, "leftOp", "5"))
				Expect(err).ToNot(HaveOccurred())
			})
			for _, op := range Operators {
				Specify("With valid operator parameters", func() {
					_, err := AddCriteria(ctx, ByField(op, "leftOp", "rightop"))
					Expect(err).ToNot(HaveOccurred())
				})
			}
		})
	})

	Describe("Context with criteria", func() {
		Context("When there are no criteria in the context", func() {
			It("Adds the new ones", func() {
				newCriteria := []Criterion{ByField(EqualsOperator, "leftOp", "rightOp")}
				newContext := ContextWithCriteria(ctx, newCriteria)
				Expect(CriteriaForContext(newContext)).To(ConsistOf(newCriteria))
			})
		})

		Context("When there are criteria already in the context", func() {
			It("Overrides them", func() {
				oldCriteria := []Criterion{ByField(EqualsOperator, "leftOp", "rightOp")}
				oldContext := ContextWithCriteria(ctx, oldCriteria)

				newCriteria := []Criterion{ByLabel(NotEqualsOperator, "leftOp1", "rightOp1")}
				newContext := ContextWithCriteria(oldContext, newCriteria)
				criteriaForNewContext := CriteriaForContext(newContext)
				Expect(criteriaForNewContext).To(ConsistOf(newCriteria))
				Expect(criteriaForNewContext).ToNot(ContainElement(oldCriteria[0]))
			})
		})
	})

	Describe("Parse query", func() {
		for _, queryType := range CriteriaTypes {
			Context("With no query", func() {
				It("Should return empty criteria", func() {
					criteria, err := Parse(queryType, "")
					Expect(err).ToNot(HaveOccurred())
					Expect(len(criteria)).To(Equal(0))
				})
			})

			Context("With missing query operator", func() {
				It("Should return an error", func() {
					criteria, err := Parse(queryType, "leftop_rightop")
					Expect(err).To(HaveOccurred())
					Expect(criteria).To(BeNil())
				})
			})

			Context("When there is an invalid field query", func() {
				It("Should return an error", func() {
					criteria, err := Parse(queryType, "leftop lt 'rightop'")
					Expect(err).To(HaveOccurred())
					Expect(criteria).To(BeNil())
				})
			})

			Context("When passing multivariate query", func() {
				It("Should be ok", func() {
					criteria, err := Parse(queryType, "leftop in ('rightop', 'rightop2')")
					Expect(err).ToNot(HaveOccurred())
					Expect(criteria).To(ConsistOf(NewCriterion("leftop", InOperator, []string{"rightop2", "rightop"}, queryType)))
				})
			})

			Context("When passing multiple queries", func() {
				It("Should build criteria", func() {
					criteria, err := Parse(queryType, "leftop1 in ('rightop','rightop2') and leftop2 eq 'rightop3'")
					Expect(err).ToNot(HaveOccurred())
					Expect(criteria).To(ConsistOf(NewCriterion("leftop1", InOperator, []string{"rightop2", "rightop"}, queryType), NewCriterion("leftop2", EqualsOperator, []string{"rightop3"}, queryType)))
				})
			})

			Context("Operator is unsupported", func() {
				It("Should return error", func() {
					criteria, err := Parse(queryType, "leftop1 @ ('rightop', 'rightop2')")
					Expect(err).To(HaveOccurred())
					Expect(criteria).To(BeNil())
				})
			})

			Context("Right operand is empty", func() {
				It("Should be ok", func() {
					criteria, err := Parse(queryType, "leftop1 in ()")
					Expect(err).ToNot(HaveOccurred())
					Expect(criteria).ToNot(BeNil())
					Expect(criteria).To(ConsistOf(NewCriterion("leftop1", InOperator, []string{""}, queryType)))
				})
			})
			Context("Multivariate operator with right operand without opening brace", func() {
				It("Should return error", func() {
					criteria, err := Parse(queryType, "leftop in 'rightop','rightop2')")
					Expect(err).To(HaveOccurred())
					Expect(criteria).To(BeNil())
				})
			})
			Context("Multivariate operator with right operand without closing brace", func() {
				It("Should return error", func() {
					criteria, err := Parse(queryType, "leftop in ('rightop','rightop2'")
					Expect(err).To(HaveOccurred())
					Expect(criteria).To(BeNil())
				})
			})
			Context("Right operand with escaped quote", func() {
				It("Should be okay", func() {
					criteria, err := Parse(queryType, "leftop1 eq 'right''op'")
					Expect(err).ToNot(HaveOccurred())
					Expect(criteria).ToNot(BeNil())
					Expect(criteria).To(ConsistOf(NewCriterion("leftop1", EqualsOperator, []string{"right'op"}, queryType)))
				})
			})
			Context("Complex right operand", func() {
				It("Should be okay", func() {
					rightOp := "this is a mixed, input example. It contains symbols   words ! -h@ppy p@rs'ng"
					escaped := strings.Replace(rightOp, "'", "''", -1)
					criteria, err := Parse(queryType, fmt.Sprintf("leftop1 eq '%s'", escaped))
					Expect(err).ToNot(HaveOccurred())
					Expect(criteria).ToNot(BeNil())
					Expect(criteria).To(ConsistOf(NewCriterion("leftop1", EqualsOperator, []string{rightOp}, queryType)))
				})
			})

			Context("Duplicate query key", func() {
				It("Should return error", func() {
					criteria, err := Parse(queryType, "leftop1 eq 'rightop' and leftop1 eq 'rightop2'")
					Expect(err).To(HaveOccurred())
					Expect(criteria).To(BeNil())
				})
			})

			Context("With different operators and query type combinations", func() {
				for _, op := range Operators {
					for _, queryType := range []CriterionType{FieldQuery, LabelQuery} {
						It("Should behave as expected", func() {
							rightOp := []string{"rightOp"}
							stringParam := rightOp[0]
							if op.Type() == MultivariateOperator {
								rightOp = []string{"rightOp1", "rightOp2"}
								stringParam = fmt.Sprintf("[%s]", strings.Join(rightOp, "||"))
							}
							criteria, err := Parse(queryType, fmt.Sprintf("leftop %s '%s'", op, stringParam))
							if op.IsNullable() && queryType == LabelQuery {
								Expect(err).To(HaveOccurred())
								Expect(criteria).To(BeNil())
							} else {
								Expect(err).ToNot(HaveOccurred())
								Expect(criteria).ToNot(BeNil())
								expectedQuery := NewCriterion("leftop1", op, rightOp, queryType)
								Expect(criteria).To(ConsistOf(expectedQuery))
							}
						})
					}
				}
			})

			Context("When separator is not properly escaped in first query value", func() {
				It("Should return error", func() {
					criteria, err := Parse(queryType, "leftop1 eq 'not'escaped' and leftOp2 eq 'rightOp'")
					Expect(err).To(HaveOccurred())
					Expect(criteria).To(BeNil())
				})
			})

			Context("When separator is not escaped in value", func() {
				It("Trims the value to the separator", func() {
					criteria, err := Parse(queryType, "leftop1 eq 'not'escaped'")
					Expect(err).To(HaveOccurred())
					Expect(criteria).To(BeNil())
				})

				It("Should fail", func() {
					criteria, err := Parse(queryType, "leftop1eq 'notescaped'")
					Expect(err).To(HaveOccurred())
					Expect(criteria).To(BeNil())
				})
			})

			Context("When using equals or operators", func() {
				It("should build the right gte query", func() {
					criteria, err := Parse(FieldQuery, "leftop gte -1.35")
					Expect(err).ToNot(HaveOccurred())
					Expect(criteria).ToNot(BeNil())
					expectedQuery := ByField(GreaterThanOrEqualOperator, "leftop", "-1.35")
					Expect(criteria).To(ConsistOf(expectedQuery))
				})

				It("should build the right lte query", func() {
					criteria, err := Parse(FieldQuery, "leftop lte 3")
					Expect(err).ToNot(HaveOccurred())
					Expect(criteria).ToNot(BeNil())
					expectedQuery := ByField(LessThanOrEqualOperator, "leftop", "3")
					Expect(criteria).To(ConsistOf(expectedQuery))
				})
			})
		}
	})
})
