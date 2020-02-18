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
	"net/http"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/storage"
	"github.com/tidwall/gjson"

	"github.com/Peripli/service-manager/pkg/web"
)

const serviceInstanceIDProperty = "service_instance_id"

const ServiceBindingVisibilityFilterName = "ServiceBindingVisibilityFilter"

// serviceBindingVisibilityFilter ensures that the tenant making the create/delete bind request
// is the actual owner of the service instance and that the bind request is for an instance created in the SM platform.
type serviceBindingVisibilityFilter struct {
	repository                    storage.Repository
	getInstanceVisibilityMetadata func(req *web.Request, repository storage.Repository) (*InstanceVisibilityMetadata, error)
}

// NewServiceBindingVisibilityFilter creates a new serviceInstanceVisibilityFilter filter
func NewServiceBindingVisibilityFilter(repository storage.Repository, getInstanceVisibilityMetadata func(req *web.Request, repository storage.Repository) (*InstanceVisibilityMetadata, error)) *serviceBindingVisibilityFilter {
	return &serviceBindingVisibilityFilter{
		repository:                    repository,
		getInstanceVisibilityMetadata: getInstanceVisibilityMetadata,
	}
}

func (*serviceBindingVisibilityFilter) Name() string {
	return ServiceBindingVisibilityFilterName
}

func (f *serviceBindingVisibilityFilter) Run(req *web.Request, next web.Handler) (*web.Response, error) {
	ctx := req.Context()

	var err error
	var instanceID string

	visibilityMetadata, err := f.getInstanceVisibilityMetadata(req, f.repository)
	if err != nil {
		return nil, err
	}

	switch req.Method {
	case http.MethodPost:
		instanceID = gjson.GetBytes(req.Body, serviceInstanceIDProperty).String()
		if instanceID == "" {
			log.C(ctx).Info("Service Instance ID is not provided in the request. Proceeding with the next handler...")
			return next.Handle(req)
		}
	case http.MethodDelete:
		bindingID := req.PathParams[web.PathParamResourceID]
		if bindingID != "" {
			log.C(ctx).Info("Service Binding ID is not provided in the request. Proceeding with the next handler...")
			return next.Handle(req)
		}
		instanceID, err = f.fetchInstanceID(ctx, visibilityMetadata.LabelKey, visibilityMetadata.LabelValue, bindingID)
		if err != nil {
			return nil, err
		}
	}

	criteria := []query.Criterion{
		query.ByField(query.EqualsOperator, platformIDProperty, visibilityMetadata.PlatformID),
		query.ByField(query.EqualsOperator, "id", instanceID),
		query.ByLabel(query.EqualsOperator, visibilityMetadata.LabelKey, visibilityMetadata.LabelValue),
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

func (f *serviceBindingVisibilityFilter) fetchInstanceID(ctx context.Context, labelKey, labelValue, bindingID string) (string, error) {
	criteria := []query.Criterion{
		query.ByField(query.EqualsOperator, "id", bindingID),
		query.ByLabel(query.EqualsOperator, labelKey, labelValue),
	}

	object, err := f.repository.Get(ctx, types.ServiceBindingType, criteria...)
	if err != nil {
		return "", util.HandleStorageError(err, types.ServiceBindingType.String())
	}

	sb := object.(*types.ServiceBinding)
	return sb.ServiceInstanceID, nil
}
