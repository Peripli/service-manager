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
	"context"
	"fmt"
	"net/http"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/web"
)

// ServiceBindingController implements api.Controller by providing service instances API logic
type ServiceBindingController struct {
	*BaseController
}

func NewServiceBindingController(ctx context.Context, options *Options) *ServiceBindingController {
	return &ServiceBindingController{
		BaseController: NewAsyncController(ctx, options, web.ServiceBindingsURL, types.ServiceBindingType, func() types.Object {
			return &types.ServiceBinding{}
		}),
	}
}

func (c *ServiceBindingController) Routes() []web.Route {
	return []web.Route{
		{
			Endpoint: web.Endpoint{
				Method: http.MethodGet,
				Path:   fmt.Sprintf("%s/{%s}", web.ServiceBindingsURL, PathParamResourceID),
			},
			Handler: c.GetSingleObject,
		},
		{
			Endpoint: web.Endpoint{
				Method: http.MethodGet,
				Path:   fmt.Sprintf("%s/{%s}%s/{%s}", c.resourceBaseURL, PathParamResourceID, web.OperationsURL, PathParamID),
			},
			Handler: c.GetOperation,
		},
		{
			Endpoint: web.Endpoint{
				Method: http.MethodGet,
				Path:   web.ServiceBindingsURL,
			},
			Handler: c.ListObjects,
		},
	}
}
