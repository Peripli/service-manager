package authn

import (
	"net/http"

	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/security"
)

type Midleware struct {
	authenticator security.Authenticator
}

func (ba *Midleware) Run(next web.Handler) web.Handler {
	return web.HandlerFunc(func(request *web.Request) (*web.Response, error) {
		_, err := ba.authenticator.Authenticate(request.Request)
		if err != nil {
			return nil, &util.HTTPError{
				ErrorType:   "Unauthorized",
				Description: "authentication failed",
				StatusCode:  http.StatusUnauthorized,
			}
		}
		return next.Handle(request)
	})
}
