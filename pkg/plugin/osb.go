package plugin

import "github.com/Peripli/service-manager/pkg/filter"

// Interfaces for OSB operations

type CatalogFetcher interface {
	FetchCatalog(req *filter.Request, next filter.Handler) (*filter.Response, error)
}

type Provisioner interface {
	Provision(req *filter.Request, next filter.Handler) (*filter.Response, error)
}

type Deprovisioner interface {
	Deprovision(req *filter.Request, next filter.Handler) (*filter.Response, error)
}

type ServiceUpdater interface {
	UpdateService(req *filter.Request, next filter.Handler) (*filter.Response, error)
}

type ServiceFetcher interface {
	FetchService(req *filter.Request, next filter.Handler) (*filter.Response, error)
}

type Binder interface {
	Bind(req *filter.Request, next filter.Handler) (*filter.Response, error)
}

type Unbinder interface {
	Unbind(req *filter.Request, next filter.Handler) (*filter.Response, error)
}

type BindingFetcher interface {
	FetchBinding(req *filter.Request, next filter.Handler) (*filter.Response, error)
}
