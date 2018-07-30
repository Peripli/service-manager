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

package web_test

import (
	"github.com/Peripli/service-manager/pkg/web"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("API", func() {
	var (
		api *web.API
	)

	BeforeEach(func() {
		api = &web.API{}
	})

	Describe("RegisterControllers", func() {
		It("increases controllers count", func() {
			originalCount := len(api.Controllers)
			api.RegisterControllers(&testController{})
			Expect(len(api.Controllers)).To(Equal(originalCount + 1))
		})
	})

	Describe("RegisterPlugins", func() {
		It("panics if argument is an empty plugin", func() {
			Expect(func() {
				api.RegisterPlugins(&invalidPlugin{})
			}).To(Panic())
		})

		It("increases filter count if successful", func() {
			originalCount := len(api.Filters)
			api.RegisterPlugins(&validPlugin{})
			Expect(len(api.Filters)).To(Equal(originalCount + 8))
		})

	})

	Describe("RegisterFilters", func() {
		It("increases filter count if successful", func() {
			originalCount := len(api.Filters)
			api.RegisterFilters(&testFilter{})
			Expect(len(api.Filters)).To(Equal(originalCount + 1))
		})
	})
})

type testController struct {
}

func (c *testController) Routes() []web.Route {
	return []web.Route{}
}

type testFilter struct {
}

func (tf testFilter) Name() string {
	return "testFilter"
}

func (tf testFilter) Run(next web.Handler) web.Handler {
	return web.HandlerFunc(func(request *web.Request) (*web.Response, error) {
		return next.Handle(request)
	})
}

func (tf testFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{}
}

type invalidPlugin struct {
}

func (p *invalidPlugin) Name() string {
	return "invalidPlugin"
}

type validPlugin struct {
}

func (c *validPlugin) UpdateService(next web.Handler) web.Handler {
	return web.HandlerFunc(func(request *web.Request) (*web.Response, error) {
		return next.Handle(request)
	})
}

func (c *validPlugin) Unbind(next web.Handler) web.Handler {
	return web.HandlerFunc(func(request *web.Request) (*web.Response, error) {
		return next.Handle(request)

	})
}

func (c *validPlugin) Bind(next web.Handler) web.Handler {
	return web.HandlerFunc(func(request *web.Request) (*web.Response, error) {
		return next.Handle(request)

	})
}

func (c *validPlugin) FetchBinding(next web.Handler) web.Handler {
	return web.HandlerFunc(func(request *web.Request) (*web.Response, error) {
		return next.Handle(request)
	})
}

func (c *validPlugin) Deprovision(next web.Handler) web.Handler {
	return web.HandlerFunc(func(request *web.Request) (*web.Response, error) {
		return next.Handle(request)
	})
}

func (c *validPlugin) Provision(next web.Handler) web.Handler {
	return web.HandlerFunc(func(request *web.Request) (*web.Response, error) {
		return next.Handle(request)
	})
}

func (c *validPlugin) FetchService(next web.Handler) web.Handler {
	return web.HandlerFunc(func(request *web.Request) (*web.Response, error) {
		return next.Handle(request)
	})
}

func (c *validPlugin) FetchCatalog(next web.Handler) web.Handler {
	return web.HandlerFunc(func(request *web.Request) (*web.Response, error) {
		return next.Handle(request)
	})
}

func (c *validPlugin) Name() string {
	return "validPlugin"
}
