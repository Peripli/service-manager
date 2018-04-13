// Package broker contains logic for building Broker REST Controller
package broker

import (
	"net/http"
	"github.com/Peripli/service-manager/rest"
	"github.com/Peripli/service-manager/storage"
)

// just to showcase usage
type Controller struct{
	BrokerStorage storage.Broker
}

// just to showcase usage
func (c Controller) Routes() []rest.Route {
	return []rest.Route{
		{
			Endpoint: rest.Endpoint{
				Method: http.MethodPost,
				Path:   "/api/v1/service_brokers",
			},

			// this is needed so that we can register the OSB API which provides a whole http.Router as handler
			Handler: rest.APIHandler(c.addBroker),
		},
	}
}

// just to showcase usage
func (c *Controller) addBroker(response http.ResponseWriter, request *http.Request) error {
		// use broker storage
		return nil
	}