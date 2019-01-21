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

func DescribeDeleteTestsfor(ctx *common.TestContext, t TestCase) bool {
	return Describe(fmt.Sprintf("DELETE %s", t.API), func() {
		var testResource common.Object
		var testResourceID string

		Context("Existing resource", func() {
			BeforeEach(func() {
				createFunc := t.ResourceBlueprint

				By(fmt.Sprintf("[SETUP]: Creating test resource of type %s", t.API))
				testResource = createFunc(ctx)
				Expect(testResource).ToNot(BeEmpty())

				By(fmt.Sprintf("[SETUP]: Verifying that test resource %v has an non empty id of type string", testResource))
				testResourceID = testResource["id"].(string)
				Expect(testResourceID).ToNot(BeEmpty())
			})

			It("returns 200", func() {
				By("[TEST]: Verify resource of type %s exists before delete")
				ctx.SMWithOAuth.GET(fmt.Sprintf("%s/%s", t.API, testResourceID)).
					Expect().
					Status(http.StatusOK).JSON().Object().ContainsMap(testResource)

				By("[TEST]: Verify resource of type %s is deleted successfully")
				ctx.SMWithOAuth.DELETE(fmt.Sprintf("%s/%s", t.API, testResourceID)).
					Expect().
					Status(http.StatusOK)

				By("[TEST]: Verify resource of type %s does not exist after delete")
				ctx.SMWithOAuth.GET(fmt.Sprintf("%s/%s", t.API, testResourceID)).
					Expect().
					Status(http.StatusNotFound)
			})
		})

		Context("Not existing resource", func() {
			BeforeEach(func() {
				testResourceID = "non-existing-id"
			})

			It("returns 404", func() {
				ctx.SMWithOAuth.DELETE(fmt.Sprintf("%s/%s", t.API, testResourceID)).
					Expect().
					Status(http.StatusNotFound).JSON().Object().Keys().Contains("error", "description")
			})
		})
	})
}
