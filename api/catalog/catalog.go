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

// Package catalog contains logic for the Service Manager aggregated catalog API
package catalog

import (
	"net/http"

	"github.com/Peripli/service-manager/pkg/web"
)

const (
	apiVersion = "v1"
	root       = "sm_catalog"
	// URL is the path of the aggregated catalog endpoint
	URL = "/" + apiVersion + "/" + root
)

// Routes returns slice of routes which handle catalog operations
func (c *Controller) Routes() []web.Route {
	return []web.Route{
		{
			Endpoint: web.Endpoint{
				Method: http.MethodGet,
				Path:   URL,
			},
			Handler: c.getCatalog,
		},
	}
}
