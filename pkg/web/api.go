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

// RegisterPlugins registers a set of plugins
func (api *API) RegisterPlugins(plugins ...Plugin) {
	for _, plugin := range plugins {
		pluginSegments := api.decomposePlugin(plugin)
		if len(pluginSegments) == 0 {
			logrus.Panicf("%T does not implement any plugin operation", plugin)
		}

		api.RegisterFilters(pluginSegments...)
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
