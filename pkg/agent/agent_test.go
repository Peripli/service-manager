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

package agent

import (
	"github.com/Peripli/service-manager/pkg/env/envfakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"context"
	"net/http"
	"net/http/httptest"

	"github.com/Peripli/service-manager/pkg/agent/platform"
	"github.com/Peripli/service-manager/pkg/agent/platform/platformfakes"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/gavv/httpexpect"
	"github.com/spf13/pflag"
)

var _ = Describe("Sbproxy", func() {
	var ctx context.Context
	var cancel context.CancelFunc
	var fakePlatformClient *platformfakes.FakeClient
	var fakeBrokerClient *platformfakes.FakeBrokerClient
	var fakeEnvironment *envfakes.FakeEnvironment

	BeforeEach(func() {
		ctx = context.TODO()
		cancel = func() {}

		fakeBrokerClient = &platformfakes.FakeBrokerClient{}

		fakePlatformClient = &platformfakes.FakeClient{}
		fakePlatformClient.BrokerReturns(fakeBrokerClient)
		fakePlatformClient.VisibilityReturns(&platformfakes.FakeVisibilityClient{})
		fakePlatformClient.CatalogFetcherReturns(&platformfakes.FakeCatalogFetcher{})

		fakeEnvironment = &envfakes.FakeEnvironment{}
	})

	Describe("New", func() {
		Context("when validating config fails", func() {
			It("should panic", func() {
				env, err := DefaultEnv(context.TODO(), func(set *pflag.FlagSet) {
					set.Set("app.url", "http://localhost:8080")
					set.Set("app.legacy_url", "http://service-broker-proxy.domain.com")
					set.Set("sm.user", "")
					set.Set("sm.password", "admin")
					set.Set("sm.url", "http://localhost:8080")
					set.Set("sm.osb_api_path", "/osb")
					set.Set("log.level", "")
				})
				Expect(err).ToNot(HaveOccurred())
				settings, err := NewSettings(env)
				Expect(err).ToNot(HaveOccurred())
				smProxyBuilder, err := New(ctx, cancel, fakeEnvironment, settings, fakePlatformClient)
				Expect(err).To(HaveOccurred())
				Expect(smProxyBuilder).To(BeNil())
			})
		})

		Context("when creating sm client fails due to missing config properties", func() {
			It("should panic", func() {
				env, err := DefaultEnv(context.TODO(), func(set *pflag.FlagSet) {
					set.Set("app.url", "http://localhost:8080")
					set.Set("app.legacy_url", "http://service-broker-proxy.domain.com")
					set.Set("sm.user", "")
					set.Set("sm.password", "admin")
					set.Set("sm.url", "http://localhost:8080")
					set.Set("sm.osb_api_path", "/osb")
				})
				Expect(err).ToNot(HaveOccurred())
				settings, err := NewSettings(env)
				Expect(err).ToNot(HaveOccurred())
				smProxyBuilder, err := New(ctx, cancel, fakeEnvironment, settings, fakePlatformClient)
				Expect(err).To(HaveOccurred())
				Expect(smProxyBuilder).To(BeNil())
			})
		})

		Context("when no errors occur", func() {
			var SMProxy *httpexpect.Expect

			BeforeEach(func() {
				fakeBrokerClient.GetBrokersReturns([]*platform.ServiceBroker{}, nil)
				env, err := DefaultEnv(context.TODO(), func(set *pflag.FlagSet) {
					set.Set("app.url", "http://localhost:8080")
					set.Set("app.legacy_url", "http://service-broker-proxy.domain.com")
					set.Set("sm.user", "admin")
					set.Set("sm.password", "admin")
					set.Set("sm.url", "http://localhost:8080")
					set.Set("sm.osb_api_path", "/osb")
				})
				Expect(err).ToNot(HaveOccurred())
				settings, err := NewSettings(env)
				Expect(err).ToNot(HaveOccurred())
				proxy, err := New(ctx, cancel, fakeEnvironment, settings, fakePlatformClient)
				Expect(err).ToNot(HaveOccurred())
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
