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

package healthcheck_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/Peripli/service-manager/api/healthcheck"
	"github.com/Peripli/service-manager/test/common"
	"github.com/gavv/httpexpect"
	. "github.com/onsi/ginkgo"
)

func Test(t *testing.T) {
	os.Chdir("../..")
	RunSpecs(t, "Healthcheck Suite")
}

var _ = Describe("Healthcheck API", func() {

	var sm *httpexpect.Expect
	var testServer *httptest.Server

	BeforeSuite(func() {
		testServer = httptest.NewServer(common.GetServerHandler(nil))
		sm = httpexpect.New(GinkgoT(), testServer.URL)
	})

	AfterSuite(func() {
		if testServer != nil {
			testServer.Close()
		}
	})

	Describe("Get info handler", func() {
		It("Returns correct response", func() {
			responseObject := sm.GET(healthcheck.URL).
				Expect().
				Status(http.StatusOK).
				JSON().Object()
			responseObject.Value("status").String().Equal("UP")
			responseObject.Value("storage").Object().Value("status").String().Equal("UP")
		})
	})
})
