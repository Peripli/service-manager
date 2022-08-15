package sm

import (
	"fmt"

	secFilters "github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/security/filters"

	httpsec "github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/security/http"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/security/http/authn"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/security/http/authz"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web"
)

// SecurityBuilder provides means by which authentication and authorization filters
// can be constructed and attached in a builder-pattern style through the use of methods such as:
// Path(...), Method(...), WithAuthentication(...), WithAuthorization(...) and more.
// A key part of the builder is that once you've chained all desired authentication
// and authorization settings for a specific API (a set of path and method parameters)
// you have to use one of the provided finisher methods - Required() or Optional().
// These finisher methods will ensure that the appropriate authentication/authorization filter is constructed for
// the desired path and methods. A finisher method also ensures a "clean slate" in terms of authorization
// so that you continue chaining and constructing new authorization filters.
type SecurityBuilder struct {
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

	authnDynamicFilter *web.DynamicMatchingFilter
	authzDynamicFilter *web.DynamicMatchingFilter
}

// NewSecurityBuilder should be used when someone needs to build security of the API.
// The returned filters should be attached where the authentication and authorization needs to be in the filter chain
func NewSecurityBuilder() (*SecurityBuilder, []web.Filter) {
	authnDynamicFilter := web.NewDynamicMatchingFilter(secFilters.AuthenticationFilterName)
	authzDynamicFilter := web.NewDynamicMatchingFilter(secFilters.AuthorizationFilterName)

	return &SecurityBuilder{
		authnDynamicFilter: authnDynamicFilter,
		authzDynamicFilter: authzDynamicFilter,
	}, []web.Filter{authnDynamicFilter, authzDynamicFilter}
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
func (sb *SecurityBuilder) Optional() *SecurityBuilder {
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
func (sb *SecurityBuilder) Required() *SecurityBuilder {
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
func (sb *SecurityBuilder) Path(paths ...string) *SecurityBuilder {
	sb.pathMatcher = web.Path(paths...)
	sb.paths = paths
	return sb
}

// Method specifies which methods will have authentication/authorization.
func (sb *SecurityBuilder) Method(methods ...string) *SecurityBuilder {
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
func (sb *SecurityBuilder) Authentication() *SecurityBuilder {
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
func (sb *SecurityBuilder) Authorization() *SecurityBuilder {
	sb.authorization = true
	return sb
}

// WithAuthentication applies the provided authenticator
func (sb *SecurityBuilder) WithAuthentication(authenticator httpsec.Authenticator) *SecurityBuilder {
	sb.authenticators = append(sb.authenticators, authenticator)
	sb.authentication = true
	return sb
}

// WithAuthorization applies the provided authorizator
func (sb *SecurityBuilder) WithAuthorization(authorizer httpsec.Authorizer) *SecurityBuilder {
	sb.authorizers = append(sb.authorizers, authorizer)
	sb.authorization = true
	return sb
}

// WithScopes applies authorization mechanism, which checks the JWT scopes for the specified scopes
func (sb *SecurityBuilder) WithScopes(scopes ...string) *SecurityBuilder {
	sb.authorization = true
	sb.authorizers = append(sb.authorizers, authz.NewScopesAuthorizer(scopes, web.GlobalAccess))
	return sb
}

// WithClientIDSuffix applies authorization mechanism, which checks the JWT client id for the specified suffix
func (sb *SecurityBuilder) WithClientIDSuffix(suffix string) *SecurityBuilder {
	return sb.WithClientIDSuffixes([]string{suffix})
}

// WithClientIDSuffix applies authorization mechanism, which checks the JWT client id for one of the specified suffixes
func (sb *SecurityBuilder) WithClientIDSuffixes(suffixes []string) *SecurityBuilder {
	sb.authorization = true
	sb.authorizers = append(sb.authorizers, authz.NewClientIDSuffixesAuthorizer(suffixes, web.GlobalAccess))
	return sb
}

// WithClientID applies authorization mechanism, which checks the JWT client id for equality with the given one
func (sb *SecurityBuilder) WithClientID(clientID string) *SecurityBuilder {
	sb.authorization = true
	sb.authorizers = append(sb.authorizers, authz.NewOauthClientAuthorizer(clientID, web.GlobalAccess))
	return sb
}

// SetAccessLevel will set the specified access level, no matter what the authorizators returned before it.
// If this is set, it will override the default access level of the authorizers
func (sb *SecurityBuilder) SetAccessLevel(accessLevel web.AccessLevel) *SecurityBuilder {
	sb.authorization = true
	sb.accessLevelSet = true
	sb.accessLevel = accessLevel
	return sb
}

func (sb *SecurityBuilder) resetAuthenticators() {
	sb.authenticators = make([]httpsec.Authenticator, 0)
	sb.authorizers = make([]httpsec.Authorizer, 0)
	sb.accessLevelSet = false
	sb.accessLevel = web.GlobalAccess
	sb.authentication = false
	sb.authorization = false
}

// Reset should be called before starting with new matchers
func (sb *SecurityBuilder) Reset() *SecurityBuilder {
	sb.resetAuthenticators()
	sb.pathMatcher = nil
	sb.methodMatcher = nil
	sb.paths = make([]string, 0)
	sb.methods = make([]string, 0)
	return sb
}

// Clear removes all authentication and authorization already build by the security builder
func (sb *SecurityBuilder) Clear() *SecurityBuilder {
	return sb.ClearAuthentication().ClearAuthorization()
}

// ClearAuthentication removes all authentication already build by the security builder
func (sb *SecurityBuilder) ClearAuthentication() *SecurityBuilder {
	sb.requiredAuthNMatchers = make([]web.FilterMatcher, 0)
	sb.authnDynamicFilter.ClearFilters()
	return sb.Reset()
}

// ClearAuthorization removes all authorization already build by the security builder
func (sb *SecurityBuilder) ClearAuthorization() *SecurityBuilder {
	sb.requiredAuthZMatchers = make([]web.FilterMatcher, 0)
	sb.authzDynamicFilter.ClearFilters()
	return sb.Reset()
}

func (sb *SecurityBuilder) register(finalMatchers []web.Matcher) {
	if len(sb.authenticators) > 0 {
		finalAuthenticator := authn.NewOrAuthenticator(sb.authenticators...)

		name := fmt.Sprintf("authN-inner-%d", len(sb.authenticators))
		sb.authnDynamicFilter.AddFilter(secFilters.NewAuthenticationFilter(finalAuthenticator, name, []web.FilterMatcher{
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
		sb.authzDynamicFilter.AddFilter(secFilters.NewAuthzFilter(finalAuthorizer, name, []web.FilterMatcher{
			{
				Matchers: finalMatchers,
			},
		}))
	}
}

// Builder should be called when security is ready and nothing else will be changed
func (sb *SecurityBuilder) Build() {
	if len(sb.requiredAuthNMatchers) > 0 {
		sb.authnDynamicFilter.AddFilter(secFilters.NewRequiredAuthnFilter(sb.requiredAuthNMatchers))
	}

	if len(sb.requiredAuthZMatchers) > 0 {
		sb.authzDynamicFilter.AddFilter(secFilters.NewRequiredAuthzFilter(sb.requiredAuthZMatchers))
	}
}

func (sb *SecurityBuilder) finalMatchers() []web.Matcher {
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
