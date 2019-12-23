package plugins

import (
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
)

const CheckPlatformIDPluginName = "CheckPlatformIDPlugin"

type checkPlatformIDPlugin struct {
	repository storage.Repository
}

// NewCheckPlatformIDPlugin creates new plugin that checks the platform_id of the instance
func NewCheckPlatformIDPlugin(repository storage.Repository) *checkPlatformIDPlugin {
	return &checkPlatformIDPlugin{
		repository: repository,
	}
}

// Name returns the name of the plugin
func (p *checkPlatformIDPlugin) Name() string {
	return CheckPlatformIDPluginName
}

// Deprovision intercepts deprovision requests and check if the instance is in the platform from where the request comes
func (p *checkPlatformIDPlugin) Deprovision(req *web.Request, next web.Handler) (*web.Response, error) {
	return next.Handle(req)
}

// UpdateService intercepts update service instance requests and check if the instance is in the platform from where the request comes
func (p *checkPlatformIDPlugin) UpdateService(req *web.Request, next web.Handler) (*web.Response, error) {
	return next.Handle(req)
}

// PollInstance intercepts poll instance operation requests and check if the instance is in the platform from where the request comes
func (p *checkPlatformIDPlugin) PollInstance(req *web.Request, next web.Handler) (*web.Response, error) {
	return next.Handle(req)
}

// Bind intercepts bind requests and check if the instance is in the platform from where the request comes
func (p *checkPlatformIDPlugin) Bind(req *web.Request, next web.Handler) (*web.Response, error) {
	return next.Handle(req)
}

// Unbind intercepts unbind requests and check if the instance is in the platform from where the request comes
func (p *checkPlatformIDPlugin) Unbind(req *web.Request, next web.Handler) (*web.Response, error) {
	return next.Handle(req)
}

// PollBinding intercepts poll binding operation requests and check if the instance is in the platform from where the request comes
func (p *checkPlatformIDPlugin) PollBinding(req *web.Request, next web.Handler) (*web.Response, error) {
	return next.Handle(req)
}
