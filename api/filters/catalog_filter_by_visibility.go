package filters

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/Peripli/service-manager/api/osb"

	"github.com/Peripli/service-manager/pkg/query"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
)

const CatalogFilterByVisibilityName = "CatalogFilterByVisibility"

func NewCatalogFilterByVisibility(repository storage.Repository) *CatalogFilterByVisibility {
	return &CatalogFilterByVisibility{
		repository: repository,
	}
}

type CatalogFilterByVisibility struct {
	repository storage.Repository
}

func (vf *CatalogFilterByVisibility) Run(req *web.Request, next web.Handler) (*web.Response, error) {
	ctx := req.Context()
	userCtx, ok := web.UserFromContext(ctx)
	if !ok {
		return nil, errors.New("no user found")
	}
	if userCtx.AuthenticationType != web.Basic {
		log.C(ctx).Debugf("Authentication is %s, not basic so proceed without visibility filter criteria", userCtx.AuthenticationType)
		return next.Handle(req)
	}
	platform := &types.Platform{}
	if err := userCtx.Data(platform); err != nil {
		return nil, err
	}
	if platform.Type != k8sPlatformType {
		log.C(ctx).Debugf("Platform type is %s, which is not kubernetes. Skip filtering on visibilities", platform.Type)
		return next.Handle(req)
	}

	res, err := next.Handle(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return res, nil
	}

	brokerID := req.PathParams[osb.BrokerIDPathParam]
	offerings, err := vf.repository.List(ctx, types.ServiceOfferingType, query.ByField(query.EqualsOperator, "broker_id", brokerID))
	if err != nil {
		return nil, err
	}
	// TODO: If offerings is empty: return
	offeringIDs := make([]string, 0, offerings.Len())
	for i := 0; i < offerings.Len(); i++ {
		offeringIDs = append(offeringIDs, offerings.ItemAt(i).GetID())
	}

	// TODO: If plans is empty: return?
	plansList, err := vf.repository.List(ctx, types.ServicePlanType, query.ByField(query.InOperator, "service_offering_id", offeringIDs...))
	if err != nil {
		return nil, err
	}
	planIDs := make([]string, 0, plansList.Len())
	for i := 0; i < plansList.Len(); i++ {
		planIDs = append(planIDs, plansList.ItemAt(i).GetID())
	}

	plans := (plansList.(*types.ServicePlans)).ServicePlans
	visibilitiesList, err := vf.repository.List(ctx, types.VisibilityType,
		query.ByField(query.EqualsOrNilOperator, "platform_id", platform.ID),
		query.ByField(query.InOperator, "service_plan_id", planIDs...))
	if err != nil {
		return nil, err
	}
	visibilities := (visibilitiesList.(*types.Visibilities)).Visibilities
	visiblePlans := make(map[string]bool)
	for _, v := range visibilities {
		visiblePlans[v.ServicePlanID] = true
	}

	visibleCatalogPlans := make(map[string]bool)
	for _, p := range plans {
		if visiblePlans[p.ID] {
			visibleCatalogPlans[p.CatalogID] = true
		}
	}

	res.Body, err = filterCatalogByVisiblePlans(res.Body, visibleCatalogPlans)
	return res, err
}

func servicesCriteria(ctx context.Context, repository storage.Repository, planQuery *query.Criterion) (*query.Criterion, error) {
	objectList, err := repository.List(ctx, types.ServicePlanType, *planQuery)
	if err != nil {
		return nil, err
	}
	plans := objectList.(*types.ServicePlans)
	if plans.Len() < 1 {
		return nil, nil
	}
	serviceIDs := make([]string, 0, plans.Len())
	for _, p := range plans.ServicePlans {
		serviceIDs = append(serviceIDs, p.ServiceOfferingID)
	}
	c := query.ByField(query.InOperator, "id", serviceIDs...)
	return &c, nil
}

func plansCriteria(ctx context.Context, repository storage.Repository, platformID string) (*query.Criterion, error) {
	objectList, err := repository.List(ctx, types.VisibilityType, query.ByField(query.EqualsOrNilOperator, "platform_id", platformID))
	if err != nil {
		return nil, err
	}
	visibilityList := objectList.(*types.Visibilities)
	if visibilityList.Len() < 1 {
		return nil, nil
	}
	planIDs := make([]string, 0, visibilityList.Len())
	for _, vis := range visibilityList.Visibilities {
		planIDs = append(planIDs, vis.ServicePlanID)
	}
	c := query.ByField(query.InOperator, "id", planIDs...)
	return &c, nil
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

func (vf *CatalogFilterByVisibility) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path(web.OSBURL + "/*/v2/catalog"),
				web.Methods(http.MethodGet),
			},
		},
	}
}

func (vf *CatalogFilterByVisibility) Name() string {
	return CatalogFilterByVisibilityName
}
