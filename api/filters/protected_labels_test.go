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

package filters_test

import (
	"net/http"

	"github.com/Peripli/service-manager/api/filters"

	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/pkg/web/webfakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Forbidden label operations filter", func() {
	var filter *filters.ProtectedLabelsFilter
	var handler *webfakes.FakeHandler
	var protectedLabels []string

	BeforeEach(func() {
		protectedLabels = []string{"forbidden", "alsoforbidden"}
	})

	JustBeforeEach(func() {
		handler = &webfakes.FakeHandler{}
		filter = filters.NewProtectedLabelsFilter(protectedLabels)
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
		When("entity has forbidden labels with upper case property", func() {
			It("should return 400", func() {
				req := mockedRequest(http.MethodPost, `{"Labels": {"alsoforbidden": ["forbidden_value"]}, "labels": {}}`)
				_, err := filter.Run(req, handler)
				httpErr, ok := err.(*util.HTTPError)
				Expect(ok).To(BeTrue())
				Expect(httpErr.StatusCode).To(Equal(http.StatusBadRequest))
				Expect(httpErr.Description).To(ContainSubstring("Set/Add values for label alsoforbidden is not allowed"))
				Expect(handler.HandleCallCount()).To(Equal(0))
			})
		})

		When("entity has no forbidden labels", func() {
			It("should call next filter in chain", func() {
				req := mockedRequest(http.MethodPost, `{"labels": {"allowed": ["value"]}}`)
				_, err := filter.Run(req, handler)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(handler.HandleCallCount()).To(Equal(1))
			})
		})

		When("there are no protected labels specified", func() {
			BeforeEach(func() {
				protectedLabels = []string{}
			})
			It("should call next filter in chain", func() {
				req := mockedRequest(http.MethodPost, `{"labels": {"alsoforbidden": ["forbidden_value"]}}`)
				_, err := filter.Run(req, handler)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(handler.HandleCallCount()).To(Equal(1))
			})
		})

		When("no labels are provided", func() {
			It("should call next filter in chain", func() {
				req := mockedRequest(http.MethodPost, `{}`)
				_, err := filter.Run(req, handler)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(handler.HandleCallCount()).To(Equal(1))
			})
		})

		When("empty labels are provided", func() {
			It("should call next filter in chain", func() {
				req := mockedRequest(http.MethodPost, `{"labels":{}}`)
				_, err := filter.Run(req, handler)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(handler.HandleCallCount()).To(Equal(1))
			})
		})

		When("labels is invalid json", func() {
			It("should return error", func() {
				req := mockedRequest(http.MethodPost, `{"labels": "invalid"}`)
				_, err := filter.Run(req, handler)
				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).Should(ContainSubstring("Invalid JSON body"))
				Expect(handler.HandleCallCount()).To(Equal(0))
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
				Expect(httpErr.Description).To(ContainSubstring("Modifying is not allowed for label forbidden"))
				Expect(handler.HandleCallCount()).To(Equal(0))
			})
		})

		When("there are no protected labels specified", func() {
			BeforeEach(func() {
				protectedLabels = []string{}
			})
			It("should call next filter in chain", func() {
				req := mockedRequest(http.MethodPatch, `{"labels": [{"op":"add", "key":"forbidden", "values":["forbidden_value"]}]}`)
				_, err := filter.Run(req, handler)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(handler.HandleCallCount()).To(Equal(1))
			})
		})

		When("labels is invalid json", func() {
			It("should return error", func() {
				req := mockedRequest(http.MethodPatch, `{"labels": "invalid"}`)
				_, err := filter.Run(req, handler)
				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).Should(ContainSubstring("Failed to decode"))
				Expect(handler.HandleCallCount()).To(Equal(0))
			})
		})

		When("entity has no forbidden labels", func() {
			It("should call next filter in chain", func() {
				req := mockedRequest(http.MethodPatch, `{"labels": [{"op":"add", "key":"allowed", "values":["value"]}]}`)
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
