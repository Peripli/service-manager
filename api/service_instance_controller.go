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
	"github.com/Peripli/service-manager/api/osb"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
	"net/http"
)

const serviceInstanceOSBURL string= "%s/v2/service_instances/%s"
// ServiceInstanceController implements api.Controller by providing service Instances API logic
type ServiceInstanceController struct {
	*BaseController
	osbVersion string
}

func NewServiceInstanceController(ctx context.Context, options *Options) *ServiceInstanceController {

	return &ServiceInstanceController{
		BaseController: NewAsyncController(ctx, options, web.ServiceInstancesURL, types.ServiceInstanceType, true, func() types.Object {
			return &types.ServiceInstance{}
		}),
		osbVersion: options.APISettings.OSBVersion,
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
	isAsync := r.URL.Query().Get(web.QueryParamAsync)
	if isAsync == "true" {
		return nil, &util.HTTPError{
			ErrorType:   "InvalidRequest",
			Description: fmt.Sprintf("requested %s api doesn't support asynchronous operation", r.URL.RequestURI()),
			StatusCode:  http.StatusBadRequest,
		}
	}
	serviceInstanceId := r.PathParams[web.PathParamResourceID]
	ctx := r.Context()
	log.C(ctx).Debugf("Getting %s with id %s", c.objectType, serviceInstanceId)

	service, err := storage.GetServiceOfferingByServiceInstanceId(c.repository, ctx, serviceInstanceId)
	if err != nil {
		return nil, err
	}
	brokerObject, err := c.repository.Get(ctx, types.ServiceBrokerType, query.ByField(query.EqualsOperator, "id", service.BrokerID))
	if err != nil {
		return nil, util.HandleStorageError(err, types.ServiceBrokerType.String())
	}
	broker := brokerObject.(*types.ServiceBroker)
	if service.InstancesRetrievable {
		serviceInstanceBytes, err := osb.Get(util.ClientRequest, c.osbVersion, ctx,
			broker,
			fmt.Sprintf(serviceInstanceOSBURL, broker.BrokerURL, serviceInstanceId),
			types.ServiceInstanceType.String())

		if err != nil {
			return nil, err
		}

		serviceResponse := &types.ServiceInstance{}
		if err := util.BytesToObject(serviceInstanceBytes, &serviceResponse); err != nil {
			return nil, &util.HTTPError{
				ErrorType:   "ServiceBrokerErr",
				Description: fmt.Sprintf("Error reading parameters of service instance with id %s from broker broker %s", serviceInstanceId, broker.BrokerURL),
				StatusCode:  http.StatusInternalServerError,
			}
		}

		return util.NewJSONResponse(http.StatusOK, &serviceResponse.Parameters)

	}

	return nil, &util.HTTPError{
		ErrorType:   "BadRequest",
		Description: fmt.Sprintf("This operation is not supported"),
		StatusCode:  http.StatusBadRequest,
	}
}
