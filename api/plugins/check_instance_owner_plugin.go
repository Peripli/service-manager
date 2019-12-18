package plugins

import (
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
	"github.com/tidwall/gjson"
	"net/http"
	"strings"
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
	ctx := req.Context()
	callerTenantID := gjson.GetBytes(req.Body, "context."+p.tenantIdentifier).String()
	pathSegments := strings.Split(req.URL.Path, "/")
	instanceID := pathSegments[len(pathSegments)-3] // /v2/service_instances/:instance_id/service_bindings/:binding_id
	byID := query.ByField(query.EqualsOperator, "id", instanceID)
	object, err := p.repository.Get(ctx, types.ServiceInstanceType, byID)
	if err != nil {
		if err != util.ErrNotFoundInStorage {
			return nil, util.HandleStorageError(err, string(types.ServiceInstanceType))
		} else {
			return next.Handle(req)
		}
	}
	instance := object.(*types.ServiceInstance)

	var instanceOwnerTenantID string
	if instance.Labels != nil {
		if tenantIDLabel, ok := instance.Labels[p.tenantIdentifier]; ok {
			instanceOwnerTenantID = tenantIDLabel[0]
		}
	}

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
