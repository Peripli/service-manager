package authn

import (
	"net/http"

	"context"

	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/security"
)

type Middleware struct {
	authenticator security.Authenticator
}

func (ba *Middleware) Run(next web.Handler) web.Handler {
	return web.HandlerFunc(func(request *web.Request) (*web.Response, error) {
		if request.Context().Value("user") != nil {
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
		request.Request = request.WithContext(context.WithValue(request.Context(), "user", user))
		return next.Handle(request)
	})
}
