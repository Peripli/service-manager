package authz

import (
	"context"
	"fmt"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/log"
	httpsec "github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/security/http"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/util/slice"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web"
)

func NewScopesAuthorizer(scopes []string, level web.AccessLevel) httpsec.Authorizer {
	return NewBaseAuthorizer(func(ctx context.Context, userContext *web.UserContext) (httpsec.Decision, web.AccessLevel, error) {
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

		return httpsec.Deny, web.NoAccess, fmt.Errorf(`none of the scopes %v are present in the user token scopes %v`, scopes, userScopes)
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

// Checks whether the user has the requested scope
func HasScope(user *web.UserContext, scope string) (bool, error) {
	var claims struct {
		Scopes []string `json:"scope"`
	}

	if err := user.Data(&claims); err != nil {
		return false, fmt.Errorf("could not extract scopes from token: %v", err)
	}
	userScopes := claims.Scopes

	return slice.StringsAnyEquals(userScopes, scope), nil
}
