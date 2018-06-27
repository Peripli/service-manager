package rest

import (
	"reflect"

	"github.com/Peripli/service-manager/pkg/filter"
	"github.com/Peripli/service-manager/pkg/plugin"
	"github.com/sirupsen/logrus"
)

type pluginMatcher struct {
	method, path string
}

type pluginContainer struct {
	methodName    string
	operationType reflect.Type
}

var pluginsMap = map[pluginMatcher]pluginContainer{
	pluginMatcher{"GET", "/v1/osb/*/v2/catalog"}:                                   pluginContainer{"FetchCatalog", reflect.TypeOf((*plugin.CatalogFetcher)(nil)).Elem()},
	pluginMatcher{"GET", "/v1/osb/*/v2/service_instances/*"}:                       pluginContainer{"FetchService", reflect.TypeOf((*plugin.ServiceFetcher)(nil)).Elem()},
	pluginMatcher{"PUT", "/v1/osb/*/v2/service_instances/*"}:                       pluginContainer{"Provision", reflect.TypeOf((*plugin.Provisioner)(nil)).Elem()},
	pluginMatcher{"PATCH", "/v1/osb/*/v2/service_instances/*"}:                     pluginContainer{"UpdateService", reflect.TypeOf((*plugin.ServiceUpdater)(nil)).Elem()},
	pluginMatcher{"DELETE", "/v1/osb/*/v2/service_instances/*"}:                    pluginContainer{"Deprovision", reflect.TypeOf((*plugin.Deprovisioner)(nil)).Elem()},
	pluginMatcher{"GET", "/v1/osb/*/v2/service_instances/*/service_bindings/*"}:    pluginContainer{"FetchBinding", reflect.TypeOf((*plugin.BindingFetcher)(nil)).Elem()},
	pluginMatcher{"PUT", "/v1/osb/*/v2/service_instances/*/service_bindings/*"}:    pluginContainer{"Bind", reflect.TypeOf((*plugin.Binder)(nil)).Elem()},
	pluginMatcher{"DELETE", "/v1/osb/*/v2/service_instances/*/service_bindings/*"}: pluginContainer{"Unbind", reflect.TypeOf((*plugin.Unbinder)(nil)).Elem()},
}

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

		ptype := reflect.TypeOf(plug)
		for k, v := range pluginsMap {
			if ptype.Implements(v.operationType) {
				method := reflect.ValueOf(plug).MethodByName(v.methodName)
				middlewareRef, ok := method.Interface().((func(*filter.Request, filter.Handler) (*filter.Response, error)))
				if ok {
					register(k.method, k.path, middlewareRef)
				}
			}
		}

		if !match {
			logrus.Panicf("%T is not a plugin", plug)
		}
	}
}
