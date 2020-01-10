package sm

import (
	"fmt"

	secFilters "github.com/Peripli/service-manager/pkg/security/filters"

	httpsec "github.com/Peripli/service-manager/pkg/security/http"
	"github.com/Peripli/service-manager/pkg/security/http/authn"
	"github.com/Peripli/service-manager/pkg/security/http/authz"
	"github.com/Peripli/service-manager/pkg/web"
)

// securityBuilder provides means by which authentication and authorization filters
// can be constructed and attached in a builder-pattern style through the use of methods such as:
// Path(...), Method(...), WithAuthentication(...), WithAuthorization(...) and more.
// A key part of the builder is that once you've chained all desired authentication
// and authorization settings for a specific API (a set of path and method parameters)
// you have to use one of the provided finisher methods - Required() or Optional().
// These finisher methods will ensure that the appropriate authentication/authorization filter is constructed for
// the desired path and methods. A finisher method also ensures a "clean slate" in terms of authorization
// so that you continue chaining and constructing new authorization filters.
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

	smb *ServiceManagerBuilder
}

// Optional makes authentication/authorization optional for the requested path pattern (meaning all subpaths if "*" is used) and methods.
// Optional will be applied only if there are any required paths
//
// Example 1:
//  	no matter if Required("/v1/service_brokers") is applied
//  	if Optional("/**") is set, then all subpaths will be optional
//
// Example 2:
//  	Required("/v1/**") is applied
//  	Optional("/v1/service_brokers") is applied,
//		then only "/v1/service_brokers" will be optional
//
// Best practice is to set optional paths in the end and be
// as specific as possible.
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
	sb.register(sb.finalMatchers())
	sb.resetAuthenticators()
	return sb
}

// Required makes authentication/authorization required for the path pattern and methods
// Example 1:
//  	no matter if Required("/v1/service_brokers") is applied
//  	if Optional("/**") is set, then all subpaths will be optional
//
// Example 2:
//  	Required("/v1/**") is applied
//  	Optional("/v1/service_brokers") is applied,
//		then only "/v1/service_brokers" will be optional
func (sb *securityBuilder) Required() *securityBuilder {
	finalMatchers := sb.finalMatchers()

	if sb.authorization {
		sb.requiredAuthZMatchers = append(sb.requiredAuthZMatchers, web.FilterMatcher{
			Matchers: finalMatchers,
		})
	}
	if sb.authentication {
		sb.requiredAuthNMatchers = append(sb.requiredAuthNMatchers, web.FilterMatcher{
			Matchers: finalMatchers,
		})
	}
	sb.register(finalMatchers)
	sb.resetAuthenticators()
	return sb
}

// Path specifies which paths will have authentication/authorization.
func (sb *securityBuilder) Path(paths ...string) *securityBuilder {
	sb.pathMatcher = web.Path(paths...)
	sb.paths = paths
	return sb
}

// Method specifies which methods will have authentication/authorization.
func (sb *securityBuilder) Method(methods ...string) *securityBuilder {
	sb.methodMatcher = web.Methods(methods...)
	sb.methods = methods
	return sb
}

// Authentication should be used to guarantee that a given path and method will have an authentication.
// Later on, a specific authentication can be applied for a given path/subpath
// Example:
// 		Path("/**").
// 		Method(http.MethodGet, http.MethodPut, http.MethodPost, http.MethodPatch, http.MethodDelete).
// 		Authentication().
// 		Required()
func (sb *securityBuilder) Authentication() *securityBuilder {
	sb.authentication = true
	return sb
}

// Authorization should be used to guarantee that a given path and method will have an authorization
// Later on, a specific authorization can be applied for a given path/subpath
// Example:
// 		Path("/**").
// 		Method(http.MethodGet, http.MethodPut, http.MethodPost, http.MethodPatch, http.MethodDelete).
// 		Authorization().
// 		Required()
func (sb *securityBuilder) Authorization() *securityBuilder {
	sb.authorization = true
	return sb
}

// WithAuthentication applies the provided authenticator
func (sb *securityBuilder) WithAuthentication(authenticator httpsec.Authenticator) *securityBuilder {
	sb.authenticators = append(sb.authenticators, authenticator)
	sb.authentication = true
	return sb
}

// WithAuthorization applies the provided authorizator
func (sb *securityBuilder) WithAuthorization(authorizer httpsec.Authorizer) *securityBuilder {
	sb.authorizers = append(sb.authorizers, authorizer)
	sb.authorization = true
	return sb
}

// WithScopes applies authorization mechanism, which checks the JWT scopes for the specified scopes
func (sb *securityBuilder) WithScopes(scopes ...string) *securityBuilder {
	sb.authorization = true
	sb.authorizers = append(sb.authorizers, authz.NewScopesAuthorizer(scopes, web.GlobalAccess))
	return sb
}

// WithClientIDSuffix applies authorization mechanism, which checks the JWT client id for the specified suffix
func (sb *securityBuilder) WithClientIDSuffix(suffix string) *securityBuilder {
	sb.authorization = true
	sb.authorizers = append(sb.authorizers, authz.NewClientIDSuffixAuthorizer(suffix, web.GlobalAccess))
	return sb
}

// WithClientID applies authorization mechanism, which checks the JWT client id for equality with the given one
func (sb *securityBuilder) WithClientID(clientID string) *securityBuilder {
	sb.authorization = true
	sb.authorizers = append(sb.authorizers, authz.NewOauthClientAuthorizer(clientID, web.GlobalAccess))
	return sb
}

// SetAccessLevel will set the specified access level, no matter what the authorizators returned before it.
// If this is set, it will override the default access level of the authorizers
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

func (sb *securityBuilder) register(finalMatchers []web.Matcher) {
	if len(sb.authenticators) > 0 {
		finalAuthenticator := authn.NewOrAuthenticator(sb.authenticators...)

		name := fmt.Sprintf("authN-inner-%d", len(sb.authenticators))
		sb.smb.authnDynamicFilter.AddFilter(secFilters.NewAuthenticationFilter(finalAuthenticator, name, []web.FilterMatcher{
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

		name := fmt.Sprintf("authZ-inner-%d", len(sb.authenticators))
		sb.smb.authzDynamicFilter.AddFilter(secFilters.NewAuthzFilter(finalAuthorizer, name, []web.FilterMatcher{
			{
				Matchers: finalMatchers,
			},
		}))
	}
}

func (sb *securityBuilder) build() {
	if len(sb.requiredAuthNMatchers) > 0 {
		sb.smb.authnDynamicFilter.AddFilter(secFilters.NewRequiredAuthnFilter(sb.requiredAuthNMatchers))
	}

	if len(sb.requiredAuthZMatchers) > 0 {
		sb.smb.authzDynamicFilter.AddFilter(secFilters.NewRequiredAuthzFilter(sb.requiredAuthZMatchers))
	}
}

func (sb *securityBuilder) finalMatchers() []web.Matcher {
	result := make([]web.Matcher, 0)
	if sb.pathMatcher != nil {
		result = append(result, sb.pathMatcher)
	}
	if sb.methodMatcher != nil {
		result = append(result, sb.methodMatcher)
	}
	return result
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
