package authz

import (
	"context"
	"fmt"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/security/http"
	httpsec "github.com/Peripli/service-manager/pkg/security/http"
	"github.com/Peripli/service-manager/pkg/util/slice"
	"github.com/Peripli/service-manager/pkg/web"
)

// NewRequiredScopesAuthorizer returns OAuth authorizer which denys if scopes not presented
func NewRequiredScopesAuthorizer(requiredScopes []string, level web.AccessLevel) httpsec.Authorizer {
	return newScopesAuthorizer(requiredScopes, true, level)
}

// NewOptionalScopesAuthorizer returns OAuth authorizer which abstains if scopes not presented
func NewOptionalScopesAuthorizer(optionalScopes []string, level web.AccessLevel) httpsec.Authorizer {
	return newScopesAuthorizer(optionalScopes, false, level)
}

func NewOauthClientAuthorizer(clientID string, level web.AccessLevel) http.Authorizer {
	return newBaseAuthorizer(func(ctx context.Context, userContext *web.UserContext) (http.Decision, web.AccessLevel, error) {
		var cid struct {
			CID string
		}
		logger := log.C(ctx)
		if err := userContext.Data(&cid); err != nil {
			return http.Deny, web.NoAccess, fmt.Errorf("invalid token: %v", err)
		}
		logger.Debugf("User token cid=%s", cid.CID)
		if cid.CID != clientID {
			logger.Debugf(`Client id "%s" from user token does not match the required client-id %s`, cid.CID, clientID)
			return http.Deny, web.NoAccess, fmt.Errorf(`client id "%s" from user token is not trusted`, cid.CID)
		}
		return http.Allow, level, nil
	})
}

func newScopesAuthorizer(scopes []string, mandatory bool, level web.AccessLevel) httpsec.Authorizer {
	return newBaseAuthorizer(func(ctx context.Context, userContext *web.UserContext) (httpsec.Decision, web.AccessLevel, error) {
		var claims struct {
			Scopes []string `json:"scope"`
		}
		if err := userContext.Data(&claims); err != nil {
			return httpsec.Deny, web.NoAccess, fmt.Errorf("could not extract scopes from token: %v", err)
		}
		userScopes := claims.Scopes
		log.C(ctx).Debugf("User token scopes: %v", userScopes)

		for _, scope := range scopes {
			if slice.StringsAnyEquals(userScopes, scope) {
				return httpsec.Allow, level, nil
			}
		}
		if mandatory {
			return httpsec.Deny, web.NoAccess, fmt.Errorf(`none of the required scopes %v are present in the user token scopes %v`,
				scopes, userScopes)
		}

		log.C(ctx).Debugf("none of the optional scopes %v are present in the user token scopes %v", scopes, userScopes)
		return httpsec.Abstain, web.NoAccess, nil
	})
}

func PrefixScopes(space string, scopes ...string) []string {
	prefixedScopes := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		prefixedScopes = append(prefixedScopes, prefixScope(space, scope))
	}
	return prefixedScopes
}

func prefixScope(space, scope string) string {
	return fmt.Sprintf("%s.%s", space, scope)
}

func findMostRestrictiveAccessLevel(levels []web.AccessLevel) web.AccessLevel {
	min := levels[0]
	for _, level := range levels {

		if level < min {
			min = level
		}
	}
	return min
}
