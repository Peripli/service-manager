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

package sbproxy

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"context"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/Peripli/service-manager/pkg/env/envfakes"
	"github.com/Peripli/service-manager/pkg/sbproxy/platform"
	"github.com/Peripli/service-manager/pkg/sbproxy/platform/platformfakes"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/gavv/httpexpect"
	"github.com/spf13/pflag"
)

var _ = Describe("Sbproxy", func() {
	var ctx context.Context
	var fakePlatformClient *platformfakes.FakeClient
	var fakeEnv *envfakes.FakeEnvironment

	BeforeEach(func() {
		ctx = context.TODO()
		fakePlatformClient = &platformfakes.FakeClient{}
		fakeEnv = &envfakes.FakeEnvironment{}
	})

	Describe("New", func() {
		Context("when setting up config fails", func() {
			It("should panic", func() {
				fakeEnv.UnmarshalReturns(fmt.Errorf("error"))

				Expect(func() {
					New(ctx, fakeEnv, fakePlatformClient)
				}).To(Panic())
			})
		})

		Context("when validating config fails", func() {
			It("should panic", func() {
				Expect(func() {
					New(ctx, DefaultEnv(func(set *pflag.FlagSet) {
						set.Set("app.url", "http://localhost:8080")
						set.Set("sm.user", "")
						set.Set("sm.password", "admin")
						set.Set("sm.url", "http://localhost:8080")
						set.Set("sm.osb_api_path", "/osb")
						set.Set("log.level", "")
					}), fakePlatformClient)
				}).To(Panic())
			})
		})

		Context("when creating sm client fails due to missing config properties", func() {
			It("should panic", func() {
				Expect(func() {
					New(ctx, DefaultEnv(func(set *pflag.FlagSet) {
						set.Set("app.url", "http://localhost:8080")
						set.Set("sm.user", "")
						set.Set("sm.password", "admin")
						set.Set("sm.url", "http://localhost:8080")
						set.Set("sm.osb_api_path", "/osb")
					}), fakePlatformClient)
				}).To(Panic())
			})
		})

		Context("when no errors occur", func() {
			var SMProxy *httpexpect.Expect

			BeforeEach(func() {
				fakePlatformClient.GetBrokersReturns([]platform.ServiceBroker{}, nil)

				proxy := New(ctx, DefaultEnv(func(set *pflag.FlagSet) {
					set.Set("app.url", "http://localhost:8080")
					set.Set("sm.user", "admin")
					set.Set("sm.password", "admin")
					set.Set("sm.url", "http://localhost:8080")
					set.Set("sm.osb_api_path", "/osb")
				}), fakePlatformClient)
				proxy.RegisterControllers(testController{})
				SMProxy = httpexpect.New(GinkgoT(), httptest.NewServer(proxy.Build().Server.Router).URL)
			})

			It("bootstraps successfully", func() {
				SMProxy.GET("/").
					Expect().
					Status(http.StatusOK)
			})

			It("recovers from panics", func() {
				SMProxy.GET("/").Expect().Status(http.StatusOK)
				SMProxy.GET("/").WithQuery("panic", "true").Expect().Status(http.StatusInternalServerError)
				SMProxy.GET("/").Expect().Status(http.StatusOK)
			})
		})
	})
})

func testHandler() web.HandlerFunc {
	return web.HandlerFunc(func(request *web.Request) (*web.Response, error) {
		if request.URL.Query().Get("panic") == "true" {
			panic("expected")
		}
		headers := http.Header{}
		headers.Add("Content-Type", "application/json")
		return &web.Response{
			StatusCode: 200,
			Header:     headers,
			Body:       []byte(`{}`),
		}, nil
	})
}

type testController struct {
}

func (tc testController) Routes() []web.Route {
	return []web.Route{
		{
			Endpoint: web.Endpoint{
				Method: http.MethodGet,
				Path:   "/",
			},
			Handler: testHandler(),
		},
	}
}
