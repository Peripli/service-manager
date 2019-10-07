package filters

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/Peripli/service-manager/pkg/util"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/web"
)

type visibilityFilteringMiddleware struct {
	IsResourceVisible       func(ctx context.Context, resourceID, platformID string) (bool, error)
	ListResourcesCriteria   func(ctx context.Context, platformID string) (*query.Criterion, error)
	EmptyObjectListProvider func() types.ObjectList
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
	if platform.Type != types.K8sPlatformType {
		log.C(ctx).Debugf("Platform type is %s, which is not kubernetes. Skip filtering on visibilities", platform.Type)
		return next.Handle(req)
	}

	resourceID := req.PathParams["id"]
	isSingleResource := (resourceID != "")

	if isSingleResource {
		if isResourceVisible, err := m.IsResourceVisible(ctx, resourceID, platform.ID); err != nil {
			return nil, err
		} else if !isResourceVisible {
			return nil, &util.HTTPError{
				ErrorType:   "NotFound",
				Description: fmt.Sprintf("could not find resource"),
				StatusCode:  http.StatusNotFound,
			}
		}

		return next.Handle(req)
	}

	finalQuery, err := m.ListResourcesCriteria(ctx, platform.ID)
	if err != nil {
		return nil, err
	}
	if finalQuery == nil {
		return util.NewJSONResponse(http.StatusOK, m.EmptyObjectListProvider())
	}

	ctx, err = query.AddCriteria(ctx, *finalQuery)

	if err != nil {
		return nil, err
	}

	req.Request = req.WithContext(ctx)
	return next.Handle(req)
}
