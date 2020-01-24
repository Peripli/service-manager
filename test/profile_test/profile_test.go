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

package profile_test

import (
	"net/http"
	"testing"

	"github.com/Peripli/service-manager/pkg/web"

	. "github.com/onsi/ginkgo"

	"github.com/Peripli/service-manager/test/common"
)

func TestProfile(t *testing.T) {
	RunSpecs(t, "Profile Suite")
}

var _ = Describe("Profile API", func() {

	var ctx *common.TestContext

	BeforeSuite(func() {
		ctx = common.DefaultTestContext()
	})

	AfterSuite(func() {
		ctx.Cleanup()
	})

	Describe("Get heap profile", func() {
		It("Returns correct response", func() {
			ctx.SM.GET(web.ProfileURL + "/heap").
				Expect().
				Status(http.StatusOK)
		})
	})

	Describe("Get unknown profile", func() {
		It("Returns 404 response", func() {
			ctx.SM.GET(web.ProfileURL + "/unknown").
				Expect().
				Status(http.StatusNotFound)
		})
	})
})
