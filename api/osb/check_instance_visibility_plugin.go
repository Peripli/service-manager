package osb

import (
	"encoding/json"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
	"github.com/tidwall/gjson"
	"net/http"
)

const CheckVisibilityPluginName = "CheckVisibilityPlugin"

var errPlanNotAccessible = &util.HTTPError{
	ErrorType:   "ServicePlanNotFound",
	Description: "service plan not found or not accessible",
	StatusCode:  http.StatusNotFound,
}

type checkVisibilityPlugin struct {
	repository storage.Repository
}

// NewCheckVisibilityPlugin creates new plugin that checks if a plan is visible to the user on provision request
func NewCheckVisibilityPlugin(repository storage.Repository) *checkVisibilityPlugin {
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
	plan, err := findServicePlanByCatalogIDs(ctx, p.repository, requestPayload.BrokerID, requestPayload.ServiceID, requestPayload.PlanID)
	if err != nil {
		return nil, err
	}
	planID := plan.GetID()
	return p.checkVisibility(req, next, planID, requestPayload.RawContext)
}

// UpdateService intercepts update service instance requests and check if the new plan is visible to the user making the request
func (p *checkVisibilityPlugin) UpdateService(req *web.Request, next web.Handler) (*web.Response, error) {
	ctx := req.Context()
	requestPayload := &updateRequest{}
	if err := decodeRequestBody(req, requestPayload); err != nil {
		return nil, err
	}
	if len(requestPayload.PlanID) == 0 { // plan is not being updated
		return next.Handle(req)
	}
	plan, err := findServicePlanByCatalogIDs(ctx, p.repository, requestPayload.BrokerID, requestPayload.ServiceID, requestPayload.PlanID)
	if err != nil {
		return nil, err
	}
	planID := plan.GetID()

	byID := query.ByField(query.EqualsOperator, "id", requestPayload.InstanceID)
	instanceObj, err := p.repository.Get(ctx, types.ServiceInstanceType, byID)
	if err != nil {
		if err == util.ErrNotFoundInStorage {
			return next.Handle(req)
		}
		return nil, util.HandleStorageError(err, string(types.ServiceInstanceType))
	}
	instance := instanceObj.(*types.ServiceInstance)
	if instance.ServicePlanID == planID { // plan is not being updated
		return next.Handle(req)
	}

	return p.checkVisibility(req, next, planID, requestPayload.RawContext)
}

func (p *checkVisibilityPlugin) checkVisibility(req *web.Request, next web.Handler, planID string, osbContext json.RawMessage) (*web.Response, error) {
	ctx := req.Context()
	platform, err := ExtractPlatformFromContext(ctx)
	if err != nil {
		return nil, err
	}

	var payloadOrgGUID string
	var tenantKey string

	if platform.Type == "cloudfoundry" {
		if len(osbContext) == 0 {
			log.C(ctx).Errorf("Could not find context in the osb request.")
			return nil, &util.HTTPError{
				ErrorType:   "BadRequest",
				Description: "missing context in request body",
				StatusCode:  http.StatusBadRequest,
			}
		}
		payloadOrgGUID = gjson.GetBytes(osbContext, "organization_guid").String()
		if len(payloadOrgGUID) == 0 {
			log.C(ctx).Errorf("Could not find organization_guid in the context of the osb request.")
			return nil, &util.HTTPError{
				ErrorType:   "BadRequest",
				Description: "organization_guid missing in osb context",
				StatusCode:  http.StatusBadRequest,
			}
		}

		tenantKey = "organization_guid"
	}

	list, err := p.repository.QueryForList(ctx, types.VisibilityType, storage.QueryForVisibilityWithPlatformAndPlan, map[string]interface{}{
		"platform_id":     platform.ID,
		"service_plan_id": planID,
		"key":             tenantKey,
		"val":             payloadOrgGUID,
	})

	if err != nil {
		return nil, err
	}

	if list.Len() > 0 {
		return next.Handle(req)
	}

	log.C(ctx).Errorf("Service plan %v is not visible on platform %v", planID, platform.ID)
	return nil, errPlanNotAccessible
}
