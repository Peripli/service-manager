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
	"testing"

	"github.com/Peripli/service-manager/pkg/web"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type bindUnbindPlugin struct {
}

func (bindUnbindPlugin) Bind(next web.Handler) web.Handler {
	return web.HandlerFunc(func(request *web.Request) (*web.Response, error) {
		return next.Handle(request)
	})
}

func (bindUnbindPlugin) Name() string {
	return "BindUnbindPlugin"
}

func (bindUnbindPlugin) Unbind(next web.Handler) web.Handler {
	return web.HandlerFunc(func(request *web.Request) (*web.Response, error) {
		return next.Handle(request)
	})
}

func TestWaeb(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Web Suite")
}

var _ = Describe("API", func() {
	var (
		api *web.API
	)

	BeforeEach(func() {
		api = &web.API{}
	})

	filterNames := func() []string {
		var names []string
		for i := range api.Filters {
			names = append(names, api.Filters[i].Name())
		}
		return names
	}

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

	Describe("Register Plugin Before", func() {
		Context("When plugin is not registered", func() {
			It("Should panic", func() {
				registerPlugin := func() { api.RegisterPluginBefore("some-plugin", &validPlugin{}) }
				Expect(registerPlugin).To(Panic())
			})
		})
		Context("When plugin is registered", func() {
			It("Should register all filters before the ones of the registered plugin", func() {
				bindPlugin := &bindUnbindPlugin{}
				provisionPlugin := &provisionDeprovisionPlugin{}
				api.RegisterPlugins(bindPlugin)
				api.RegisterPluginBefore(bindPlugin.Name(), provisionPlugin)
				names := filterNames()
				provisionFilters := names[:2]
				bindFilters := names[2:]
				Expect(len(names)).To(Equal(4))
				Expect(provisionFilters).To(ConsistOf([]string{provisionPlugin.Name() + ":Provision", provisionPlugin.Name() + ":Deprovision"}))
				Expect(bindFilters).To(ConsistOf([]string{bindPlugin.Name() + ":Bind", bindPlugin.Name() + ":Unbind"}))
			})
		})
	})

	Describe("Register Plugin After", func() {
		Context("When plugin is not registered", func() {
			It("Should panic", func() {
				registerPlugin := func() { api.RegisterPluginAfter("some-plugin", &validPlugin{}) }
				Expect(registerPlugin).To(Panic())
			})
		})
		Context("When plugin is registered", func() {
			It("Should register all filters after the ones of the registered plugin", func() {
				bindPlugin := &bindUnbindPlugin{}
				provisionPlugin := &provisionDeprovisionPlugin{}
				api.RegisterPlugins(bindPlugin)
				api.RegisterPluginAfter(bindPlugin.Name(), provisionPlugin)
				names := filterNames()
				bindFilters := names[:2]
				provisionFilters := names[2:]
				Expect(len(names)).To(Equal(4))
				Expect(provisionFilters).To(ConsistOf([]string{provisionPlugin.Name() + ":Provision", provisionPlugin.Name() + ":Deprovision"}))
				Expect(bindFilters).To(ConsistOf([]string{bindPlugin.Name() + ":Bind", bindPlugin.Name() + ":Unbind"}))
			})
		})
	})

	Describe("Replace Plugin", func() {
		Context("When both have equal number of filters", func() {
			It("Should replace all", func() {
				provisionerPlugin := &provisionDeprovisionPlugin{}
				binderPlugin := &bindUnbindPlugin{}
				firstFilter := &testFilter{}
				lastFilter := &testFilter2{}

				api.RegisterFilters(firstFilter)
				api.RegisterPlugins(provisionerPlugin)
				api.RegisterFilters(lastFilter)
				api.ReplacePlugin(provisionerPlugin.Name(), binderPlugin)

				expectedResult := []string{firstFilter.Name(), binderPlugin.Name() + ":Bind", binderPlugin.Name() + ":Unbind", lastFilter.Name()}
				names := filterNames()
				Expect(names).Should(ConsistOf(expectedResult))
			})
		})

		Context("When replaced has more filters than the new", func() {
			It("Should remove old and add new", func() {
				validPlugin := &validPlugin{}
				binderPlugin := &bindUnbindPlugin{}
				firstFilter := &testFilter{}
				lastFilter := &testFilter2{}

				api.RegisterFilters(firstFilter)
				api.RegisterPlugins(validPlugin)
				api.RegisterFilters(lastFilter)
				api.ReplacePlugin(validPlugin.Name(), binderPlugin)

				expectedResult := []string{firstFilter.Name(), binderPlugin.Name() + ":Bind", binderPlugin.Name() + ":Unbind", lastFilter.Name()}
				names := filterNames()
				Expect(names).Should(ConsistOf(expectedResult))
			})
		})
		Context("When replaced has less filters than the new", func() {
			It("Should remove old and add new", func() {
				validPlugin := &validPlugin{}
				binderPlugin := &bindUnbindPlugin{}
				firstFilter := &testFilter{}
				lastFilter := &testFilter2{}

				api.RegisterFilters(firstFilter)
				api.RegisterPlugins(binderPlugin)
				api.RegisterFilters(lastFilter)
				api.ReplacePlugin(binderPlugin.Name(), validPlugin)

				expectedResult := []string{
					firstFilter.Name(), validPlugin.Name() + ":Bind", validPlugin.Name() + ":Unbind",
					validPlugin.Name() + ":UpdateService", validPlugin.Name() + ":FetchService",
					validPlugin.Name() + ":FetchBinding", validPlugin.Name() + ":FetchCatalog",
					validPlugin.Name() + ":Provision", validPlugin.Name() + ":Deprovision", lastFilter.Name()}
				names := filterNames()
				Expect(names).Should(ConsistOf(expectedResult))
			})
		})
	})

	Describe("Replace Filter", func() {
		Context("When filter with such name does not exist", func() {
			It("Panics", func() {
				replaceFilter := func() {
					api.ReplaceFilter("some-filter", &testFilter{})
				}
				Expect(replaceFilter).To(Panic())
			})
		})
		Context("When filter with such name exists", func() {
			It("Replaces the filter", func() {
				filter := &testFilter{}
				newFilter := &testFilter2{}
				api.RegisterFilters(filter)
				api.ReplaceFilter(filter.Name(), newFilter)
				names := filterNames()
				Expect(names).To(ConsistOf([]string{newFilter.Name()}))
			})
		})
	})

	Describe("Register Filter Before", func() {
		Context("When filter with such name does not exist", func() {
			It("Panics", func() {
				replaceFilter := func() {
					api.RegisterFilterBefore("some-filter", &testFilter{})
				}
				Expect(replaceFilter).To(Panic())
			})
		})
		Context("When filter with such name exists", func() {
			It("Adds the filter before it", func() {
				filter := &testFilter{}
				newFilter := &testFilter2{}
				api.RegisterFilters(filter)
				api.RegisterFilterBefore(filter.Name(), newFilter)
				names := filterNames()
				Expect(names).To(Equal([]string{newFilter.Name(), filter.Name()}))
			})
		})
	})

	Describe("Register Filter After", func() {
		Context("When filter with such name does not exist", func() {
			It("Panics", func() {
				replaceFilter := func() {
					api.RegisterFilterAfter("some-filter", &testFilter{})
				}
				Expect(replaceFilter).To(Panic())
			})
		})
		Context("When filter with such name exists", func() {
			It("Adds the filter before it", func() {
				filter := &testFilter{}
				newFilter := &testFilter2{}
				api.RegisterFilters(filter)
				api.RegisterFilterAfter(filter.Name(), newFilter)
				names := filterNames()
				Expect(names).To(Equal([]string{filter.Name(), newFilter.Name()}))
			})
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

type testFilter2 struct {
}

func (testFilter2) Name() string {
	return "testFilter2"
}

func (testFilter2) Run(next web.Handler) web.Handler {
	return web.HandlerFunc(func(request *web.Request) (*web.Response, error) {
		return next.Handle(request)
	})
}

func (testFilter2) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{}
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

type provisionDeprovisionPlugin struct {
}

func (provisionDeprovisionPlugin) Deprovision(next web.Handler) web.Handler {
	return web.HandlerFunc(func(request *web.Request) (*web.Response, error) {
		return next.Handle(request)
	})
}

func (provisionDeprovisionPlugin) Name() string {
	return "ProvisionDeprovisionPlugin"
}

func (provisionDeprovisionPlugin) Provision(next web.Handler) web.Handler {
	return web.HandlerFunc(func(request *web.Request) (*web.Response, error) {
		return next.Handle(request)
	})
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
