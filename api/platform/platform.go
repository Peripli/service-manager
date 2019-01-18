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

// Package platform contains logic for the Service Manager Platform Management API
package platform

import (
	"net/http"

	"github.com/Peripli/service-manager/pkg/web"
)

// Routes returns slice of routes which handle platform operations
func (c *Controller) Routes() []web.Route {
	return []web.Route{
		{
			Endpoint: web.Endpoint{
				Method: http.MethodPost,
				Path:   web.PlatformsURL,
			},
			Handler: c.createPlatform,
		},
		{
			Endpoint: web.Endpoint{
				Method: http.MethodGet,
				Path:   web.PlatformsURL + "/{platform_id}",
			},
			Handler: c.getPlatform,
		},
		{
			Endpoint: web.Endpoint{
				Method: http.MethodGet,
				Path:   web.PlatformsURL,
			},
			Handler: c.listPlatforms,
		},
		{
			Endpoint: web.Endpoint{
				Method: http.MethodDelete,
				Path:   web.PlatformsURL,
			},
			Handler: c.deletePlatforms,
		},
		{
			Endpoint: web.Endpoint{
				Method: http.MethodDelete,
				Path:   web.PlatformsURL + "/{platform_id}",
			},
			Handler: c.deletePlatform,
		},
		{
			Endpoint: web.Endpoint{
				Method: http.MethodPatch,
				Path:   web.PlatformsURL + "/{platform_id}",
			},
			Handler: c.patchPlatform,
		},
	}
}
