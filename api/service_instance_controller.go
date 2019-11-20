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

package api

import (
	"fmt"
	"net/http"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"

	"github.com/Peripli/service-manager/pkg/web"
)

// ServiceInstanceController implements api.Controller by providing service instances API logic
type ServiceInstanceController struct {
	*BaseController
}

func NewServiceInstanceController(repository storage.Repository, defaultPageSize, maxPageSize int) *ServiceInstanceController {
	return &ServiceInstanceController{
		BaseController: NewController(repository, web.ServiceInstancesURL, types.ServiceInstanceType, func() types.Object {
			return &types.ServiceInstance{}
		}, defaultPageSize, maxPageSize),
	}
}
func (c *ServiceInstanceController) Routes() []web.Route {
	return []web.Route{
		{
			Endpoint: web.Endpoint{
				Method: http.MethodGet,
				Path:   fmt.Sprintf("%s/{%s}", web.ServiceInstancesURL, PathParamID),
			},
			Handler: c.GetSingleObject,
		},
		{
			Endpoint: web.Endpoint{
				Method: http.MethodGet,
				Path:   web.ServiceInstancesURL,
			},
			Handler: c.ListObjects,
		},
	}
}
