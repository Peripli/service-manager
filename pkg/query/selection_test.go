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

package query

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/Peripli/service-manager/pkg/web"

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
			Specify("Nullable operator applied to label query", func() {
				addInvalidCriterion(ByLabel(EqualsOrNilOperator, "leftOp", "1"))
			})
			Specify("Numeric operator to non-numeric right operand", func() {
				addInvalidCriterion(ByField(GreaterThanOperator, "leftOp", "non-numeric"))
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
			for _, op := range operators {
				Specify("With valid operator parameters", func() {
					_, err := AddCriteria(ctx, ByField(op, "leftOp", "rightop"))
					Expect(err).ToNot(HaveOccurred())
				})
			}
		})
	})

	Describe("Build criteria from request", func() {
		var request *web.Request
		BeforeEach(func() {
			request = &web.Request{}
		})

		buildCriteria := func(url string) ([]Criterion, error) {
			newRequest, err := http.NewRequest(http.MethodGet, url, nil)
			Expect(err).ToNot(HaveOccurred())
			request = &web.Request{Request: newRequest}
			return BuildCriteriaFromRequest(request)
		}

		Context("When build from request with no query parameters", func() {
			It("Should return empty criteria", func() {
				criteriaFromRequest, err := buildCriteria("http://localhost:8080")
				Expect(err).ToNot(HaveOccurred())
				Expect(len(criteriaFromRequest)).To(Equal(0))
			})
		})

		Context("With missing query operator", func() {
			It("Should return an error", func() {
				criteriaFromRequest, err := buildCriteria("http://localhost:8080/v1/visibilities?fieldQuery=leftop_rightop")
				Expect(err).To(HaveOccurred())
				Expect(criteriaFromRequest).To(BeNil())
			})
		})

		Context("When there is an invalid field query", func() {
			It("Should return an error", func() {
				criteriaFromRequest, err := buildCriteria("http://localhost:8080/v1/visibilities?fieldQuery=leftop lt rightop")
				Expect(err).To(HaveOccurred())
				Expect(criteriaFromRequest).To(BeNil())
			})
		})

		Context("When passing multivariate query", func() {
			It("Should be ok", func() {
				criteriaFromRequest, err := buildCriteria("http://localhost:8080/v1/visibilities?fieldQuery=leftop in [rightop||rightop2]")
				Expect(err).ToNot(HaveOccurred())
				Expect(criteriaFromRequest).To(ConsistOf(ByField(InOperator, "leftop", "rightop", "rightop2")))
			})
		})

		Context("When passing label query", func() {
			It("Should be ok", func() {
				criteriaFromRequest, err := buildCriteria("http://localhost:8080/v1/visibilities?labelQuery=leftop in [rightop||rightop2]")
				Expect(err).ToNot(HaveOccurred())
				Expect(criteriaFromRequest).To(ConsistOf(ByLabel(InOperator, "leftop", "rightop", "rightop2")))
			})
		})

		Context("When passing label and field query", func() {
			It("Should be ok", func() {
				criteriaFromRequest, err := buildCriteria("http://localhost:8080/v1/visibilities?fieldQuery=leftop in [rightop||rightop2]&labelQuery=leftop in [rightop||rightop2]")
				Expect(err).ToNot(HaveOccurred())
				Expect(criteriaFromRequest).To(ConsistOf(ByField(InOperator, "leftop", "rightop", "rightop2"), ByLabel(InOperator, "leftop", "rightop", "rightop2")))
			})
		})

		Context("When passing multiple field queries", func() {
			It("Should build criteria", func() {
				criteriaFromRequest, err := buildCriteria("http://localhost:8080/v1/visibilities?fieldQuery=leftop1 in [rightop||rightop2]|leftop2 = rightop3")
				Expect(err).ToNot(HaveOccurred())
				Expect(criteriaFromRequest).To(ConsistOf(ByField(InOperator, "leftop1", "rightop", "rightop2"), ByField(EqualsOperator, "leftop2", "rightop3")))
			})
		})

		Context("When passing multiple label queries", func() {
			It("Should build criteria", func() {
				criteriaFromRequest, err := buildCriteria("http://localhost:8080/v1/visibilities?labelQuery=leftop1 in [rightop||rightop2]|leftop2 = rightop3")
				Expect(err).ToNot(HaveOccurred())
				Expect(criteriaFromRequest).To(ConsistOf(ByLabel(InOperator, "leftop1", "rightop", "rightop2"), ByLabel(EqualsOperator, "leftop2", "rightop3")))
			})
		})

		Context("Operator is unsupported", func() {
			It("Should return error", func() {
				criteriaFromRequest, err := buildCriteria("http://localhost:8080/v1/visibilities?fieldQuery=leftop1 @ [rightop||rightop2]")
				Expect(err).To(HaveOccurred())
				Expect(criteriaFromRequest).To(BeNil())
			})
		})

		Context("Operand has encoded value", func() {
			It("Should be ok", func() {
				criteriaFromRequest, err := buildCriteria("http://localhost:8080/v1/visibilities?fieldQuery=leftop1 in [%2Frightop||rightop2]")
				Expect(err).ToNot(HaveOccurred())
				Expect(criteriaFromRequest).ToNot(BeNil())
				expectedQuery := ByField(InOperator, "leftop1", "/rightop", "rightop2")
				Expect(criteriaFromRequest).To(ConsistOf(expectedQuery))
			})
		})
		Context("Right operand is empty", func() {
			It("Should be ok", func() {
				criteriaFromRequest, err := buildCriteria("http://localhost:8080/v1/visibilities?fieldQuery=leftop1 in ")
				Expect(err).ToNot(HaveOccurred())
				Expect(criteriaFromRequest).ToNot(BeNil())
				expectedQuery := ByField(InOperator, "leftop1", "")
				Expect(criteriaFromRequest).To(ConsistOf(expectedQuery))
			})
		})
		Context("Multivariate operator with right operand without opening brace", func() {
			It("Should return error", func() {
				criteriaFromRequest, err := buildCriteria("http://localhost:8080/v1/visibilities?fieldQuery=leftop in rightop||rightop2]")
				Expect(err).To(HaveOccurred())
				Expect(criteriaFromRequest).To(BeNil())
			})
		})
		Context("Multivariate operator with right operand without closing brace", func() {
			It("Should return error", func() {
				criteriaFromRequest, err := buildCriteria("http://localhost:8080/v1/visibilities?fieldQuery=leftop in [rightop||rightop2")
				Expect(err).To(HaveOccurred())
				Expect(criteriaFromRequest).To(BeNil())
			})
		})
		Context("Right operand with escaped separator", func() {
			It("Should be okay", func() {
				criteriaFromRequest, err := buildCriteria(`http://localhost:8080/v1/visibilities?fieldQuery=leftop1 = right\|op`)
				Expect(err).ToNot(HaveOccurred())
				Expect(criteriaFromRequest).ToNot(BeNil())
				expectedQuery := ByField(EqualsOperator, "leftop1", "right|op")
				Expect(criteriaFromRequest).To(ConsistOf(expectedQuery))
			})
		})
		Context("Complex right operand", func() {
			It("Should be okay", func() {
				rightOp := "this is a mixed, input example. It contains symbols   words ! -h@ppy p@rs|ng"
				escaped := url.QueryEscape(strings.Replace(rightOp, "|", "\\|", -1))
				criteriaFromRequest, err := buildCriteria(`http://localhost:8080/v1/visibilities?fieldQuery=leftop1 = ` + escaped)
				Expect(err).ToNot(HaveOccurred())
				Expect(criteriaFromRequest).ToNot(BeNil())
				expectedQuery := ByField(EqualsOperator, "leftop1", rightOp)
				Expect(criteriaFromRequest).To(ConsistOf(expectedQuery))
			})
		})

		Context("Duplicate field query key", func() {
			It("Should return error", func() {
				criteriaFromRequest, err := buildCriteria(`http://localhost:8080/v1/visibilities?fieldQuery=leftop1 = rightop|leftop1 = rightop2`)
				Expect(err).To(HaveOccurred())
				Expect(criteriaFromRequest).To(BeNil())
			})
		})

		Context("Duplicate label query key", func() {
			It("Should return error", func() {
				criteriaFromRequest, err := buildCriteria(`http://localhost:8080/v1/visibilities?labelQuery=leftop1 = rightop|leftop1 = rightop2`)
				Expect(err).To(HaveOccurred())
				Expect(criteriaFromRequest).To(BeNil())
			})
		})

		Context("With different operators and query type combinations", func() {
			for _, op := range operators {
				for _, queryType := range []CriterionType{FieldQuery, LabelQuery} {
					It("Should behave as expected", func() {
						rightOp := []string{"rightOp"}
						stringParam := rightOp[0]
						if op.IsMultiVariate() {
							rightOp = []string{"rightOp1", "rightOp2"}
							stringParam = fmt.Sprintf("[%s]", strings.Join(rightOp, "||"))
						}
						criteriaFromRequest, err := buildCriteria(fmt.Sprintf("http://localhost:8080/v1/visibilities?%s=leftop %s %s", queryType, op, stringParam))
						if op.IsNullable() && queryType == LabelQuery {
							Expect(err).To(HaveOccurred())
							Expect(criteriaFromRequest).To(BeNil())
						} else {
							Expect(err).ToNot(HaveOccurred())
							Expect(criteriaFromRequest).ToNot(BeNil())
							expectedQuery := newCriterion("leftop1", op, rightOp, queryType)
							Expect(criteriaFromRequest).To(ConsistOf(expectedQuery))
						}
					})
				}
			}
		})

		Context("When separator is not properly escaped in first query value", func() {
			It("Should return error", func() {
				criteriaFromRequest, err := buildCriteria(`http://localhost:8080/v1/visibilities?fieldQuery=leftop1 = not|escaped|leftOp2 = rightOp`)
				Expect(err).To(HaveOccurred())
				Expect(criteriaFromRequest).To(BeNil())
			})
		})

		Context("When separator is not escaped in value", func() {
			It("Trims the value to the separator", func() {
				criteriaFromRequest, err := buildCriteria(`http://localhost:8080/v1/visibilities?fieldQuery=leftop1 = not|escaped`)
				Expect(err).ToNot(HaveOccurred())
				Expect(criteriaFromRequest).ToNot(BeNil())
				expectedQuery := ByField(EqualsOperator, "leftop1", "not")
				Expect(criteriaFromRequest).To(ConsistOf(expectedQuery))
			})
		})
	})
})
