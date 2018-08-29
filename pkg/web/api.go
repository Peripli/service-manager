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

// Package web exposes the extension points of the Service Manager. One can add additional controllers, filters
// and plugins to already built SM.
package web

import (
	"net/http"

	"github.com/Peripli/service-manager/pkg/util/slice"
	"github.com/sirupsen/logrus"
)

// API is the primary point for REST API registration
type API struct {
	// Controllers contains the registered controllers
	Controllers []Controller

	// Filters contains the registered filters
	Filters []Filter
}

// pluginSegment represents one piece of a web.invalidPlugin. Each web.invalidPlugin is decomposed into as many plugin segments as
// the count of OSB operations it provides. Each pluginSegment is treated as a web.Filter.
type pluginSegment struct {
	NameValue          string
	PluginOp           Middleware
	RouteMatchersValue []FilterMatcher
}

// newPluginSegment creates a plugin segment with the specified Middleware function and name matching the
// specified path and method
func newPluginSegment(name, method, pathPattern string, f Middleware) *pluginSegment {
	return &pluginSegment{
		NameValue: name,
		PluginOp:  f,
		RouteMatchersValue: []FilterMatcher{
			{
				Matchers: []Matcher{
					Methods(method),
					Path(pathPattern),
				},
			},
		},
	}
}

func (dp *pluginSegment) Run(request *Request, next Handler) (*Response, error) {
	return dp.PluginOp.Run(request, next)
}

func (dp *pluginSegment) Name() string {
	return dp.NameValue
}

func (dp *pluginSegment) FilterMatchers() []FilterMatcher {
	return dp.RouteMatchersValue
}

// RegisterControllers registers a set of controllers
func (api *API) RegisterControllers(controllers ...Controller) {
	for _, controller := range controllers {
		api.Controllers = append(api.Controllers, controller)
	}
}

// RegisterFilters registers a set of filters
func (api *API) RegisterFilters(filters ...Filter) {
	api.validateFilters(filters...)
	api.Filters = append(api.Filters, filters...)
}

// RegisterFilterBefore registers the specified filter before the one with the given name.
// If for some routes, the filter with the given name does not match the routes of the provided filter, then
// the filter will be registered at the place at which the filter with this name would have been, had it been
// configured to match route.
func (api *API) RegisterFilterBefore(beforeFilterName string, filter Filter) {
	logrus.Debugf("Registering filter %s before %s", filter.Name(), beforeFilterName)
	api.validateFilters(filter)
	api.registerFilterRelatively(beforeFilterName, filter, func(beforeFilterPosition int) int {
		return beforeFilterPosition
	})
}

// RegisterFilterAfter registers the specified filter after the one with the given name.
// If for some routes, the filter with the given name does not match the routes of the provided filter, then
// the filter will be registered after the place at which the filter with this name would have been, had it been
// configured to match route.
func (api *API) RegisterFilterAfter(afterFilterName string, filter Filter) {
	logrus.Debugf("Registering filter %s after %s", filter.Name(), afterFilterName)
	api.validateFilters(filter)
	api.registerFilterRelatively(afterFilterName, filter, func(filterPosition int) int {
		return filterPosition + 1
	})
}

// ReplaceFilter registers the given filter in the place of the filter with the given name.
func (api *API) ReplaceFilter(replacedFilterName string, filter Filter) {
	logrus.Debugf("Replacing filter %s with %s", replacedFilterName, filter.Name())
	api.validateFilters(filter)
	registeredFilterPosition := api.findFilterPosition(replacedFilterName)
	api.Filters[registeredFilterPosition] = filter
}

// RemoveFilter removes the filter with the given name
func (api *API) RemoveFilter(name string) {
	position := api.findFilterPosition(name)
	copy(api.Filters[position:], api.Filters[position+1:])
	api.Filters[len(api.Filters)-1] = nil
	api.Filters = api.Filters[:len(api.Filters)-1]
}

// RegisterPlugins registers a set of plugins
func (api *API) RegisterPlugins(plugins ...Plugin) {
	for _, plugin := range plugins {
		registeredFilterNames := api.filterNames(api.Filters)
		if slice.StringsAnyPrefix(registeredFilterNames, plugin.Name()+":") {
			logrus.Panicf("Plugin %s is already registered", plugin.Name())
		}
		pluginSegments := api.decomposePluginOrDie(plugin)
		api.Filters = append(api.Filters, pluginSegments...)
	}
}

