package authn

import (
	"net/http"

	"strings"

	"context"

	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/security/oidc"
)

type bearerAuthnFilter struct {
	Middleware
}

func NewBearerAuthnFilter(ctx context.Context, tokenIssuer, clientID string) (*bearerAuthnFilter, error) {
	authenticator, err := oidc.NewAuthenticator(ctx, oidc.Options{
		IssuerURL: tokenIssuer,
		ClientID:  clientID,
	})
	if err != nil {
		return nil, err
	}
	return &bearerAuthnFilter{
		Middleware: Middleware{
			authenticator: authenticator,
		},
	}, nil
}

func (ba *bearerAuthnFilter) Name() string {
	return "BearerAuthenticationFilter"
}

func (ba *bearerAuthnFilter) RouteMatchers() []web.RouteMatcher {
	return []web.RouteMatcher{
		{
			Matchers: []web.Matcher{
				web.Methods(http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch),
				web.Path("/v1/service_brokers/**"),
			},
		},
		{
			Matchers: []web.Matcher{
				web.Methods(http.MethodGet),
				web.Path("/v1/service_brokers/**"),
			},
		},
		{
			Matchers: []web.Matcher{
				web.Path("/v1/platforms/**"),
			},
		},
		{
			Matchers: []web.Matcher{
				web.Path("/v1/sm_catalog"),
			},
		},
	}
}

func (ba *bearerAuthnFilter) Matches(request *web.Request) bool {
	return strings.HasPrefix(request.Header.Get("Authorization"), "Bearer ")
}
