/*
 * Copyright 2018 The Service Manager Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package info contains logic for the Service Manager info API
package info

import (
	"net/http"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web"
)

// URL is the path of the info endpoint
const URL = web.InfoURL

// Routes returns a slice of the routs that handle info operations
func (c *Controller) Routes() []web.Route {
	return []web.Route{
		{
			Endpoint: web.Endpoint{
				Method: http.MethodGet,
				Path:   URL,
			},
			Handler: c.getInfo,
		},
	}
}
