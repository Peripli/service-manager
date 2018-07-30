package authn

import (
	"net/http"

	"context"

	"errors"

	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/security"
)

// userKey represents the authenticated user from the request context
type contextKey string

var userKey = contextKey("service-manager-authenticated-user")

func (c contextKey) String() string {
	return string(c)
}

// UserFromContext gets the authenticated user from the context.
func UserFromContext(ctx context.Context) (*security.User, bool) {
	userStr, ok := ctx.Value(userKey).(*security.User)
	return userStr, ok
}

var (
	errUnauthorized = &util.HTTPError{
		ErrorType:   "Unauthorized",
		Description: "authentication failed",
		StatusCode:  http.StatusUnauthorized,
	}
	errUserNotFound = errors.New("user identity must be provided when allowing authentication")
)

// Middleware type represents an anthentication middleware
type Middleware struct {
	authenticator security.Authenticator
	name          string
}

// Run represents the authentication middleware function that delegates the authentication
// to the provided authenticator
func (ba *Middleware) Run(next web.Handler) web.Handler {
	return web.HandlerFunc(func(request *web.Request) (*web.Response, error) {
		if _, ok := UserFromContext(request.Context()); ok {
			return next.Handle(request)
		}

		user, decision, err := ba.authenticator.Authenticate(request.Request)
		if err != nil {
			if decision == security.Deny {
				return nil, &util.HTTPError{
					ErrorType:   "Unauthorized",
					Description: err.Error(),
					StatusCode:  http.StatusUnauthorized,
				}
			}
			return nil, err
		}

		switch decision {
		case security.Allow:
			if user == nil {
				return nil, errUserNotFound
			}
			request.Request = request.WithContext(context.WithValue(request.Context(), userKey, user))
		case security.Deny:
			return nil, errUnauthorized
		}

		return next.Handle(request)
	})
}
