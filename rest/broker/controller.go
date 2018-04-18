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

// Package broker contains logic for building Broker REST Controller
package broker

import (
	"net/http"

	"github.com/Peripli/service-manager/rest"
	"github.com/Peripli/service-manager/storage"
)

// Controller just to showcase usage
type Controller struct {
	BrokerStorage storage.Broker
}

// Routes just to showcase usage
func (c *Controller) Routes() []rest.Route {
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

// addBroker just to showcase usage
func (c *Controller) addBroker(response http.ResponseWriter, request *http.Request) error {
	// use broker storage
	return nil
}
