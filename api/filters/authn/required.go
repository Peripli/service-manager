package authn

import (
	"net/http"

	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/security"
	"github.com/sirupsen/logrus"
)

type requiredAuthnFilter struct {
}

func NewRequiredAuthnFilter() *requiredAuthnFilter {
	return &requiredAuthnFilter{}
}

func (raf *requiredAuthnFilter) Name() string {
	return "AuthenticationRequiredFilter"
}

func (raf *requiredAuthnFilter) Run(next web.Handler) web.Handler {
	return web.HandlerFunc(func(request *web.Request) (*web.Response, error) {
		logrus.Debug("Entering filter: ", raf.Name())

		user := request.Context().Value(UserKey)
		if _, ok := user.(*security.User); ok {
			resp, err := next.Handle(request)
			logrus.Debug("Exiting filter: ", raf.Name())
			return resp, err
		}

		logrus.Error("No authenticated user found in request context during execution of filter ", raf.Name())
		return nil, &util.HTTPError{
			ErrorType:   "Unauthorized",
			Description: "unsupported authentication scheme",
			StatusCode:  http.StatusUnauthorized,
		}
	})
}

func (raf *requiredAuthnFilter) RouteMatchers() []web.RouteMatcher {
	return []web.RouteMatcher{
		{
			Matchers: []web.Matcher{
				web.Path("/v1/service_brokers/**", "/v1/platforms/**", "/v1/sm_catalog", "v1/osb/**"),
			},
		},
	}
}
