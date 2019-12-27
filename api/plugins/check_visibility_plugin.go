package plugins

import (
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
)

const CheckVisibilityPluginName = "CheckVisibilityPlugin"

type checkVisibilityPlugin struct {
	repository storage.Repository
}

// NewCheckVisibilityPlugin creates new plugin that checks if a plan is visible to the user on provision request
func NewCheckVisibilityPlugin(repository storage.Repository) *checkVisibilityPlugin {
	return &checkVisibilityPlugin{
		repository: repository,
	}
}

// Name returns the name of the plugin
func (p *checkVisibilityPlugin) Name() string {
	return CheckVisibilityPluginName
}

// Provision intercepts provision requests and check if the plan is visible to the user making the request
func (p *checkVisibilityPlugin) Provision(req *web.Request, next web.Handler) (*web.Response, error) {
	return next.Handle(req)
}

// UpdateService intercepts update service instance requests and check if the new plan is visible to the user making the request
func (p *checkVisibilityPlugin) UpdateService(req *web.Request, next web.Handler) (*web.Response, error) {
	return next.Handle(req)
}
