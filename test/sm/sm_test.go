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

package sm_test

import (
	"context"
	"net/http/httptest"
	"os"

	"github.com/Peripli/service-manager/pkg/web"

	"github.com/gavv/httpexpect"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"

	"github.com/Peripli/service-manager/pkg/sm"
	"github.com/Peripli/service-manager/test/common"
)

// TestServiceManager tests servermanager package
func TestServiceManager(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Manager Suite")
}

var _ = Describe("SM", func() {

	var serviceManagerServer *httptest.Server

	BeforeSuite(func() {
		os.Chdir("../..")
		os.Setenv("FILE_LOCATION", "test/common")
		os.Setenv("API_TOKEN_ISSUER_URL", common.SetupMockOAuthServer().URL)
	})

	AfterSuite(func() {
		os.Unsetenv("FILE_LOCATION")
		os.Unsetenv("API_TOKEN_ISSUER_URL")
	})

	AfterEach(func() {
		if serviceManagerServer != nil {
			serviceManagerServer.Close()
		}
	})

	Describe("New", func() {
		Context("with no filters or plugins", func() {
			It("should return server", func() {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				servicemanager := sm.New(ctx, cancel, sm.DefaultEnv()).Build()

				serviceManagerServer = httptest.NewServer(servicemanager.Server.Router)
				assertResponse(serviceManagerServer, "/v1/info", 200, "")
			})
		})

		Context("with filters", func() {
			It("should return server", func() {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				smanager := sm.New(ctx, cancel, sm.DefaultEnv())
				smanager.RegisterFilters(testFilter{})
				serviceManagerServer = httptest.NewServer(smanager.Build().Server.Router)
				assertResponse(serviceManagerServer, "/v1/info", 200, "")
			})
		})

	})
})

func assertResponse(serviceManagerServer *httptest.Server, url string, statusCode int, body string) {
	SM := httpexpect.New(GinkgoT(), serviceManagerServer.URL)
	resp := SM.GET(url).Expect().Status(statusCode)
	if body != "" {
		resp.Body().Equal(body)
	}
}

type testFilter struct {
}

func (tf testFilter) Name() string {
	return "testFilter"
}

func (tf testFilter) Run(next web.Handler) web.Handler {
	return web.HandlerFunc(func(request *web.Request) (*web.Response, error) {
		return &web.Response{
			StatusCode: 200,
			Body:       []byte("OK"),
		}, nil
	})
}

func (tf testFilter) RouteMatchers() []web.RouteMatcher {
	return []web.RouteMatcher{
		{
			Matchers: []web.Matcher{
				web.Path("**"),
			},
		},
	}
}
