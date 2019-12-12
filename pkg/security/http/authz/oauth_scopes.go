package authz

import (
	"context"
	"fmt"

	"github.com/Peripli/service-manager/pkg/log"
	httpsec "github.com/Peripli/service-manager/pkg/security/http"
	"github.com/Peripli/service-manager/pkg/util/slice"
	"github.com/Peripli/service-manager/pkg/web"
)

// // NewRequiredScopesAuthorizer returns OAuth authorizer which denys if scopes not presented
// func NewRequiredScopesAuthorizer(requiredScopes []string, level web.AccessLevel) httpsec.Authorizer {
// 	return newScopesAuthorizer(requiredScopes, true, level)
// }

// // NewOptionalScopesAuthorizer returns OAuth authorizer which abstains if scopes not presented
// func NewOptionalScopesAuthorizer(optionalScopes []string, level web.AccessLevel) httpsec.Authorizer {
// 	return newScopesAuthorizer(optionalScopes, false, level)
// }

func NewScopesAuthorizer(scopes []string, level web.AccessLevel) httpsec.Authorizer {
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
