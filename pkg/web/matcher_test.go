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

package web_test

import (
	"net/http"

	"net/http/httptest"
	"strings"

	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/pkg/web/webfakes"
	"github.com/Peripli/service-manager/test/testutil"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

var _ = Describe("Filters", func() {
	fakeFilter := func(name string, f web.MiddlewareFunc, matchers []web.FilterMatcher) web.Filter {
		filter := &webfakes.FakeFilter{}
		filter.RunStub = f
		filter.NameReturns(name)
		filter.FilterMatchersReturns(matchers)
		return filter
	}

	delegatingMiddleware := func(request *web.Request, next web.Handler) (*web.Response, error) {
		return next.Handle(request)
	}

	loggingValidationMiddleware := func(prevFilterName, currFilterName, nextFilterName string, hook *testutil.LogInterceptor) web.MiddlewareFunc {
		return func(request *web.Request, next web.Handler) (*web.Response, error) {
			if prevFilterName != "" {
				Expect(hook).To(ContainSubstring("Entering Filter: " + prevFilterName))
			}
			Expect(hook).To(ContainSubstring("Entering Filter: " + currFilterName))
			resp, err := next.Handle(request)
			if nextFilterName != "" {
				Expect(hook).To(ContainSubstring("Exiting Filter: " + nextFilterName))
			}
			return resp, err
		}
	}

	Describe("Matching", func() {
		Context("Panics", func() {

			tests := []struct {
				description string
				endpoint    web.Endpoint
				filters     []web.Filter
				result      []string
			}{
				{
					"Empty method matches should panic",
					web.Endpoint{http.MethodGet, "/a/b/c"},
					[]web.Filter{
						fakeFilter("filter1", delegatingMiddleware, []web.FilterMatcher{
							{
								Matchers: []web.Matcher{
									web.Methods(),
								},
							},
						}),
					},
					[]string{},
				},
				{
					"Empty path pattern should panic",
					web.Endpoint{http.MethodGet, "/a/b/c"},
					[]web.Filter{
						fakeFilter("filter2", delegatingMiddleware, []web.FilterMatcher{
							{
								Matchers: []web.Matcher{
									web.Path(),
								},
							},
						}),
					},
					[]string{},
				},
			}

			for _, t := range tests {
				t := t
				It(t.description, func() {
					Expect(func() { web.Filters(t.filters).Matching(t.endpoint) }).To(Panic())
				})

			}
		})

		tests := []struct {
			description string
			endpoint    web.Endpoint
			filters     []web.Filter
			result      []string
		}{
			{
				"** matches multiple path segments",
				web.Endpoint{http.MethodGet, "/a/b/c"},
				[]web.Filter{
					fakeFilter("filter1", delegatingMiddleware, []web.FilterMatcher{
						{
							Matchers: []web.Matcher{
								web.Methods(http.MethodGet),
								web.Path("/a/**"),
							},
						},
					}),
					fakeFilter("filter2", delegatingMiddleware, []web.FilterMatcher{
						{
							Matchers: []web.Matcher{
								web.Methods(http.MethodGet),
								web.Path("/b/**"),
							},
						},
					}),
				},
				[]string{"filter1"},
			},
			{
				"No matchers matches anything",
				web.Endpoint{http.MethodGet, "/a/b/c"},
				[]web.Filter{
					fakeFilter("filter1", delegatingMiddleware, []web.FilterMatcher{}),
				},
				[]string{"filter1"},
			},
			{
				"Path ending with ** matches prefixes",
				web.Endpoint{http.MethodGet, "/a/b/c"},
				[]web.Filter{
					fakeFilter("filter1", delegatingMiddleware, []web.FilterMatcher{
						{
							Matchers: []web.Matcher{
								web.Path("/a/**"),
							},
						},
					}),
				},
				[]string{"filter1"},
			},
			{
				"Non strict trailing slash",
				web.Endpoint{http.MethodGet, "/a/b/c"},
				[]web.Filter{
					fakeFilter("filter1", delegatingMiddleware, []web.FilterMatcher{
						{
							Matchers: []web.Matcher{
								web.Path("/a/b/c/**"),
							},
						},
					}),
					fakeFilter("filter2", delegatingMiddleware, []web.FilterMatcher{
						{
							Matchers: []web.Matcher{
								web.Path("/a/b/c/*"),
							},
						},
					}),
					fakeFilter("filter3", delegatingMiddleware, []web.FilterMatcher{
						{
							Matchers: []web.Matcher{
								web.Path("/a/b/c/"),
							},
						},
					}),
					fakeFilter("filter4", delegatingMiddleware, []web.FilterMatcher{
						{
							Matchers: []web.Matcher{
								web.Path("/a/b/c"),
							},
						},
					}),
					fakeFilter("filter5", delegatingMiddleware, []web.FilterMatcher{
						{
							Matchers: []web.Matcher{
								web.Path("/a/b/**"),
							},
						},
					}),
				},
				[]string{"filter1", "filter2", "filter3", "filter4", "filter5"},
			},
		}

		for _, t := range tests {
			t := t
			It(t.description, func() {
				matchedFilters := web.Filters(t.filters).Matching(t.endpoint)
				matchedNames := make([]string, len(matchedFilters))
				for i, f := range matchedFilters {
					matchedNames[i] = f.Name()
				}
				Expect(matchedNames).To(Equal(t.result))
			})
		}
	})

	Describe("Chain", func() {
		handler := web.HandlerFunc(func(request *web.Request) (*web.Response, error) {
			headers := http.Header{}
			headers.Add("handler", "handler")
			resp := &web.Response{
				StatusCode: http.StatusOK,
				Header:     headers,
				Body:       []byte(`{}`),
			}

			return resp, nil
		})

		var filters []web.Filter
		var level logrus.Level

		BeforeEach(func() {
			hook := &testutil.LogInterceptor{}
			level = logrus.GetLevel()
			logrus.SetLevel(logrus.DebugLevel)
			logrus.AddHook(hook)
			filters = []web.Filter{
				fakeFilter("filter1", loggingValidationMiddleware("", "filter1", "filter2", hook), []web.FilterMatcher{}),
				fakeFilter("filter2", loggingValidationMiddleware("filter1", "filter2", "filter3", hook), []web.FilterMatcher{}),
				fakeFilter("filter3", loggingValidationMiddleware("filter2", "filter3", "", hook), []web.FilterMatcher{}),
			}
		})
		AfterEach(func() {
			logrus.SetLevel(level)
		})

		It("executes all chained filters in the correct order", func() {
			webReq := web.Request{
				Request: httptest.NewRequest("GET", "http://example.com", strings.NewReader("")),
			}
			chainedHandler := web.Filters(filters).Chain(handler)
			resp, err := chainedHandler.Handle(&webReq)

			Expect(err).ShouldNot(HaveOccurred())
			Expect(resp.Header.Get("handler")).To(Equal("handler"))
		})

	})
})
