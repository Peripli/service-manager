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
	"errors"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/gofrs/uuid"
	"github.com/tidwall/gjson"
	"net/http"
	"strings"
	"time"

	"github.com/Peripli/service-manager/storage"

	"github.com/Peripli/service-manager/pkg/web"
)

const (
	ReferenceInstanceFilterName        = "ReferenceInstanceFilter"
	ReferenceParametersAreNotSupported = "Reference service instance parameters are not supported"
)

// serviceInstanceVisibilityFilter ensures that the tenant making the provisioning/update request
// has the necessary visibilities - i.e. that he has the right to consume the requested plan.
type referenceInstanceFilter struct {
	repository storage.TransactionalRepository
}

func NewReferenceInstanceFilter(repository storage.TransactionalRepository) *referenceInstanceFilter {
	return &referenceInstanceFilter{
		repository: repository,
	}
}

func (*referenceInstanceFilter) Name() string {
	return ReferenceInstanceFilterName
}

func (f *referenceInstanceFilter) Run(req *web.Request, next web.Handler) (*web.Response, error) {
	ctx := req.Context()

	// todo: manage operations
	// todo: allow async=true
	//resourceID := req.PathParams["resource_id"]
	//isAsync := req.URL.Query().Get(web.QueryParamAsync)
	switch req.Request.Method {
	/*case http.MethodGet:
	if isParametersRequest(req) {
		return nil, &util.HTTPError{
			ErrorType:   "BadRequest",
			Description: ReferenceParametersAreNotSupported,
			StatusCode:  http.StatusBadRequest,
		}
	}
	if isGetOperationRequest(req) {
		return next.Handle(req)
	}
	if isGetInstanceRequest(req) {
		instance, err := f.getObjectByID(ctx, types.ServiceInstanceType, resourceID)
		if err != nil {
			return nil, util.HandleStorageError(err, types.ServiceInstanceType.String())
		}
		// todo: maybe we dont need this casting..
		return util.NewJSONResponse(http.StatusOK, instance.(*types.ServiceInstance))
	}*/
	case http.MethodPost:
		isReferencePlan, err := f.isReferencePlan(ctx, req)
		if err != nil {
			return nil, err
		}
		if !isReferencePlan {
			return next.Handle(req)
		}

		referencedKey := "referenced_instance_id"
		parameters := gjson.GetBytes(req.Body, "parameters").Map()

		referencedInstanceID, exists := parameters[referencedKey]
		isReferencedShared, err := f.isReferencedShared(ctx, referencedInstanceID.Str)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, errors.New("Missing referenced_instance_id")
		} else if exists && !isReferencedShared {
			return nil, errors.New("Referenced instance is not shared")
		}
		return next.Handle(req)
		//return f.handleReferenceProvision(req, referencedInstanceID.Str, isAsync)
		/*case http.MethodPatch:
			// we don't want to allow plan_id and/or parameters changes on a reference service instance
			if resourceID == "" {
				return nil, errors.New("Missing resource ID")
			}
			currentInstance, err := f.getObjectByID(ctx, types.ServiceInstanceType, resourceID)
			currentInstanceObject := currentInstance.(*types.ServiceInstance)
			if err != nil {
				return nil, util.HandleStorageError(err, types.ServiceInstanceType.String())
			}
			isValidPatchRequest, err := f.isValidPatchRequest(req, currentInstanceObject)
			if err != nil {
				return nil, err
			}
			if isValidPatchRequest {
				return f.handleReferenceUpdate(req, currentInstanceObject)
			}
		case http.MethodDelete:
			// deleting a reference service instance with bindings is not allowed
			bindings, err := f.getBindings(resourceID)
			if err != nil {
				return nil, err
			}
			if bindings.Len() > 0 {
				return nil, errors.New("The service instance has bindins which should be deleted first.")
			}
			return f.handleDeletion(req)*/
	}
	return nil, nil
}

func (f *referenceInstanceFilter) getBindings(resourceID string) (types.ObjectList, error) {
	bindings, err := f.repository.List(
		context.Background(),
		types.ServiceBindingType,
		query.ByField(query.EqualsOperator, "service_instance_id", resourceID))
	if err != nil {
		return nil, err
	}
	return bindings, nil
}

func (*referenceInstanceFilter) FilterMatchers() []web.FilterMatcher {
	return []web.FilterMatcher{
		{
			Matchers: []web.Matcher{
				web.Path(web.ServiceInstancesURL + "/**"),
				web.Methods(http.MethodPost),
			},
		},
	}
}

