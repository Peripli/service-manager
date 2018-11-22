package service_plan

import (
	"net/http"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
)

const reqServicePlanID = "service_plan_id"

type Controller struct {
	ServicePlanStorage storage.ServicePlan
}

func (c *Controller) getServicePlan(r *web.Request) (*web.Response, error) {
	servicePlanID := r.PathParams[reqServicePlanID]
	ctx := r.Context()
	log.C(ctx).Debugf("Getting service plan with id %s", servicePlanID)

	servicePlan, err := c.ServicePlanStorage.Get(ctx, servicePlanID)
	if err = util.HandleStorageError(err, "service_plan", servicePlanID); err != nil {
		return nil, err
	}
	return util.NewJSONResponse(http.StatusOK, servicePlan)
}

func (c *Controller) ListServicePlans(r *web.Request) (*web.Response, error) {
	var servicePlans []*types.ServicePlan
	var err error
	ctx := r.Context()
	log.C(ctx).Debug("Listing service plans")

	query := r.URL.Query()
	catalogName := query.Get("catalog_name")
	if catalogName != "" {
		log.C(ctx).Debugf("Filtering list by catalog_name=%s", catalogName)
		servicePlans, err = c.ServicePlanStorage.ListByCatalogName(ctx, catalogName)
	} else {
		servicePlans, err = c.ServicePlanStorage.List(ctx)
	}
	if err != nil {
		return nil, err
	}

	return util.NewJSONResponse(http.StatusOK, struct {
		ServicePlans []*types.ServicePlan `json:"service_plans"`
	}{
		ServicePlans: servicePlans,
	})
}
