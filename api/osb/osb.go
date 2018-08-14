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

	"github.com/Peripli/service-manager/pkg/web"
)

const (
	// v1 is a prefix the first version of the OSB API
	v1 = "/v1"

	// root is a prefix for the OSB API
	root = "/osb"

	// BrokerIDPathParam is a service broker ID path parameter
	BrokerIDPathParam = "brokerID"

	// baseURL is the OSB API controller path
	baseURL = v1 + root + "/{" + BrokerIDPathParam + "}"

	catalogURL                      = baseURL + "/v2/catalog"
	serviceInstanceURL              = baseURL + "/v2/service_instances/{instance_id}"
	serviceInstanceLastOperationURL = baseURL + "/v2/service_instances/{instance_id}/last_operation"
	serviceBindingURL               = baseURL + "/v2/service_instances/{instance_id}/service_bindings/{binding_id}"
	serviceBindingLastOperationURL  = baseURL + "/v2/service_instances/{instance_id}/service_bindings/{binding_id}/last_operation"
)

// Routes implements api.Controller.Routes by providing the routes for the OSB API
func (c *controller) Routes() []web.Route {
	handler := c.adapter.Handler()

	return []web.Route{
		{Endpoint: web.Endpoint{Method: http.MethodGet, Path: catalogURL}, Handler: handler},

		{Endpoint: web.Endpoint{Method: http.MethodGet, Path: serviceInstanceURL}, Handler: handler},
		{Endpoint: web.Endpoint{Method: http.MethodPut, Path: serviceInstanceURL}, Handler: handler},
		{Endpoint: web.Endpoint{Method: http.MethodPatch, Path: serviceInstanceURL}, Handler: handler},
		{Endpoint: web.Endpoint{Method: http.MethodDelete, Path: serviceInstanceURL}, Handler: handler},

		{Endpoint: web.Endpoint{Method: http.MethodGet, Path: serviceBindingURL}, Handler: handler},
		{Endpoint: web.Endpoint{Method: http.MethodPut, Path: serviceBindingURL}, Handler: handler},
		{Endpoint: web.Endpoint{Method: http.MethodDelete, Path: serviceBindingURL}, Handler: handler},

		{Endpoint: web.Endpoint{Method: http.MethodGet, Path: serviceInstanceLastOperationURL}, Handler: handler},
		{Endpoint: web.Endpoint{Method: http.MethodGet, Path: serviceBindingLastOperationURL}, Handler: handler},
	}
}
