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
	"strings"

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

func (dp *pluginSegment) Run(next Handler) Handler {
	return dp.PluginOp.Run(next)
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
	api.Filters = append(api.Filters, filters...)
}

func (api *API) registerFilterRelatively(filterName string, newFilter Filter, newFilterPosition func(filterPosition int) int) {
	registeredFilterPosition := api.filterPosition(filterName)
	filterPosition := newFilterPosition(registeredFilterPosition)
	api.Filters = append(api.Filters, nil)
	copy(api.Filters[filterPosition+1:], api.Filters[filterPosition:])
	api.Filters[filterPosition] = newFilter
}

func (api *API) RegisterFilterBefore(beforeFilterName string, filter Filter) {
	logrus.Debugf("Registering filter %s before %s", filter.Name(), beforeFilterName)
	api.registerFilterRelatively(beforeFilterName, filter, func(beforeFilterPosition int) int {
		if beforeFilterPosition == 0 {
			return 0
		}
		return beforeFilterPosition - 1
	})
}

// RegisterFilterAfter adds the given filter after the filter with the specified name
// If the filter with the specified name is not registered, the method panics
// If for some routes, the filter with the given name does not match the routes of the provided filter, then
// the filter will be registered at the place at which the filter with this name would have been, had it been
// configured to match route
//
// Example:
//  Flter with name A matches routes /api/v1/service_brokers and /api/v1/platforms.
//  Filter with name B matches routes /api/v1/service_brokers and /api/v1/service_instances
//  Filter with name C matches routes /api/v1/platforms and /api/v1/service_instances
//  Filter B is registered after filter A
// Calling RegisterFilterAfter("A", filterC) will result in the following:
//  calling /api/v1/service_brokers will go through A, then B, then reach the handler. Event though filter C is registered after filter A
// it is not executed because it doesn't match the route
//  calling /api/v1/platforms will go through A, then C, then reach the handler
//  calling /api/v1/service_instances will go through C, then B, then reach the handler
func (api *API) RegisterFilterAfter(afterFilterName string, filter Filter) {
	logrus.Debugf("Registering filter %s after %s", filter.Name(), afterFilterName)
	api.registerFilterRelatively(afterFilterName, filter, func(filterPosition int) int {
		return filterPosition + 1
	})
}

func (api *API) ReplaceFilter(replacedFilterName string, filter Filter) {
	logrus.Debugf("Replacing filter %s with %s", replacedFilterName, filter.Name())
	registeredFilterPosition := api.filterPosition(replacedFilterName)
	api.Filters[registeredFilterPosition] = filter
}

func (api *API) filterPosition(filterName string) int {
	registeredFilterPosition := -1
	for i := range api.Filters {
		registeredFilter := api.Filters[i]
		if registeredFilter.Name() == filterName {
			registeredFilterPosition = i
		}
	}
	if registeredFilterPosition < 0 {
		logrus.Panicf("Filter with name %s is not registered", filterName)
	}
	return registeredFilterPosition
}

// RegisterPlugins registers a set of plugins
func (api *API) RegisterPlugins(plugins ...Plugin) {
	for _, plugin := range plugins {
		pluginSegments := api.decomposePluginOrDie(plugin)
		api.RegisterFilters(pluginSegments...)
	}
}

func (api *API) RegisterPluginBefore(name string, plugin Plugin) {
	logrus.Debugf("Registering plugin %s before %s", plugin.Name(), name)
	api.registerPluginRelatively(name, plugin, api.RegisterFilterBefore)
}

func (api *API) RegisterPluginAfter(name string, plugin Plugin) {
	logrus.Debugf("Registering plugin %s after %s", plugin.Name(), name)
	api.registerPluginRelatively(name, plugin, api.RegisterFilterAfter)
}

func (api *API) registerPluginRelatively(relativeName string, plugin Plugin, relationFunc func(string, Filter)) {
	pluginSegments := api.decomposePluginOrDie(plugin)
	for i := range pluginSegments {
		pluginSegment := pluginSegments[i]
		relationFunc(relativeName, pluginSegment)
	}
}

func (api *API) decomposePluginOrDie(plugin Plugin) []Filter {
	pluginSegments := api.decomposePlugin(plugin)
	if len(pluginSegments) == 0 {
		logrus.Panicf("%T does not implement any plugin operation", plugin)
	}
	return pluginSegments
}

func (api *API) ReplacePlugin(name string, plugin Plugin) {
	logrus.Debugf("Replacing plugin %s with %s", name, plugin.Name())
	var replacedPluginPositions []int
	for i := range api.Filters {
		filter := api.Filters[i]
		if strings.HasPrefix(filter.Name(), name+":") {
			replacedPluginPositions = append(replacedPluginPositions, i)
		}
	}
	pluginSegments := api.decomposePluginOrDie(plugin)
	replacedPluginFiltersCount := len(replacedPluginPositions)
	newPluginFiltersCount := len(pluginSegments)

	i := 0
	for _, filterPosition := range replacedPluginPositions {
		if i >= newPluginFiltersCount {
			break
		}
		api.Filters[filterPosition] = pluginSegments[i]
		i++
	}

	// remove leftover filters
	// TODO: do while have filter with name prefix, remove them
	for j := i+1; j < replacedPluginFiltersCount; j++ {
		api.Filters = append(api.Filters[:j], api.Filters[j+1:]...)
	}

	// add new filters after the last one
	for j := i; j < newPluginFiltersCount; j++ {
		api.RegisterFilterAfter(pluginSegments[i-1].Name(), pluginSegments[i])
	}
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
