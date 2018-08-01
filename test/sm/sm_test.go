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

	"github.com/gavv/httpexpect"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/spf13/pflag"

	"testing"

	"net/http"

	"errors"

	"github.com/Peripli/service-manager/api/healthcheck"
	"github.com/Peripli/service-manager/pkg/env/envfakes"
	"github.com/Peripli/service-manager/pkg/sm"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/test/common"
)

func TestServiceManager(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Manager Suite")
}

var _ = Describe("SM", func() {
	var (
		serviceManagerServer *httptest.Server
		ctx                  context.Context
		cancel               context.CancelFunc
	)

	BeforeSuite(func() {
		// TODO: storage must be refactored and so that context be in BeforeEach
		ctx, cancel = context.WithCancel(context.Background())
		os.Chdir("../..")
		os.Setenv("FILE_LOCATION", "test/common")
	})

	AfterSuite(func() {
		defer cancel()
		os.Unsetenv("FILE_LOCATION")
	})

	BeforeEach(func() {
		os.Setenv("API_TOKEN_ISSUER_URL", common.SetupFakeOAuthServer().URL)
	})

	AfterEach(func() {
		os.Unsetenv("API_TOKEN_ISSUER_URL")
		if serviceManagerServer != nil {
			serviceManagerServer.Close()
		}
	})

	Describe("New", func() {
		Context("when setting up config fails", func() {
			It("should panic", func() {
				fakeEnv := &envfakes.FakeEnvironment{}
				fakeEnv.UnmarshalReturns(errors.New("error"))

				Expect(func() {
					sm.New(ctx, cancel, fakeEnv)
				}).To(Panic())
			})
		})

		Context("when validating config fails", func() {
			It("should panic", func() {
				Expect(func() {
					sm.New(ctx, cancel, sm.DefaultEnv(func(set *pflag.FlagSet) {
						set.Set("log.level", "")
					}))
				}).To(Panic())
			})
		})

		Context("when setting up storage fails", func() {
			It("should panic", func() {
				Expect(func() {
					sm.New(ctx, cancel, sm.DefaultEnv(func(set *pflag.FlagSet) {
						set.Set("storage.uri", "invalid")
					}))
				}).To(Panic())
			})
		})

		Context("when setting up API fails", func() {
			It("should panic", func() {
				Expect(func() {
					sm.New(ctx, cancel, sm.DefaultEnv(func(set *pflag.FlagSet) {
						set.Set("api.token_issuer_url", "")
					}))
				}).To(Panic())
			})
		})

		Context("when no API extensions are registered", func() {
			It("should return working service manager", func() {
				smanager := sm.New(ctx, cancel, sm.DefaultEnv())

				verifyServiceManagerStartsSuccessFully(httptest.NewServer(smanager.Build().Server.Router))

			})
		})

		Context("when additional filter is registered", func() {
			It("should return working service manager with a new filter", func() {
				smanager := sm.New(ctx, cancel, sm.DefaultEnv())
				smanager.RegisterFilters(testFilter{})

				SM := verifyServiceManagerStartsSuccessFully(httptest.NewServer(smanager.Build().Server.Router))

				SM.GET("/v1/info").
					Expect().
					Status(http.StatusOK).JSON().Object().Value("invoked").Equal("filter")
			})
		})

		Context("when additional controller is registered", func() {
			It("should return working service manager with additional controller", func() {
				smanager := sm.New(ctx, cancel, sm.DefaultEnv())
				smanager.RegisterControllers(testController{})

				SM := verifyServiceManagerStartsSuccessFully(httptest.NewServer(smanager.Build().Server.Router))

				SM.GET("/v1/test").
					Expect().
					Status(http.StatusOK).JSON().Object().Value("invoked").Equal("controller")

			})
		})
	})
})

func verifyServiceManagerStartsSuccessFully(serviceManagerServer *httptest.Server) *httpexpect.Expect {
	SM := httpexpect.New(GinkgoT(), serviceManagerServer.URL)
	SM.GET(healthcheck.URL).
		Expect().
		Status(http.StatusOK).JSON().Object().ContainsMap(map[string]interface{}{
		"status": "UP",
		"storage": map[string]interface{}{
			"status": "UP",
		},
	})
	return SM
}

func testHandler(identifier string) web.HandlerFunc {
	return web.HandlerFunc(func(request *web.Request) (*web.Response, error) {
		headers := http.Header{}
		headers.Add("Content-Type", "application/json")
		return &web.Response{
			StatusCode: 200,
			Header:     headers,
			Body:       []byte(`{"invoked": "` + identifier + `"}`),
		}, nil
	})
}

type testFilter struct {
}

func (tf testFilter) Name() string {
	return "testFilter"
}

func (tf testFilter) Run(next web.Handler) web.Handler {
	return testHandler("filter")
}

func (tf testFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path("/v1/info/*"),
			},
		},
	}
}

type testController struct {
}

func (tc testController) Routes() []web.Route {
	return []web.Route{
		{
			Endpoint: web.Endpoint{
				Method: http.MethodGet,
				Path:   "/v1/test",
			},
			Handler: testHandler("controller"),
		},
	}
}
