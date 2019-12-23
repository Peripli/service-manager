package plugins

import (
	"fmt"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
	"github.com/tidwall/gjson"
	"net/http"
)

const CheckInstanceOwnerPluginName = "CheckInstanceOwnerPlugin"

type checkInstanceOwnerPlugin struct {
	repository       storage.Repository
	tenantIdentifier string
}

// NewCheckInstanceOwnerPlugin creates new plugin that checks the owner of the instance
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
	return p.assertOwner(req, next)
}

// UpdateService intercepts update service instance requests and check if the instance owner is the same as the one requesting the operation
func (p *checkInstanceOwnerPlugin) UpdateService(req *web.Request, next web.Handler) (*web.Response, error) {
	return p.assertOwner(req, next)
}

func (p *checkInstanceOwnerPlugin) assertOwner(req *web.Request, next web.Handler) (*web.Response, error) {
	ctx := req.Context()
	callerTenantID := gjson.GetBytes(req.Body, "context."+p.tenantIdentifier).String()
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

	if instance.Labels == nil {
		return nil, fmt.Errorf("could not determine owner of service instance with id: %s", instanceID)
	}
	tenantIDLabel, ok := instance.Labels[p.tenantIdentifier]
	if !ok {
		return nil, fmt.Errorf("could not determine owner of service instance with id: %s", instanceID)
	}
	instanceOwnerTenantID := tenantIDLabel[0]

	if instanceOwnerTenantID != callerTenantID {
		log.C(ctx).Errorf("Instance owner %s is not the same as the caller %s", instanceOwnerTenantID, callerTenantID)
		return nil, &util.HTTPError{
			ErrorType:   "NotFound",
			Description: "Service instance not found or not visible to the current user.",
			StatusCode:  http.StatusNotFound,
		}
	}

	return next.Handle(req)
}
