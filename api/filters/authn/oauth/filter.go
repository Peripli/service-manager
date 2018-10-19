package oauth

import (
	"context"

	"github.com/Peripli/service-manager/pkg/security/middlewares"
	"github.com/Peripli/service-manager/pkg/web"
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
				),
			},
		},
	}
}

// NewFilter returns a web.Filter for Bearer authentication
func NewFilter(ctx context.Context, tokenIssuer, clientID string) (web.Filter, error) {
	authenticator, err := newAuthenticator(ctx, &options{
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
