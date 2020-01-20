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
	"github.com/Peripli/service-manager/storage"
	"github.com/tidwall/gjson"
	"net/http"

	"github.com/Peripli/service-manager/pkg/web"
)

const ServiceBindingOwnershipFilterName = "ServiceBindingOwnershipFilter"

// serviceBindingOwnershipFilter ensures that the tenant making the create/delete binding request
// is the actual instance owner
type serviceBindingOwnershipFilter struct {
	repository       storage.Repository
	tenantIdentifier string
}

// NewServiceInstanceOwnershipFilter creates a new serviceBindingOwnershipFilter filter
func NewServiceBindingOwnershipFilter(repository storage.Repository, tenantIdentifier string) *serviceBindingOwnershipFilter {
	return &serviceBindingOwnershipFilter{
		repository:       repository,
		tenantIdentifier: tenantIdentifier,
	}
}

func (*serviceBindingOwnershipFilter) Name() string {
	return ServiceBindingOwnershipFilterName
}

func (f *serviceBindingOwnershipFilter) Run(req *web.Request, next web.Handler) (*web.Response, error) {
	ctx := req.Context()
	var resourceID string
	var objectType types.ObjectType

	switch req.Method {
	case http.MethodPost:
		objectType = types.ServiceInstanceType
		resourceID = gjson.GetBytes(req.Body, serviceInstanceIDProperty).Str
	case http.MethodDelete:
		objectType = types.ServiceBindingType
		resourceID = req.PathParams[web.PathParamResourceID]
		if resourceID == "" {
			resourceID = query.RetrieveFromCriteria("id", query.CriteriaForContext(ctx)...)
		}
	default:
		return nil, &util.HTTPError{
			ErrorType:   "BadRequest",
			Description: fmt.Sprintf("Invalid HTTP method"),
			StatusCode:  http.StatusBadRequest,
		}
	}

	if resourceID == "" {
		log.C(ctx).Info("Service Instance/Binding ID is not provided in the request. Proceeding with the next handler...")
		return next.Handle(req)
	}

	tenantID := query.RetrieveFromCriteria(f.tenantIdentifier, query.CriteriaForContext(ctx)...)
	if tenantID == "" {
		log.C(ctx).Info("Tenant identifier not found in request criteria. Proceeding with the next handler...")
		return next.Handle(req)
	}

	criteria := []query.Criterion{
		query.ByField(query.EqualsOperator, "id", resourceID),
		query.ByLabel(query.InOperator, f.tenantIdentifier, tenantID),
	}

	_, err := f.repository.Get(ctx, objectType, criteria...)
	if err != nil {
		return nil, util.HandleStorageError(err, objectType.String())
	}

	return next.Handle(req)
}

func (*serviceBindingOwnershipFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path(web.ServiceBindingsURL + "/**"),
				web.Methods(http.MethodPost, http.MethodDelete),
			},
		},
	}
}
