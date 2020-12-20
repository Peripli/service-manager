package filters

import (
	"context"
	"net/http"

	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"

	"github.com/Peripli/service-manager/storage"

	"github.com/Peripli/service-manager/pkg/web"
)

const ServicesVisibilityFilterName = "ServicesFilterByVisibility"

func NewServicesFilterByVisibility(repository storage.Repository) *ServicesFilterByVisibility {
	return &ServicesFilterByVisibility{
		visibilityFilteringMiddleware: &visibilityFilteringMiddleware{
			ListResourcesCriteria: servicesCriteriaFunc(repository),
			IsResourceVisible:     isServiceVisible(repository),
		},
	}
}

type ServicesFilterByVisibility struct {
	*visibilityFilteringMiddleware
}

func isServiceVisible(repository storage.Repository) func(ctx context.Context, serviceID, platformID string) (bool, error) {
	return func(ctx context.Context, serviceID, platformID string) (bool, error) {
		plansList, err := repository.List(ctx, types.ServicePlanType, query.ByField(query.EqualsOperator, "service_offering_id", serviceID))
		if err != nil {
			return false, err
		}
		planIds := make([]string, 0, plansList.Len())
		for i := 0; i < plansList.Len(); i++ {
			planIds = append(planIds, plansList.ItemAt(i).GetID())
		}
		visibilities, err := repository.QueryForList(ctx, types.VisibilityType, storage.QueryForVisibilitiesWithLabelsByPlatformsAndServiceIds, map[string]interface{}{
			"service_plan_ids": planIds,
			"platform_ids":     []string{platformID},
		})

		if err != nil {
			return false, err
		}

		return visibilities.Len() > 0, nil
	}
}

func servicesCriteriaFunc(repository storage.Repository) func(ctx context.Context, platformID string) (*query.Criterion, error) {
	return func(ctx context.Context, platformID string) (*query.Criterion, error) {
		planQuery, err := plansCriteria(ctx, repository, platformID)
		if err != nil {
			return nil, err
		}
		if planQuery == nil {
			return nil, nil
		}

		return servicesCriteria(ctx, repository, planQuery)
	}
}

func (vf *ServicesFilterByVisibility) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path(web.ServiceOfferingsURL + "/*"),
				web.Methods(http.MethodGet),
			},
		},
	}
}

func (vf *ServicesFilterByVisibility) Name() string {
	return ServicesVisibilityFilterName
}
