package osb

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
	"github.com/tidwall/gjson"
)

const CheckVisibilityPluginName = "CheckVisibilityPlugin"

type checkVisibilityPlugin struct {
	repository   storage.Repository
	platformType string
	labelKey     string
}

// NewCheckVisibilityPlugin creates new plugin that checks if a plan is visible to the user on provision request
func NewCheckVisibilityPlugin(repository storage.Repository, platformType, labelKey string) *checkVisibilityPlugin {
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
	byPlanID := query.ByField(query.EqualsOperator, "service_plan_id", planID)
	visibilitiesList, err := p.repository.List(ctx, types.VisibilityType, byPlanID)
	if err != nil {
		return nil, util.HandleStorageError(err, string(types.VisibilityType))
	}
	visibilities := visibilitiesList.(*types.Visibilities)

	switch platform.Type {
	case p.platformType:
		payloadOrgGUID := gjson.GetBytes(osbContext, p.labelKey).String()
		if len(payloadOrgGUID) == 0 {
			log.C(ctx).Errorf("Could not find organization_guid in the context of the osb request.")
			return nil, fmt.Errorf("organization_guid missing in osb context")
		}
		for _, v := range visibilities.Visibilities {
			if v.PlatformID == "" {
				return next.Handle(req)
			}
			if v.PlatformID == platform.ID {
				if v.Labels == nil {
					return next.Handle(req)
				}
				orgGUIDs, ok := v.Labels[p.labelKey]
				if !ok {
					return next.Handle(req)
				}
				for _, orgGUID := range orgGUIDs {
					if payloadOrgGUID == orgGUID {
						return next.Handle(req)
					}
				}
			}
		}
		return nil, &util.HTTPError{
			ErrorType:   "NotFound",
			Description: fmt.Sprintf("could not find such %s", string(types.ServicePlanType)),
			StatusCode:  http.StatusNotFound,
		}
	default:
		for _, v := range visibilities.Visibilities {
			if v.PlatformID == "" {
				return next.Handle(req)
			}
			if v.PlatformID == platform.ID {
				return next.Handle(req)
			}
		}
		return nil, &util.HTTPError{
			ErrorType:   "NotFound",
			Description: fmt.Sprintf("could not find such %s", string(types.ServicePlanType)),
			StatusCode:  http.StatusNotFound,
		}
	}
}
