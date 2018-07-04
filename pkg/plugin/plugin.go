package plugin

import "github.com/Peripli/service-manager/pkg/web"

// Plugin can intercept Service Manager operations and
// augment them with additional logic.
// To intercept SM operations a plugin implements one or more of
// the interfaces defined in this package.
type Plugin interface {
	Name() string
}

// Interfaces for OSB operations

// CatalogFetcher should be implemented by plugins that need to intercept OSB call for get catalog operation
type CatalogFetcher interface {
	FetchCatalog(req *web.Request, next web.Handler) (*web.Response, error)
}

// Provisioner should be implemented by plugins that need to intercept OSB call for provision operation
type Provisioner interface {
	Provision(req *web.Request, next web.Handler) (*web.Response, error)
}

// Deprovisioner should be implemented by plugins that need to intercept OSB call for deprovision operation
type Deprovisioner interface {
	Deprovision(req *web.Request, next web.Handler) (*web.Response, error)
}

// ServiceUpdater should be implemented by plugins that need to intercept OSB call for update service operation
type ServiceUpdater interface {
	UpdateService(req *web.Request, next web.Handler) (*web.Response, error)
}

// ServiceFetcher should be implemented by plugins that need to intercept OSB call for get service operation
type ServiceFetcher interface {
	FetchService(req *web.Request, next web.Handler) (*web.Response, error)
}

// Binder should be implemented by plugins that need to intercept OSB call for bind service operation
type Binder interface {
	Bind(req *web.Request, next web.Handler) (*web.Response, error)
}

// Unbinder should be implemented by plugins that need to intercept OSB call for unbind service operation
type Unbinder interface {
	Unbind(req *web.Request, next web.Handler) (*web.Response, error)
}

// BindingFetcher should be implemented by plugins that need to intercept OSB call for unbind service operation
type BindingFetcher interface {
	FetchBinding(req *web.Request, next web.Handler) (*web.Response, error)
}
