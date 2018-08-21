package authn

import (
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/sirupsen/logrus"
)

// RequiredAuthnFilter type verifies that authentication has been performed for APIs that are secured
type RequiredAuthnFilter struct {
}

// NewRequiredAuthnFilter returns RequiredAuthnFilter
func NewRequiredAuthnFilter() *RequiredAuthnFilter {
	return &RequiredAuthnFilter{}
}

// Name implements the web.Filter interface and returns the identifier of the filter
func (raf *RequiredAuthnFilter) Name() string {
	return "RequiredAuthenticationFilter"
}

// Run implements web.Filter and represents the authentication middleware function that verifies the user is
// authenticated
func (raf *RequiredAuthnFilter) Run(request *web.Request, next web.Handler) (*web.Response, error) {
	if _, ok := web.UserFromContext(request.Context()); !ok {
		logrus.Error("No authenticated user found in request context during execution of filter ", raf.Name())
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
