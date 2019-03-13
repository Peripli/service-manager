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

package info_test

import (
	"net/http"
	"testing"

	"github.com/Peripli/service-manager/pkg/env"

	"github.com/Peripli/service-manager/api/info"
	"github.com/Peripli/service-manager/test/common"
	. "github.com/onsi/ginkgo"
)

func TestInfo(t *testing.T) {
	RunSpecs(t, "Info Suite")
}

var _ = Describe("Info API", func() {
	cases := []struct {
		description     string
		configBasicAuth bool
		expectBasicAuth bool
	}{
		{"Returns token_issuer_url and token_basic_auth: true", true, true},
		{"Returns token_issuer_url and token_basic_auth: false", false, false},
	}

	for _, tc := range cases {
		tc := tc
		var ctx *common.TestContext

		BeforeEach(func() {
			postHook := func(e env.Environment, servers map[string]common.FakeServer) {
				e.Set("api.token_basic_auth", tc.configBasicAuth)
			}
			ctx = common.NewTestContextBuilder().WithEnvPostExtensions(postHook).Build()
		})

		AfterEach(func() {
			ctx.Cleanup()
		})

		It(tc.description, func() {
			ctx.SM.GET(info.URL).
				Expect().
				Status(http.StatusOK).
				JSON().Object().Equal(common.Object{
				"token_issuer_url": ctx.Servers[common.OauthServer].URL(),
				"token_basic_auth": tc.expectBasicAuth,
			})
		})
	}
})
