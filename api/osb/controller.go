/*
 * Copyright 2018 The Service Manager Authors
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */

// Package osb contains logic for building the Service Manager OSB API
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

// Controller implements rest.Controller by providing OSB API logic
type Controller struct {
	BrokerStorage storage.Broker
}

var _ rest.Controller = &Controller{}

// Routes implements rest.Controller.Routes by providing the routes for the OSB API
func (c *Controller) Routes() []rest.Route {
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

func (c *Controller) osbHandler() http.Handler {
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
