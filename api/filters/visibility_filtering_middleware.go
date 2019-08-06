package filters

import (
	"context"
	"errors"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/web"
)

type visibilityFilteringMiddleware struct {
	FilteringFunc func(context.Context, string) (*query.Criterion, error)
}

func (m visibilityFilteringMiddleware) Run(req *web.Request, next web.Handler) (*web.Response, error) {
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

	finalQuery, err := m.FilteringFunc(ctx, platform.ID)
	if err != nil {
		return nil, err
	}

	resourceID := req.PathParams["id"]
	foundResource := false
	if finalQuery != nil && resourceID != "" {
		for _, serviceID := range finalQuery.RightOp {
			if serviceID == resourceID {
				foundResource = true
				break
			}
		}
	}

	if finalQuery == nil || (!foundResource && resourceID != "") {
		ctx = query.ContextWithCriteria(ctx, []query.Criterion{query.LimitResultBy(0)})
	} else if resourceID == "" {
		ctx = query.ContextWithCriteria(ctx, []query.Criterion{*finalQuery})
	}

	req.Request = req.WithContext(ctx)
	return next.Handle(req)
}
