package authn

import (
	"context"

	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/security/oidc"
)

// BearerAuthnFilterName is the name of the bearer authentication filter
const BearerAuthnFilterName string = "BearerAuthnFilter"

// BearerAuthnFilter performs Bearer authentication by validating the Authorization header
type BearerAuthnFilter struct {
	Middleware
}

// NewBearerAuthnFilter returns a BearerAuthnFilter
func NewBearerAuthnFilter(ctx context.Context, tokenIssuer, clientID string) (*BearerAuthnFilter, error) {
	authenticator, err := oidc.NewAuthenticator(ctx, &oidc.Options{
		IssuerURL: tokenIssuer,
		ClientID:  clientID,
	})
	if err != nil {
		return nil, err
	}
	return &BearerAuthnFilter{
		Middleware: Middleware{
			authenticator: authenticator,
			name:          BearerAuthnFilterName,
		},
	}, nil
}

// Name implements the web.Filter interface and returns the identifier of the filter
func (ba *BearerAuthnFilter) Name() string {
	return BearerAuthnFilterName
}

// FilterMatchers implements the web.Filter interface and returns the conditions on which the filter should be executed
func (ba *BearerAuthnFilter) FilterMatchers() []web.FilterMatcher {
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
