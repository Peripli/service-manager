package authz

import (
	"context"
	"fmt"
	"strings"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/log"
	httpsec "github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/security/http"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web"
)

// NewClientIDSuffixAuthorizer returns OAuth authorizer
func NewClientIDSuffixAuthorizer(trustedClientIDSuffix string, level web.AccessLevel) httpsec.Authorizer {
	return NewClientIDSuffixesAuthorizer([]string{trustedClientIDSuffix}, level)
}

// NewClientIDSuffixesAuthorizer returns OAuth authorizer
func NewClientIDSuffixesAuthorizer(trustedClientIDSuffixes []string, level web.AccessLevel) httpsec.Authorizer {
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

		for _, suffix := range trustedClientIDSuffixes {
			if strings.HasSuffix(claims.CID, suffix) {
				return httpsec.Allow, level, nil
			}
		}

		return httpsec.Deny, web.NoAccess, fmt.Errorf(`client id "%s" from user token does not have the required suffix`, claims.CID)
	})
}
