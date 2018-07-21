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

package api

import (
	"testing"

	"github.com/Peripli/service-manager/pkg/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestHandler(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "SMHandler Suite")
}

var _ = Describe("SMHandler", func() {
	Describe("NewHTTPHandler", func() {
		It("Panics if a filter has no middleware function", func() {
			filters := []types.Filter{{
				Name: "test-filter",
				RouteMatcher: types.RouteMatcher{
					PathPattern: "*",
				},
			}}
			handler := func(*types.Request) (*types.Response, error) { return nil, nil }

			Expect(func() {
				NewHTTPHandler(filters, handler)
			}).To(Panic())

			filters[0].Middleware = func(*types.Request, types.SMHandler) (*types.Response, error) {
				return nil, nil
			}
			Expect(func() {
				NewHTTPHandler(filters, handler)
			}).ToNot(Panic())
		})
	})
})
