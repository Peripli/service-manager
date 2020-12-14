package filters

import (
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
	"net/http"
	"regexp"

	"github.com/Peripli/service-manager/pkg/web"
)

var operationsUrl = regexp.MustCompile("(/operations/.*)$")

const (
	TechnicalPlatformFilterName = "TechnicalPlatformFilter"
	technicalKeyName            = "technical"
)

// PlatformFilter filters out technical platforms and marks it as active on creation
type TechnicalPlatformFilter struct {
	Storage storage.Repository
}

func (f *TechnicalPlatformFilter) Name() string {
	return TechnicalPlatformFilterName
}

func (f *TechnicalPlatformFilter) Run(req *web.Request, next web.Handler) (*web.Response, error) {
	switch req.Method {
	case http.MethodGet:
		return f.get(req, next)
	default:
		return next.Handle(req)
	}
}

func (f *TechnicalPlatformFilter) get(req *web.Request, next web.Handler) (*web.Response, error) {
	var criteria query.Criterion
	if operationRequest(req) {
		platformID := req.PathParams[web.PathParamResourceID]
		isTechnicalPlatformCriteria := []query.Criterion{
			query.ByField(query.EqualsOperator, "id", platformID),
			query.ByField(query.EqualsOperator, technicalKeyName, "true"),
		}

		cnt, err := f.Storage.Count(req.Context(), types.PlatformType, isTechnicalPlatformCriteria...)
		if err != nil {
			return nil, err
		}
		if cnt == 0 {
			return next.Handle(req)
		}
		criteria = query.ByField(query.NotEqualsOperator, "resource_id", platformID)
	} else {
		criteria = query.ByField(query.EqualsOperator, technicalKeyName, "false")
	}

	newCtx, err := query.AddCriteria(req.Context(), criteria)
	if err != nil {
		return nil, err
	}

	req.Request = req.WithContext(newCtx)
	return next.Handle(req)
}

func operationRequest(req *web.Request) bool {
	m := operationsUrl.FindStringSubmatch(req.URL.Path)
	return len(m) > 1
}

func (f *TechnicalPlatformFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Methods(http.MethodGet),
				web.Path(web.PlatformsURL + "/**"),
			},
		},
	}
}
