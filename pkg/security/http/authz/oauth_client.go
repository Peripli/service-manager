package authz

import (
	"context"
	"fmt"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/log"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/security/http"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web"
)

func NewOauthClientAuthorizer(clientID string, level web.AccessLevel) http.Authorizer {
	return NewBaseAuthorizer(func(ctx context.Context, userContext *web.UserContext) (http.Decision, web.AccessLevel, error) {
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
