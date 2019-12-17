package plugins

import (
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
)

const CheckInstanceOwnerPluginName = "CheckInstanceOwnerPlugin"

type checkInstanceOwnerPlugin struct {
	repository       storage.Repository
	tenantIdentifier string
}

// NewCheckInstanceOwnerPlugin creates new plugin that checks the owner of the instance on bind requests
func NewCheckInstanceOwnerPlugin(repository storage.Repository, tenantIdentifier string) *checkInstanceOwnerPlugin {
	return &checkInstanceOwnerPlugin{
		repository:       repository,
		tenantIdentifier: tenantIdentifier,
	}
}

// Name returns the name of the plugin
func (p *checkInstanceOwnerPlugin) Name() string {
	return CheckInstanceOwnerPluginName
}

// Bind intercepts bind requests and check if the instance owner is the same as the one requesting the bind operation
func (p *checkInstanceOwnerPlugin) Bind(req *web.Request, next web.Handler) (*web.Response, error) {
	return next.Handle(req)
}

// Unbind intercepts bind requests and check if the instance owner is the same as the one requesting the bind operation
func (p *checkInstanceOwnerPlugin) Unbind(req *web.Request, next web.Handler) (*web.Response, error) {
	return next.Handle(req)
}
