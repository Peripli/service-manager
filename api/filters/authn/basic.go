package authn

import (
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/security/basic"
	"github.com/Peripli/service-manager/storage"
)

type basicAuthnFilter struct {
	Middleware
}

func NewBasicAuthnFilter(storage storage.Credentials) *basicAuthnFilter {
	return &basicAuthnFilter{
		Middleware: Middleware{
			authenticator: basic.NewAuthenticator(storage),
			name:          "BasicAuthenticationFilter",
		},
	}
}

func (ba *basicAuthnFilter) Name() string {
	return "BasicAuthenticationFilter"
}

func (ba *basicAuthnFilter) RouteMatchers() []web.RouteMatcher {
	return []web.RouteMatcher{
		{
			Matchers: []web.Matcher{
				web.Path("/v1/osb/**"),
			},
		},
	}
}
