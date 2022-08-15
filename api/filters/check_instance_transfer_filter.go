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

package filters

import (
	"fmt"
	"net/http"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/log"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/query"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/util"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/storage"

	"github.com/tidwall/gjson"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web"
)

const ServiceInstanceTransferFilterName = "ServiceInstanceTransferFilter"

type serviceInstanceTransferFilter struct {
	repository             storage.Repository
	enableInstanceTransfer bool
}

func NewServiceInstanceTransferFilter(repository storage.Repository, enableInstanceTransfer bool) *serviceInstanceTransferFilter {
	return &serviceInstanceTransferFilter{
		repository:             repository,
		enableInstanceTransfer: enableInstanceTransfer,
	}
}

func (*serviceInstanceTransferFilter) Name() string {
	return ServiceInstanceTransferFilterName
}

func (f *serviceInstanceTransferFilter) Run(req *web.Request, next web.Handler) (*web.Response, error) {
	ctx := req.Context()

	platformID := gjson.GetBytes(req.Body, "platform_id").String()
	if platformID == "" {
		return next.Handle(req)
	}

	if !f.enableInstanceTransfer {
		return nil, &util.HTTPError{
			ErrorType:   "TransferDisabled",
			Description: "Instance transfer is disabled in this service-manager installation",
			StatusCode:  http.StatusBadRequest,
		}
	}

	instanceID := req.PathParams[web.PathParamResourceID]
	if instanceID == "" {
		return next.Handle(req)
	}

	planID := gjson.GetBytes(req.Body, "service_plan_id").String()
	if planID == "" {
		log.C(ctx).Debug("Plan ID is not provided in the request. Fetching instance from SMDB...")
		byID := query.ByField(query.EqualsOperator, "id", instanceID)
		instanceObject, err := f.repository.Get(ctx, types.ServiceInstanceType, byID)
		if err != nil {
			return nil, util.HandleStorageError(err, types.ServiceInstanceType.String())
		}
		planID = instanceObject.(*types.ServiceInstance).ServicePlanID
	}

	byID := query.ByField(query.EqualsOperator, "id", planID)
	planObject, err := f.repository.Get(ctx, types.ServicePlanType, byID)
	if err != nil {
		return nil, util.HandleStorageError(err, types.ServicePlanType.String())
	}
	plan := planObject.(*types.ServicePlan)

	byID = query.ByField(query.EqualsOperator, "id", platformID)
	platformObject, err := f.repository.Get(ctx, types.PlatformType, byID)
	if err != nil {
		return nil, util.HandleStorageError(err, types.PlatformType.String())
	}
	platform := platformObject.(*types.Platform)
	if !plan.SupportsPlatformInstance(*platform) {
		return nil, &util.HTTPError{
			ErrorType:   "UnsupportedPlatform",
			Description: fmt.Sprintf("Instance transfer to platform of type %s failed because instance plan %s does not support this platform", platform.Type, plan.Name),
			StatusCode:  http.StatusBadRequest,
		}
	}

	byID = query.ByField(query.EqualsOperator, "id", plan.ServiceOfferingID)
	serviceObject, err := f.repository.Get(ctx, types.ServiceOfferingType, byID)
	if err != nil {
		return nil, util.HandleStorageError(err, types.ServiceOfferingType.String())
	}
	service := serviceObject.(*types.ServiceOffering)

	if !service.AllowContextUpdates {
		return nil, &util.HTTPError{
			ErrorType:   "UnsupportedContextUpdate",
			Description: fmt.Sprintf("Instance transfer to platform of type %s failed because instance service offering %s does not support context updates", platform.Type, service.Name),
			StatusCode:  http.StatusBadRequest,
		}
	}

	return next.Handle(req)
}

func (*serviceInstanceTransferFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path(web.ServiceInstancesURL + "/*"),
				web.Methods(http.MethodPatch),
			},
		},
	}
}
