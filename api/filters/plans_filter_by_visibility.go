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

const PlanVisibilityFilterName = "PlanFilterByVisibility"

func NewPlanFilterByVisibility(repository storage.Repository) *PlanFilterByVisibility {
	return &PlanFilterByVisibility{
		repository: repository,
	}
}

type PlanFilterByVisibility struct {
	repository storage.Repository
}

func (vf *PlanFilterByVisibility) Run(req *web.Request, next web.Handler) (*web.Response, error) {
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

	planQuery, err := plansCriteria(ctx, vf.repository, platform.ID)
	if err != nil {
		return nil, err
	}
	objectID := req.PathParams["id"]
	hasID := false
	if planQuery != nil {
		for _, planID := range planQuery.RightOp {
			if planID == objectID {
				hasID = true
				break
			}
		}
	}
	if planQuery == nil || (!hasID && objectID != "") {
		ctx = query.ContextWithCriteria(ctx, []query.Criterion{query.LimitResultBy(0)})
	} else if objectID == "" {
		ctx = query.ContextWithCriteria(ctx, []query.Criterion{*planQuery})
	}

	req.Request = req.WithContext(ctx)
	return next.Handle(req)
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
