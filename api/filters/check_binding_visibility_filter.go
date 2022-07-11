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
	"encoding/json"
	"fmt"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"net/http"
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
	var instanceID string

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

	if req.Method == http.MethodPatch {
		bindingID := req.PathParams[web.PathParamResourceID]
		bindingObj, err := f.repository.Get(ctx, types.ServiceBindingType, query.ByField(query.EqualsOperator, "id", bindingID))
		if err != nil {
			return nil, util.HandleStorageError(err, types.ServiceBindingType.String())
		}
		instanceID = bindingObj.(*types.ServiceBinding).ServiceInstanceID
	} else {
		instanceID = gjson.GetBytes(req.Body, serviceInstanceIDProperty).String()
	}

	if instanceID == "" {
		log.C(ctx).Info("Service Instance ID is not provided in the request. Proceeding with the next handler...")
		return next.Handle(req)
	}

	platformID := types.SMPlatform
	if web.IsSMAAPOperated(req.Context()) {
		platformID = gjson.GetBytes(req.Body, "platform_id").String()
		var err error
		if req.Body, err = sjson.DeleteBytes(req.Body, "platform_id"); err != nil {
			return nil, err
		}
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
		criteria := []query.Criterion{
			query.ByField(query.EqualsOperator, "id", instanceID),
			query.ByLabel(query.EqualsOperator, f.tenantIdentifier, tenantID),
		}

		count, err := f.repository.Count(ctx, types.ServiceInstanceType, criteria...)
		if err != nil {
			return nil, util.HandleStorageError(err, types.ServiceInstanceType.String())
		}

		if err = addClusterIdAndNameSpaceToReqCtx(req); err != nil {
			return nil, err
		}

		byID := query.ByField(query.EqualsOperator, "resource_id", instanceID)
		orderDesc := query.OrderResultBy("paging_sequence", query.DescOrder)
		lastOperationObject, err := f.repository.Get(ctx, types.OperationType, byID, orderDesc)
		if err != nil {
			return nil, err
		}

		lastOperation := lastOperationObject.(*types.Operation)
		deletionFailed := lastOperation.Type == types.DELETE && lastOperation.State == types.FAILED

		if !deletionFailed || count != 1 {
			return nil, &util.HTTPError{
				ErrorType:   "NotFound",
				Description: "service instance not found or not accessible",
				StatusCode:  http.StatusNotFound,
			}
		}
	}

	return next.Handle(req)
}

func addClusterIdAndNameSpaceToReqCtx(req *web.Request) error {
	clusterIdFieldName := "_clusterid"
	nameSpaceFieldName := "_namespace"
	labels := types.Labels{}

	labelsString := gjson.GetBytes(req.Body, "labels").Raw
	if len(labelsString) > 0 {
		err := json.Unmarshal([]byte(labelsString), &labels)
		if err != nil {
			return fmt.Errorf("could not get labels from request body: %s", err)
		}
	}

	if labels[clusterIdFieldName] == nil || labels[nameSpaceFieldName] == nil {
		return fmt.Errorf("clusterid or namespace field is missing in the body of the request")
	}

	clusterLabel := labels[clusterIdFieldName]
	namespaceLabel := labels[nameSpaceFieldName]
	clusterID := clusterLabel[0]
	namespace := namespaceLabel[0]

	var err error
	req.Body, err = sjson.SetBytes(req.Body, "context.clusterid", clusterID)
	if err != nil {
		return fmt.Errorf("could not add clusterid to context: %s", err)
	}

	req.Body, err = sjson.SetBytes(req.Body, "context.namespace", namespace)
	if err != nil {
		return fmt.Errorf("could not add namespace to context: %s", err)
	}

	return nil
}

func (*serviceBindingVisibilityFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path(web.ServiceBindingsURL + "/**"),
				web.Methods(http.MethodPost, http.MethodDelete, http.MethodPatch),
			},
		},
	}
}
