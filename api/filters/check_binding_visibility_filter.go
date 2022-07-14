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
	"encoding/json"
	"fmt"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
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
		return returnHttpError("BadRequest", "no tenant identifier provided", http.StatusBadRequest)
	}

	if req.Method == http.MethodDelete {
		return next.Handle(req)
	}

	var instanceID string
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

	if count == 1 {
		return next.Handle(req)
	}

	criteria = []query.Criterion{
		query.ByField(query.EqualsOperator, "id", instanceID),
		query.ByLabel(query.EqualsOperator, f.tenantIdentifier, tenantID),
	}

	count, err = f.repository.Count(ctx, types.ServiceInstanceType, criteria...)
	if err != nil {
		return nil, util.HandleStorageError(err, types.ServiceInstanceType.String())
	}

	serviceBindingForceParam := isForceBindingFlagExist(req)
	if count != 1 || !serviceBindingForceParam {
		return returnHttpError("NotFound", "service instance not found or not accessible", http.StatusNotFound)
	}

	if count == 1 {
		deletionFailed, err := isLastOperationIsDeletedFailed(f, instanceID, ctx)
		if !deletionFailed || err != nil {
			return returnHttpError("NotFound", "service instance not found, not accessible or not in deletion failed", http.StatusNotFound)
		}
		if err = addClusterIdAndNameSpaceToReqCtx(req); err != nil {
			return returnHttpError("InvalidRequest", err.Error(), http.StatusBadRequest)
		}

		criteria := []query.Criterion{query.ByField(query.EqualsOperator, "id", instanceID)}
		serviceInstance, err := f.repository.Get(ctx, types.ServiceInstanceType, criteria...)
		if err != nil {
			return nil, util.HandleStorageError(err, types.ServiceInstanceType.String())
		}
		if !isOperatedBySmaaP(serviceInstance.(*types.ServiceInstance)) {
			return returnHttpError("NotFound", "Instnace is not originated by operator", http.StatusNotFound)
		}
		return next.Handle(req)
	}

	return returnHttpError("NotFound", "service instance not found or not accessible", http.StatusNotFound)
}

func returnHttpError(errorType string, description string, statusCode int) (*web.Response, error) {
	return nil, &util.HTTPError{
		ErrorType:   errorType,
		Description: description,
		StatusCode:  statusCode,
	}
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

func isLastOperationIsDeletedFailed(f *serviceBindingVisibilityFilter, instanceID string, ctx context.Context) (bool, error) {
	byID := query.ByField(query.EqualsOperator, "resource_id", instanceID)
	orderDesc := query.OrderResultBy("paging_sequence", query.DescOrder)
	lastOperationObject, err := f.repository.Get(ctx, types.OperationType, byID, orderDesc)
	if err != nil {
		return false, err
	}

	lastOperation := lastOperationObject.(*types.Operation)
	return lastOperation.Type == types.DELETE && lastOperation.State == types.FAILED, nil
}

func isOperatedBySmaaP(instance *types.ServiceInstance) bool {
	return instance.Labels != nil && len(instance.Labels["operated_by"]) > 0
}

func isForceBindingFlagExist(req *web.Request) bool {
	parameters := gjson.GetBytes(req.Body, "parameters").Map()
	forceBinding, exists := parameters["force_bind"]
	return exists && forceBinding.String() == "true"
}
