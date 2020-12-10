package filters

import (
	"github.com/Peripli/service-manager/pkg/query"
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
	m := operationsUrl.FindStringSubmatch(req.URL.Path)
	if len(m) > 1 {
		return next.Handle(req)
	}

	criteria := query.ByField(query.EqualsOperator, technicalKeyName, "false")
	newCtx, err := query.AddCriteria(req.Context(), criteria)
	if err != nil {
		return nil, err
	}

	req.Request = req.WithContext(newCtx)
	return next.Handle(req)
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
