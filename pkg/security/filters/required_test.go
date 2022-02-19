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

package filters

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/pkg/web/webfakes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Security Filters Suite")
}

var _ = Describe("Filters tests", func() {
	var (
		req     *web.Request
		handler *webfakes.FakeHandler
	)

	BeforeEach(func() {
		req = &web.Request{
			Request: httptest.NewRequest("GET", "/", nil),
		}
		handler = &webfakes.FakeHandler{}
	})

	Describe("Authz required filter", func() {
		var requiredAuthzFilter web.Filter
		var matchers []web.FilterMatcher

		BeforeEach(func() {
			matchers = []web.FilterMatcher{}
			requiredAuthzFilter = NewRequiredAuthzFilter(matchers)
		})

		Describe("when Filter.Run is invoked with no authorization confirmation", func() {
			It("should return 403", func() {
				resp, err := requiredAuthzFilter.Run(req, handler)
				Expect(resp).To(BeNil())
				httpErr, ok := err.(*util.HTTPError)
				Expect(ok).To(BeTrue())
				Expect(httpErr.StatusCode).To(Equal(http.StatusForbidden))
			})
		})

		Describe("when Filter.Run is invoked with authorization confirmation", func() {
			It("should continue", func() {
				req.Request = req.WithContext(web.ContextWithAuthorization(req.Context()))
				_, err := requiredAuthzFilter.Run(req, handler)
				Expect(err).ToNot(HaveOccurred())
				Expect(handler.HandleCallCount()).To(Equal(1))
			})
		})

		Describe("when Filter.FilterMatchers is invoked", func() {
			It("should return the matchers passed to the c-tor", func() {
				Expect(requiredAuthzFilter.FilterMatchers()).To(Equal(matchers))
			})
		})

	})

	Describe("Authn required filter", func() {

		Describe("when Filter.Run is invoked with no user in context", func() {
			It("should return 401", func() {
				requiredAuthnFilter := NewRequiredAuthnFilter(nil)
				resp, err := requiredAuthnFilter.Run(req, handler)
				Expect(resp).To(BeNil())
				httpErr, ok := err.(*util.HTTPError)
				Expect(ok).To(BeTrue())
				Expect(httpErr.StatusCode).To(Equal(http.StatusUnauthorized))
			})
		})

		Describe("when Filter.Run is invoked with user in context", func() {
			It("should continue", func() {
				requiredAuthzFilter := NewRequiredAuthnFilter(nil)
				req.Request = req.WithContext(web.ContextWithUser(req.Context(), &web.UserContext{}))
				_, err := requiredAuthzFilter.Run(req, handler)
				Expect(err).ToNot(HaveOccurred())
				Expect(handler.HandleCallCount()).To(Equal(1))
			})
		})

		Describe("when Filter.FilterMatchers is invoked", func() {
			It("should return an empty array", func() {
				Expect(NewRequiredAuthnFilter([]web.FilterMatcher{}).FilterMatchers()).To(HaveLen(0))
			})
		})

	})
})
