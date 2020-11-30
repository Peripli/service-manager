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
	"github.com/tidwall/gjson"
	"net/http"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
)

const serviceInstanceIDProperty = "service_instance_id"

const ServiceBindingVisibilityFilterName = "ServiceBindingVisibilityFilter"

// serviceBindingVisibilityFilter ensures that the tenant making the create/delete bind request
// is the actual owner of the service instance and that the bind request is for an instance created in the SM platform.
type serviceBindingVisibilityFilter struct {
	repository       storage.Repository
	tenantIdentifier string
}

// NewServiceBindingVisibilityFilter creates a new serviceInstanceVisibilityFilter filter
func NewServiceBindingVisibilityFilter(repository storage.Repository, tenantIdentifier string) *serviceBindingVisibilityFilter {
	return &serviceBindingVisibilityFilter{
		repository:       repository,
		tenantIdentifier: tenantIdentifier,
	}
}

func (*serviceBindingVisibilityFilter) Name() string {
	return ServiceBindingVisibilityFilterName
}

func (f *serviceBindingVisibilityFilter) Run(req *web.Request, next web.Handler) (*web.Response, error) {
	ctx := req.Context()

	tenantID := query.RetrieveFromCriteria(f.tenantIdentifier, query.CriteriaForContext(ctx)...)
	if tenantID == "" {
		log.C(ctx).Errorf("Tenant identifier not found in request criteria.")
		return nil, &util.HTTPError{
			ErrorType:   "BadRequest",
			Description: "no tenant identifier provided",
			StatusCode:  http.StatusBadRequest,
		}
	}

	if req.Method == http.MethodDelete {
		return next.Handle(req)
	}

	instanceID := gjson.GetBytes(req.Body, serviceInstanceIDProperty).String()
	if instanceID == "" {
		log.C(ctx).Info("Service Instance ID is not provided in the request. Proceeding with the next handler...")
		return next.Handle(req)
	}

	platformID := types.SMPlatform
	if web.IsSMAAPOperated(req.Context()) {
		platformID = gjson.GetBytes(req.Body, "platform_id").String()
	}

	criteria := []query.Criterion{
		query.ByField(query.EqualsOperator, platformIDProperty, platformID),
		query.ByField(query.EqualsOperator, "id", instanceID),
		query.ByLabel(query.EqualsOperator, f.tenantIdentifier, tenantID),
	}

	count, err := f.repository.Count(ctx, types.ServiceInstanceType, criteria...)
	if err != nil {
		return nil, util.HandleStorageError(err, types.ServiceInstanceType.String())
	}

	if count != 1 {
		return nil, &util.HTTPError{
			ErrorType:   "NotFound",
			Description: "service instance not found or not accessible",
			StatusCode:  http.StatusNotFound,
		}
	}

	return next.Handle(req)
}

func (*serviceBindingVisibilityFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path(web.ServiceBindingsURL + "/**"),
				web.Methods(http.MethodPost, http.MethodDelete),
			},
		},
	}
}
