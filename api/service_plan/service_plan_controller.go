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
	"net/http"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
)

const reqServicePlanID = "service_plan_id"

// Controller implements api.Controller by providing service plans API logic
type Controller struct {
	ServicePlanStorage storage.ServicePlan
}

func (c *Controller) getServicePlan(r *web.Request) (*web.Response, error) {
	servicePlanID := r.PathParams[reqServicePlanID]
	ctx := r.Context()
	log.C(ctx).Debugf("Getting service plan with id %s", servicePlanID)

	servicePlan, err := c.ServicePlanStorage.Get(ctx, servicePlanID)
	if err = util.HandleStorageError(err, "service_plan", servicePlanID); err != nil {
		return nil, err
	}
	return util.NewJSONResponse(http.StatusOK, servicePlan)
}

func (c *Controller) ListServicePlans(r *web.Request) (*web.Response, error) {
	var servicePlans []*types.ServicePlan
	var err error
	ctx := r.Context()
	log.C(ctx).Debug("Listing service plans")

	query := r.URL.Query()
	catalogName := query.Get("catalog_name")
	if catalogName != "" {
		log.C(ctx).Debugf("Filtering list by catalog_name=%s", catalogName)
		servicePlans, err = c.ServicePlanStorage.ListByCatalogName(ctx, catalogName)
	} else {
		servicePlans, err = c.ServicePlanStorage.List(ctx)
	}
	if err != nil {
		return nil, err
	}

	return util.NewJSONResponse(http.StatusOK, struct {
		ServicePlans []*types.ServicePlan `json:"service_plans"`
	}{
		ServicePlans: servicePlans,
	})
}
