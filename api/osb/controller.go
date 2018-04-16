package osb

import (
	"net/http"

	"github.com/Peripli/service-manager/rest"
	"github.com/Peripli/service-manager/storage"
	osbc "github.com/pmorie/go-open-service-broker-client/v2"
	"github.com/pmorie/osb-broker-lib/pkg/metrics"
	osbrest "github.com/pmorie/osb-broker-lib/pkg/rest"
	"github.com/pmorie/osb-broker-lib/pkg/server"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

type Controller struct {
	BrokerStorage storage.Broker
}

func (c Controller) Routes() []rest.Route {
	return []rest.Route{
		{
			Endpoint: rest.Endpoint{
				Method: rest.AllMethods,
				Path:   "/osb/{brokerID}",
			},
			Handler: c.osbHandler(),
		},
	}
}

func (c Controller) osbHandler() http.Handler {
	businessLogic := NewBusinessLogic(osbc.NewClient, c.BrokerStorage)

	reg := prom.NewRegistry()
	osbMetrics := metrics.New()
	reg.MustRegister(osbMetrics)

	api, err := osbrest.NewAPISurface(businessLogic, osbMetrics)
	if err != nil {
		logrus.Fatalf("Error creating OSB API surface: %s", err)
	}

	return server.NewHTTPHandler(api)
}
