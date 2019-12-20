package authz

import (
	"context"
	"fmt"

	"github.com/Peripli/service-manager/pkg/log"
	httpsec "github.com/Peripli/service-manager/pkg/security/http"
	"github.com/Peripli/service-manager/pkg/util/slice"
	"github.com/Peripli/service-manager/pkg/web"
)

// NewClientIDSuffixAuthorizer returns OAuth authorizer
func NewClientIDSuffixAuthorizer(trustedClientIDSuffix string, level web.AccessLevel) httpsec.Authorizer {
	return NewBaseAuthorizer(func(ctx context.Context, userContext *web.UserContext) (httpsec.Decision, web.AccessLevel, error) {
		var claims struct {
			ZID string
			CID string
		}
		logger := log.C(ctx)
		if err := userContext.Data(&claims); err != nil {
			return httpsec.Deny, web.NoAccess, fmt.Errorf("invalid token: %v", err)
		}
		logger.Debugf("User token: zid=%s cid=%s", claims.ZID, claims.CID)

		if !slice.StringsAnySuffix([]string{claims.CID}, trustedClientIDSuffix) {
			return httpsec.Deny, web.NoAccess, fmt.Errorf(`client id "%s" from user token does not have the required suffix`, claims.CID)
		}

		return httpsec.Allow, level, nil
	})
}
