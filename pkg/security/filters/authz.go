package filters

import (
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/security/filters/middlewares"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/security/http"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web"
)

const (
	AuthorizationFilterName string = "Authorization"
)

// NewAuthzFilter returns a web.Filter for a specific scope and endpoint
func NewAuthzFilter(authorizer http.Authorizer, name string, matchers []web.FilterMatcher) *AuthorizationFilter {
	return &AuthorizationFilter{
		Authorization: &middlewares.Authorization{
			Authorizer: authorizer,
		},
		matchers: matchers,
		name:     name,
	}
}

type AuthorizationFilter struct {
	*middlewares.Authorization
	matchers []web.FilterMatcher
	name     string
}

func (af *AuthorizationFilter) Name() string {
	return af.name
}

// FilterMatchers implements the web.Filter interface and returns the conditions
// on which the filter should be executed
func (af *AuthorizationFilter) FilterMatchers() []web.FilterMatcher {
	return af.matchers
}
