package authn

import (
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/web"
	)

// RequiredAuthenticationFilterName is the name of RequiredAuthenticationFilter
const RequiredAuthenticationFilterName = "RequiredAuthenticationFilter"

// RequiredAuthnFilter type verifies that authentication has been performed for APIs that are secured
type RequiredAuthnFilter struct {
}

// NewRequiredAuthnFilter returns RequiredAuthnFilter
func NewRequiredAuthnFilter() *RequiredAuthnFilter {
	return &RequiredAuthnFilter{}
}

// Name implements the web.Filter interface and returns the identifier of the filter
func (raf *RequiredAuthnFilter) Name() string {
	return RequiredAuthenticationFilterName
}

// Run implements web.Filter and represents the authentication middleware function that verifies the user is
// authenticated
func (raf *RequiredAuthnFilter) Run(request *web.Request, next web.Handler) (*web.Response, error) {
	ctx := request.Context()
	if _, ok := web.UserFromContext(ctx); !ok {
		log.C(ctx).Error("No authenticated user found in request context during execution of filter ", raf.Name())
		return nil, errUnauthorized
	}

	return next.Handle(request)
}

// FilterMatchers implements the web.Filter interface and returns the conditions on which the filter should be executed
func (raf *RequiredAuthnFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path(
					web.BrokersURL+"/**",
					web.PlatformsURL+"/**",
					web.SMCatalogURL,
					web.OSBURL+"/**",
				),
			},
		},
	}
}
