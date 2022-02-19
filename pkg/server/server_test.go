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

	"github.com/Peripli/service-manager/api/filters"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/gavv/httpexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Server Suite")
}

var sm *httpexpect.Expect

var _ = Describe("Server", Ordered, func() {

	BeforeAll(func() {
		api := &web.API{}
		route := web.Route{
			Endpoint: web.Endpoint{
				Path:   "/",
				Method: http.MethodGet,
			},
			Handler: testHandler,
		}
		testCtl := &testController{}
		testCtl.RegisterRoutes(route)
		api.RegisterControllers(testCtl)
		api.RegisterFilters(&testFilter{})
		serverSettings := &Settings{
			Port:            0,
			RequestTimeout:  time.Second * 3,
			ShutdownTimeout: time.Second * 3,
		}
		server := New(serverSettings, api)
		server.Router.Use(filters.NewRecoveryMiddleware())
		Expect(server).ToNot(BeNil())
		testServer := httptest.NewServer(server.Router)
		sm = httpexpect.New(GinkgoT(), testServer.URL)
	})

	Describe("Panic Recovery", func() {
		Context("when controller has panicing http.handler", func() {
			It("should return 500", func() {
				assertRecover("fail=true")
			})
		})

		Context("when controller has panicing filter before delegating to next handler", func() {
			It("should return 500", func() {
				assertRecover("filter_fail_before=true")
			})
		})

		Context("when controller has panicing filter after delegating to next handler", func() {
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
	testRoutes []web.Route
}

func (t *testController) RegisterRoutes(routes ...web.Route) {
	t.testRoutes = append(t.testRoutes, routes...)
}

func (t *testController) Routes() []web.Route {
	return t.testRoutes
}

type testFilter struct {
}

func (tf testFilter) Name() string {
	return "testFilter"
}

func (tf testFilter) Run(request *web.Request, next web.Handler) (*web.Response, error) {
	if request.URL.Query().Get("filter_fail_before") == "true" {
		panic("expected")
	}
	res, err := next.Handle(request)
	if request.URL.Query().Get("filter_fail_after") == "true" {
		panic("expected")
	}
	return res, err
}

func (tf testFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path("**"),
			},
		},
	}
}

func testHandler(req *web.Request) (*web.Response, error) {
	if req.URL.Query().Get("fail") == "true" {
		panic("expected")
	}
	return &web.Response{
		StatusCode: http.StatusOK,
	}, nil
}
