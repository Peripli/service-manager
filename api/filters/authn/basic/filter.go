package basic

import (
	"net/http"

	"github.com/Peripli/service-manager/pkg/security"
	"github.com/Peripli/service-manager/pkg/security/middlewares"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
)

// BasicAuthnFilterName is the name of the basic authentication filter
const BasicAuthnFilterName string = "BasicAuthnFilter"

// NewFilter returns a web.Filter for basic auth using the provided
// credentials storage in order to validate the credentials
func NewFilter(storage storage.Credentials, encrypter security.Encrypter) web.Filter {
	return &basicAuthnFilter{
		Filter: middlewares.NewAuthnMiddleware(
			BasicAuthnFilterName,
			&basicAuthenticator{CredentialStorage: storage, Encrypter: encrypter},
		),
	}
}

// basicAuthnFilter performs Basic authentication by validating the Authorization header
type basicAuthnFilter struct {
	web.Filter
}

// FilterMatchers implements the web.Filter interface and returns the conditions on which the filter should be executed
func (ba *basicAuthnFilter) FilterMatchers() []web.FilterMatcher {
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
		{
			Matchers: []web.Matcher{
				web.Methods(http.MethodGet),
				web.Path(web.ServicePlansURL + "/**"),
			},
		},
		{
			Matchers: []web.Matcher{
				web.Methods(http.MethodGet),
				web.Path(web.VisibilitiesURL + "/**"),
			},
		},
	}
}