func (api *API) validateFilters(filters ...Filter) {
	newFilterNames := api.filterNames(filters)
	if slice.StringsAnyEquals(newFilterNames, "") {
		logrus.Panicf("Filters cannot have empty names")
	}
	registeredFilterNames := api.filterNames(api.Filters)
	commonFilterNames := slice.StringsIntersection(registeredFilterNames, newFilterNames)
	if len(commonFilterNames) > 0 {
		logrus.Panicf("Filters %q are already registered", commonFilterNames)
	}
	filterNamesWithColon := slice.StringsContaining(newFilterNames, ":")
	if  len(filterNamesWithColon) > 0 {
		logrus.Panicf("Cannot register filters with : in their names. Invalid filter names: %q", filterNamesWithColon)
	}
}

func (api *API) registerFilterRelatively(filterName string, newFilter Filter, newFilterPosition func(filterPosition int) int) {
	registeredFilterPosition := api.findFilterPosition(filterName)
	filterPosition := newFilterPosition(registeredFilterPosition)
	api.Filters = append(api.Filters, nil)
	copy(api.Filters[filterPosition+1:], api.Filters[filterPosition:])
	api.Filters[filterPosition] = newFilter
}

func (api *API) findFilterPosition(filterName string) int {
	filterPosition := -1
	for i := range api.Filters {
		registeredFilter := api.Filters[i]
		if registeredFilter.Name() == filterName {
			filterPosition = i
			break
		}
	}
	if filterPosition < 0 {
		logrus.Panicf("Filter with name %s is not found", filterName)
	}
	return filterPosition
}

func (api *API) filterNames(filters []Filter) []string {
	var filterNames []string
	for i := range filters {
		filter := filters[i]
		filterNames = append(filterNames, filter.Name())
	}
	return filterNames
}

func (api *API) decomposePluginOrDie(plugin Plugin) []Filter {
	pluginSegments := api.decomposePlugin(plugin)
	if len(pluginSegments) == 0 {
		logrus.Panicf("%T does not implement any plugin operation", plugin)
	}
	return pluginSegments
}

// decomposePlugin decomposes a Plugin into multiple Filters
func (api *API) decomposePlugin(plug Plugin) []Filter {
	filters := make([]Filter, 0)

	if p, ok := plug.(CatalogFetcher); ok {
		filter := newPluginSegment(plug.Name()+":FetchCatalog", http.MethodGet, "/v1/osb/*/v2/catalog/*", MiddlewareFunc(p.FetchCatalog))
		filters = append(filters, filter)
	}
	if p, ok := plug.(ServiceFetcher); ok {
		filter := newPluginSegment(plug.Name()+":FetchService", http.MethodGet, "/v1/osb/*/v2/service_instances/*", MiddlewareFunc(p.FetchService))
		filters = append(filters, filter)
	}
	if p, ok := plug.(Provisioner); ok {
		filter := newPluginSegment(plug.Name()+":Provision", http.MethodPut, "/v1/osb/*/v2/service_instances/*", MiddlewareFunc(p.Provision))
		filters = append(filters, filter)
	}
	if p, ok := plug.(ServiceUpdater); ok {
		filter := newPluginSegment(plug.Name()+":UpdateService", http.MethodPatch, "/v1/osb/*/v2/service_instances/*", MiddlewareFunc(p.UpdateService))
		filters = append(filters, filter)
	}
	if p, ok := plug.(Deprovisioner); ok {
		filter := newPluginSegment(plug.Name()+":Deprovision", http.MethodDelete, "/v1/osb/*/v2/service_instances/*", MiddlewareFunc(p.Deprovision))
		filters = append(filters, filter)
	}
	if p, ok := plug.(BindingFetcher); ok {
		filter := newPluginSegment(plug.Name()+":FetchBinding", http.MethodGet, "/v1/osb/*/v2/service_instances/*/service_bindings/*", MiddlewareFunc(p.FetchBinding))
		filters = append(filters, filter)
	}
	if p, ok := plug.(Binder); ok {
		filter := newPluginSegment(plug.Name()+":Bind", http.MethodPut, "/v1/osb/*/v2/service_instances/*/service_bindings/*", MiddlewareFunc(p.Bind))
		filters = append(filters, filter)
	}
	if p, ok := plug.(Unbinder); ok {
		filter := newPluginSegment(plug.Name()+":Unbind", http.MethodDelete, "/v1/osb/*/v2/service_instances/*/service_bindings/*", MiddlewareFunc(p.Unbind))
		filters = append(filters, filter)
	}
	if p, ok := plug.(InstancePoller); ok {
		filter := newPluginSegment(plug.Name()+":PollInstance", http.MethodGet, "/v1/osb/*/v2/service_instances/*/last_operation", MiddlewareFunc(p.PollInstance))
		filters = append(filters, filter)
	}
	if p, ok := plug.(BindingPoller); ok {
		filter := newPluginSegment(plug.Name()+":PollBinding", http.MethodGet, "/v1/osb/*/v2/service_instances/*/service_bindings/*/last_operation", MiddlewareFunc(p.PollBinding))
		filters = append(filters, filter)
	}

	return filters
}
