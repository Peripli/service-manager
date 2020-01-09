package sm

import (
	"net/http"

	"github.com/Peripli/service-manager/pkg/web"
)

type dynamicMiddlewares struct {
	matchers   []web.FilterMatcher
	middleware web.Middleware
}

type DynamicMatchingFilter struct {
	middlewares []dynamicMiddlewares
	name        string
}

func NewDynamicMatchingFilter(name string) *DynamicMatchingFilter {
	return &DynamicMatchingFilter{
		name: name,
	}
}

func (dmf *DynamicMatchingFilter) Name() string {
	return dmf.name
}

func (dmf *DynamicMatchingFilter) Run(req *web.Request, next web.Handler) (*web.Response, error) {
	endpoint := web.Endpoint{
		Method: req.Method,
		Path:   req.URL.Path,
	}

	filters := web.Filters{}
	for _, dm := range dmf.middlewares {
		matched, err := web.Matching(dm.matchers, endpoint)
		if err != nil {
			return nil, err
		}
		if matched {
			filters = append(filters, &emptyFilter{
				Middleware: dm.middleware,
			})
		}
	}
	return filters.Chain(next).Handle(req)
}

func (dmf *DynamicMatchingFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path("/**"),
				web.Methods(http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodHead),
			},
		},
	}
}

type emptyFilter struct {
	web.Middleware
}

func (ef *emptyFilter) Name() string {
	return ""
}

func (ef *emptyFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{}
}
