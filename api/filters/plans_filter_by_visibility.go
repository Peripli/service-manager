package filters

import (
	"context"
	"net/http"

	"github.com/Peripli/service-manager/pkg/types"

	"github.com/Peripli/service-manager/pkg/query"

	"github.com/Peripli/service-manager/storage"

	"github.com/Peripli/service-manager/pkg/web"
)

const PlanVisibilityFilterName = "PlanFilterByVisibility"

func NewPlansFilterByVisibility(repository storage.Repository) *PlanFilterByVisibility {
	return &PlanFilterByVisibility{
		visibilityFilteringMiddleware: &visibilityFilteringMiddleware{
			ListResourcesCriteria: plansCriteriaFunc(repository),
			IsResourceVisible:     isPlanVisibile(repository),
		},
	}
}

type PlanFilterByVisibility struct {
	*visibilityFilteringMiddleware
}

func isPlanVisibile(repository storage.Repository) func(ctx context.Context, planID, platformID string) (bool, error) {
	return func(ctx context.Context, planID, platformID string) (bool, error) {
		visibilities, err := repository.QueryForList(ctx, types.VisibilityType, storage.QueryForVisibilitiesWithLabelsByPlan, map[string]interface{}{
			"service_plan_ids": []string{platformID},
		})

		if err != nil {
			return false, err
		}

		return visibilities.Len() > 0, nil
	}
}

func plansCriteriaFunc(repository storage.Repository) func(context.Context, string) (*query.Criterion, error) {
	return func(ctx context.Context, platformID string) (*query.Criterion, error) {
		planQuery, err := plansCriteria(ctx, repository, platformID)
		if err != nil {
			return nil, err
		}
		return planQuery, nil
	}
}

func (vf *PlanFilterByVisibility) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path(web.ServicePlansURL + "/*"),
				web.Methods(http.MethodGet),
			},
		},
	}
}

func (vf *PlanFilterByVisibility) Name() string {
	return PlanVisibilityFilterName
}
