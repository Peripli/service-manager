package filters

import (
	"fmt"
	"strings"

	"github.com/Peripli/service-manager/pkg/security/filters/middlewares"
	"github.com/Peripli/service-manager/pkg/security/http"
	"github.com/Peripli/service-manager/pkg/web"
)

// NewAuthzFilter returns a web.Filter for a specific scope and endpoint
func NewAuthzFilter(methods []string, path string, authorizer http.Authorizer) *AuthorizationFilter {
	filterName := fmt.Sprintf("%s-AuthzFilter@%s", strings.Join(methods, "/"), path)
	return &AuthorizationFilter{
		Authorization: &middlewares.Authorization{
			Authorizer: authorizer,
		},
		methods: methods,
		path:    path,
		name:    filterName,
	}
}

type AuthorizationFilter struct {
	*middlewares.Authorization

	methods []string
	path    string
	name    string
}

func (af *AuthorizationFilter) Name() string {
	return af.name
}

// FilterMatchers implements the web.Filter interface and returns the conditions
// on which the filter should be executed
func (af *AuthorizationFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Methods(af.methods...),
				web.Path(af.path),
			},
		},
	}
}