func (f *referenceInstanceFilter) isValidPatchRequest(req *web.Request, instance *types.ServiceInstance) (bool, error) {
	// todo: How can we update labels and do we want to allow the change?
	newPlanID := gjson.GetBytes(req.Body, planIDProperty).String()
	if instance.ServicePlanID != newPlanID {
		return false, errors.New("Can't modify reference's plan")
	}

	parametersRaw := gjson.GetBytes(req.Body, "parameters").Raw
	if parametersRaw != "" {
		return false, errors.New("Can't modify reference's parameters")
	}

	return true, nil
}

func (f *referenceInstanceFilter) handleReferenceUpdate(req *web.Request, instance *types.ServiceInstance) (*web.Response, error) {
	ctx := req.Context()

	newName := gjson.GetBytes(req.Body, "name").String()
	instance.Name = newName
	// todo: How can we update labels and do we want to allow the change?
	if _, err := f.repository.Update(ctx, instance, types.LabelChanges{}); err != nil {
		return nil, util.HandleStorageError(err, string(instance.GetType()))
	}
	return util.NewJSONResponse(http.StatusOK, instance)
}

func decodeRequestToObject(body []byte) provisionRequest {
	var instanceRequest = provisionRequest{}
	util.BytesToObject(body, instanceRequest)
	return instanceRequest
}

func (f *referenceInstanceFilter) isReferencePlan(ctx context.Context, req *web.Request) (bool, error) {
	servicePlanID := gjson.GetBytes(req.Body, planIDProperty).Str
	plan, err := f.getObjectByID(ctx, types.ServicePlanType, servicePlanID)
	if err != nil {
		return false, err
	}
	planObject := plan.(*types.ServicePlan)
	return planObject.Name == "reference-plan", nil
}

func (f *referenceInstanceFilter) createInstanceOnDB(ctx context.Context, instance *types.ServiceInstance) error {
	inTransactionErr := f.repository.InTransaction(ctx, func(ctx context.Context, storage storage.Repository) error {
		_, err := storage.Create(ctx, instance)
		if err != nil {
			log.C(ctx).Errorf("Could not create a new object on the DB for the instance (%s): %v", instance.ID, err)
			return err
		}
		return nil
	})
	return inTransactionErr
}
func (f *referenceInstanceFilter) createOperationOnDB(ctx context.Context, operation *types.Operation) error {
	inTransactionErr := f.repository.InTransaction(ctx, func(ctx context.Context, storage storage.Repository) error {
		_, err := storage.Create(ctx, operation)
		if err != nil {
			log.C(ctx).Errorf("Could not create a new operation (%s): %v", operation.ID, err)
			return err
		}
		return nil
	})
	return inTransactionErr
}

type provisionRequest struct {
	Name               string          `json:"name"`
	PlanID             string          `json:"plan_id"`
	PlatformID         string          `json:"platform_id"`
	RawContext         json.RawMessage `json:"context"`
	RawMaintenanceInfo json.RawMessage `json:"maintenance_info"`
}

func isParametersRequest(req *web.Request) bool {
	resourceID := req.PathParams["resource_id"]
	path := req.URL.Path
	if resourceID != "" && strings.Contains(path, web.ServiceInstancesURL) && strings.Contains(path, web.ParametersURL) {
		return true
	}
	return false
}

func isGetOperationRequest(req *web.Request) bool {
	resourceID := req.PathParams["resource_id"]
	path := req.URL.Path
	if resourceID != "" && strings.Contains(path, web.ServiceInstancesURL) && strings.Contains(path, web.OperationsURL) {
		return true
	}
	return false
}

func isGetInstanceRequest(req *web.Request) bool {
	instanceID := req.PathParams["resource_id"]
	path := req.URL.Path
	if instanceID != "" && strings.Contains(path, web.ServiceInstancesURL) && !strings.Contains(path, web.ParametersURL) {
		return true
	}
	return false
}

func (f *referenceInstanceFilter) getObjectByID(ctx context.Context, objectType types.ObjectType, resourceID string) (types.Object, error) {
	byID := query.ByField(query.EqualsOperator, "id", resourceID)
	object, err := f.repository.Get(ctx, objectType, byID)
	if err != nil {
		return nil, util.HandleStorageError(err, objectType.String())
	}
	return object, nil
}
func (f *referenceInstanceFilter) isReferencedShared(ctx context.Context, referencedInstanceID string) (bool, error) {
	byID := query.ByField(query.EqualsOperator, "id", referencedInstanceID)
	referencedObject, err := f.repository.Get(ctx, types.ServiceInstanceType, byID)
	if err != nil {
		return false, util.HandleStorageError(err, types.ServiceInstanceType.String())
	}
	instance := referencedObject.(*types.ServiceInstance)

	if !instance.Shared {
		return false, errors.New("referenced instance is not shared")
	}
	return true, nil
}

