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
	"context"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/storage"
	"github.com/tidwall/gjson"
	"net/http"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/web"
)

const SharingInstanceFilterName = "SharingInstanceFilter"

// ServiceInstanceStripFilter checks patch request body for unmodifiable properties
type sharingInstanceFilter struct {
	repository        storage.TransactionalRepository
	storageRepository storage.Repository
	labelKey          string
}

func NewSharingInstanceFilter(repository storage.TransactionalRepository, storageRepository storage.Repository, labelKey string) *sharingInstanceFilter {
	return &sharingInstanceFilter{
		repository:        repository,
		storageRepository: storageRepository,
		labelKey:          labelKey,
	}
}

func (*sharingInstanceFilter) Name() string {
	return SharingInstanceFilterName
}

func (f *sharingInstanceFilter) Run(req *web.Request, next web.Handler) (*web.Response, error) {
	// Ignore the filter if has no shared property
	sharedBytes := gjson.GetBytes(req.Body, "shared")
	if len(sharedBytes.Raw) == 0 {
		return next.Handle(req)
	}

	ctx := req.Context()
	instanceID := req.PathParams["resource_id"]
	byID := query.ByField(query.EqualsOperator, "id", instanceID)
	instanceObject, err := f.repository.Get(ctx, types.ServiceInstanceType, byID)
	if err != nil {
		return nil, util.HandleStorageError(err, types.ServiceInstanceType.String())
	}
	instance := instanceObject.(*types.ServiceInstance)
	planID := instance.ServicePlanID
	byID = query.ByField(query.EqualsOperator, "id", planID)
	planObject, err := f.repository.Get(ctx, types.ServicePlanType, byID)
	if err != nil {
		return nil, util.HandleStorageError(err, types.ServicePlanType.String())
	}
	plan := planObject.(*types.ServicePlan)
	if !plan.IsShareablePlan() {
		return nil, &util.UnsupportedQueryError{
			Message: "Plan is non-shared",
		}
	}

	return next.Handle(req)
}

func (f *sharingInstanceFilter) retrieveTenantID(ctx context.Context) (string, error) {
	tenantID := query.RetrieveFromCriteria(f.labelKey, query.CriteriaForContext(ctx)...)
	if tenantID == "" {
		log.C(ctx).Errorf("Tenant identifier not found in request criteria.")
		return "", &util.HTTPError{
			ErrorType:   "BadRequest",
			Description: "no tenant identifier provided",
			StatusCode:  http.StatusBadRequest,
		}
	}
	return tenantID, nil
}

func (*sharingInstanceFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path(web.ServiceInstancesURL + "/**"),
				web.Methods(http.MethodPatch),
			},
		},
	}
}

type shareInstanceType struct {
	shared bool
}
