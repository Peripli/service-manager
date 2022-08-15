package authz

import (
	"context"

	httpsec "github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/security/http"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web"
)

type baseAuthorizer struct {
	userProcessingFunc func(context.Context, *web.UserContext) (httpsec.Decision, web.AccessLevel, error)
}

func NewBaseAuthorizer(userProcessingFunc func(context.Context, *web.UserContext) (httpsec.Decision, web.AccessLevel, error)) *baseAuthorizer {
	return &baseAuthorizer{userProcessingFunc: userProcessingFunc}
}

func (a *baseAuthorizer) Authorize(request *web.Request) (httpsec.Decision, web.AccessLevel, error) {
	ctx := request.Context()

	user, ok := web.UserFromContext(ctx)
	if !ok {
		return httpsec.Abstain, web.NoAccess, nil
	}

	if user.AuthenticationType != web.Bearer {
		return httpsec.Abstain, web.NoAccess, nil // Not oauth
	}

	decision, accessLevel, err := a.userProcessingFunc(ctx, user)
	if err != nil {
		// denying with an error is allowed so in case of an error we return the decision as well
		return decision, accessLevel, err
	}

	request.Request = request.WithContext(web.ContextWithUser(ctx, user))

	return decision, accessLevel, nil
}
