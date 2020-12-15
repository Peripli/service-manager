package osb

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/security"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
)

const CatalogFilterByVisibilityPluginName = "CatalogFilterByVisibilityPlugin"

func NewCatalogFilterByVisibilityPlugin(repository storage.Repository) *CatalogFilterByVisibilityPlugin {
	return &CatalogFilterByVisibilityPlugin{
		repository: repository,
	}
}

func (c *CatalogFilterByVisibilityPlugin) Name() string {
	return CatalogFilterByVisibilityPluginName
}

type CatalogFilterByVisibilityPlugin struct {
	repository storage.Repository
}

func (c *CatalogFilterByVisibilityPlugin) FetchCatalog(req *web.Request, next web.Handler) (*web.Response, error) {
	ctx := req.Context()
	userCtx, ok := web.UserFromContext(ctx)
	if !ok {
		return nil, security.UnauthorizedHTTPError("no user found")
	}

	res, err := next.Handle(req)

	if err != nil || res.StatusCode != http.StatusOK {
		return res, err
	}

	if userCtx.AuthenticationType != web.Basic {
		log.C(ctx).Debugf("Authentication is %s, not basic. Skip filtering on visibilities", userCtx.AuthenticationType)
		return res, nil
	}
	platform := &types.Platform{}
	if err := userCtx.Data(platform); err != nil {
		return nil, err
	}

	brokerID := req.PathParams[BrokerIDPathParam]
	visibleCatalogPlans, err := getVisiblePlansByBrokerIDAndPlatformID(ctx, c.repository, brokerID, platform)
	if err != nil {
		return nil, err
	}
	res.Body, err = filterCatalogByVisiblePlans(res.Body, visibleCatalogPlans)
	return res, err
}

func getVisiblePlansByBrokerIDAndPlatformID(ctx context.Context, repository storage.Repository, brokerID string, platform *types.Platform) (map[string]bool, error) {
	offeringIDs, err := getOfferingIDsByBrokerID(ctx, repository, brokerID)
	if err != nil {
		return nil, err
	}

	if len(offeringIDs) == 0 {
		return map[string]bool{}, nil
	}

	plansList, err := repository.List(ctx, types.ServicePlanType, query.ByField(query.InOperator, "service_offering_id", offeringIDs...))
	if err != nil {
		log.C(ctx).Errorf("Could not get %s: %v", types.ServicePlanType, err)
		return nil, err
	}

	visibleCatalogPlans := make(map[string]bool)
	if platform.Type == types.CFPlatformType {
		for i := 0; i < plansList.Len(); i++ {
			plan := plansList.ItemAt(i).(*types.ServicePlan)
			if plan.SupportsPlatformInstance(*platform) {
				visibleCatalogPlans[plan.CatalogID] = true
			}
		}

		return visibleCatalogPlans, nil
	}

	planIDs := make([]string, 0, plansList.Len())
	for i := 0; i < plansList.Len(); i++ {
		planIDs = append(planIDs, plansList.ItemAt(i).GetID())
	}

	visibilitiesList, err := repository.ListNoLabels(ctx, types.VisibilityType,
		query.ByField(query.EqualsOrNilOperator, "platform_id", platform.ID),
		query.ByField(query.InOperator, "service_plan_id", planIDs...))
	if err != nil {
		log.C(ctx).Errorf("Could not get %s: %v", types.VisibilityType, err)
		return nil, err
	}
	visibilities := (visibilitiesList.(*types.Visibilities)).Visibilities
	visiblePlans := make(map[string]bool)
	for _, v := range visibilities {
		visiblePlans[v.ServicePlanID] = true
	}

	plans := (plansList.(*types.ServicePlans)).ServicePlans
	for _, p := range plans {
		if visiblePlans[p.ID] {
			visibleCatalogPlans[p.CatalogID] = true
		}
	}
	return visibleCatalogPlans, nil
}

func getOfferingIDsByBrokerID(ctx context.Context, repository storage.Repository, brokerID string) ([]string, error) {
	offerings, err := repository.List(ctx, types.ServiceOfferingType, query.ByField(query.EqualsOperator, "broker_id", brokerID))
	if err != nil {
		log.C(ctx).Errorf("Could not get %s: %v", types.ServiceOfferingType, err)
		return nil, err
	}

	offeringIDs := make([]string, 0, offerings.Len())
	for i := 0; i < offerings.Len(); i++ {
		offeringIDs = append(offeringIDs, offerings.ItemAt(i).GetID())
	}
	return offeringIDs, nil
}

func filterCatalogByVisiblePlans(catalog []byte, visibleCatalogPlans map[string]bool) ([]byte, error) {
	var err error
	services := gjson.GetBytes(catalog, "services").Array()

	// loop services in reverse order to ease the removal of elements
	for i := len(services) - 1; i >= 0; i-- {
		servicePath := fmt.Sprintf(`services.%d`, i)
		plans := services[i].Get("plans").Array()

		for j := len(plans) - 1; j >= 0; j-- {
			catalogPlanID := plans[j].Get("id").String()
			if !visibleCatalogPlans[catalogPlanID] {
				planPath := fmt.Sprintf(servicePath+`.plans.%d`, j)
				catalog, err = sjson.DeleteBytes(catalog, planPath)
				if err != nil {
					return nil, err
				}
			}
		}

		if len(gjson.GetBytes(catalog, servicePath+".plans").Array()) == 0 {
			catalog, err = sjson.DeleteBytes(catalog, servicePath)
			if err != nil {
				return nil, err
			}
		}
	}

	return catalog, nil
}
