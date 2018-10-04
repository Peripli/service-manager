package authn

import (
	"context"

	"github.com/Peripli/service-manager/pkg/security/middlewares"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/security/oidc"
)

// BearerAuthnFilterName is the name of the bearer authentication filter
const BearerAuthnFilterName string = "BearerAuthnFilter"

// bearerAuthnFilter performs Bearer authentication by validating the Authorization header
type bearerAuthnFilter struct {
	web.Filter
}

// FilterMatchers implements the web.Filter interface and returns the conditions on which the filter should be executed
func (ba *bearerAuthnFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path(
					web.BrokersURL+"/**",
					web.PlatformsURL+"/**",
					web.SMCatalogURL,
				),
			},
		},
	}
}

// NewBearerAuthnFilter returns a web.Filter for Bearer authentication
func NewBearerAuthnFilter(ctx context.Context, tokenIssuer, clientID string) (web.Filter, error) {
	authenticator, err := oidc.NewAuthenticator(ctx, &oidc.Options{
		IssuerURL: tokenIssuer,
		ClientID:  clientID,
	})
	if err != nil {
		return nil, err
	}
	return &bearerAuthnFilter{
		Filter: middlewares.NewAuthnMiddleware(BearerAuthnFilterName, authenticator),
	}, nil
}
