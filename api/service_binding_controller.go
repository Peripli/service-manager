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
	"github.com/tidwall/gjson"
	"net/http"
)

const serviceBindingOSBURL = "%s/v2/service_instances/%s/service_bindings/%s"

// ServiceBindingController implements api.Controller by providing service bindings API logic
type ServiceBindingController struct {
	*BaseController
	osbVersion string
}

func NewServiceBindingController(ctx context.Context, options *Options) *ServiceBindingController {
	return &ServiceBindingController{
		BaseController: NewAsyncController(ctx, options, web.ServiceBindingsURL, types.ServiceBindingType, true, func() types.Object {
			return &types.ServiceBinding{}
		}, true),
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
		{
			Endpoint: web.Endpoint{
				Method: http.MethodPatch,
				Path:   fmt.Sprintf("%s/{%s}", c.resourceBaseURL, web.PathParamResourceID),
			},
			Handler: c.PatchObjectNameAndLabels,
		},
	}
}
func (c *ServiceBindingController) GetParameters(r *web.Request) (*web.Response, error) {
	isAsync := r.URL.Query().Get(web.QueryParamAsync)
	if isAsync == "true" {
		return nil, &util.HTTPError{
			ErrorType:   "InvalidRequest",
			Description: fmt.Sprintf("requested %s api doesn't support asynchronous operation.", r.URL.RequestURI()),
			StatusCode:  http.StatusBadRequest,
		}
	}
	ctx := r.Context()
	serviceBindingId := r.PathParams[web.PathParamResourceID]
	byID := query.ByField(query.EqualsOperator, "id", serviceBindingId)
	serviceBindingObject, err := c.repository.Get(ctx, types.ServiceBindingType, byID)
	if err != nil {
		return nil, util.HandleStorageError(err, types.ServiceBindingType.String())
	}
	serviceBinding := serviceBindingObject.(*types.ServiceBinding)

	// Check whether the instance is reference type or not:
	byID = query.ByField(query.EqualsOperator, "id", serviceBinding.ServiceInstanceID)
	dbInstanceObject, err := c.repository.Get(ctx, types.ServiceInstanceType, byID)
	if err != nil {
		return nil, err
	}
	instance := dbInstanceObject.(*types.ServiceInstance)
	if len(instance.ReferencedInstanceID) > 0 {
		serviceBinding.ServiceInstanceID = instance.ReferencedInstanceID
	}

	service, plan, err := storage.GetServiceOfferingAndPlanByServiceInstanceId(c.repository, ctx, serviceBinding.ServiceInstanceID)
	if err != nil {
		return nil, err
	}
	brokerObject, err := c.repository.Get(ctx, types.ServiceBrokerType, query.ByField(query.EqualsOperator, "id", service.BrokerID))
	if err != nil {
		return nil, util.HandleStorageError(err, types.ServiceBrokerType.String())
	}
	broker := brokerObject.(*types.ServiceBroker)
	if !service.BindingsRetrievable {
		return nil, &util.HTTPError{
			ErrorType:   "BadRequest",
			Description: "this operation is not supported",
			StatusCode:  http.StatusBadRequest,
		}
	}

	osbUrl := fmt.Sprintf("%s?service_id=%s&plan_id=%s", fmt.Sprintf(serviceBindingOSBURL, broker.BrokerURL, serviceBinding.ServiceInstanceID, serviceBindingId), service.CatalogID, plan.CatalogID)
	log.C(ctx).Infof("fetch binding request: %s", osbUrl)
	serviceBindingBytes, err := osb.Get(util.ClientRequest, c.osbVersion, ctx,
		broker,
		osbUrl,
		types.ServiceBindingType.String())
	if err != nil {
		return nil, err
	}

	serviceBindingResponse := &types.ServiceBinding{}
	if err := util.BytesToObject(serviceBindingBytes, &serviceBindingResponse); err != nil {
		return nil, &util.HTTPError{
			ErrorType:   "ServiceBrokerErr",
			Description: fmt.Sprintf("error reading parameters of service binding with id %s from broker %s", serviceBindingId, broker.BrokerURL),
			StatusCode:  http.StatusBadGateway,
		}
	}

	return util.NewJSONResponse(http.StatusOK, &serviceBindingResponse.Parameters)

}

func (c *ServiceBindingController) PatchObjectNameAndLabels(r *web.Request) (*web.Response, error) {
	isAsync := r.URL.Query().Get(web.QueryParamAsync)
	if isAsync == "true" {
		return nil, &util.HTTPError{
			ErrorType:   "InvalidRequest",
			Description: fmt.Sprintf("requested %s api doesn't support asynchronous operation.", r.URL.RequestURI()),
			StatusCode:  http.StatusBadRequest,
		}
	}

	bodyMap := gjson.ParseBytes(r.Body).Map()
	delete(bodyMap, "name")
	delete(bodyMap, "labels")

	if len(bodyMap) > 0 {
		return nil, &util.HTTPError{
			ErrorType:   "InvalidRequest",
			Description: fmt.Sprintf("requested %s api only supports name and label changes", r.URL.RequestURI()),
			StatusCode:  http.StatusBadRequest,
		}
	}

	return c.PatchObject(r)
}
