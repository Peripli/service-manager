package osb

import (
	"encoding/json"

	"github.com/Peripli/service-manager/pkg/visibility"

	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
)

const CheckVisibilityPluginName = "CheckVisibilityPlugin"

type checkVisibilityPlugin struct {
	repository storage.Repository
	checker    *visibility.Checker
}

// NewCheckVisibilityPlugin creates new plugin that checks if a plan is visible to the user on provision request
func NewCheckVisibilityPlugin(repository storage.Repository) *checkVisibilityPlugin {
	return &checkVisibilityPlugin{
		repository: repository,
		checker:    visibility.NewChecker(repository, "cloudfoundry", "organization_guid"),
	}
}

// Name returns the name of the plugin
func (p *checkVisibilityPlugin) Name() string {
	return CheckVisibilityPluginName
}

// Provision intercepts provision requests and check if the plan is visible to the user making the request
func (p *checkVisibilityPlugin) Provision(req *web.Request, next web.Handler) (*web.Response, error) {
	ctx := req.Context()
	requestPayload := &provisionRequest{}
	if err := decodeRequestBody(req, requestPayload); err != nil {
		return nil, err
	}
	planID, err := findServicePlanIDByCatalogIDs(ctx, p.repository, requestPayload.BrokerID, requestPayload.ServiceID, requestPayload.PlanID)
	if err != nil {
		return nil, err
	}
	return p.checkVisibility(req, next, planID, requestPayload.RawContext)
}

// UpdateService intercepts update service instance requests and check if the new plan is visible to the user making the request
func (p *checkVisibilityPlugin) UpdateService(req *web.Request, next web.Handler) (*web.Response, error) {
	ctx := req.Context()
	requestPayload := &updateRequest{}
	if err := decodeRequestBody(req, requestPayload); err != nil {
		return nil, err
	}
	if len(requestPayload.PlanID) == 0 { // plan is not changed
		return next.Handle(req)
	}
	planID, err := findServicePlanIDByCatalogIDs(ctx, p.repository, requestPayload.BrokerID, requestPayload.ServiceID, requestPayload.PlanID)
	if err != nil {
		return nil, err
	}
	return p.checkVisibility(req, next, planID, requestPayload.RawContext)
}

func (p *checkVisibilityPlugin) checkVisibility(req *web.Request, next web.Handler, planID string, osbContext json.RawMessage) (*web.Response, error) {
	ctx := req.Context()
	platform, err := extractPlatformFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if err := p.checker.CheckVisibility(req, platform, planID, osbContext); err != nil {
		return nil, err
	}
	return next.Handle(req)
}
