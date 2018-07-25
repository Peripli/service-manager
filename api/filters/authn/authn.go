package authn

import (
	"net/http"

	"context"

	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/security"
	"github.com/sirupsen/logrus"
)

// UserKey represents the authenticated user from the request context
const UserKey = "user"

type Middleware struct {
	authenticator security.Authenticator
	name          string
}

func (ba *Middleware) Run(next web.Handler) web.Handler {
	return web.HandlerFunc(func(request *web.Request) (*web.Response, error) {
		logrus.Debug("Entering filter: ", ba.name)
		if request.Context().Value(UserKey) != nil {
			return next.Handle(request)
		}
		user, err := ba.authenticator.Authenticate(request.Request)
		if err != nil {
			return nil, &util.HTTPError{
				ErrorType:   "Unauthorized",
				Description: "authentication failed",
				StatusCode:  http.StatusUnauthorized,
			}
		}
		request.Request = request.WithContext(context.WithValue(request.Context(), UserKey, user))
		resp, err := next.Handle(request)
		logrus.Debug("Exiting filter: ", ba.name)

		return resp, err
	})
}
