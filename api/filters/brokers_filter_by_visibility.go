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
			ListResourcesCriteria: brokersCriteriaFunc(repository),
			IsResourceVisible:     isBrokerVisible(repository),
		},
	}
}

type BrokersFilterByVisibility struct {
	*visibilityFilteringMiddleware
}

func isBrokerVisible(repository storage.Repository) func(ctx context.Context, brokerID, platformID string) (bool, error) {
	return func(ctx context.Context, brokerID, platformID string) (bool, error) {
		servicesList, err := repository.List(ctx, types.ServiceOfferingType, query.ByField(query.EqualsOperator, "broker_id", brokerID))
		if err != nil {
			return false, err
		}
		serviceIDs := make([]string, 0, servicesList.Len())
		for i := 0; i < servicesList.Len(); i++ {
			serviceIDs = append(serviceIDs, servicesList.ItemAt(i).GetID())
		}

		plansList, err := repository.List(ctx, types.ServicePlanType, query.ByField(query.InOperator, "service_offering_id", serviceIDs...))
		if err != nil {
			return false, err
		}
		planIds := make([]string, 0, plansList.Len())
		for i := 0; i < plansList.Len(); i++ {
			planIds = append(planIds, plansList.ItemAt(i).GetID())
		}

		cnt, err := repository.Count(ctx, types.VisibilityType, query.ByField(query.InOperator, "service_plan_id", planIds...),
			query.ByField(query.EqualsOrNilOperator, "platform_id", platformID))
		return cnt > 0, err
	}
}

func brokersCriteriaFunc(repository storage.Repository) func(ctx context.Context, platformID string) (*query.Criterion, error) {
	return func(ctx context.Context, platformID string) (*query.Criterion, error) {
		planQuery, err := plansCriteria(ctx, repository, platformID)
		if err != nil {
			return nil, err
		}
		if planQuery == nil {
			return nil, nil
		}

		servicesQuery, err := servicesCriteria(ctx, repository, planQuery)
		if err != nil {
			return nil, err
		}
		if servicesQuery == nil {
			return nil, nil
		}

		return brokersCriteria(ctx, repository, servicesQuery)
	}
}

func (vf *BrokersFilterByVisibility) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path(web.ServiceBrokersURL + "/*"),
				web.Methods(http.MethodGet),
			},
		},
	}
}

func (vf *BrokersFilterByVisibility) Name() string {
	return BrokersVisibilityFilterName
}
