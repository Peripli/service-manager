package osb

import (
	"fmt"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/log"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/query"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/util"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/storage"
	"net/http"
)

const PlatformTerminationPluginName = "PlatformTerminationPlugin"

type platformTerminationPlugin struct {
	repository storage.Repository
}

// NewPlatformTerminationPlugin creates new plugin that checks the platform deletion status
func NewPlatformTerminationPlugin(repository storage.Repository) *platformTerminationPlugin {
	return &platformTerminationPlugin{
		repository: repository,
	}
}

// Name returns the name of the plugin
func (p *platformTerminationPlugin) Name() string {
	return PlatformTerminationPluginName
}

// UpdateService intercepts update service instance requests and check if the instance is in the platform from where the request comes
func (p *platformTerminationPlugin) UpdateService(req *web.Request, next web.Handler) (*web.Response, error) {
	return p.validateNotPendingTermination(req, next)
}

// Bind intercepts bind requests and check if the instance is in the platform from where the request comes
func (p *platformTerminationPlugin) Bind(req *web.Request, next web.Handler) (*web.Response, error) {
	return p.validateNotPendingTermination(req, next)
}

func (p *platformTerminationPlugin) Provision(req *web.Request, next web.Handler) (*web.Response, error) {
	return p.validateNotPendingTermination(req, next)
}

func (p *platformTerminationPlugin) validateNotPendingTermination(req *web.Request, next web.Handler) (*web.Response, error) {
	ctx := req.Context()
	user, _ := web.UserFromContext(ctx)
	if user == nil {
		return next.Handle(req)
	}
	platform := &types.Platform{}
	if err := user.Data(platform); err != nil {
		return nil, err
	}
	if err := platform.Validate(); err != nil {
		log.C(ctx).WithError(err).Errorf("Invalid platform found in context")
		return nil, err
	}
	platformID := platform.GetID()
	criteria := []query.Criterion{
		query.ByField(query.NotEqualsOperator, "cascade_root_id", ""),
		query.ByField(query.EqualsOperator, "type", string(types.DELETE)),
		query.ByField(query.EqualsOperator, "resource_type", string(types.PlatformType)),
		query.ByField(query.InOperator, "state", string(types.PENDING), string(types.IN_PROGRESS)),
		query.ByField(query.EqualsOperator, "resource_id", platformID),
	}
	operationCount, err := p.repository.Count(ctx, types.OperationType, criteria...)
	if err != nil {
		return nil, err
	}
	if operationCount > 0 {
		return nil, &util.HTTPError{
			ErrorType:   http.StatusText(http.StatusUnprocessableEntity),
			Description: fmt.Sprintf("Operations are not possible for resources of platform %s which is during termination", platformID),
			StatusCode:  http.StatusUnprocessableEntity,
		}
	}
	return next.Handle(req)
}
