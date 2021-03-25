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
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/tidwall/gjson"
	"net/http"

	"github.com/Peripli/service-manager/storage"

	"github.com/Peripli/service-manager/pkg/web"
)

const ReferenceInstanceOwnershipFilterName = "ReferenceInstanceOwnershipFilter"

// serviceInstanceVisibilityFilter ensures that the tenant making the provisioning/update request
// has the necessary visibilities - i.e. that he has the right to consume the requested plan.
type referenceInstanceOwnershipFilter struct {
	repository       storage.Repository
	tenantIdentifier string
}

func NewReferenceInstanceOwnershipFilter(repository storage.Repository, tenantIdentifier string) *referenceInstanceOwnershipFilter {
	return &referenceInstanceOwnershipFilter{
		repository:       repository,
		tenantIdentifier: tenantIdentifier,
	}
}

func (*referenceInstanceOwnershipFilter) Name() string {
	return ReferenceInstanceOwnershipFilterName
}

func (f *referenceInstanceOwnershipFilter) Run(req *web.Request, next web.Handler) (*web.Response, error) {
	ctx := req.Context()
	userContext, _ := web.UserFromContext(ctx)
	fmt.Print(userContext)
	planID := gjson.GetBytes(req.Body, planIDProperty).String()

	byID := query.ByField(query.EqualsOperator, "id", planID)
	planObject, err := f.repository.Get(ctx, types.ServicePlanType, byID)

	if err != nil {
		return nil, util.HandleStorageError(err, types.ServicePlanType.String())
	}

	plan := planObject.(*types.ServicePlan)

	if plan.Name != "reference-plan" {
		return next.Handle(req)
	}

	referencedInstanceID := gjson.GetBytes(req.Body, "parameters.referenced_instance_id").String()
	byID = query.ByField(query.EqualsOperator, "id", referencedInstanceID)

	referencedObject, err := f.repository.Get(ctx, types.ServiceInstanceType, byID)
	if err != nil {
		return nil, util.HandleStorageError(err, types.ServiceInstanceType.String())
	}
	instance := referencedObject.(*types.ServiceInstance)
	referencedOwnerTenantID := instance.Labels["tenant"][0]

	if !*instance.Shared {
		return nil, &util.HTTPError{
			ErrorType:   "BadRequest",
			Description: "The referenced instance is not shared",
			StatusCode:  http.StatusBadRequest,
		}
	}

	callerTenantID := gjson.GetBytes(req.Body, "contextt."+f.tenantIdentifier).String()
	if referencedOwnerTenantID != callerTenantID {
		log.C(ctx).Errorf("Instance owner %s is not the same as the caller %s", referencedOwnerTenantID, callerTenantID)
		return nil, &util.HTTPError{
			ErrorType:   "NotFound",
			Description: "could not find such service instance",
			StatusCode:  http.StatusNotFound,
		}
	}

	return next.Handle(req)
	//return nil, nil
}

func (*referenceInstanceOwnershipFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path(web.ServiceInstancesURL + "/**"),
				web.Methods(http.MethodPost),
			},
		},
	}
}