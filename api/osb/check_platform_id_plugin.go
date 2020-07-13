package osb

import (
	"net/http"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
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
	return p.assertPlatformID(req, next)
}

// UpdateService intercepts update service instance requests and check if the instance is in the platform from where the request comes
func (p *checkPlatformIDPlugin) UpdateService(req *web.Request, next web.Handler) (*web.Response, error) {
	return p.assertPlatformID(req, next)
}

// PollInstance intercepts poll instance operation requests and check if the instance is in the platform from where the request comes
func (p *checkPlatformIDPlugin) PollInstance(req *web.Request, next web.Handler) (*web.Response, error) {
	return p.assertPlatformID(req, next)
}

// Bind intercepts bind requests and check if the instance is in the platform from where the request comes
func (p *checkPlatformIDPlugin) Bind(req *web.Request, next web.Handler) (*web.Response, error) {
	return p.assertPlatformID(req, next)
}

// Unbind intercepts unbind requests and check if the instance is in the platform from where the request comes
func (p *checkPlatformIDPlugin) Unbind(req *web.Request, next web.Handler) (*web.Response, error) {
	return p.assertPlatformID(req, next)
}

// PollBinding intercepts poll binding operation requests and check if the instance is in the platform from where the request comes
func (p *checkPlatformIDPlugin) PollBinding(req *web.Request, next web.Handler) (*web.Response, error) {
	return p.assertPlatformID(req, next)
}

// FetchService intercepts get service instance requests and check if the instance owner is the same as the one requesting the operation
func (p *checkPlatformIDPlugin) FetchService(req *web.Request, next web.Handler) (*web.Response, error) {
	return p.assertPlatformID(req, next)
}

// FetchBinding intercepts get service binding requests and check if the instance owner is the same as the one requesting the operation
func (p *checkPlatformIDPlugin) FetchBinding(req *web.Request, next web.Handler) (*web.Response, error) {
	return p.assertPlatformID(req, next)
}

func (p *checkPlatformIDPlugin) assertPlatformID(req *web.Request, next web.Handler) (*web.Response, error) {
	ctx := req.Context()
	user, _ := web.UserFromContext(ctx)
	platform := &types.Platform{}
	if err := user.Data(platform); err != nil {
		return nil, err
	}
	if err := platform.Validate(); err != nil {
		log.C(ctx).WithError(err).Errorf("Invalid platform found in context")
		return nil, err
	}

	instanceID := req.PathParams["instance_id"]
	byID := query.ByField(query.EqualsOperator, "id", instanceID)
	object, err := p.repository.Get(ctx, types.ServiceInstanceType, byID)
	if err != nil {
		if err == util.ErrNotFoundInStorage {
			return next.Handle(req)
		}
		return nil, util.HandleStorageError(err, string(types.ServiceInstanceType))
	}

	instance, ok := object.(*types.ServiceInstance)
	if !ok {
		log.C(ctx).Errorf("Instance with id %s has not found", instance.ID)
		return nil, &util.HTTPError{
			ErrorType:   "NotFound",
			Description: "service instance not found",
			StatusCode:  http.StatusNotFound,
		}
	}
	req.Request = req.WithContext(web.ContextWithInstance(req.Context(), instance))

	if platform.ID != instance.PlatformID {
		log.C(ctx).Errorf("Instance with id %s and platform id %s does not belong to platform with id %s", instance.ID, instance.PlatformID, platform.ID)
		return nil, &util.HTTPError{
			ErrorType:   "NotFound",
			Description: "could not find such service instance",
			StatusCode:  http.StatusNotFound,
		}
	}

	return next.Handle(req)
}
