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
	"context"
	"net/http"

	"github.com/Peripli/service-manager/pkg/log"

	"net/http/httptest"
	"strings"

	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/pkg/web/webfakes"
	"github.com/Peripli/service-manager/test/testutil"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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

	loggingValidationMiddleware := func(filterName string) web.MiddlewareFunc {
		return func(request *web.Request, next web.Handler) (*web.Response, error) {
			log.D().Debugf("pre-%s", filterName)
			resp, err := next.Handle(request)
			log.D().Debugf("post-%s", filterName)
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
					"NewInstance method matches should panic",
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
					"NewInstance path pattern should panic",
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
			log.D().Debug("handler")
			return resp, nil
		})

		var filters []web.Filter
		var oldSettings log.Settings
		var hook *testutil.LogInterceptor

		BeforeEach(func() {
			hook = &testutil.LogInterceptor{}
			oldSettings = log.Configuration()
			_, err := log.Configure(context.TODO(), &log.Settings{
				Level:  "debug",
				Format: "text",
				Output: "/dev/stdout",
			})
			Expect(err).ToNot(HaveOccurred())
			log.AddHook(hook)
			filters = []web.Filter{
				fakeFilter("filter1", loggingValidationMiddleware("filter1"), []web.FilterMatcher{}),
				fakeFilter("filter2", loggingValidationMiddleware("filter2"), []web.FilterMatcher{}),
				fakeFilter("filter3", loggingValidationMiddleware("filter3"), []web.FilterMatcher{}),
			}
		})
		AfterEach(func() {
			_, err := log.Configure(context.TODO(), &oldSettings)
			Expect(err).ToNot(HaveOccurred())
		})

		It("executes all chained filters in the correct order", func() {
			webReq := web.Request{
				Request: httptest.NewRequest("GET", "http://example.com", strings.NewReader("")),
			}
			chainedHandler := web.Filters(filters).Chain(handler)
			resp, err := chainedHandler.Handle(&webReq)

			Expect(err).ShouldNot(HaveOccurred())
			Expect(resp.Header.Get("handler")).To(Equal("handler"))
			Expect(hook.String()).To(MatchRegexp(
				"(?s)pre-filter1.*pre-filter2.*pre-filter3.*handler.*post-filter3.*post-filter2.*post-filter1"))
		})

	})
})
