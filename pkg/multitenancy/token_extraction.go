package multitenancy

import (
	"encoding/json"
	"fmt"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/tidwall/gjson"
)

// ExtractTenatFromTokenWrapperFunc returns function which extracts tenant from JWT token
func ExtractTenatFromTokenWrapperFunc(clientID, clientIDTokenClaim, tenantTokenClaim string) func(request *web.Request) (string, error) {
	return func(request *web.Request) (string, error) {
		if len(clientID) == 0 {
			return "", fmt.Errorf("clientID should be provided")
		}
		if len(clientIDTokenClaim) == 0 {
			return "", fmt.Errorf("clientIDTokenClaim should be provided")
		}
		if len(tenantTokenClaim) == 0 {
			return "", fmt.Errorf("tenantTokenClaim should be provided")
		}

		ctx := request.Context()
		logger := log.C(ctx)

		user, ok := web.UserFromContext(ctx)
		if !ok {
			logger.Infof("No user found in user context. Proceeding with next filter...")
			return "", nil
		}

		if user.AuthenticationType != web.Bearer {
			logger.Infof("Authentication type is not Bearer. Proceeding with next filter...")
			return "", nil
		}

		var userData json.RawMessage
		if err := user.Data(&userData); err != nil {
			return "", fmt.Errorf("could not unmarshal claims from token: %s", err)
		}

		clientIDFromToken := gjson.GetBytes([]byte(userData), clientIDTokenClaim).String()
		if clientID != clientIDFromToken {
			logger.Infof("Token in user context was issued by %s and not by the tenant aware client %s. Proceeding with next filter...", clientIDFromToken, clientID)
			return "", nil
		}

		delimiterClaimValue := gjson.GetBytes([]byte(userData), tenantTokenClaim).String()
		if len(delimiterClaimValue) == 0 {
			return "", fmt.Errorf("invalid token: could not find delimiter %s in token claims", tenantTokenClaim)
		}
		return delimiterClaimValue, nil
	}
}
