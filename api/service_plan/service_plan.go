package service_plan

import (
	"net/http"

	"github.com/Peripli/service-manager/pkg/web"
)

// Routes returns slice of routes which handle service plan operations
func (c *Controller) Routes() []web.Route {
	return []web.Route{
		{
			Endpoint: web.Endpoint{
				Method: http.MethodGet,
				Path:   web.ServicePlansURL + "/{service_plan_id}",
			},
			Handler: c.getServicePlan,
		},
		{
			Endpoint: web.Endpoint{
				Method: http.MethodGet,
				Path:   web.ServicePlansURL,
			},
			Handler: c.ListServicePlans,
		},
	}
}
