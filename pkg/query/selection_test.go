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
	"net/http"

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
		Context("Invalid", func() {
			Specify("Univariate operator with multiple right operands", func() {
				_, err := AddCriteria(ctx, ByField(EqualsOperator, "leftOp", "1", "2"))
				Expect(err).To(HaveOccurred())
			})
			Specify("Nullable operator applied to label query", func() {
				_, err := AddCriteria(ctx, ByLabel(EqualsOrNilOperator, "leftOp", "1"))
				Expect(err).To(HaveOccurred())
			})
			Specify("Numeric operator to non-numeric right operand", func() {
				_, err := AddCriteria(ctx, ByField(GreaterThanOperator, "leftOp", "non-numeric"))
				Expect(err).To(HaveOccurred())
			})
			Specify("Field query with duplicate key", func() {
				ctx, err := AddCriteria(ctx, validCriterion)
				Expect(err).ToNot(HaveOccurred())
				duplicateKeyCriterion := ByField(EqualsOrNilOperator, validCriterion.LeftOp, "right op")
				_, err = AddCriteria(ctx, duplicateKeyCriterion)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("duplicate"))
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
		})
	})

	Describe("Build criteria from request", func() {
		var request *web.Request
		BeforeEach(func() {
			request = &web.Request{}
		})

		buildCriteria := func(url string) (criteria, error) {
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
				criteriaFromRequest, err := buildCriteria("http://localhost:8080/v1/visibilities?fieldQuery=leftop+lt+rightop")
				Expect(err).To(HaveOccurred())
				Expect(criteriaFromRequest).To(BeNil())
			})
		})

		Context("When passing multivariate query", func() {
			It("Should be ok", func() {
				criteriaFromRequest, err := buildCriteria("http://localhost:8080/v1/visibilities?fieldQuery=leftop+in+[rightop,rightop2]")
				Expect(err).ToNot(HaveOccurred())
				Expect(criteriaFromRequest).To(ConsistOf(ByField(InOperator, "leftop", "rightop", "rightop2")))
			})
		})

		Context("When passing label query", func() {
			It("Should be ok", func() {
				criteriaFromRequest, err := buildCriteria("http://localhost:8080/v1/visibilities?labelQuery=leftop+in+[rightop,rightop2]")
				Expect(err).ToNot(HaveOccurred())
				Expect(criteriaFromRequest).To(ConsistOf(ByLabel(InOperator, "leftop", "rightop", "rightop2")))
			})
		})

		Context("When passing label and field query", func() {
			It("Should be ok", func() {
				criteriaFromRequest, err := buildCriteria("http://localhost:8080/v1/visibilities?fieldQuery=leftop+in+[rightop,rightop2]&labelQuery=leftop+in+[rightop,rightop2]")
				Expect(err).ToNot(HaveOccurred())
				Expect(criteriaFromRequest).To(ConsistOf(ByField(InOperator, "leftop", "rightop", "rightop2"), ByLabel(InOperator, "leftop", "rightop", "rightop2")))
			})
		})
	})
})
