package profile

import (
	"net/http"
	"net/http/pprof"

	"github.com/Peripli/service-manager/pkg/web"
)

const profileNameParam = "profile_name"

// Controller profile controller
type Controller struct{}

// Routes provides the REST endpoints
func (c *Controller) Routes() []web.Route {
	return []web.Route{
		{
			Endpoint: web.Endpoint{
				Method: http.MethodGet,
				Path:   web.ProfileURL + "/{" + profileNameParam + "}",
			},
			Handler: c.profile,
		},
	}
}

func (c *Controller) profile(req *web.Request) (*web.Response, error) {
	profileName := req.PathParams[profileNameParam]
	resp := req.HijackResponseWriter()
	if profileName == "profile" {
		pprof.Profile(resp, req.Request)
	} else {
		pprof.Handler(profileName).ServeHTTP(resp, req.Request)
	}

	return &web.Response{}, nil
}
