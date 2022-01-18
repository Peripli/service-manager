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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/Peripli/service-manager/pkg/query"
)

var _ = Describe("Selection", func() {

	var ctx context.Context

	BeforeEach(func() {
		ctx = context.TODO()
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
			Specify("Multiple limit criteria", func() {
				var err error
				ctx, err = AddCriteria(ctx, LimitResultBy(10))
				Expect(err).ShouldNot(HaveOccurred())
				addInvalidCriterion(LimitResultBy(5))
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
				newContext, err := ContextWithCriteria(ctx, newCriteria...)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(CriteriaForContext(newContext)).To(ConsistOf(newCriteria))
			})
		})
		Context("When there are criteria already in the context", func() {
			It("Overrides them", func() {
				oldCriteria := []Criterion{ByField(EqualsOperator, "leftOp", "rightOp")}
				oldContext, err := ContextWithCriteria(ctx, oldCriteria...)
				Expect(err).ShouldNot(HaveOccurred())

				newCriteria := []Criterion{ByLabel(NotEqualsOperator, "leftOp1", "rightOp1")}
				newContext, err := ContextWithCriteria(oldContext, newCriteria...)
				Expect(err).ShouldNot(HaveOccurred())

				criteriaForNewContext := CriteriaForContext(newContext)
				Expect(criteriaForNewContext).To(ConsistOf(newCriteria))
				Expect(criteriaForNewContext).ToNot(ContainElement(oldCriteria[0]))
			})
		})
		Context("When limit is already in the context adding it again", func() {
			It("should return error", func() {
				ctx, err := ContextWithCriteria(ctx, LimitResultBy(10), LimitResultBy(5))
				Expect(err).Should(HaveOccurred())
				Expect(ctx).Should(BeNil())

			})
		})
	})

	Describe("Parse query", func() {
		for _, queryType := range CriteriaTypes {
			Describe(string("queryType - "+queryType), func() {
				queryType := queryType
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
						//ExistQuery type doesn't support left operands nor it excepts any operators aside from NotExist/Exist operators
						if queryType == LabelQuery {
							criteria, err := Parse(queryType, "leftop1 eq 'rightop' and leftop1 eq 'rightop2'")
							Expect(err).To(HaveOccurred())
							Expect(criteria).To(BeNil())
						}
					})
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
				})

				Context("When using equals or operators", func() {
					It("should build the right ge query", func() {
						criteria, err := Parse(FieldQuery, "leftop ge -1.35")
						Expect(err).ToNot(HaveOccurred())
						Expect(criteria).ToNot(BeNil())
						expectedQuery := ByField(GreaterThanOrEqualOperator, "leftop", "-1.35")
						Expect(criteria).To(ConsistOf(expectedQuery))
					})

					It("should build the right le query", func() {
						criteria, err := Parse(FieldQuery, "leftop le 3")
						Expect(err).ToNot(HaveOccurred())
						Expect(criteria).ToNot(BeNil())
						expectedQuery := ByField(LessThanOrEqualOperator, "leftop", "3")
						Expect(criteria).To(ConsistOf(expectedQuery))
					})
				})
			})
			for _, op := range Operators {
				op := op
				for _, queryType := range []CriterionType{FieldQuery, LabelQuery} {
					queryType := queryType
					Context(fmt.Sprintf("With %s operator and %s query type", op.String(), queryType), func() {
						It("Should behave as expected", func() {
							leftOp := "leftOp"
							rightOp := []string{"rightOp"}
							stringParam := fmt.Sprintf("'%s'", rightOp[0])
							if op.IsNumeric() {
								rightOp = []string{"5"}
								stringParam = rightOp[0]
							}
							if op.Type() == MultivariateOperator {
								rightOp = []string{"rightOp1", "rightOp2"}
								stringParam = fmt.Sprintf("('%s')", strings.Join(rightOp, "','"))
							}
							query := fmt.Sprintf("%s %s %s", leftOp, op, stringParam)
							criteria, err := Parse(queryType, query)
							if op.IsNullable() && queryType == LabelQuery {
								Expect(err).To(HaveOccurred())
								Expect(criteria).To(BeNil())
							} else {
								Expect(err).ToNot(HaveOccurred())
								Expect(criteria).ToNot(BeNil())
								c := criteria[0]
								expectedQuery := NewCriterion(leftOp, op, rightOp, queryType)
								Expect(c.LeftOp).To(Equal(expectedQuery.LeftOp))
								Expect(c.Operator).To(Equal(expectedQuery.Operator))
								Expect(c.RightOp).To(ConsistOf(expectedQuery.RightOp))
								Expect(c.Type).To(Equal(expectedQuery.Type))
							}
						})
					})
				}
			}
		}

	})

	DescribeTable("Validate Criterion",
		func(c Criterion, expectedErr ...string) {
			err := c.Validate()
			if len(expectedErr) == 0 {
				Expect(err).To(BeNil())
			} else {
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(ContainSubstring(expectedErr[0]))
			}
		},
		Entry("Empty right operand is not allowed",
			ByField(InOperator, "left"),
			"missing right operand"),
		Entry("Limit that is not a number is not allowed",
			NewCriterion(Limit, NoOperator, []string{"not a number"}, ResultQuery),
			"could not convert string to int"),
		Entry("Negative limit is not allowed",
			LimitResultBy(-1),
			"should be positive"),
		Entry("Valid limit is allowed",
			LimitResultBy(10)),
		Entry("Missing order type is not allowed",
			NewCriterion(OrderBy, NoOperator, []string{"field"}, ResultQuery),
			"expects field name and order type"),
		Entry("Valid order by is allowed",
			OrderResultBy("field", AscOrder)),
		Entry("Multiple right operands are not allowed for univariate operators",
			ByField(EqualsOperator, "left", "right1", "right2"),
			"single value operation"),
		Entry("Operators working with nil are not allowed for label queries",
			ByLabel(EqualsOrNilOperator, "left", "right"),
			"only for field queries"),
		Entry("Non-numeric operand is not allowed by comparison operators",
			ByField(LessThanOperator, "left", "not numeric"),
			"not numeric or datetime"),
		Entry("Separator word 'and' is allowed to appear in left operand as a substring",
			ByField(EqualsOperator, "band", "music")),
		Entry("Separator word 'and' is not allowed to be the only word in the left operand",
			ByField(EqualsOperator, "and", "this"),
			"separator and is not allowed"),
		Entry("Separator word 'and' is not allowed to appear as a standalone word in middle of left operand",
			ByField(EqualsOperator, "a and b", "this"),
			"separator and is not allowed"),
		Entry("Separator word 'and' is not allowed to appear as a standalone word in beginning of left operand",
			ByField(EqualsOperator, "and b", "this"),
			"separator and is not allowed"),
		Entry("Separator word 'and' is not allowed to appear as a standalone word in end of left operand",
			ByField(EqualsOperator, "a and", "this"),
			"separator and is not allowed"),
		Entry("New line character is not allowed in right operand",
			ByField(EqualsOperator, "left", "one\ntwo"),
			"forbidden new line character"),
	)
})
