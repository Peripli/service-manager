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

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web"
)

const (
	// BrokerIDPathParam is a service broker ID path parameter
	BrokerIDPathParam   = "brokerID"
	InstanceIDPathParam = "instance_id"
	BindingIDPathParam  = "binding_id"

	// baseURL is the OSB API Controller path
	baseURL = web.OSBURL + "/{" + BrokerIDPathParam + "}"

	catalogURL                        = baseURL + "/v2/catalog"
	serviceInstanceURL                = baseURL + "/v2/service_instances/{instance_id}"
	serviceInstanceLastOperationURL   = baseURL + "/v2/service_instances/{instance_id}/last_operation"
	serviceBindingURL                 = baseURL + "/v2/service_instances/{instance_id}/service_bindings/{binding_id}"
	serviceBindingLastOperationURL    = baseURL + "/v2/service_instances/{instance_id}/service_bindings/{binding_id}/last_operation"
	serviceBindingAdaptCredentialsURL = baseURL + "/v2/service_instances/{instance_id}/service_bindings/{binding_id}/adapt_credentials"
)

// Routes implements api.Controller.Routes by providing the routes for the OSB API
func (c *Controller) Routes() []web.Route {
	return []web.Route{
		{Endpoint: web.Endpoint{Method: http.MethodGet, Path: catalogURL}, Handler: c.catalogHandler},

		{Endpoint: web.Endpoint{Method: http.MethodGet, Path: serviceInstanceURL}, Handler: c.proxyHandler},
		{Endpoint: web.Endpoint{Method: http.MethodPut, Path: serviceInstanceURL}, Handler: c.proxyHandler},
		{Endpoint: web.Endpoint{Method: http.MethodPatch, Path: serviceInstanceURL}, Handler: c.proxyHandler},
		{Endpoint: web.Endpoint{Method: http.MethodDelete, Path: serviceInstanceURL}, Handler: c.proxyHandler},

		{Endpoint: web.Endpoint{Method: http.MethodGet, Path: serviceBindingURL}, Handler: c.proxyHandler},
		{Endpoint: web.Endpoint{Method: http.MethodPut, Path: serviceBindingURL}, Handler: c.proxyHandler},
		{Endpoint: web.Endpoint{Method: http.MethodDelete, Path: serviceBindingURL}, Handler: c.proxyHandler},

		{Endpoint: web.Endpoint{Method: http.MethodGet, Path: serviceInstanceLastOperationURL}, Handler: c.proxyHandler},
		{Endpoint: web.Endpoint{Method: http.MethodGet, Path: serviceBindingLastOperationURL}, Handler: c.proxyHandler},

		{Endpoint: web.Endpoint{Method: http.MethodPost, Path: serviceBindingAdaptCredentialsURL}, Handler: c.proxyHandler},
	}
}
