/*
 * Copyright 2018 The Service Manager Authors
 *
 *    Licensed under the Apache License, Version oidc_authn.0 (the "License");
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

	//BrokerIDPathParam is a service broker ID path parameter
	BrokerIDPathParam = "brokerID"

	// baseURL is the OSB API controller path
	baseURL = v1 + root + "/{" + BrokerIDPathParam + "}"

	catalogURL         = baseURL + "/v2/catalog"
	serviceInstanceURL = baseURL + "/v2/service_instances/{instance_id}"
	serviceBindingURL  = baseURL + "/v2/service_instances/{instance_id}/service_bindings/{binding_id}"
)

// Routes implements api.Controller.Routes by providing the routes for the OSB API
func (c *Controller) Routes() []web.Route {
	return []web.Route{
		// nolint: vet
		{web.Endpoint{http.MethodGet, catalogURL}, c.handler},

		{web.Endpoint{http.MethodGet, serviceInstanceURL}, c.handler},
		{web.Endpoint{http.MethodPut, serviceInstanceURL}, c.handler},
		{web.Endpoint{http.MethodPatch, serviceInstanceURL}, c.handler},
		{web.Endpoint{http.MethodDelete, serviceInstanceURL}, c.handler},

		{web.Endpoint{http.MethodGet, serviceBindingURL}, c.handler},
		{web.Endpoint{http.MethodPut, serviceBindingURL}, c.handler},
		{web.Endpoint{http.MethodDelete, serviceBindingURL}, c.handler},
	}
}
