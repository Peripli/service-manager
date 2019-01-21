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

package test

import (
	"fmt"
	"net/http"

	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func DescribeGetTestsfor(ctx *common.TestContext, t TestCase) bool {
	return Describe("GET", func() {
		var testResource common.Object
		var testResourceID string

		Context(fmt.Sprintf("Existing resource of type %s", t.API), func() {
			BeforeEach(func() {
				testResource = t.ResourceBlueprint(ctx)
				By(fmt.Sprintf("[SETUP]: Verifying that test resource %v is not empty", testResource))
				Expect(testResource).ToNot(BeEmpty())

				By(fmt.Sprintf("[SETUP]: Verifying that test resource %v has an id of type string", testResource))
				testResourceID = testResource["id"].(string)
				Expect(testResourceID).ToNot(BeEmpty())
			})

			It("returns 200", func() {
				ctx.SMWithOAuth.GET(fmt.Sprintf("%s/%s", t.API, testResourceID)).
					Expect().
					Status(http.StatusOK).JSON().Object().ContainsMap(testResource)
			})
		})

		Context(fmt.Sprintf("Not existing resource of type %s", t.API), func() {
			BeforeEach(func() {
				testResourceID = "non-existing-id"
			})

			It("returns 404", func() {
				ctx.SMWithOAuth.GET(fmt.Sprintf("%s/%s", t.API, testResourceID)).
					Expect().
					Status(http.StatusNotFound).JSON().Object().Keys().Contains("error", "description")
			})
		})
	})
}
