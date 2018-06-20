package rest

import (
	"github.com/Peripli/service-manager/pkg/filter"
	"github.com/Peripli/service-manager/pkg/plugin"
	"github.com/sirupsen/logrus"
)

// API is the primary point for REST API registration
type API struct {
	// Controllers contains the registered controllers
	Controllers []Controller

	// Filters contains the registered filters
	Filters []filter.Filter
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
func (api *API) RegisterFilters(filters ...filter.Filter) {
	api.Filters = append(api.Filters, filters...)
}

// RegisterPlugins registers a set of plugins
func (api *API) RegisterPlugins(plugins ...plugin.Plugin) {
	for _, plug := range plugins {
		if plug == nil {
			logrus.Panicln("Cannot add nil plugins")
		}
		match := false
		register := func(method string, pathPattern string, middleware filter.Middleware) {
			api.RegisterFilters(filter.Filter{
				RouteMatcher: filter.RouteMatcher{
					Methods:     []string{method},
					PathPattern: pathPattern,
				},
				Middleware: middleware,
			})
			match = true
		}

		if p, ok := plug.(plugin.CatalogFetcher); ok {
			register("GET", "/v1/osb/*/v2/catalog", p.FetchCatalog)
		}
		if p, ok := plug.(plugin.ServiceFetcher); ok {
			register("GET", "/v1/osb/*/v2/service_instances/*", p.FetchService)
		}
		if p, ok := plug.(plugin.Provisioner); ok {
			register("PUT", "/v1/osb/*/v2/service_instances/*", p.Provision)
		}
		if p, ok := plug.(plugin.ServiceUpdater); ok {
			register("PATCH", "/v1/osb/*/v2/service_instances/*", p.UpdateService)
		}
		if p, ok := plug.(plugin.Deprovisioner); ok {
			register("DELETE", "/v1/osb/*/v2/service_instances/*", p.Deprovision)
		}
		if p, ok := plug.(plugin.BindingFetcher); ok {
			register("GET", "/v1/osb/*/v2/service_instances/*/service_bindings/*", p.FetchBinding)
		}
		if p, ok := plug.(plugin.Binder); ok {
			register("PUT", "/v1/osb/*/v2/service_instances/*/service_bindings/*", p.Bind)
		}
		if p, ok := plug.(plugin.Unbinder); ok {
			register("DELETE", "/v1/osb/*/v2/service_instances/*/service_bindings/*", p.Unbind)
		}
		if !match {
			logrus.Panicf("%T is not a plugin", plug)
		}
	}

}
