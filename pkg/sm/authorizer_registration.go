package sm

import (
	"github.com/Peripli/service-manager/pkg/security/filters"

	"github.com/Peripli/service-manager/pkg/security/http/authz"

	"github.com/Peripli/service-manager/pkg/log"
	httpsec "github.com/Peripli/service-manager/pkg/security/http"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/web"
)

var TypeToPath = map[types.ObjectType]string{
	types.ServiceBrokerType:   web.ServiceBrokersURL + "/**",
	types.PlatformType:        web.PlatformsURL + "/**",
	types.ServiceOfferingType: web.ServiceOfferingsURL + "/**",
	types.ServicePlanType:     web.ServicePlansURL + "/**",
	types.VisibilityType:      web.VisibilitiesURL + "/**",
	types.NotificationType:    web.NotificationsURL + "/**",
	types.ServiceInstanceType: web.ServiceInstancesURL + "/**",
}

type authorizerBuilder struct {
	parent *authorizerBuilder

	objectType types.ObjectType
	path       string
	methods    []string

	authorizers           []httpsec.Authorizer
	optional              bool
	cloneSpace            string
	clientID              string
	trustedClientIDSuffix string
	attachFunc            func(web.Filter)
	done                  func() *ServiceManagerBuilder
}

func (ab *authorizerBuilder) Configure(cloneSpace, clientID, trustedClientIDSuffix string) *authorizerBuilder {
	ab.cloneSpace = cloneSpace
	ab.clientID = clientID
	ab.trustedClientIDSuffix = trustedClientIDSuffix
	return ab
}

func (ab *authorizerBuilder) Custom(authorizer httpsec.Authorizer) *authorizerBuilder {
	ab.authorizers = append(ab.authorizers, authorizer)
	return ab
}

func (ab *authorizerBuilder) Global(scopes ...string) *authorizerBuilder {
	ab.authorizers = append(ab.authorizers, authz.NewAndAuthorizer(
		authz.NewOAuthCloneAuthorizer(ab.trustedClientIDSuffix, web.GlobalAccess),
		authz.NewRequiredScopesAuthorizer(authz.PrefixScopes(ab.cloneSpace, scopes...), web.GlobalAccess),
	))
	return ab
}

func (ab *authorizerBuilder) Tenant(tenantScopes ...string) *authorizerBuilder {
	ab.authorizers = append(ab.authorizers, authz.NewAndAuthorizer(
		authz.NewOrAuthorizer(
			authz.NewOauthClientAuthorizer(ab.clientID, web.GlobalAccess),
			authz.NewOAuthCloneAuthorizer(ab.trustedClientIDSuffix, web.GlobalAccess),
		),
		authz.NewRequiredScopesAuthorizer(authz.PrefixScopes(ab.cloneSpace, tenantScopes...), web.TenantAccess),
	))
	return ab
}

func (ab *authorizerBuilder) AllTenant(allTenantScopes ...string) *authorizerBuilder {
	ab.authorizers = append(ab.authorizers, authz.NewAndAuthorizer(
		authz.NewOAuthCloneAuthorizer(ab.trustedClientIDSuffix, web.GlobalAccess),
		authz.NewRequiredScopesAuthorizer(authz.PrefixScopes(ab.cloneSpace, allTenantScopes...), web.AllTenantAccess),
	))
	return ab
}

func (ab *authorizerBuilder) Basic(access web.AccessLevel) *authorizerBuilder {
	ab.authorizers = append(ab.authorizers, authz.NewBasic(access))
	return ab
}

func (ab *authorizerBuilder) For(methods ...string) *authorizerBuilder {
	ab.methods = methods
	return ab
}

func (ab *authorizerBuilder) Optional() *authorizerBuilder {
	ab.optional = true
	return ab
}

func (ab *authorizerBuilder) And() *authorizerBuilder {
	return &authorizerBuilder{
		parent:                ab,
		path:                  ab.path,
		objectType:            ab.objectType,
		cloneSpace:            ab.cloneSpace,
		clientID:              ab.clientID,
		trustedClientIDSuffix: ab.trustedClientIDSuffix,
		attachFunc:            ab.attachFunc,
		done:                  ab.done,
		optional:              false,
	}
}

func (ab *authorizerBuilder) Register() *ServiceManagerBuilder {
	current := ab
	for current != nil {
		path := current.path
		if len(path) == 0 {
			path = TypeToPath[current.objectType]
		}
		if len(current.methods) == 0 {
			log.D().Panicf("Cannot register authorizers at %s with no methods", path)
		}
		if len(current.authorizers) == 0 {
			log.D().Panicf("Cannot register 0 authorizers at %s for %v", path, current.methods)
		}
		finalAuthorizer := authz.NewOrAuthorizer(current.authorizers...)
		if !ab.optional {
			finalAuthorizer = authz.NewAndAuthorizer(finalAuthorizer, NewRequiredAuthz())
		}
		filter := filters.NewAuthzFilter(current.methods, path, finalAuthorizer)
		current.attachFunc(filter)
		current = current.parent
	}
	return ab.done()
}

func NewRequiredAuthz() httpsec.Authorizer {
	return &requiredAuthz{}
}

type requiredAuthz struct{}

func (raz *requiredAuthz) Authorize(req *web.Request) (httpsec.Decision, web.AccessLevel, error) {
	ctx := req.Context()
	user, found := web.UserFromContext(ctx)
	if web.IsAuthorized(ctx) && found {
		return httpsec.Allow, user.AccessLevel, nil
	}
	return httpsec.Deny, web.NoAccess, nil
}
