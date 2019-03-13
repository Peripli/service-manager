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

package service_plan

import (
	"fmt"
	"net/http"

	"github.com/Peripli/service-manager/api/base"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
)

// Controller implements api.Controller by providing service plans API logic
type Controller struct {
	*base.Controller
}

func (c *Controller) Routes() []web.Route {
	return []web.Route{
		{
			Endpoint: web.Endpoint{
				Method: http.MethodGet,
				Path:   fmt.Sprintf("%s/{%s}", web.ServicePlansURL, base.PathParamID),
			},
			Handler: c.GetSingleObject,
		},
		{
			Endpoint: web.Endpoint{
				Method: http.MethodGet,
				Path:   web.ServicePlansURL,
			},
			Handler: c.ListObjects,
		},
	}
}

func NewController(repository storage.Repository) *Controller {
	return &Controller{
		Controller: base.NewController(repository, web.ServicePlansURL, func() types.Object {
			return &types.ServicePlan{}
		}),
	}
}
