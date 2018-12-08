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

// Package visibility contains logic for building the Service Manager visibilities API
package visibility

import (
	"net/http"

	"github.com/Peripli/service-manager/pkg/web"
)

// Routes returns slice of routes which handle broker operations
func (c *Controller) Routes() []web.Route {
	return []web.Route{
		{
			Endpoint: web.Endpoint{
				Method: http.MethodPost,
				Path:   web.VisibilitiesURL,
			},
			Handler: c.createVisibility,
		},
		{
			Endpoint: web.Endpoint{
				Method: http.MethodGet,
				Path:   web.VisibilitiesURL + "/{visibility_id}",
			},
			Handler: c.getVisibility,
		},
		{
			Endpoint: web.Endpoint{
				Method: http.MethodGet,
				Path:   web.VisibilitiesURL,
			},
			Handler: c.listVisibilities,
		},
		{
			Endpoint: web.Endpoint{
				Method: http.MethodDelete,
				Path:   web.VisibilitiesURL + "/{visibility_id}",
			},
			Handler: c.deleteVisibility,
		},
		{
			Endpoint: web.Endpoint{
				Method: http.MethodPatch,
				Path:   web.VisibilitiesURL + "/{visibility_id}",
			},
			Handler: c.patchVisibility,
		},
	}
}
