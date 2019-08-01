package filters

import (
	"errors"
	"net/http"

	"github.com/Peripli/service-manager/pkg/util"

	"github.com/Peripli/service-manager/pkg/query"

	"github.com/Peripli/service-manager/pkg/types"

	"github.com/Peripli/service-manager/storage"

	"github.com/Peripli/service-manager/pkg/log"

	"github.com/Peripli/service-manager/pkg/web"
)

const VisibilityFilterName = "VisibilityFilter"

func NewVisibilityFilter(repository storage.Repository) *VisibilityFilter {
	return &VisibilityFilter{
		repository: repository,
	}
}

type VisibilityFilter struct {
	repository storage.Repository
}

func (vf *VisibilityFilter) Run(req *web.Request, next web.Handler) (*web.Response, error) {
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
	objectList, err := vf.repository.List(ctx, types.VisibilityType, query.ByField(query.EqualsOperator, "platform_id", platform.ID))
	if err != nil {
		return nil, util.HandleStorageError(err, "Visibility")
	}
	visibilityList := objectList.(*types.Visibilities)
	planIDs := make([]string, 0, visibilityList.Len())
	for _, vis := range visibilityList.Visibilities {
		planIDs = append(planIDs, vis.ServicePlanID)
	}

	if len(planIDs) > 0 {
		ctx = query.ContextWithCriteria(ctx, []query.Criterion{query.ByField(query.InOperator, "id", planIDs...)})
	} else {
		// Paging must come before this filter
		ctx = query.ContextWithCriteria(ctx, []query.Criterion{query.LimitResultBy(0)})
	}

	req.Request = req.WithContext(ctx)
	return next.Handle(req)
}

func (vf *VisibilityFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path(web.ServicePlansURL + "/**"),
				web.Methods(http.MethodGet),
			},
		},
	}
}

func (vf *VisibilityFilter) Name() string {
	return VisibilityFilterName
}
