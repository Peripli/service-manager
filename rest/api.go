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

package rest

import (
	"net/http"

	"github.com/Peripli/service-manager/pkg/plugin"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/sirupsen/logrus"
)

// API is the primary point for REST API registration
type API struct {
	// Controllers contains the registered controllers
	Controllers []Controller

	// Filters contains the registered filters
	Filters []web.Filter
}

// RegisterControllers registers a set of controllers
func (api *API) RegisterControllers(controllers ...Controller) {
	for _, controller := range controllers {
		if controller == nil {
			logrus.Panicln("Cannot add nil controllers")
		}
		api.Controllers = append(api.Controllers, controller)
	}
}

// RegisterFilters registers a set of filters
func (api *API) RegisterFilters(filters ...web.Filter) {
	api.Filters = append(api.Filters, filters...)
}

// RegisterPlugins registers a set of plugins
// nolint: gocyclo
func (api *API) RegisterPlugins(plugins ...plugin.Plugin) {
	for _, plug := range plugins {
		if plug == nil {
			logrus.Panicln("Cannot add nil plugins")
		}
		match := false
		register := func(opName, method, pathPattern string, middleware web.Middleware) {
			api.RegisterFilters(web.Filter{
				Name: plug.Name() + "-" + opName,
				RouteMatcher: web.RouteMatcher{
					Methods:     []string{method},
					PathPattern: pathPattern,
				},
				Middleware: middleware,
			})
			match = true
		}

		if p, ok := plug.(plugin.CatalogFetcher); ok {
			register("FetchCatalog", http.MethodGet, "/v1/osb/*/v2/catalog", p.FetchCatalog)
		}
		if p, ok := plug.(plugin.ServiceFetcher); ok {
			register("FetchService", http.MethodGet, "/v1/osb/*/v2/service_instances/*", p.FetchService)
		}
		if p, ok := plug.(plugin.Provisioner); ok {
			register("Provision", http.MethodPut, "/v1/osb/*/v2/service_instances/*", p.Provision)
		}
		if p, ok := plug.(plugin.ServiceUpdater); ok {
			register("UpdateService", http.MethodPatch, "/v1/osb/*/v2/service_instances/*", p.UpdateService)
		}
		if p, ok := plug.(plugin.Deprovisioner); ok {
			register("Deprovision", http.MethodDelete, "/v1/osb/*/v2/service_instances/*", p.Deprovision)
		}
		if p, ok := plug.(plugin.BindingFetcher); ok {
			register("FetchBinding", http.MethodGet, "/v1/osb/*/v2/service_instances/*/service_bindings/*", p.FetchBinding)
		}
		if p, ok := plug.(plugin.Binder); ok {
			register("Bind", http.MethodPut, "/v1/osb/*/v2/service_instances/*/service_bindings/*", p.Bind)
		}
		if p, ok := plug.(plugin.Unbinder); ok {
			register("Unbind", http.MethodDelete, "/v1/osb/*/v2/service_instances/*/service_bindings/*", p.Unbind)
		}
		if !match {
			logrus.Panicf("%T does not implement any plugin operation", plug)
		}
	}

}
