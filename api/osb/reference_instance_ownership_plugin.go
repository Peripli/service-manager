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

const ReferenceInstanceOwnershipPluginName = "ReferenceInstanceOwnershipPlugin"

type referenceInstanceOwnershipPlugin struct {
	repository       storage.Repository
	tenantIdentifier string
}

// NewCheckPlatformIDPlugin creates new plugin that checks the platform_id of the instance
func NewReferenceInstanceOwnershipPlugin(repository storage.Repository, tenantIdentifier string) *referenceInstanceOwnershipPlugin {
	return &referenceInstanceOwnershipPlugin{
		repository:       repository,
		tenantIdentifier: tenantIdentifier,
	}
}

// Name returns the name of the plugin
func (p *referenceInstanceOwnershipPlugin) Name() string {
	return ReferenceInstanceOwnershipPluginName
}

// Deprovision intercepts deprovision requests and check if the instance is in the platform from where the request comes
func (p *referenceInstanceOwnershipPlugin) Deprovision(req *web.Request, next web.Handler) (*web.Response, error) {
	return p.assertPlatformID(req, next)
}

// UpdateService intercepts update service instance requests and check if the instance is in the platform from where the request comes
func (p *referenceInstanceOwnershipPlugin) UpdateService(req *web.Request, next web.Handler) (*web.Response, error) {
	return p.assertPlatformID(req, next)
}

// PollInstance intercepts poll instance operation requests and check if the instance is in the platform from where the request comes
func (p *referenceInstanceOwnershipPlugin) PollInstance(req *web.Request, next web.Handler) (*web.Response, error) {
	return p.assertPlatformID(req, next)
}

// Bind intercepts bind requests and check if the instance is in the platform from where the request comes
func (p *referenceInstanceOwnershipPlugin) Bind(req *web.Request, next web.Handler) (*web.Response, error) {
	return p.assertPlatformID(req, next)
}

// Unbind intercepts unbind requests and check if the instance is in the platform from where the request comes
func (p *referenceInstanceOwnershipPlugin) Unbind(req *web.Request, next web.Handler) (*web.Response, error) {
	return p.assertPlatformID(req, next)
}

// PollBinding intercepts poll binding operation requests and check if the instance is in the platform from where the request comes
func (p *referenceInstanceOwnershipPlugin) PollBinding(req *web.Request, next web.Handler) (*web.Response, error) {
	return p.assertPlatformID(req, next)
}

// FetchService intercepts get service instance requests and check if the instance owner is the same as the one requesting the operation
func (p *referenceInstanceOwnershipPlugin) FetchService(req *web.Request, next web.Handler) (*web.Response, error) {
	return p.assertPlatformID(req, next)
}

// FetchBinding intercepts get service binding requests and check if the instance owner is the same as the one requesting the operation
func (p *referenceInstanceOwnershipPlugin) FetchBinding(req *web.Request, next web.Handler) (*web.Response, error) {
	return p.assertPlatformID(req, next)
}

func (p *referenceInstanceOwnershipPlugin) assertPlatformID(req *web.Request, next web.Handler) (*web.Response, error) {
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
	req.Request = req.WithContext(types.ContextWithInstance(req.Context(), instance))

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
