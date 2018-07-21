/*
 *    Copyright 2018 The Service Manager Authors
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
package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Peripli/service-manager/api"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/gavv/httpexpect"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Server Suite")
}

var sm *httpexpect.Expect

var _ = Describe("Server", func() {

	BeforeSuite(func() {
		api := &api.API{}
		route := api.Route{
			Endpoint: api.Endpoint{
				Path:   "/",
				Method: http.MethodGet,
			},
			Handler: testHandler,
		}
		testCtl := &testController{}
		testCtl.RegisterRoutes(route)
		api.RegisterControllers(testCtl)
		api.RegisterFilters(types.Filter{
			RouteMatcher: types.RouteMatcher{
				PathPattern: "**",
			},
			Middleware: testMiddleware,
		})
		serverSettings := Settings{
			Port:            0,
			RequestTimeout:  time.Second * 3,
			ShutdownTimeout: time.Second * 3,
		}
		server := New(api, serverSettings)
		Expect(server).ToNot(BeNil())
		testServer := httptest.NewServer(server.Handler)
		sm = httpexpect.New(GinkgoT(), testServer.URL)
	})

	Describe("newPluginSegment", func() {
		Context("when controller has panicing http.handler", func() {
			It("should return 500", func() {
				assertRecover("fail=true")
			})
		})

		Context("when controller has panicing filter", func() {
			It("should return 500", func() {
				assertRecover("filter_fail_before=true")
			})
		})

		Context("when controller has panicing filter", func() {
			It("should return 500", func() {
				assertRecover("filter_fail_after=true")
			})
		})
	})

})

func assertRecover(query string) {
	sm.GET("/").Expect().Status(http.StatusOK)
	sm.GET("/").WithQueryString(query).Expect().Status(http.StatusInternalServerError)
	sm.GET("/").Expect().Status(http.StatusOK)
}

type testController struct {
	testRoutes []api.Route
}

func (t *testController) RegisterRoutes(routes ...api.Route) {
	t.testRoutes = append(t.testRoutes, routes...)
}

func (t *testController) Routes() []api.Route {
	return t.testRoutes
}

func testHandler(req *types.Request) (*types.Response, error) {
	if req.URL.Query().Get("fail") == "true" {
		panic("expected")
	}
	resp := types.Response{}
	resp.StatusCode = http.StatusOK
	return &resp, nil
}

func testMiddleware(req *types.Request, next types.SMHandler) (*types.Response, error) {
	if req.URL.Query().Get("filter_fail_before") == "true" {
		panic("expected")
	}
	res, err := next(req)
	if req.URL.Query().Get("filter_fail_after") == "true" {
		panic("expected")
	}
	return res, err
}