/*func (f *referenceInstanceFilter) handleReferenceProvision(req *web.Request, referencedInstanceID string, isAsync string) (*web.Response, error) {
	ctx := req.Context()

	planID := gjson.GetBytes(req.Body, planIDProperty).String()
	byID := query.ByField(query.EqualsOperator, "id", planID)
	planObject, err := f.repository.Get(ctx, types.ServicePlanType, byID)
	if err != nil {
		return nil, util.HandleStorageError(err, types.ServicePlanType.String())
	}
	plan := planObject.(*types.ServicePlan)

	if isReferencePlan(plan) {
		// set as !isReferencePlan
		return nil, errors.New("plan_id is not a reference plan")
	}

	byID = query.ByField(query.EqualsOperator, "id", referencedInstanceID)
	referencedObject, err := f.repository.Get(ctx, types.ServiceInstanceType, byID)
	if err != nil {
		return nil, util.HandleStorageError(err, types.ServiceInstanceType.String())
	}
	instance := referencedObject.(*types.ServiceInstance)

	if !instance.Shared {
		return nil, errors.New("referenced instance is not shared")
	}

	//instanceRequestBody := decodeRequestToObject(req.Body)
	generatedReferenceInstance := f.generateReferenceInstance(req.Body)
	err = f.createInstanceOnDB(ctx, generatedReferenceInstance)
	if err != nil {
		return nil, util.HandleStorageError(err, types.ServiceInstanceType.String())
	}
	generatedOperation := f.generateOperation(types.SUCCEEDED, generatedReferenceInstance.ID)
	err = f.createOperationOnDB(ctx, generatedOperation)
	if err != nil {
		return nil, util.HandleStorageError(err, types.OperationType.String())
	}

	return handleResponse(isAsync, generatedOperation, generatedReferenceInstance)
}*/
func handleResponse(isAsync string, generatedOperation *types.Operation, generatedReferenceInstance *types.ServiceInstance) (*web.Response, error) {
	if isAsync == "true" {
		return util.NewJSONResponse(http.StatusCreated, generatedOperation)
	}
	return util.NewJSONResponse(http.StatusCreated, generatedReferenceInstance)
}

func (f *referenceInstanceFilter) generateReferenceInstance(body []byte) *types.ServiceInstance {
	UUID, _ := uuid.NewV4()
	currentTime := time.Now().UTC()

	referencedInstanceID := gjson.GetBytes(body, "parameters.referenced_instance_id").String()
	if referencedInstanceID == "" {
		referencedInstanceID = gjson.GetBytes(body, "referenced_instance_id").String()
	}

	instance := &types.ServiceInstance{
		Base: types.Base{
			ID:        UUID.String(),
			CreatedAt: currentTime,
			UpdatedAt: currentTime,
			Labels:    make(map[string][]string),
			Ready:     true,
		},
		Name:                 gjson.GetBytes(body, "name").String(),
		ServicePlanID:        gjson.GetBytes(body, "service_plan_id").String(),
		PlatformID:           gjson.GetBytes(body, "platform_id").String(),
		MaintenanceInfo:      json.RawMessage(gjson.GetBytes(body, "maintenance_info").String()),
		Context:              json.RawMessage(gjson.GetBytes(body, "contextt").String()),
		Usable:               true,
		ReferencedInstanceID: referencedInstanceID,
	}

	return instance
}

func (f *referenceInstanceFilter) generateOperation(state types.OperationState, resourceID string) *types.Operation {
	UUID, _ := uuid.NewV4()
	currentTime := time.Now().UTC()

	operation := &types.Operation{
		Base: types.Base{
			ID:        UUID.String(),
			CreatedAt: currentTime,
			UpdatedAt: currentTime,
			Labels:    make(map[string][]string),
			Ready:     true,
		},
		Description:   "test",
		Type:          types.CREATE,
		State:         state,
		ResourceID:    resourceID,
		ResourceType:  types.ServiceInstanceType,
		CorrelationID: UUID.String(), // Can correlation ID equal operation ID?
	}
	return operation
}

func (f *referenceInstanceFilter) handleDeletion(req *web.Request) (*web.Response, error) {
	// todo: can we use the DeleteSingleObject from the controller and avoid communication with the interceptor?

	/*isAsync := req.URL.Query().Get(web.QueryParamAsync)
	if isAsync == "true" {
		return util.NewJSONResponse(http.StatusAccepted, common.Object{})
	}
	return util.NewJSONResponse(http.StatusOK, common.Object{})*/
	return nil, nil
}
