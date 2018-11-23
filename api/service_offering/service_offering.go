package service_offering

import (
	"net/http"

	"github.com/Peripli/service-manager/pkg/web"
)

// Routes returns slice of routes which handle service offering operations
func (c *Controller) Routes() []web.Route {
	return []web.Route{
		{
			Endpoint: web.Endpoint{
				Method: http.MethodGet,
				Path:   web.ServiceOfferingsURL + "/{service_offering_id}",
			},
			Handler: c.getServiceOffering,
		},
		{
			Endpoint: web.Endpoint{
				Method: http.MethodGet,
				Path:   web.ServiceOfferingsURL,
			},
			Handler: c.listServiceOfferings,
		},
	}
}
