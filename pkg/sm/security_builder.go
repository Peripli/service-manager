package sm

import (
	"fmt"

	"github.com/Peripli/service-manager/api/filters"
	secFilters "github.com/Peripli/service-manager/pkg/security/filters"

	httpsec "github.com/Peripli/service-manager/pkg/security/http"
	"github.com/Peripli/service-manager/pkg/security/http/authn"
	"github.com/Peripli/service-manager/pkg/security/http/authz"
	"github.com/Peripli/service-manager/pkg/web"
)

type securityBuilder struct {
	currentMatchers []web.Matcher
	paths           []string
	methods         []string
	authenticators  []httpsec.Authenticator
	authorizers     []httpsec.Authorizer

	requiredAuthNMatchers []web.FilterMatcher
	requiredAuthZMatchers []web.FilterMatcher
	authnFilters          []web.Filter
	authzFilters          []web.Filter

	smb *ServiceManagerBuilder
}

type authenticationBuilder struct {
	sb *securityBuilder
}

func (sb *securityBuilder) Optional() *ServiceManagerBuilder {
	sb.register()
	sb.reset()
	return sb.smb
}

func (sb *securityBuilder) Required() *ServiceManagerBuilder {
	if len(sb.authorizers) > 0 {
		sb.requiredAuthZMatchers = append(sb.requiredAuthZMatchers, web.FilterMatcher{sb.currentMatchers})
	}
	if len(sb.authenticators) > 0 {
		sb.requiredAuthNMatchers = append(sb.requiredAuthNMatchers, web.FilterMatcher{sb.currentMatchers})
	}
	sb.register()
	sb.reset()
	return sb.smb
}

func (sb *securityBuilder) Path(paths ...string) *securityBuilder {
	sb.currentMatchers = append(sb.currentMatchers, web.Path(paths...))
	sb.paths = append(sb.paths, paths...)
	return sb
}

func (sb *securityBuilder) Method(methods ...string) *securityBuilder {
	sb.currentMatchers = append(sb.currentMatchers, web.Methods(methods...))
	sb.methods = append(sb.methods, methods...)
	return sb
}

func (sb *securityBuilder) WithAuthentication(authenticator httpsec.Authenticator) *securityBuilder {
	sb.authenticators = append(sb.authenticators, authenticator)
	return sb
}

func (sb *securityBuilder) WithAuthorization(authorizer httpsec.Authorizer) *securityBuilder {
	sb.authorizers = append(sb.authorizers, authorizer)
	return sb
}

func (sb *securityBuilder) WithScopes(scopes ...string) *securityBuilder {
	sb.authorizers = append(sb.authorizers, authz.NewScopesAuthorizer(scopes, web.GlobalAccess))
	return sb
}

func (sb *securityBuilder) WithClientIDSuffix(suffix string) *securityBuilder {
	sb.authorizers = append(sb.authorizers, authz.NewOAuthCloneAuthorizer(suffix, web.GlobalAccess))
	return sb
}

func (sb *securityBuilder) WithClientID(clientID string) *securityBuilder {
	sb.authorizers = append(sb.authorizers, authz.NewOauthClientAuthorizer(clientID, web.GlobalAccess))
	return sb
}

func (sb *securityBuilder) reset() {
	sb.currentMatchers = make([]web.Matcher, 0)
	sb.paths = make([]string, 0)
	sb.methods = make([]string, 0)
	sb.authenticators = make([]httpsec.Authenticator, 0)
	sb.authorizers = make([]httpsec.Authorizer, 0)
}

func (sb *securityBuilder) register() {
	if len(sb.authenticators) > 0 {
		finalAuthenticator := authn.NewOrAuthenticator(sb.authenticators...)

		sb.authnFilters = append(sb.authnFilters,
			secFilters.NewAuthenticationFilter(
				finalAuthenticator,
				fmt.Sprintf("%v-AuthNFilter%d@%v", sb.methods, len(sb.authnFilters), sb.paths),
				[]web.FilterMatcher{
					{
						sb.currentMatchers,
					},
				}))
	}

	if len(sb.authorizers) > 0 {
		finalAuthorizer := authz.NewAndAuthorizer(sb.authorizers...)
		sb.authzFilters = append(sb.authzFilters,
			secFilters.NewAuthzFilter(
				finalAuthorizer,
				fmt.Sprintf("%v-AuthZFilter%d@%v", sb.methods, len(sb.authzFilters), sb.paths),
				[]web.FilterMatcher{
					{
						sb.currentMatchers,
					},
				}))
	}
}

func (sb *securityBuilder) finalize() {
	finalFilters := sb.authnFilters
	if len(sb.requiredAuthNMatchers) > 0 {
		requiredAuthN := secFilters.NewRequiredAuthnFilter(sb.requiredAuthNMatchers)
		finalFilters = append(finalFilters, requiredAuthN)
	}
	finalFilters = append(finalFilters, sb.authzFilters...)
	if len(sb.requiredAuthZMatchers) > 0 {
		requiredAuthZ := secFilters.NewRequiredAuthzFilter(sb.requiredAuthZMatchers)
		finalFilters = append(finalFilters, requiredAuthZ)
	}
	sb.smb.RegisterFiltersAfter(filters.LoggingFilterName, finalFilters...)
}
