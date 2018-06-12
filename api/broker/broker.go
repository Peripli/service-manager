/*
 *    Copyright 2018 The Service Manager Authors
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

package broker

import (
	"net/http"

	"github.com/Peripli/service-manager/rest"
)

const (
	apiVersion = "v1"
	root       = "service_brokers"
	url        = "/" + apiVersion + "/" + root
)

// Routes returns slice of routes which handle broker operations
func (c *Controller) Routes() []rest.Route {
	return []rest.Route{
		{
			Endpoint: rest.Endpoint{
				Method: http.MethodPost,
				Path:   url,
			},
			Handler: rest.APIHandler(c.createBroker),
		},
		{
			Endpoint: rest.Endpoint{
				Method: http.MethodGet,
				Path:   url + "/{broker_id}",
			},
			Handler: rest.APIHandler(c.getBroker),
		},
		{
			Endpoint: rest.Endpoint{
				Method: http.MethodGet,
				Path:   url,
			},
			Handler: rest.APIHandler(c.getAllBrokers),
		},
		{
			Endpoint: rest.Endpoint{
				Method: http.MethodDelete,
				Path:   url + "/{broker_id}",
			},
			Handler: rest.APIHandler(c.deleteBroker),
		},
		{
			Endpoint: rest.Endpoint{
				Method: http.MethodPatch,
				Path:   url + "/{broker_id}",
			},
			Handler: rest.APIHandler(c.updateBroker),
		},
	}
}
