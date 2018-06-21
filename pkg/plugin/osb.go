package plugin

import "github.com/Peripli/service-manager/pkg/filter"

// Interfaces for OSB operations

// CatalogFetcher should be implemented by plugins that need to intercept OSB call for get catalog operation
type CatalogFetcher interface {
	FetchCatalog(req *filter.Request, next filter.Handler) (*filter.Response, error)
}

// Provisioner should be implemented by plugins that need to intercept OSB call for provision operation
type Provisioner interface {
	Provision(req *filter.Request, next filter.Handler) (*filter.Response, error)
}

// Deprovisioner should be implemented by plugins that need to intercept OSB call for deprovision operation
type Deprovisioner interface {
	Deprovision(req *filter.Request, next filter.Handler) (*filter.Response, error)
}

// ServiceUpdater should be implemented by plugins that need to intercept OSB call for update service operation
type ServiceUpdater interface {
	UpdateService(req *filter.Request, next filter.Handler) (*filter.Response, error)
}

// ServiceFetcher should be implemented by plugins that need to intercept OSB call for get service operation
type ServiceFetcher interface {
	FetchService(req *filter.Request, next filter.Handler) (*filter.Response, error)
}

// Binder should be implemented by plugins that need to intercept OSB call for bind service operation
type Binder interface {
	Bind(req *filter.Request, next filter.Handler) (*filter.Response, error)
}

// Unbinder should be implemented by plugins that need to intercept OSB call for unbind service operation
type Unbinder interface {
	Unbind(req *filter.Request, next filter.Handler) (*filter.Response, error)
}

// BindingFetcher should be implemented by plugins that need to intercept OSB call for unbind service operation
type BindingFetcher interface {
	FetchBinding(req *filter.Request, next filter.Handler) (*filter.Response, error)
}
