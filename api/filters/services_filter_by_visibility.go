package filters

import (
	"context"
	"net/http"

	"github.com/Peripli/service-manager/pkg/query"

	"github.com/Peripli/service-manager/storage"

	"github.com/Peripli/service-manager/pkg/web"
)

const ServicesVisibilityFilterName = "ServicesFilterByVisibility"

func NewServicesFilterByVisibility(repository storage.Repository) *ServicesFilterByVisibility {
	return &ServicesFilterByVisibility{
		visibilityFilteringMiddleware: &visibilityFilteringMiddleware{
			FilteringFunc: newServicesFilterFunc(repository),
		},
	}
}

type ServicesFilterByVisibility struct {
	*visibilityFilteringMiddleware
}

func newServicesFilterFunc(repository storage.Repository) func(ctx context.Context, platformID string) (*query.Criterion, error) {
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
		return servicesQuery, nil
	}
}

func (vf *ServicesFilterByVisibility) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path(web.ServiceOfferingsURL + "/**"),
				web.Methods(http.MethodGet),
			},
		},
	}
}

func (vf *ServicesFilterByVisibility) Name() string {
	return ServicesVisibilityFilterName
}
