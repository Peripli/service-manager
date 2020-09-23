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
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/storage"
	"net/http"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/web"
)

// ServiceBindingController implements api.Controller by providing service bindings API logic
type ServiceBindingController struct {
	*BaseController
	osbVersion string
}

func NewServiceBindingController(ctx context.Context, options *Options) *ServiceBindingController {
	return &ServiceBindingController{
		BaseController: NewAsyncController(ctx, options, web.ServiceBindingsURL, types.ServiceBindingType, true, func() types.Object {
			return &types.ServiceBinding{}
		}),
		osbVersion: options.APISettings.OSBVersion,
	}
}

func (c *ServiceBindingController) Routes() []web.Route {
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
				Method: http.MethodGet,
				Path:   fmt.Sprintf("%s/{%s}/{%s}", c.resourceBaseURL, web.PathParamResourceID, web.ParametersURL),
			},
			Handler: c.GetParameters,
		},
		{
			Endpoint: web.Endpoint{
				Method: http.MethodDelete,
				Path:   fmt.Sprintf("%s/{%s}", c.resourceBaseURL, web.PathParamResourceID),
			},
			Handler: c.DeleteSingleObject,
		},
	}
}
func (c *ServiceBindingController) GetParameters(r *web.Request) (*web.Response, error) {
	ctx := r.Context()
	serviceBindingId := r.PathParams[web.PathParamResourceID]
	service, err := storage.GetServiceByServiceBinding(c.repository, ctx, serviceBindingId)
	if err != nil {
		return nil, err
	}
	brokerObject, err := c.repository.Get(ctx, types.ServiceBrokerType, query.ByField(query.EqualsOperator, "id", service.BrokerID))
	if err != nil {
		return nil, util.HandleStorageError(err, types.ServiceBrokerType.String())
	}
	broker := brokerObject.(*types.ServiceBroker)
	if service.BindingsRetrievable {
		serviceBindingBytes, err := osb.Get(util.ClientRequest, c.osbVersion, ctx,
			broker,
			fmt.Sprintf(osb.ServiceBindingURL, broker.BrokerURL, service.ID, serviceBindingId),
			types.ServiceBindingType.String(),
			serviceBindingId)
		if err != nil {
			return nil, &util.HTTPError{
				ErrorType:   "ServiceBrokerErr",
				Description: fmt.Sprintf("Error sending request to the broker %s", broker.BrokerURL),
				StatusCode:  http.StatusInternalServerError,
			}
		}

		serviceBindingResponse := &types.ServiceBinding{}
		if err := util.BytesToObject(serviceBindingBytes, &serviceBindingResponse); err != nil {
			return nil, &util.HTTPError{
				ErrorType:   "ServiceBrokerErr",
				Description: fmt.Sprintf("Error reading parameters of service binding with id %s from broker broker %s", serviceBindingId, broker.BrokerURL),
				StatusCode:  http.StatusInternalServerError,
			}
		}

		return util.NewJSONResponse(http.StatusOK, &serviceBindingResponse.Parameters)

	}

	return nil, &util.HTTPError{
		ErrorType:   "BadRequest",
		Description: fmt.Sprintf("This operation is not supported"),
		StatusCode:  http.StatusBadRequest,
	}
}
