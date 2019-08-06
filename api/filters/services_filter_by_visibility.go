package filters

import (
	"errors"
	"net/http"

	"github.com/Peripli/service-manager/pkg/query"

	"github.com/Peripli/service-manager/pkg/types"

	"github.com/Peripli/service-manager/storage"

	"github.com/Peripli/service-manager/pkg/log"

	"github.com/Peripli/service-manager/pkg/web"
)

const ServicesVisibilityFilterName = "ServicesFilterByVisibility"

func NewServicesFilterByVisibility(repository storage.Repository) *ServicesFilterByVisibility {
	return &ServicesFilterByVisibility{
		repository: repository,
	}
}

type ServicesFilterByVisibility struct {
	repository storage.Repository
}

func (vf *ServicesFilterByVisibility) Run(req *web.Request, next web.Handler) (*web.Response, error) {
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

	zeroResult := false

	planQuery, err := plansCriteria(ctx, vf.repository, platform.ID)
	if err != nil {
		return nil, err
	}
	if planQuery == nil {
		zeroResult = true
	}

	if !zeroResult {
		objectID := req.PathParams["id"]
		servicesQuery, err := servicesCriteria(ctx, vf.repository, planQuery)
		if err != nil {
			return nil, err
		}

		hasID := false
		if servicesQuery != nil {
			for _, serviceID := range servicesQuery.RightOp {
				if serviceID == objectID {
					hasID = true
					break
				}
			}
		}

		if servicesQuery == nil || (!hasID && objectID != "") {
			zeroResult = true
		} else if objectID == "" {
			ctx = query.ContextWithCriteria(ctx, []query.Criterion{*servicesQuery})
		}
	}

	if zeroResult {
		ctx = query.ContextWithCriteria(ctx, []query.Criterion{query.LimitResultBy(0)})
	}
	req.Request = req.WithContext(ctx)
	return next.Handle(req)
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
