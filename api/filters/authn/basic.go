package authn

import (
	"net/http"

	"github.com/Peripli/service-manager/pkg/security"
	"github.com/Peripli/service-manager/pkg/security/middlewares"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/security/basic"
	"github.com/Peripli/service-manager/storage"
)

// BasicAuthnFilterName is the name of the basic authentication filter
const BasicAuthnFilterName string = "BasicAuthnFilter"

// BasicAuthnFilter performs Basic authentication by validating the Authorization header
type BasicAuthnFilter struct {
	web.Filter
}

// NewBasicAuthnFilter returns a BasicAuthnFilter using the provided credentials storage
// in order to validate the credentials
func NewBasicAuthnFilter(storage storage.Credentials, encrypter security.Encrypter) *BasicAuthnFilter {
	return &BasicAuthnFilter{
		Filter: middlewares.NewAuthnMiddleware(BasicAuthnFilterName, basic.NewAuthenticator(storage, encrypter)),
	}
}

// FilterMatchers implements the web.Filter interface and returns the conditions on which the filter should be executed
func (ba *BasicAuthnFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path(web.OSBURL + "/**"),
			},
		},
		{
			Matchers: []web.Matcher{
				web.Methods(http.MethodGet),
				web.Path(web.BrokersURL + "/**"),
			},
		},
	}
}
