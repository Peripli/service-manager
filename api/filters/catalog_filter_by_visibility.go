package filters

import (
	"context"
	"errors"
	"fmt"
	"net/http"

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
		return nil, errors.New("No user found")
	}
	if userCtx.AuthenticationType != web.Basic {
		log.C(ctx).Debugf("Authentication is %s, not basic so proceed without visibility filter criteria", userCtx.AuthenticationType)
		return next.Handle(req)
	}
	platform := &types.Platform{}
	if err := userCtx.Data(platform); err != nil {
		return nil, err
	}

	res, err := next.Handle(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return res, nil
	}

	plansQuery, err := plansCriteria(ctx, vf.repository, platform.ID)
	if err != nil {
		return nil, err
	}
	plans, err := vf.getPlans(ctx, plansQuery)
	if err != nil {
		return nil, err
	}

	var services []*types.ServiceOffering
	if plansQuery != nil {
		servicesQuery, err := servicesCriteria(ctx, vf.repository, plansQuery)
		if err != nil {
			return nil, err
		}
		services, err = vf.getServices(ctx, servicesQuery)
		if err != nil {
			return nil, err
		}
	}

	planIDs := make([]string, 0, len(plans))
	for _, plan := range plans {
		planIDs = append(planIDs, plan.ID)
	}
	offeringIDs := make([]string, 0, len(services))
	for _, offering := range services {
		offeringIDs = append(offeringIDs, offering.ID)
	}

	res.Body, err = deleteNotVisibleServices(ctx, vf.repository, res.Body, planIDs, offeringIDs)
	return res, err
	// services := gjson.GetBytes(res.Body, "services")
}

// deleteNotVisibleServices deletes the unsupported services from the catalog
func deleteNotVisibleServices(ctx context.Context, repository storage.Repository, catalog []byte, visiblePlanIDs []string, visibleServiceIDs []string) ([]byte, error) {
	var err error
	services := gjson.GetBytes(catalog, "services").Array()

	// loop services in reverse order to ease the removal of elements
	for i := len(services) - 1; i >= 0; i-- {
		servicePath := fmt.Sprintf(`services.%d`, i)
		plans := services[i].Get("plans").Array()

		catalog, err = deleteNotVisiblePlans(ctx, repository, plans, visiblePlanIDs, visibleServiceIDs, catalog, servicePath)
		if err != nil {
			return nil, err
		}

		plans = gjson.GetBytes(catalog, servicePath+".plans").Array()
		if len(plans) == 0 {
			catalog, err = sjson.DeleteBytes(catalog, servicePath)
			if err != nil {
				return nil, err
			}
		}
	}

	return catalog, err

}

func deleteNotVisiblePlans(ctx context.Context, repository storage.Repository, plans []gjson.Result, visiblePlanIDs []string, visibleServiceIDs []string, catalog []byte, servicePath string) ([]byte, error) {
	// loop plans in reverse order to ease the removal of elements
	for j := len(plans) - 1; j >= 0; j-- {
		isSupported, err := isPlanVisible(ctx, repository, plans[j], visiblePlanIDs, visibleServiceIDs)
		if err != nil {
			return nil, err
		}
		if !isSupported {
			planPath := fmt.Sprintf(servicePath+`.plans.%d`, j)
			var err error
			catalog, err = sjson.DeleteBytes(catalog, planPath)
			if err != nil {
				return nil, err
			}
		}
	}
	return catalog, nil
}

func isPlanVisible(ctx context.Context, repository storage.Repository, plan gjson.Result, visiblePlanIDs []string, visibleServiceIDs []string) (bool, error) {
	if len(visiblePlanIDs) < 1 || len(visibleServiceIDs) < 1 {
		return false, nil
	}
	catalogPlanID := plan.Get("id").String()
	list, err := repository.List(ctx, types.ServicePlanType,
		query.ByField(query.InOperator, "id", visiblePlanIDs...),
		query.ByField(query.InOperator, "service_offering_id", visibleServiceIDs...),
		query.ByField(query.EqualsOperator, "catalog_id", catalogPlanID))
	if err != nil {
		return false, err
	}
	return list.Len() > 0, nil
}

func (vf *CatalogFilterByVisibility) getPlans(ctx context.Context, q *query.Criterion) ([]*types.ServicePlan, error) {
	if q == nil {
		return nil, nil
	}
	list, err := vf.repository.List(ctx, types.ServicePlanType, *q)
	return (list.(*types.ServicePlans)).ServicePlans, err
}

func (vf *CatalogFilterByVisibility) getServices(ctx context.Context, q *query.Criterion) ([]*types.ServiceOffering, error) {
	if q == nil {
		return nil, nil
	}
	list, err := vf.repository.List(ctx, types.ServiceOfferingType, *q)
	return (list.(*types.ServiceOfferings)).ServiceOfferings, err
}

func (vf *CatalogFilterByVisibility) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path(web.OSBURL + "/**"),
				web.Methods(http.MethodGet),
			},
		},
	}
}

func (vf *CatalogFilterByVisibility) Name() string {
	return CatalogFilterByVisibilityName
}
