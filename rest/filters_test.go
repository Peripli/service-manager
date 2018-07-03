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
	"net/http"
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
							Methods: []string{http.MethodGet},
						},
					},
				})
			}).To(Panic())
		})

		tests := []struct {
			description string
			endpoint    *Endpoint
			filters     []web.Filter
			result      []int
		}{
			{
				"",
				&Endpoint{"GET", "/a/b/c"},
				[]web.Filter{
					{
						RouteMatcher: web.RouteMatcher{
							Methods:     []string{"GET"},
							PathPattern: "/a/**",
						},
					},
					{
						RouteMatcher: web.RouteMatcher{
							Methods:     []string{"PUT"},
							PathPattern: "/a/**",
						},
					},
					{
						RouteMatcher: web.RouteMatcher{
							Methods:     []string{"GET"},
							PathPattern: "/a/b/*",
						},
					},
				},
				[]int{0, 2},
			},
		}
		for _, t := range tests {
			It(t.description, func() {
				result := make([]web.Filter, len(t.result))
				for i, r := range t.result {
					result[i] = t.filters[r]
				}
				Expect(MatchFilters(t.endpoint, t.filters)).To(Equal(result))
			})
		}
	})
})
