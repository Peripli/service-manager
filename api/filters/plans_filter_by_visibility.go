package filters

import (
	"context"
	"net/http"

	"github.com/Peripli/service-manager/pkg/query"

	"github.com/Peripli/service-manager/storage"

	"github.com/Peripli/service-manager/pkg/web"
)

const PlanVisibilityFilterName = "PlanFilterByVisibility"

func NewPlanFilterByVisibility(repository storage.Repository) *PlanFilterByVisibility {
	return &PlanFilterByVisibility{
		visibilityFilteringMiddleware: &visibilityFilteringMiddleware{
			FilteringFunc: newPlansFilterFunc(repository),
		},
	}
}

type PlanFilterByVisibility struct {
	*visibilityFilteringMiddleware
}

func newPlansFilterFunc(repository storage.Repository) func(ctx context.Context, platformID string) (*query.Criterion, error) {
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
				web.Path(web.ServicePlansURL + "/**"),
				web.Methods(http.MethodGet),
			},
		},
	}
}

func (vf *PlanFilterByVisibility) Name() string {
	return PlanVisibilityFilterName
}
