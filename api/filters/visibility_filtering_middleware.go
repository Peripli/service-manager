package filters

import (
	"context"
	"errors"
	"net/http"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/util"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/log"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/query"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web"
)

type visibilityFilteringMiddleware struct {
	IsResourceVisible     func(ctx context.Context, resourceID, platformID string) (bool, error)
	ListResourcesCriteria func(ctx context.Context, platformID string) (*query.Criterion, error)
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

	resourceID := req.PathParams["resource_id"]
	isSingleResource := resourceID != ""

	if isSingleResource {
		if isResourceVisible, err := m.IsResourceVisible(ctx, resourceID, platform.ID); err != nil {
			return nil, err
		} else if !isResourceVisible {
			return nil, &util.HTTPError{
				ErrorType:   "NotFound",
				Description: "could not find resource",
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
		return util.NewJSONResponse(http.StatusOK, types.ObjectPage{Items: make([]types.Object, 0)})
	}

	ctx, err = query.AddCriteria(ctx, *finalQuery)

	if err != nil {
		return nil, err
	}

	req.Request = req.WithContext(ctx)
	return next.Handle(req)
}
