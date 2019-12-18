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
	pathMatcher   web.Matcher
	methodMatcher web.Matcher

	paths          []string
	methods        []string
	authenticators []httpsec.Authenticator
	authorizers    []httpsec.Authorizer
	accessLevelSet bool
	accessLevel    web.AccessLevel

	authentication bool
	authorization  bool

	requiredAuthNMatchers []web.FilterMatcher
	requiredAuthZMatchers []web.FilterMatcher
	authnFilters          []web.Filter
	authzFilters          []web.Filter

	smb *ServiceManagerBuilder
}

type authenticationBuilder struct {
	sb *securityBuilder
}

func (sb *securityBuilder) Optional() *securityBuilder {
	matcher := web.Not(
		web.Path(sb.paths...),
		web.Methods(sb.methods...),
	)
	if sb.authorization {
		for i := range sb.requiredAuthZMatchers {
			sb.requiredAuthZMatchers[i].Matchers = append(sb.requiredAuthZMatchers[i].Matchers, matcher)
		}
	}
	if sb.authentication {
		for i := range sb.requiredAuthNMatchers {
			sb.requiredAuthNMatchers[i].Matchers = append(sb.requiredAuthNMatchers[i].Matchers, matcher)
		}
	}
	sb.register()
	sb.resetAuthenticators()
	return sb
}

func (sb *securityBuilder) Required() *securityBuilder {
	finalMatchers := make([]web.Matcher, 0)
	if sb.pathMatcher != nil {
		finalMatchers = append(finalMatchers, sb.pathMatcher)
	}
	if sb.methodMatcher != nil {
		finalMatchers = append(finalMatchers, sb.methodMatcher)
	}

	if sb.authorization {
		sb.requiredAuthZMatchers = append(sb.requiredAuthZMatchers, web.FilterMatcher{finalMatchers})
	}
	if sb.authentication {
		sb.requiredAuthNMatchers = append(sb.requiredAuthNMatchers, web.FilterMatcher{finalMatchers})
	}
	sb.register()
	sb.resetAuthenticators()
	return sb
}

func (sb *securityBuilder) Path(paths ...string) *securityBuilder {
	sb.pathMatcher = web.Path(paths...)
	sb.paths = paths
	return sb
}

func (sb *securityBuilder) Method(methods ...string) *securityBuilder {
	sb.methodMatcher = web.Methods(methods...)
	sb.methods = methods
	return sb
}

func (sb *securityBuilder) Authentication() *securityBuilder {
	sb.authentication = true
	return sb
}

func (sb *securityBuilder) Authorization() *securityBuilder {
	sb.authorization = true
	return sb
}

func (sb *securityBuilder) WithAuthentication(authenticator httpsec.Authenticator) *securityBuilder {
	sb.authenticators = append(sb.authenticators, authenticator)
	sb.authentication = true
	return sb
}

func (sb *securityBuilder) WithAuthorization(authorizer httpsec.Authorizer) *securityBuilder {
	sb.authorizers = append(sb.authorizers, authorizer)
	sb.authorization = true
	return sb
}

func (sb *securityBuilder) WithScopes(scopes ...string) *securityBuilder {
	sb.authorization = true
	sb.authorizers = append(sb.authorizers, authz.NewScopesAuthorizer(scopes, web.GlobalAccess))
	return sb
}

func (sb *securityBuilder) WithClientIDSuffix(suffix string) *securityBuilder {
	sb.authorization = true
	sb.authorizers = append(sb.authorizers, authz.NewClientIDSuffixAuthorizer(suffix, web.GlobalAccess))
	return sb
}

func (sb *securityBuilder) WithClientID(clientID string) *securityBuilder {
	sb.authorization = true
	sb.authorizers = append(sb.authorizers, authz.NewOauthClientAuthorizer(clientID, web.GlobalAccess))
	return sb
}

func (sb *securityBuilder) SetAccessLevel(accessLevel web.AccessLevel) *securityBuilder {
	sb.authorization = true
	sb.accessLevelSet = true
	sb.accessLevel = accessLevel
	return sb
}

func (sb *securityBuilder) resetAuthenticators() {
	sb.authenticators = make([]httpsec.Authenticator, 0)
	sb.authorizers = make([]httpsec.Authorizer, 0)
	sb.accessLevelSet = false
	sb.accessLevel = web.GlobalAccess
	sb.authentication = false
	sb.authorization = false
}

func (sb *securityBuilder) reset() *securityBuilder {
	sb.resetAuthenticators()
	sb.pathMatcher = nil
	sb.methodMatcher = nil
	sb.paths = make([]string, 0)
	sb.methods = make([]string, 0)
	return sb
}

func (sb *securityBuilder) register() {
	finalMatchers := make([]web.Matcher, 0)
	if sb.pathMatcher != nil {
		finalMatchers = append(finalMatchers, sb.pathMatcher)
	}
	if sb.methodMatcher != nil {
		finalMatchers = append(finalMatchers, sb.methodMatcher)
	}

	if len(sb.authenticators) > 0 {
		finalAuthenticator := authn.NewOrAuthenticator(sb.authenticators...)

		sb.authnFilters = append(sb.authnFilters,
			secFilters.NewAuthenticationFilter(
				finalAuthenticator,
				fmt.Sprintf("%v-AuthNFilter%d@%v", sb.methods, len(sb.authnFilters), sb.paths),
				[]web.FilterMatcher{
					{
						Matchers: finalMatchers,
					},
				}))
	}

	if len(sb.authorizers) > 0 {
		finalAuthorizer := authz.NewAndAuthorizer(sb.authorizers...)
		if sb.accessLevelSet {
			finalAuthorizer = newAccessLevelAuthorizer(finalAuthorizer, sb.accessLevel)
		}
		sb.authzFilters = append(sb.authzFilters,
			secFilters.NewAuthzFilter(
				finalAuthorizer,
				fmt.Sprintf("%v-AuthZFilter%d@%v", sb.methods, len(sb.authzFilters), sb.paths),
				[]web.FilterMatcher{
					{
						Matchers: finalMatchers,
					},
				}))
	}
}

func (sb *securityBuilder) build() {
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

func newAccessLevelAuthorizer(authorizer httpsec.Authorizer, accessLevel web.AccessLevel) httpsec.Authorizer {
	return &accessLevelAuthorizer{
		authorizer:  authorizer,
		accessLevel: accessLevel,
	}
}

type accessLevelAuthorizer struct {
	authorizer  httpsec.Authorizer
	accessLevel web.AccessLevel
}

func (ala *accessLevelAuthorizer) Authorize(req *web.Request) (httpsec.Decision, web.AccessLevel, error) {
	decision, _, err := ala.authorizer.Authorize(req)
	return decision, ala.accessLevel, err
}
