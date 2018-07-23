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
	RouteMatchersValue []RouteMatcher
}

func newPluginSegment(name, method, pathPattern string, f Middleware) *pluginSegment {
	return &pluginSegment{
		NameValue: name,
		PluginOp:  f,
		RouteMatchersValue: []RouteMatcher{
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

func (dp *pluginSegment) RouteMatchers() []RouteMatcher {
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

//func(api *API) decomposePlugin(opName, method, pathPattern string, middleware types.Handler) {
func (api *API) decomposePlugin(plug Plugin) []Filter {
	filters := make([]Filter, 0)

	if p, ok := plug.(CatalogFetcher); ok {
		filter := newPluginSegment("FetchCatalog", http.MethodGet, "/v1/osb/*/v2/catalog/*", MiddlewareFunc(p.FetchCatalog))
		filters = append(filters, filter)
	}
	if p, ok := plug.(ServiceFetcher); ok {
		filter := newPluginSegment("FetchService", http.MethodGet, "/v1/osb/*/v2/service_instances/*", MiddlewareFunc(p.FetchService))
		filters = append(filters, filter)
	}
	if p, ok := plug.(Provisioner); ok {
		filter := newPluginSegment("Provision", http.MethodPut, "/v1/osb/*/v2/service_instances/*", MiddlewareFunc(p.Provision))
		filters = append(filters, filter)
	}
	if p, ok := plug.(ServiceUpdater); ok {
		filter := newPluginSegment("UpdateService", http.MethodPatch, "/v1/osb/*/v2/service_instances/*", MiddlewareFunc(p.UpdateService))
		filters = append(filters, filter)
	}
	if p, ok := plug.(Deprovisioner); ok {
		filter := newPluginSegment("Deprovision", http.MethodDelete, "/v1/osb/*/v2/service_instances/*", MiddlewareFunc(p.Deprovision))
		filters = append(filters, filter)
	}
	if p, ok := plug.(BindingFetcher); ok {
		filter := newPluginSegment("FetchBinding", http.MethodGet, "/v1/osb/*/v2/service_instances/*/service_bindings/*", MiddlewareFunc(p.FetchBinding))
		filters = append(filters, filter)
	}
	if p, ok := plug.(Binder); ok {
		filter := newPluginSegment("Bind", http.MethodPut, "/v1/osb/*/v2/service_instances/*/service_bindings/*", MiddlewareFunc(p.Bind))
		filters = append(filters, filter)
	}
	if p, ok := plug.(Unbinder); ok {
		filter := newPluginSegment("Unbind", http.MethodDelete, "/v1/osb/*/v2/service_instances/*/service_bindings/*", MiddlewareFunc(p.Unbind))
		filters = append(filters, filter)
	}

	return filters
}
