package multitenancy

import (
	"encoding/json"
	"fmt"

	"github.com/tidwall/gjson"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/log"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web"
)

// ExtractTenantFromTokenWrapperFunc returns function which extracts tenant from JWT token. The specified tenantTokenClaim
// represents the key in the token that contains the tenant identifier value
func ExtractTenantFromTokenWrapperFunc(tenantTokenClaim string) func(request *web.Request) (string, error) {
	return func(request *web.Request) (string, error) {
		if len(tenantTokenClaim) == 0 {
			return "", fmt.Errorf("tenantTokenClaim should be provided")
		}

		ctx := request.Context()
		logger := log.C(ctx)

		user, ok := web.UserFromContext(ctx)
		if !ok {
			logger.Infof("No user found in user context. Proceeding with empty tenant ID value...")
			return "", nil
		}

		if user.AuthenticationType != web.Bearer {
			logger.Infof("Authentication type is not Bearer. Proceeding with empty tenant ID value...")
			return "", nil
		}

		var userData json.RawMessage
		if err := user.Data(&userData); err != nil {
			return "", fmt.Errorf("could not unmarshal claims from token: %s", err)
		}

		delimiterClaimValue := gjson.GetBytes([]byte(userData), tenantTokenClaim).String()
		if len(delimiterClaimValue) == 0 {
			return "", fmt.Errorf("invalid token: could not find delimiter %s in token claims", tenantTokenClaim)
		}

		logger.Infof("Successfully set tenant ID to %s", delimiterClaimValue)
		return delimiterClaimValue, nil
	}
}
