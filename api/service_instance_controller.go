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
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"net/http"
)

// ServiceInstanceController implements api.Controller by providing service Instances API logic
type ServiceInstanceController struct {
	*BaseController
}

func NewServiceInstanceController(ctx context.Context, options *Options) *ServiceInstanceController {
	return &ServiceInstanceController{
		BaseController: NewAsyncController(ctx, options, web.ServiceInstancesURL, types.ServiceInstanceType, true, func() types.Object {
			return &types.ServiceInstance{}
		}),
	}
}

func (c *ServiceInstanceController) Routes() []web.Route {
	return []web.Route{
		{
			Endpoint: web.Endpoint{
				Method: http.MethodPost,
				Path:   c.resourceBaseURL,
			},
			Handler: c.CreateObject,
		},
		{
			Endpoint: web.Endpoint{
				Method: http.MethodGet,
				Path:   fmt.Sprintf("%s/{%s}", c.resourceBaseURL, web.PathParamResourceID),
			},
			Handler: c.GetSingleObject,
		},

		{
			Endpoint: web.Endpoint{
				Method: http.MethodGet,
				Path:   fmt.Sprintf("%s/{%s}%s", c.resourceBaseURL, web.PathParamResourceID, web.ParametersURL),
			},
			Handler: c.GetParameters,
		},

		{
			Endpoint: web.Endpoint{
				Method: http.MethodGet,
				Path:   fmt.Sprintf("%s/{%s}%s/{%s}", c.resourceBaseURL, web.PathParamResourceID, web.ResourceOperationsURL, web.PathParamID),
			},
			Handler: c.GetOperation,
		},
		{
			Endpoint: web.Endpoint{
				Method: http.MethodGet,
				Path:   c.resourceBaseURL,
			},
			Handler: c.ListObjects,
		},
		{
			Endpoint: web.Endpoint{
				Method: http.MethodDelete,
				Path:   fmt.Sprintf("%s/{%s}", c.resourceBaseURL, web.PathParamResourceID),
			},
			Handler: c.DeleteSingleObject,
		},
		{
			Endpoint: web.Endpoint{
				Method: http.MethodPatch,
				Path:   fmt.Sprintf("%s/{%s}", c.resourceBaseURL, web.PathParamResourceID),
			},
			Handler: c.PatchObject,
		},
	}
}

func (c *ServiceInstanceController) GetParameters(r *web.Request) (*web.Response, error) {
	// c.repository , c.objectType
	serviceId := r.PathParams[web.PathParamResourceID]
	ctx := r.Context()
	log.C(ctx).Debugf("Getting %s with id %s", c.objectType, serviceId)
	byID := query.ByField(query.EqualsOperator, "id", serviceId)
	criteria := query.CriteriaForContext(ctx)
	obj, err := c.repository.Get(ctx, types.ServiceInstanceType, append(criteria, byID)...)
	serviceInstance := obj.(*types.ServiceInstance)
	if err != nil {
		return nil, util.HandleStorageError(err, types.ServiceInstanceType.String())
	}
	planObject, err := c.repository.Get(context.TODO(), types.ServicePlanType, query.ByField(query.EqualsOperator, "id", serviceInstance.ServicePlanID))
	if err != nil {
		//TODO construct a web response in case of error
		return nil, err
	}
	plan := planObject.(*types.ServicePlan)

	serviceObject, err := c.repository.Get(context.TODO(), types.ServiceOfferingType, query.ByField(query.EqualsOperator, "id", plan.ServiceOfferingID))
	if err!=nil{
		//contsruct an error web reponse
	}
	service := serviceObject.(*types.ServiceOffering)
	fmt.Println("service:", service)
	if service.InstancesRetrievable{
		//TODO go to osb2
	}else{
		//Construct a response "not supported"
	}
	return nil, nil
}
