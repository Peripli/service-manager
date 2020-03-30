package osb

import (
	"context"
	"net/http"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
	"github.com/tidwall/gjson"
)

const CheckInstanceOwnerhipPluginName = "CheckInstanceOwnershipPlugin"

type checkInstanceOwnershipPlugin struct {
	repository       storage.Repository
	tenantIdentifier string
}

// NewCheckInstanceOwnershipPlugin creates new plugin that checks the owner of the instance
func NewCheckInstanceOwnershipPlugin(repository storage.Repository, tenantIdentifier string) *checkInstanceOwnershipPlugin {
	return &checkInstanceOwnershipPlugin{
		repository:       repository,
		tenantIdentifier: tenantIdentifier,
	}
}

// Name returns the name of the plugin
func (p *checkInstanceOwnershipPlugin) Name() string {
	return CheckInstanceOwnerhipPluginName
}

// Bind intercepts bind requests and check if the instance owner is the same as the one requesting the bind operation
func (p *checkInstanceOwnershipPlugin) Bind(req *web.Request, next web.Handler) (*web.Response, error) {
	return p.assertOwner(req, next)
}

// UpdateService intercepts update service instance requests and check if the instance owner is the same as the one requesting the operation
func (p *checkInstanceOwnershipPlugin) UpdateService(req *web.Request, next web.Handler) (*web.Response, error) {
	return p.assertOwner(req, next)
}

// FetchService intercepts get service instance requests and check if the instance owner is the same as the one requesting the operation
func (p *checkInstanceOwnershipPlugin) FetchService(req *web.Request, next web.Handler) (*web.Response, error) {
	return p.assertOwner(req, next)
}

// FetchBinding intercepts get service binding requests and check if the instance owner is the same as the one requesting the operation
func (p *checkInstanceOwnershipPlugin) FetchBinding(req *web.Request, next web.Handler) (*web.Response, error) {
	return p.assertOwner(req, next)
}

func (p *checkInstanceOwnershipPlugin) assertOwner(req *web.Request, next web.Handler) (*web.Response, error) {
	ctx := req.Context()
	callerTenantID := gjson.GetBytes(req.Body, "context."+p.tenantIdentifier).String()
	// if the request is GET there will be no request body, so we have to check the context for platform
	if req.Method == http.MethodGet {
		platform, err := extractPlatformFromContext(ctx)
		if err != nil {
			return nil, err
		}

		if platform.HasLabel(p.tenantIdentifier) {
			callerTenantID = platform.Labels[p.tenantIdentifier][0]
		}
	}

	if len(callerTenantID) == 0 {
		log.C(ctx).Info("Tenant identifier not found in request context.")
		return next.Handle(req)
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

	var instanceOwnerTenantID string
	if instance.Labels != nil {
		if tenantIDLabel, ok := instance.Labels[p.tenantIdentifier]; ok {
			instanceOwnerTenantID = tenantIDLabel[0]
		}
	}
	if instanceOwnerTenantID == "" {
		log.C(ctx).Infof("Tenant label for instance with id %s is missing.", instanceID)
		return next.Handle(req)
	}

	if instanceOwnerTenantID != callerTenantID {
		log.C(ctx).Errorf("Instance owner %s is not the same as the caller %s", instanceOwnerTenantID, callerTenantID)
		return nil, &util.HTTPError{
			ErrorType:   "NotFound",
			Description: "could not find such service instance",
			StatusCode:  http.StatusNotFound,
		}
	}

	return next.Handle(req)
}

func extractPlatformFromContext(ctx context.Context) (*types.Platform, error) {
	user, found := web.UserFromContext(ctx)
	if !found {
		return nil, &util.HTTPError{
			ErrorType:   "NotFound",
			Description: "No authenticated user found",
			StatusCode:  http.StatusNotFound,
		}
	}
	var platform types.Platform
	if err := user.Data(&platform); err != nil {
		return nil, err
	}

	return &platform, nil
}
