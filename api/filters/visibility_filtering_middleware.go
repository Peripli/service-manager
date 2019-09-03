package filters

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"

	"github.com/Peripli/service-manager/pkg/util"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/web"
)

const k8sPlatformType string = "kubernetes"

type visibilityFilteringMiddleware struct {
	SingleResourceFilterFunc func(ctx context.Context, resourceID, platformID string) (bool, error)
	MultiResourceFilterFunc  func(ctx context.Context, platformID string) (*query.Criterion, error)
	EmptyResourceArrayFunc   func() types.ObjectList
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
	isSingleResource := (req.PathParams["id"] != "")
	var err error
	var finalQuery *query.Criterion

	if isSingleResource {
		foundResource, err := m.SingleResourceFilterFunc(ctx, resourceID, platform.ID)
		if err != nil {
			return nil, err
		}
		if foundResource {
			return next.Handle(req)
		}
		return nil, &util.HTTPError{
			ErrorType:   "NotFound",
			Description: fmt.Sprintf("could not find resource"),
			StatusCode:  http.StatusNotFound,
		}
	}

	finalQuery, err = m.MultiResourceFilterFunc(ctx, platform.ID)
	if err != nil {
		return nil, err
	}
	if finalQuery == nil {
		return util.NewJSONResponse(http.StatusOK, m.EmptyResourceArrayFunc())
	}

	criterion := query.CriteriaForContext(ctx)
	merged := false
	for i, c := range criterion {
		if c.LeftOp == "id" {
			merged = true
			if c.Operator == query.EqualsOperator ||
				c.Operator == query.EqualsOrNilOperator ||
				c.Operator == query.InOperator {

				finalQuery.RightOp = intersect(finalQuery.RightOp, c.RightOp)
				criterion[i] = *finalQuery
			} else if c.Operator == query.NotEqualsOperator || c.Operator == query.NotInOperator {
				finalQuery.RightOp = subtract(finalQuery.RightOp, c.RightOp)
				criterion[i] = *finalQuery
			}

			if len(criterion[i].RightOp) == 0 {
				return util.NewJSONResponse(http.StatusOK, m.EmptyResourceArrayFunc())
			}
		}
	}
	if !merged {
		ctx, err = query.AddCriteria(ctx, *finalQuery)
	} else {
		ctx = query.ContextWithCriteria(ctx, criterion)
	}

	if err != nil {
		return nil, err
	}

	req.Request = req.WithContext(ctx)
	return next.Handle(req)
}

func elementMap(a []string) map[string]bool {
	inMap := make(map[string]bool)
	for _, el := range a {
		inMap[el] = true
	}
	return inMap
}

func subtract(a, b []string) []string {
	inB := elementMap(b)
	result := make([]string, 0, len(a))
	for _, elA := range a {
		if !inB[elA] {
			result = append(result, elA)
		}
	}
	return result
}

func intersect(a, b []string) []string {
	inB := elementMap(b)
	result := make([]string, 0, int(math.Min(float64(len(a)), float64(len(b)))))
	for _, elA := range a {
		if inB[elA] {
			result = append(result, elA)
		}
	}
	return result
}
