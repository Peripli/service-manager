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

package rest

import (
	"testing"

	"github.com/Peripli/service-manager/pkg/web"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestFilters(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Filters Suite")
}

var _ = Describe("Filters", func() {
	Describe("MatchFilters", func() {
		It("Panics if filter path is empty", func() {
			Expect(func() {
				MatchFilters(&Endpoint{"GET", "/"}, []web.Filter{
					{
						RouteMatcher: web.RouteMatcher{
							Methods: []string{"GET"},
						},
					},
				})
			}).To(Panic())
		})

		tests := []struct {
			description string
			endpoint    *Endpoint
			filters     []web.Filter
			result      []string
		}{
			{
				"** matches multiple path segments",
				&Endpoint{"GET", "/a/b/c"},
				[]web.Filter{
					{
						Name: "a",
						RouteMatcher: web.RouteMatcher{
							Methods:     []string{"GET"},
							PathPattern: "/a/**",
						},
					},
					{
						Name: "b",
						RouteMatcher: web.RouteMatcher{
							Methods:     []string{"GET"},
							PathPattern: "/b/**",
						},
					},
				},
				[]string{"a"},
			},
			{
				"No method matches any method",
				&Endpoint{"GET", "/a/b/c"},
				[]web.Filter{
					{
						Name: "a",
						RouteMatcher: web.RouteMatcher{
							PathPattern: "/a/**",
						},
					},
				},
				[]string{"a"},
			},
			{
				"Non strict trailing slash",
				&Endpoint{"GET", "/a/b/c"},
				[]web.Filter{
					{
						Name: "a",
						RouteMatcher: web.RouteMatcher{
							PathPattern: "/a/b/c/**",
						},
					},
					{
						Name: "b",
						RouteMatcher: web.RouteMatcher{
							PathPattern: "/a/b/c/*",
						},
					},
					{
						Name: "c",
						RouteMatcher: web.RouteMatcher{
							PathPattern: "/a/b/c/",
						},
					},
				},
				[]string{"a", "b", "c"},
			},
		}

		for _, t := range tests {
			It(t.description, func() {
				matchedFilters := MatchFilters(t.endpoint, t.filters)
				matchedNames := make([]string, len(matchedFilters))
				for i, f := range matchedFilters {
					matchedNames[i] = f.Name
				}
				Expect(matchedNames).To(Equal(t.result))
			})
		}
	})
})
