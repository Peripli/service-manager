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

package labels

import (
	"net/http"
	"testing"

	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/pkg/web/webfakes"

	"github.com/Peripli/service-manager/pkg/query"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestLabelsFilter(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Labels filter Suite")
}

var _ = Describe("Forbidden label operations filter", func() {
	var filter *ForibiddenLabelOperationsFilter
	var handler *webfakes.FakeHandler
	var protectedLabels map[string][]query.LabelOperation

	BeforeEach(func() {
		protectedLabels = map[string][]query.LabelOperation{
			"forbidden": []query.LabelOperation{"add", "add_values", "remove", "remove_values"},
		}
	})

	JustBeforeEach(func() {
		handler = &webfakes.FakeHandler{}
		filter = NewForbiddenLabelOperationsFilter(protectedLabels)
	})

	Context("POST", func() {
		When("entity has forbidden labels", func() {
			It("should return 400", func() {
				req := mockedRequest(http.MethodPost, `{"labels": {"forbidden": ["forbidden_value"]}}`)
				_, err := filter.Run(req, handler)
				httpErr, ok := err.(*util.HTTPError)
				Expect(ok).To(BeTrue())
				Expect(httpErr.StatusCode).To(Equal(http.StatusBadRequest))
				Expect(httpErr.Description).To(ContainSubstring("Set/Add values for label forbidden is not allowed"))
				Expect(handler.HandleCallCount()).To(Equal(0))
			})
		})

		When("there are no protected labels specified", func() {
			BeforeEach(func() {
				protectedLabels = map[string][]query.LabelOperation{}
			})
			It("should call next filter in chain", func() {
				req := mockedRequest(http.MethodPost, `{"labels": {"forbidden": ["forbidden_value"]}}`)
				_, err := filter.Run(req, handler)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(handler.HandleCallCount()).To(Equal(1))
			})
		})
	})

	Context("PATCH", func() {
		When("entity is modified with forbidden labels", func() {
			It("should return 400", func() {
				req := mockedRequest(http.MethodPatch, `{"labels": [{"op": "add", "key":"forbidden", "values":["forbidden_value"]}]}`)
				_, err := filter.Run(req, handler)
				httpErr, ok := err.(*util.HTTPError)
				Expect(ok).To(BeTrue())
				Expect(httpErr.StatusCode).To(Equal(http.StatusBadRequest))
				Expect(httpErr.Description).To(ContainSubstring("Operation add is not allowed for label forbidden"))
				Expect(handler.HandleCallCount()).To(Equal(0))
			})
		})

		When("add operation is allowed, but remove forbidden", func() {
			BeforeEach(func() {
				protectedLabels = map[string][]query.LabelOperation{
					"forbidden": []query.LabelOperation{"remove", "remove_values"},
				}
			})
			It("should call next filter in chain", func() {
				req := mockedRequest(http.MethodPatch, `{"labels": [{"op":"add", "key":"forbidden", "values":["forbidden_value"]}]}`)
				_, err := filter.Run(req, handler)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(handler.HandleCallCount()).To(Equal(1))
			})

			It("should not call next filter in chain", func() {
				req := mockedRequest(http.MethodPatch, `{"labels": [{"op":"remove", "key":"forbidden", "values":["forbidden_value"]}]}`)
				_, err := filter.Run(req, handler)
				httpErr, ok := err.(*util.HTTPError)
				Expect(ok).To(BeTrue())
				Expect(httpErr.StatusCode).To(Equal(http.StatusBadRequest))
				Expect(httpErr.Description).To(ContainSubstring("Operation remove is not allowed for label forbidden"))
				Expect(handler.HandleCallCount()).To(Equal(0))
			})
		})

		When("there are no protected labels specified", func() {
			BeforeEach(func() {
				protectedLabels = map[string][]query.LabelOperation{}
			})
			It("should call next filter in chain", func() {
				req := mockedRequest(http.MethodPatch, `{"labels": [{"op":"add", "key":"forbidden", "values":["forbidden_value"]}]}`)
				_, err := filter.Run(req, handler)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(handler.HandleCallCount()).To(Equal(1))
			})
		})
	})
})

func mockedRequest(method, json string) *web.Request {
	req, err := http.NewRequest(method, "", nil)
	Expect(err).ShouldNot(HaveOccurred())
	return &web.Request{Request: req, Body: []byte(json)}
}
