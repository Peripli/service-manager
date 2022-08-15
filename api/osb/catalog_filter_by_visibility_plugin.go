package osb

import (
	"context"
	"fmt"
	"net/http"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/query"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/security"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/log"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/storage"
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
	visibleCatalogPlans, serviceOfferings, servicePlans, err := getVisiblePlansByBrokerIDAndPlatformID(ctx, c.repository, brokerID, platform)
	if err != nil {
		return nil, err
	}
	res.Body, err = filterCatalogByVisiblePlans(res.Body, visibleCatalogPlans, serviceOfferings, servicePlans)
	return res, err
}

func getVisiblePlansByBrokerIDAndPlatformID(ctx context.Context, repository storage.Repository, brokerID string, platform *types.Platform) (map[string]bool, map[string]string, map[string]string, error) {
	offeringIDs, offeringsMap, err := getOfferingIDsByBrokerID(ctx, repository, brokerID)
	if err != nil {
		return nil, nil, nil, err
	}

	if len(offeringIDs) == 0 {
		return map[string]bool{}, nil, nil, nil
	}

	plansList, err := repository.List(ctx, types.ServicePlanType, query.ByField(query.InOperator, "service_offering_id", offeringIDs...))
	if err != nil {
		log.C(ctx).Errorf("Could not get %s: %v", types.ServicePlanType, err)
		return nil, nil, nil, err
	}

	visibleCatalogPlans := make(map[string]bool)
	plansMap := make(map[string]string)
	if platform.Type == types.CFPlatformType {
		for i := 0; i < plansList.Len(); i++ {
			plan := plansList.ItemAt(i).(*types.ServicePlan)
			if plan.SupportsPlatformInstance(*platform) {
				visibleCatalogPlans[plan.CatalogID] = true
				plansMap[fmt.Sprintf("%s_%s", plan.CatalogID, plan.Name)] = plan.ID
			}
		}

		return visibleCatalogPlans, offeringsMap, plansMap, nil
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
		return nil, offeringsMap, plansMap, err
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
			plansMap[fmt.Sprintf("%s_%s", p.CatalogID, p.Name)] = p.ID
		}
	}
	return visibleCatalogPlans, offeringsMap, plansMap, nil
}

func getOfferingIDsByBrokerID(ctx context.Context, repository storage.Repository, brokerID string) ([]string, map[string]string, error) {
	offerings, err := repository.List(ctx, types.ServiceOfferingType, query.ByField(query.EqualsOperator, "broker_id", brokerID))
	if err != nil {
		log.C(ctx).Errorf("Could not get %s: %v", types.ServiceOfferingType, err)
		return nil, nil, err
	}

	offeringMap := make(map[string]string)
	offeringIDs := make([]string, 0, offerings.Len())
	for i := 0; i < offerings.Len(); i++ {
		offering, ok := offerings.ItemAt(i).(*types.ServiceOffering)
		if !ok {
			return nil, nil, fmt.Errorf("unable to cast object to service offering")
		}
		offeringMap[fmt.Sprintf("%s_%s", offering.CatalogID, offering.Name)] = offering.ID
		offeringIDs = append(offeringIDs, offerings.ItemAt(i).GetID())
	}
	return offeringIDs, offeringMap, nil
}

func filterCatalogByVisiblePlans(catalog []byte, visibleCatalogPlans map[string]bool, serviceOfferings map[string]string, servicePlans map[string]string) ([]byte, error) {
	var err error
	services := gjson.GetBytes(catalog, "services").Array()

	// loop services in reverse order to ease the removal of elements
	for i := len(services) - 1; i >= 0; i-- {
		servicePath := fmt.Sprintf(`services.%d`, i)
		serviceKey := fmt.Sprintf("%s_%s", services[i].Get("id").String(), services[i].Get("name").String())
		catalog, err = sjson.SetBytes(catalog, fmt.Sprintf(servicePath+`.metadata.sm_offering_id`), serviceOfferings[serviceKey])
		if err != nil {
			return nil, err
		}
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

		updatedService := gjson.GetBytes(catalog, servicePath)
		for j, p := range updatedService.Get("plans").Array() {
			planPath := fmt.Sprintf(servicePath+`.plans.%d`, j)
			planKey := fmt.Sprintf("%s_%s", p.Get("id").String(), p.Get("name").String())
			catalog, err = sjson.SetBytes(catalog, fmt.Sprintf(planPath+`.metadata.sm_plan_id`), servicePlans[planKey])
			if err != nil {
				return nil, err
			}
		}
	}

	return catalog, nil
}
