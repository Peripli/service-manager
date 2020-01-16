package web

import (
	"net/http"
)

type DynamicMatchingFilter struct {
	filters Filters
	name    string
}

func NewDynamicMatchingFilter(name string) *DynamicMatchingFilter {
	return &DynamicMatchingFilter{
		name: name,
	}
}

func (dmf *DynamicMatchingFilter) AddFilter(dynamicFilter Filter) {
	dmf.filters = append(dmf.filters, dynamicFilter)
}

func (dmf *DynamicMatchingFilter) ClearFilters() {
	dmf.filters = Filters{}
}

func (dmf *DynamicMatchingFilter) Name() string {
	return dmf.name
}

func (dmf *DynamicMatchingFilter) Run(req *Request, next Handler) (*Response, error) {
	endpoint := Endpoint{
		Method: req.Method,
		Path:   req.URL.Path,
	}

	route := Route{
		Endpoint: endpoint,
		Handler:  next.Handle,
	}
	return dmf.filters.ChainMatching(route).Handle(req)
}

func (dmf *DynamicMatchingFilter) FilterMatchers() []FilterMatcher {
	return []FilterMatcher{
		{
			Matchers: []Matcher{
				Path("/**"),
				Methods(http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodHead),
			},
		},
	}
}
