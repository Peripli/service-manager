package filters

import (
	"context"
	"net/http"

	"github.com/Peripli/service-manager/pkg/query"

	"github.com/Peripli/service-manager/pkg/types"

	"github.com/Peripli/service-manager/storage"

	"github.com/Peripli/service-manager/pkg/web"
)

const BrokersVisibilityFilterName = "BrokersFilterByVisibility"

func NewBrokersFilterByVisibility(repository storage.Repository) *BrokersFilterByVisibility {
	return &BrokersFilterByVisibility{
		visibilityFilteringMiddleware: &visibilityFilteringMiddleware{
			FilteringFunc: newBrokersFilterFunc(repository),
		},
	}
}

type BrokersFilterByVisibility struct {
	*visibilityFilteringMiddleware
}

func newBrokersFilterFunc(repository storage.Repository) func(ctx context.Context, platformID string) (*query.Criterion, error) {
	return func(ctx context.Context, platformID string) (*query.Criterion, error) {
		planQuery, err := plansCriteria(ctx, repository, platformID)
		if err != nil {
			return nil, err
		}
		if planQuery == nil {
			return nil, nil
		}

		var servicesQuery *query.Criterion
		servicesQuery, err = servicesCriteria(ctx, repository, planQuery)
		if err != nil {
			return nil, err
		}
		if servicesQuery == nil {
			return nil, nil
		}

		brokersQuery, err := brokersCriteria(ctx, repository, servicesQuery)
		if err != nil {
			return nil, err
		}
		if brokersQuery == nil {
			return nil, nil
		}

		return brokersQuery, nil
	}
}

func brokersCriteria(ctx context.Context, repository storage.Repository, serviceQuery *query.Criterion) (*query.Criterion, error) {
	objectList, err := repository.List(ctx, types.ServiceOfferingType, *serviceQuery)
	if err != nil {
		return nil, err
	}
	offerings := objectList.(*types.ServiceOfferings)
	if offerings.Len() < 1 {
		return nil, nil
	}
	brokerIDs := make([]string, 0, offerings.Len())
	for _, p := range offerings.ServiceOfferings {
		brokerIDs = append(brokerIDs, p.BrokerID)
	}
	c := query.ByField(query.InOperator, "id", brokerIDs...)
	return &c, nil
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

func (vf *BrokersFilterByVisibility) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path(web.ServiceBrokersURL + "/**"),
				web.Methods(http.MethodGet),
			},
		},
	}
}

func (vf *BrokersFilterByVisibility) Name() string {
	return BrokersVisibilityFilterName
}
