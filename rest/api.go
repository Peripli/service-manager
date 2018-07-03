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
		register := func(method string, pathPattern string, middleware web.Middleware) {
			api.RegisterFilters(web.Filter{
				RouteMatcher: web.RouteMatcher{
					//Methods:     []string{method},
					PathPattern: pathPattern,
				},
				Middleware: middleware,
			})
			match = true
		}

		if p, ok := plug.(plugin.CatalogFetcher); ok {
			register(http.MethodGet, "/v1/osb/*/v2/catalog", p.FetchCatalog)
		}
		if p, ok := plug.(plugin.ServiceFetcher); ok {
			register(http.MethodGet, "/v1/osb/*/v2/service_instances/*", p.FetchService)
		}
		if p, ok := plug.(plugin.Provisioner); ok {
			register(http.MethodPut, "/v1/osb/*/v2/service_instances/*", p.Provision)
		}
		if p, ok := plug.(plugin.ServiceUpdater); ok {
			register(http.MethodPatch, "/v1/osb/*/v2/service_instances/*", p.UpdateService)
		}
		if p, ok := plug.(plugin.Deprovisioner); ok {
			register(http.MethodDelete, "/v1/osb/*/v2/service_instances/*", p.Deprovision)
		}
		if p, ok := plug.(plugin.BindingFetcher); ok {
			register(http.MethodGet, "/v1/osb/*/v2/service_instances/*/service_bindings/*", p.FetchBinding)
		}
		if p, ok := plug.(plugin.Binder); ok {
			register(http.MethodPut, "/v1/osb/*/v2/service_instances/*/service_bindings/*", p.Bind)
		}
		if p, ok := plug.(plugin.Unbinder); ok {
			register(http.MethodDelete, "/v1/osb/*/v2/service_instances/*/service_bindings/*", p.Unbind)
		}
		if !match {
			logrus.Panicf("%T is not a plugin", plug)
		}
	}

}
