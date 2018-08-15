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
		Context("When argument is empty plugin", func() {
			It("Panics", func() {
				Expect(func() {
					api.RegisterPlugins(&invalidPlugin{})
				}).To(Panic())
			})
		})

		Context("When plugin is valid", func() {
			It("Successfully registers plugin", func() {
				originalCount := len(api.Filters)
				api.RegisterPlugins(&validPlugin{})
				Expect(len(api.Filters)).To(Equal(originalCount + 8))
			})
		})

		Context("When plugin with the same name is already registered", func() {
			It("Panics", func() {
				registerPlugin := func() {
					api.RegisterPlugins(&validPlugin{})
				}
				registerPlugin()
				Expect(registerPlugin).To(Panic())
			})
		})
	})

	Describe("Replace Filter", func() {
		Context("When filter with such name does not exist", func() {
			It("Panics", func() {
				replaceFilter := func() {
					api.ReplaceFilter("some-filter", &testFilter{"testFilter"})
				}
				Expect(replaceFilter).To(Panic())
			})
		})
		Context("When filter with such name exists", func() {
			It("Replaces the filter", func() {
				filter := &testFilter{"testFilter"}
				newFilter := &testFilter{"testFilter2"}
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
					api.RegisterFilterBefore("some-filter", &testFilter{"testFilter"})
				}
				Expect(replaceFilter).To(Panic())
			})
		})

		Context("When filter with such name exists", func() {
			It("Adds the filter before it", func() {
				filter1 := &testFilter{"testFilter"}
				filter2 := &testFilter{"testFilter2"}
				filter3 := &testFilter{"testFilter3"}
				api.RegisterFilters(filter1, filter2)
				api.RegisterFilterBefore(filter2.Name(), filter3)
				names := filterNames()
				Expect(names).To(Equal([]string{filter1.Name(), filter3.Name(), filter2.Name()}))
			})
		})
	})

	Describe("Register Filter After", func() {
		Context("When filter with such name does not exist", func() {
			It("Panics", func() {
				replaceFilter := func() {
					api.RegisterFilterAfter("some-filter", &testFilter{"testFilter"})
				}
				Expect(replaceFilter).To(Panic())
			})
		})
		Context("When filter with such name exists", func() {
			It("Adds the filter before it", func() {
				filter := &testFilter{"testFilter"}
				newFilter := &testFilter{"testFilter2"}
				api.RegisterFilters(filter)
				api.RegisterFilterAfter(filter.Name(), newFilter)
				names := filterNames()
				Expect(names).To(Equal([]string{filter.Name(), newFilter.Name()}))
			})
		})
	})

	Describe("Remove Filter", func() {
		Context("When filter with such name doest not exist", func() {
			It("Panics", func() {
				removeFilter := func() {
					api.RemoveFilter("some-filter")
				}
				Expect(removeFilter).To(Panic())
			})
		})
		Context("When filter exists", func() {
			It("Should remove it", func() {
				filter := &testFilter{"testFilter"}
				api.RegisterFilters(filter)
				names := filterNames()
				Expect(names).To(ConsistOf(filter.Name()))

				api.RemoveFilter(filter.Name())
				names = filterNames()
				Expect(names).To(BeEmpty())
			})
		})
	})

	Describe("RegisterFilters", func() {
		Context("When filter with such name does not exist", func() {
			It("increases filter count if successful", func() {
				originalCount := len(api.Filters)
				api.RegisterFilters(&testFilter{"testFilter"})
				Expect(len(api.Filters)).To(Equal(originalCount + 1))
			})
		})

		Context("When filter with such name already exists", func() {
			It("Panics", func() {
				registerFilter := func() {
					api.RegisterFilters(&testFilter{"testFilter"})
				}
				registerFilter()
				Expect(registerFilter).To(Panic())
			})
		})

		Context("When filter name contains :", func() {
			It("Panics", func() {
				registerFilter := func() {
					api.RegisterFilters(&testFilter{"name:"})
				}
				Expect(registerFilter).To(Panic())
			})
		})

		Context("When filter name is empty", func() {
			It("Panics", func() {
				registerFilter := func() {
					api.RegisterFilters(&testFilter{""})
				}
				Expect(registerFilter).To(Panic())
			})
		})
	})
})

type testController struct {
}

func (c *testController) Routes() []web.Route {
	return []web.Route{}
}

type testFilter struct {
	name string
}

func (tf testFilter) Name() string {
	return tf.name
}

func (tf testFilter) Run(request *web.Request, next web.Handler) (*web.Response, error) {
	return next.Handle(request)
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

func (c *validPlugin) UpdateService(request *web.Request, next web.Handler) (*web.Response, error) {
	return next.Handle(request)
}

func (c *validPlugin) Unbind(request *web.Request, next web.Handler) (*web.Response, error) {
	return next.Handle(request)
}

func (c *validPlugin) Bind(request *web.Request, next web.Handler) (*web.Response, error) {
	return next.Handle(request)
}

func (c *validPlugin) FetchBinding(request *web.Request, next web.Handler) (*web.Response, error) {
	return next.Handle(request)
}

func (c *validPlugin) Deprovision(request *web.Request, next web.Handler) (*web.Response, error) {
	return next.Handle(request)
}

func (c *validPlugin) Provision(request *web.Request, next web.Handler) (*web.Response, error) {
	return next.Handle(request)
}

func (c *validPlugin) FetchService(request *web.Request, next web.Handler) (*web.Response, error) {
	return next.Handle(request)
}

func (c *validPlugin) FetchCatalog(request *web.Request, next web.Handler) (*web.Response, error) {
	return next.Handle(request)
}

func (c *validPlugin) Name() string {
	return "validPlugin"
}
