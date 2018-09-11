package authn

import (
	"errors"
	"net/http"

	"github.com/Peripli/service-manager/pkg/audit"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/security"
)

var (
	errUnauthorized = &util.HTTPError{
		ErrorType:   "Unauthorized",
		Description: "authentication failed",
		StatusCode:  http.StatusUnauthorized,
	}
	errUserNotFound = errors.New("user identity must be provided when allowing authentication")
)

const (
	authenticationDecisionKey = "authentication_decision"
	authenticationReasonKey   = "authentication_reason"
)

// Middleware type represents an authentication middleware
type Middleware struct {
	authenticator security.Authenticator
	name          string
}

// Run represents the authentication middleware function that delegates the authentication
// to the provided authenticator
func (ba *Middleware) Run(request *web.Request, next web.Handler) (*web.Response, error) {
	ctx := request.Context()
	if _, ok := web.UserFromContext(ctx); ok {
		return next.Handle(request)
	}

	event := audit.EventFromContext(ctx)
	user, decision, err := ba.authenticator.Authenticate(request.Request)
	if err != nil {
		if decision == security.Deny {
			err = &util.HTTPError{
				ErrorType:   "Unauthorized",
				Description: err.Error(),
				StatusCode:  http.StatusUnauthorized,
			}
		}
		audit.LogMetadata(event, authenticationDecisionKey, decision.String())
		audit.LogMetadata(event, authenticationReasonKey, err.Error())
		return nil, err
	}

	switch decision {
	case security.Allow:
		if user == nil {
			audit.LogMetadata(event, authenticationDecisionKey, security.Deny.String())
			audit.LogMetadata(event, authenticationReasonKey, "User not found")
			return nil, errUserNotFound
		}
		audit.LogMetadata(event, authenticationDecisionKey, security.Allow.String())
		request.Request = request.WithContext(web.NewContextWithUser(ctx, user))
	case security.Deny:
		audit.LogMetadata(event, authenticationDecisionKey, security.Deny.String())
		audit.LogMetadata(event, authenticationReasonKey, "Wrong credentials")
		return nil, errUnauthorized
	}
	return next.Handle(request)
}
