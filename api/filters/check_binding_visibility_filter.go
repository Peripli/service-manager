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
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/storage"
	"github.com/tidwall/gjson"
	"net/http"

	"github.com/Peripli/service-manager/pkg/web"
)

const serviceInstanceIDProperty = "service_instance_id"

const ServiceBindingVisibilityFilterName = "ServiceBindingVisibilityFilter"

// serviceBindingVisibilityFilter ensures that the tenant making the provisioning/update request
// has the necessary visibilities.
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

	tenantID := query.RetrieveFromCriteria(f.tenantIdentifier, query.CriteriaForContext(ctx)...)[0]
	if tenantID == "" {
		log.C(ctx).Info("Tenant identifier not found in request criteria. Proceeding with the next handler...")
		return next.Handle(req)
	}

	var err error
	instanceIDs := make([]string, 0)

	switch req.Method {
	case http.MethodPost:
		instanceID := gjson.GetBytes(req.Body, serviceInstanceIDProperty)
		if !instanceID.Exists() {
			log.C(ctx).Info("Service Instance ID is not provided in the request. Proceeding with the next handler...")
			return next.Handle(req)
		}
		instanceIDs[0] = instanceID.String()
	case http.MethodDelete:
		bindingIDs := make([]string, 0)
		bindingID := req.PathParams[web.PathParamResourceID]
		if bindingID != "" {
			bindingIDs[0] = bindingID
		} else {
			bindingIDs := query.RetrieveFromCriteria("id", query.CriteriaForContext(ctx)...)
			if len(bindingIDs) == 0 {
				log.C(ctx).Info("Service Binding ID is not provided in the request. Proceeding with the next handler...")
				return next.Handle(req)
			}
		}
		instanceIDs, err = f.fetchInstanceIDs(ctx, tenantID, bindingIDs...)
	}

	if len(instanceIDs) == 0 {
		return nil, &util.HTTPError{
			ErrorType:   "NotFound",
			Description: "could not find any tenant-specific instances related to the provided binding(s)",
			StatusCode:  http.StatusNotFound,
		}
	}

	criteria := []query.Criterion{
		query.ByField(query.EqualsOperator, platformIDProperty, types.SMPlatform),
		query.ByField(query.InOperator, serviceInstanceIDProperty, instanceIDs...),
		query.ByLabel(query.InOperator, f.tenantIdentifier, tenantID),
	}

	count, err := f.repository.Count(ctx, types.ServiceInstanceType, criteria...)
	if err != nil {
		return nil, util.HandleStorageError(err, types.ServiceInstanceType.String())
	}

	if count == 0 {
		return nil, &util.HTTPError{
			ErrorType:   "NotFound",
			Description: "could not find such service binding(s)",
			StatusCode:  http.StatusNotFound,
		}
	}

	return next.Handle(req)
}

func (*serviceBindingVisibilityFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path(web.ServiceBindingsURL),
				web.Methods(http.MethodPost),
			},
		},
	}
}

func (f *serviceBindingVisibilityFilter) fetchInstanceIDs(ctx context.Context, tenantID string, bindingIDs ...string) ([]string, error) {
	criteria := []query.Criterion{
		query.ByField(query.InOperator, serviceInstanceIDProperty, bindingIDs...),
		query.ByLabel(query.InOperator, f.tenantIdentifier, tenantID),
	}

	objectList, err := f.repository.List(ctx, types.ServiceBindingType, criteria...)
	if err != nil {
		return nil, util.HandleStorageError(err, types.ServiceInstanceType.String())
	}

	serviceBindings := objectList.(*types.ServiceBindings).ServiceBindings

	instanceIDs := make([]string, 0)
	for _, sb := range serviceBindings {
		instanceIDs = append(instanceIDs, sb.ServiceInstanceID)
	}
	return instanceIDs, nil
}
