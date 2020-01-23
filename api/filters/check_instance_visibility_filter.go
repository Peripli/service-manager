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
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/util/slice"
	"github.com/Peripli/service-manager/storage"
	"net/http"

	"github.com/Peripli/service-manager/pkg/web"
	"github.com/tidwall/gjson"
)

const planIDProperty = "service_plan_id"

const ServiceInstanceVisibilityFilterName = "ServiceInstanceVisibilityFilter"

// serviceInstanceVisibilityFilter ensures that the tenant making the provisioning/update request
// has the necessary visibilities - i.e. that he has the right to consume the requested plan.
type serviceInstanceVisibilityFilter struct {
	repository       storage.Repository
	tenantIdentifier string
}

// NewServiceInstanceVisibilityFilter creates a new serviceInstanceVisibilityFilter filter
func NewServiceInstanceVisibilityFilter(repository storage.Repository, tenantIdentifier string) *serviceInstanceVisibilityFilter {
	return &serviceInstanceVisibilityFilter{
		repository:       repository,
		tenantIdentifier: tenantIdentifier,
	}
}

func (*serviceInstanceVisibilityFilter) Name() string {
	return ServiceInstanceVisibilityFilterName
}

func (f *serviceInstanceVisibilityFilter) Run(req *web.Request, next web.Handler) (*web.Response, error) {
	ctx := req.Context()
	planID := gjson.GetBytes(req.Body, planIDProperty).String()

	if planID == "" {
		log.C(ctx).Info("Plan ID is not provided in the request. Proceeding with the next handler...")
		return next.Handle(req)
	}

	byPlanID := query.ByField(query.EqualsOperator, "id", planID)
	obj, err := f.repository.Get(ctx, types.ServicePlanType, byPlanID)
	if err != nil {
		return nil, util.HandleStorageError(err, types.ServicePlanType.String())
	}

	plan := obj.(*types.ServicePlan)
	if plan.Free && !plan.SupportsPlatform(types.SMPlatform) {
		return nil, &util.HTTPError{
			ErrorType:   "BadRequest",
			Description: "provisioning instances of the provided plan is not supported for the Service Manager platform",
			StatusCode:  http.StatusBadRequest,
		}
	}

	criteria := []query.Criterion{
		query.ByField(query.EqualsOrNilOperator, platformIDProperty, types.SMPlatform),
		query.ByField(query.EqualsOperator, planIDProperty, planID),
	}

	list, err := f.repository.List(ctx, types.VisibilityType, criteria...)
	if err != nil && err != util.ErrNotFoundInStorage {
		return nil, util.HandleStorageError(err, types.VisibilityType.String())
	}

	visibilityError := &util.HTTPError{
		ErrorType:   "NotFound",
		Description: "could not find such service plan",
		StatusCode:  http.StatusNotFound,
	}
	if list.Len() == 0 {
		return nil, visibilityError
	}

	tenantID := query.RetrieveFromCriteria(f.tenantIdentifier, query.CriteriaForContext(ctx)...)
	if tenantID == "" {
		log.C(ctx).Info("Tenant identifier not found in request criteria. Proceeding with the next handler...")
		return next.Handle(req)
	}

	// There may be at most one visibility for the query - for SM platform or public for this plan
	visibility := list.ItemAt(0).(*types.Visibility)
	if len(visibility.PlatformID) == 0 { // public visibility
		return next.Handle(req)
	}
	tenantLabels, ok := visibility.Labels[f.tenantIdentifier]
	if ok && slice.StringsAnyEquals(tenantLabels, tenantID) {
		return next.Handle(req)
	}

	return nil, visibilityError
}

func (*serviceInstanceVisibilityFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path(web.ServiceInstancesURL + "/**"),
				web.Methods(http.MethodPost, http.MethodPatch),
			},
		},
	}
}
