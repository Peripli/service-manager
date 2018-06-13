package plugin

import . "github.com/Peripli/service-manager/pkg/filter"

type CatalogFetcher interface {
	FetchCatalog(req *Request, next Handler) (*Response, error)
}

type Provisioner interface {
	Provision(req *Request, next Handler) (*Response, error)
}

type Deprovisioner interface {
	Deprovision(req *Request, next Handler) (*Response, error)
}

type ServiceUpdater interface {
	UpdateService(req *Request, next Handler) (*Response, error)
}

type ServiceFetcher interface {
	FetchService(req *Request, next Handler) (*Response, error)
}

type Binder interface {
	Bind(req *Request, next Handler) (*Response, error)
}

type Unbinder interface {
	Unbind(req *Request, next Handler) (*Response, error)
}

type BindingFetcher interface {
	FetchBinding(req *Request, next Handler) (*Response, error)
}
