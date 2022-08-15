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
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types"
	"net/http"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/util"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/storage"
)

const ExtractPlanIDByServiceAndPlanName = "ExtractPlanIDByServiceAndPlanName"

// NewExtractPlanIDByServiceAndPlanNameFilter creates a new extractPlanIDByServiceAndPlanNameFilter filter
func NewExtractPlanIDByServiceAndPlanNameFilter(repository storage.Repository, getVisibilityMetadata func(req *web.Request, repository storage.Repository) (*VisibilityMetadata, error)) *extractPlanIDByServiceAndPlanNameFilter {
	return &extractPlanIDByServiceAndPlanNameFilter{
		repository:            repository,
		getVisibilityMetadata: getVisibilityMetadata,
	}
}

// extractPlanIDByServiceAndPlanNameFilter convert service offering name and plan name to plan id and add it to the
// provision request body
type extractPlanIDByServiceAndPlanNameFilter struct {
	repository            storage.Repository
	getVisibilityMetadata func(req *web.Request, repository storage.Repository) (*VisibilityMetadata, error)
}

func (*extractPlanIDByServiceAndPlanNameFilter) Name() string {
	return ExtractPlanIDByServiceAndPlanName
}

func (f *extractPlanIDByServiceAndPlanNameFilter) Run(req *web.Request, next web.Handler) (*web.Response, error) {
	ctx := req.Context()

	visibilityMetadata, err := f.getVisibilityMetadata(req, f.repository)
	if err != nil {
		return nil, err
	}

	servicePlanID := gjson.GetBytes(req.Body, "service_plan_id")
	serviceOfferingName := gjson.GetBytes(req.Body, "service_offering_name")
	servicePlanName := gjson.GetBytes(req.Body, "service_plan_name")

	if servicePlanID.Exists() && !servicePlanName.Exists() && !serviceOfferingName.Exists() {
		return next.Handle(req)
	} else if !servicePlanID.Exists() && servicePlanName.Exists() && serviceOfferingName.Exists() {

		plansList, err := f.repository.QueryForList(ctx, types.ServicePlanType, storage.QueryForPlanByNameAndOfferingsWithVisibility, map[string]interface{}{
			"platform_id":           visibilityMetadata.PlatformID,
			"service_plan_name":     servicePlanName.String(),
			"service_offering_name": serviceOfferingName.String(),
			"key":                   visibilityMetadata.LabelKey,
			"val":                   visibilityMetadata.LabelValue,
		})
		if err != nil {
			return nil, util.HandleStorageError(err, types.ServicePlanType.String())
		}
		if plansList == nil || plansList.Len() == 0 {
			return nil, &util.HTTPError{
				ErrorType:   "BadRequest",
				Description: fmt.Sprintf("service plan %s not found for service offering %s", servicePlanName, serviceOfferingName),
				StatusCode:  http.StatusBadRequest,
			}
		}
		if plansList.Len() > 1 {
			return nil, &util.HTTPError{
				ErrorType:   "BadRequest",
				Description: fmt.Sprintf("ambiguity error, found more than one resource matching the provided offering name %s and plan name %s, provide the desired servicePlanID", serviceOfferingName, servicePlanName),
				StatusCode:  http.StatusBadRequest,
			}
		}
		bytes, err := sjson.SetBytes(req.Body, "service_plan_id", plansList.(*types.ServicePlans).ServicePlans[0].GetID())
		if err != nil {
			return nil, err
		}
		req.Body = bytes
		return next.Handle(req)
	} else {
		return nil, &util.HTTPError{
			ErrorType:   "BadRequest",
			Description: "Constraint violated: you have to provide parameters as following: service offering name and service plan name, or  service plan id.",
			StatusCode:  http.StatusBadRequest,
		}
	}
}

func (*extractPlanIDByServiceAndPlanNameFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path(web.ServiceInstancesURL + "/**"),
				web.Methods(http.MethodPost),
			},
		},
	}
}
