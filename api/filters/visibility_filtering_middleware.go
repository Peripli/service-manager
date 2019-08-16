package filters

import (
	"context"
	"errors"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/web"
)

const k8sPlatformType string = "kubernetes"

type visibilityFilteringMiddleware struct {
	SingleResourceFilterFunc func(ctx context.Context, resourceID, platformID string) (bool, error)
	MultiResourceFilterFunc  func(ctx context.Context, platformID string) (*query.Criterion, error)
}

func (m visibilityFilteringMiddleware) Run(req *web.Request, next web.Handler) (*web.Response, error) {
	ctx := req.Context()
	userCtx, ok := web.UserFromContext(ctx)
	if !ok {
		return nil, errors.New("no user found")
	}
	if userCtx.AuthenticationType != web.Basic {
		log.C(ctx).Debugf("Authentication is %s, not basic so proceed without visibility filter criteria", userCtx.AuthenticationType)
		return next.Handle(req)
	}
	platform := &types.Platform{}
	if err := userCtx.Data(platform); err != nil {
		return nil, err
	}
	if platform.Type != k8sPlatformType {
		log.C(ctx).Debugf("Platform type is %s, which is not kubernetes. Skip filtering on visibilities", platform.Type)
		return next.Handle(req)
	}

	resourceID := req.PathParams["id"]
	foundResource := false
	var err error
	var finalQuery *query.Criterion

	if resourceID != "" {
		foundResource, err = m.SingleResourceFilterFunc(ctx, resourceID, platform.ID)
		if err != nil {
			return nil, err
		}
		if foundResource {
			return next.Handle(req)
		}
	} else {
		finalQuery, err = m.MultiResourceFilterFunc(ctx, platform.ID)
		if err != nil {
			return nil, err
		}
		if finalQuery != nil {
			foundResource = true
		}
	}

	if !foundResource {
		ctx = query.ContextWithCriteria(ctx, []query.Criterion{query.LimitResultBy(0)})
	} else {
		ctx = query.ContextWithCriteria(ctx, []query.Criterion{*finalQuery})
	}

	req.Request = req.WithContext(ctx)
	return next.Handle(req)
}
