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
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/Peripli/service-manager/pkg/types"

	"github.com/Peripli/service-manager/pkg/env/envfakes"

	"github.com/Peripli/service-manager/config"
	"github.com/Peripli/service-manager/pkg/env"

	"github.com/gavv/httpexpect"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"net/http"

	"github.com/Peripli/service-manager/api/healthcheck"
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
		ctx         context.Context
		cancel      context.CancelFunc
		oauthServer *common.OAuthServer
		fakeEnv     *envfakes.FakeEnvironment
	)

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())
		oauthServer = common.NewOAuthServer()
		fakeEnv = &envfakes.FakeEnvironment{}
	})

	AfterEach(func() {
		defer cancel()
		oauthServer.Close()
	})

	Describe("New", func() {
		Context("when validating config fails", func() {
			It("should return error", func() {
				env, err := env.DefaultLogger(context.TODO(), config.AddPFlags, common.SetTestFileLocation)
				Expect(err).ToNot(HaveOccurred())
				env.Set("api.token_issuer_url", oauthServer.URL())
				env.Set("log.level", "invalid")

				cfg, err := config.New(env)
				Expect(err).ToNot(HaveOccurred())

				_, err = sm.New(ctx, cancel, fakeEnv, cfg)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("error validating configuration"))
			})
		})

		Context("when setting up storage with invalid uri", func() {
			It("should throw error during migrations setup", func() {
				env, err := env.DefaultLogger(context.TODO(), config.AddPFlags, common.SetTestFileLocation)
				Expect(err).ToNot(HaveOccurred())
				env.Set("api.token_issuer_url", oauthServer.URL())
				env.Set("storage.uri", "invalid")

				cfg, err := config.New(env)
				Expect(err).ToNot(HaveOccurred())

				_, err = sm.New(ctx, cancel, fakeEnv, cfg)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("error opening storage"))
			})
		})

		Context("when setting up API fails", func() {
			It("should return error", func() {
				env, err := env.DefaultLogger(context.TODO(), config.AddPFlags, common.SetTestFileLocation)
				Expect(err).ToNot(HaveOccurred())
				env.Set("api.token_issuer_url", "")

				cfg, err := config.New(env)
				Expect(err).ToNot(HaveOccurred())

				_, err = sm.New(ctx, cancel, fakeEnv, cfg)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when no API extensions are registered", func() {
			It("should return working service manager", func() {
				env, err := env.DefaultLogger(context.TODO(), config.AddPFlags, common.SetTestFileLocation)
				Expect(err).ToNot(HaveOccurred())
				env.Set("api.token_issuer_url", oauthServer.URL())

				cfg, err := config.New(env)
				Expect(err).ToNot(HaveOccurred())

				smanager, err := sm.New(ctx, cancel, fakeEnv, cfg)
				Expect(err).ToNot(HaveOccurred())

				verifyServiceManagerStartsSuccessFully(httptest.NewServer(smanager.Build().Server.Router))
			})
		})

		Context("when additional filter is registered", func() {
			It("should return working service manager with a new filter", func() {
				env, err := env.DefaultLogger(context.TODO(), config.AddPFlags, common.SetTestFileLocation)
				Expect(err).ToNot(HaveOccurred())
				env.Set("api.token_issuer_url", oauthServer.URL())

				cfg, err := config.New(env)
				Expect(err).ToNot(HaveOccurred())

				smanager, err := sm.New(ctx, cancel, fakeEnv, cfg)
				Expect(err).ToNot(HaveOccurred())

				smanager.RegisterFilters(testFilter{})

				SM := verifyServiceManagerStartsSuccessFully(httptest.NewServer(smanager.Build().Server.Router))

				SM.GET(web.InfoURL).
					Expect().
					Status(http.StatusOK).JSON().Object().Value("invoked").Equal("filter")
			})
		})

		Context("when additional controller is registered", func() {
			It("should return working service manager with additional controller", func() {
				env, err := env.DefaultLogger(context.TODO(), config.AddPFlags, common.SetTestFileLocation)
				Expect(err).ToNot(HaveOccurred())
				env.Set("api.token_issuer_url", oauthServer.URL())

				cfg, err := config.New(env)
				Expect(err).ToNot(HaveOccurred())

				smanager, err := sm.New(ctx, cancel, fakeEnv, cfg)
				Expect(err).ToNot(HaveOccurred())

				smanager.RegisterControllers(testController{})

				SM := verifyServiceManagerStartsSuccessFully(httptest.NewServer(smanager.Build().Server.Router))

				SM.GET("/v1/test").
					Expect().
					Status(http.StatusOK).JSON().Object().Value("invoked").Equal("controller")

			})
		})

		Context("Service Manager platform", func() {
			var ctx *common.TestContext
			var smPlatformID string
			verifySMPlatformExists := func() string {
				resp := ctx.SMWithOAuth.GET(web.PlatformsURL).WithQueryString(fmt.Sprintf("fieldQuery=name eq '%s'", types.SMPlatform)).
					Expect().Status(http.StatusOK).JSON()
				resp.Path("$.num_items").Number().Equal(1)
				resp.Path("$.items[*]").Array().First().Object().ContainsMap(map[string]interface{}{
					"id":   types.SMPlatform,
					"type": types.SMPlatform,
					"name": types.SMPlatform,
				})
				return resp.Path("$.items[*]").Array().First().Object().Value("id").String().Raw()
			}

			BeforeEach(func() {
				ctx = common.NewTestContextBuilder().Build()
				smPlatformID = verifySMPlatformExists()

			})

			AfterEach(func() {
				verifySMPlatformExists()
			})

			auths := []func() *common.SMExpect{
				func() *common.SMExpect {
					return ctx.SM
				},
				func() *common.SMExpect {
					return ctx.SMWithOAuth
				},
				func() *common.SMExpect {
					return ctx.SMWithOAuthForTenant
				},
				func() *common.SMExpect {
					return ctx.SMWithBasic
				},
			}

			for _, auth := range auths {
				auth := auth
				It("disallows bulk deleting of service manager platform", func() {

					auth().DELETE(web.PlatformsURL).WithQueryString(fmt.Sprintf("fieldQuery=name eq '%s'", types.SMPlatform)).
						Expect().Status(http.StatusNotFound)
				})

				It("disallows single deleting of service manager platform", func() {
					auth().DELETE(web.PlatformsURL + "/" + smPlatformID).
						Expect().Status(http.StatusNotFound)
				})

				It("disallows patching the service manager platform", func() {
					auth().PATCH(web.PlatformsURL + "/" + smPlatformID).WithJSON(common.Object{}).
						Expect().Status(http.StatusNotFound)
				})
			}
		})
	})
})

func verifyServiceManagerStartsSuccessFully(serviceManagerServer *httptest.Server) *common.SMExpect {
	SM := httpexpect.New(GinkgoT(), serviceManagerServer.URL)
	SM.GET(healthcheck.URL).
		Expect().
		Status(http.StatusOK).JSON().Object().ContainsMap(map[string]interface{}{
		"status": "UP",
	})
	return &common.SMExpect{Expect: SM}
}

func testHandler(identifier string) web.HandlerFunc {
	return func(request *web.Request) (*web.Response, error) {
		headers := http.Header{}
		headers.Add("Content-Type", "application/json")
		return &web.Response{
			StatusCode: 200,
			Header:     headers,
			Body:       []byte(`{"invoked": "` + identifier + `"}`),
		}, nil
	}
}

type testFilter struct {
}

func (tf testFilter) Name() string {
	return "testFilter"
}

func (tf testFilter) Run(request *web.Request, next web.Handler) (*web.Response, error) {
	return testHandler("filter")(request)
}

func (tf testFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path(web.InfoURL + "/*"),
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
