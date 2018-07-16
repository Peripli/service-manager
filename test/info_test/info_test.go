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
	"net/http/httptest"
	"os"
	"testing"

	"github.com/Peripli/service-manager/api/info"
	"github.com/Peripli/service-manager/test/common"
	"github.com/gavv/httpexpect"
	. "github.com/onsi/ginkgo"
)

func Test(t *testing.T) {
	os.Chdir("../..")
	RunSpecs(t, "Info Suite")
}

var _ = Describe("Info API", func() {
	var SM *httpexpect.Expect
	var smServer *httptest.Server
	var mockOauthServer *httptest.Server

	BeforeSuite(func() {
		mockOauthServer = common.SetupMockOAuthServer()
		smServer = httptest.NewServer(common.GetServerHandler(nil, mockOauthServer.URL))
		SM = httpexpect.New(GinkgoT(), smServer.URL)
	})

	AfterSuite(func() {
		if smServer != nil {
			smServer.Close()
		}
		if mockOauthServer != nil {
			mockOauthServer.Close()
		}
	})

	Describe("Get info handler", func() {
		It("Returns correct response", func() {
			SM.GET(info.URL).
				Expect().
				Status(http.StatusOK).
				JSON().Object().Value("token_issuer_url").String().Equal(mockOauthServer.URL)
		})
	})
})
