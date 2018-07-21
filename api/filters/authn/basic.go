package authn

import (
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/security/basic"
	"github.com/Peripli/service-manager/storage"
)

type basicAuthnFilter struct {
	Midleware
}

func NewBasicAuthnFilter(storage storage.Credentials) *basicAuthnFilter {
	return &basicAuthnFilter{
		Midleware: Midleware{
			authenticator: basic_authn.NewAuthenticator(storage),
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

func (ba *basicAuthnFilter) Matches(request *web.Request) bool {
	_, _, ok := request.BasicAuth()
	return ok
}
