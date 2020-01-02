package osb

import (
	"fmt"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
	"net/http"
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
	instance := object.(*types.ServiceInstance)

	if platform.ID != instance.PlatformID {
		log.C(ctx).Errorf("Operation with instance with id %s in platform with id %s "+
			"was performed from platform with id %s", instance.ID, instance.PlatformID, platform.ID)
		return nil, &util.HTTPError{
			ErrorType:   "NotFound",
			Description: fmt.Sprintf("could not find such %s", string(types.ServiceInstanceType)),
			StatusCode:  http.StatusNotFound,
		}
	}

	return next.Handle(req)
}
