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

package osb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Peripli/service-manager/operations/opcontext"
	"github.com/Peripli/service-manager/pkg/instance_sharing"
	"net/http"
	"time"

	"github.com/tidwall/sjson"

	"github.com/tidwall/gjson"

	"github.com/Peripli/service-manager/pkg/query"

	"github.com/gofrs/uuid"

	"github.com/Peripli/service-manager/pkg/util"

	"github.com/Peripli/service-manager/storage"

	"github.com/Peripli/service-manager/pkg/log"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/web"
)

const (
	// OSBStorePluginName is the plugin name
	OSBStorePluginName = "OSBStorePluginName"
	smServicePlanIDKey = "sm_service_plan_id"
	smContextKey       = "sm_context_key"
)

type entityOperation string

const (
	READY    entityOperation = "ready"
	ROLLBACK entityOperation = "rollback"
	DELETE   entityOperation = "delete"
	NONE     entityOperation = "none"
)

type provisionRequest struct {
	commonRequestDetails

	ServiceID          string          `json:"service_id"`
	PlanID             string          `json:"plan_id"`
	OrganizationGUID   string          `json:"organization_guid"`
	SpaceGUID          string          `json:"space_guid"`
	RawContext         json.RawMessage `json:"context"`
	RawParameters      json.RawMessage `json:"parameters"`
	RawMaintenanceInfo json.RawMessage `json:"maintenance_info"`
}

func (pr *provisionRequest) Validate() error {
	if len(pr.ServiceID) == 0 {
		return errors.New("service_id cannot be empty")
	}
	if len(pr.PlanID) == 0 {
		return errors.New("plan_id cannot be empty")
	}

	return nil
}

type provisionResponse struct {
	OperationData  string `json:"operation"`
	Error          string `json:"error"`
	Description    string `json:"description"`
	DashboardURL   string `json:"dashboard_url"`
	InstanceUsable bool   `json:"instance_usable"`
}

func (b *provisionResponse) GetError() string {
	return b.Error
}

func (b *provisionResponse) GetDescription() string {
	return b.Description
}

type lastInstanceOperationResponse struct {
	provisionResponse
	State types.OperationState `json:"state"`
}

type lastBindOperationResponse struct {
	State       types.OperationState `json:"state"`
	Error       string               `json:"error"`
	Description string               `json:"description"`
}

func (lb *lastBindOperationResponse) GetError() string {
	return lb.Error
}

func (lb *lastBindOperationResponse) GetDescription() string {
	return lb.Description
}

type bindRequest struct {
	commonRequestDetails

	ServiceID    string                 `json:"service_id"`
	PlanID       string                 `json:"plan_id"`
	BindingID    string                 `json:"binding_id"`
	RawContext   json.RawMessage        `json:"context"`
	BindResource json.RawMessage        `json:"bind_resource"`
	Parameters   map[string]interface{} `json:"parameters"`
}

func (br *bindRequest) Validate() error {
	if len(br.ServiceID) == 0 {
		return errors.New("service_id cannot be empty")
	}
	if len(br.PlanID) == 0 {
		return errors.New("plan_id cannot be empty")
	}

	return nil
}

type bindResponse struct {
	OperationData   string          `json:"operation"`
	Error           string          `json:"error"`
	Description     string          `json:"description"`
	RouteServiceUrl string          `json:"route_service_url"`
	SyslogDrainUrl  string          `json:"syslog_drain_url"`
	VolumeMounts    json.RawMessage `json:"volume_mounts"`
	Endpoints       json.RawMessage `json:"endpoints"`
}

func (b *bindResponse) GetError() string {
	return b.Error
}

func (b *bindResponse) GetDescription() string {
	return b.Description
}

type unbindRequest struct {
	commonRequestDetails

	ServiceID string `json:"service_id"`
	PlanID    string `json:"plan_id"`
	BindingID string `json:"binding_id"`
}

func (br *unbindRequest) Validate() error {
	if len(br.ServiceID) == 0 {
		return errors.New("service_id cannot be empty")
	}
	if len(br.PlanID) == 0 {
		return errors.New("plan_id cannot be empty")
	}

	return nil
}

type unbindResponse struct {
	OperationData string `json:"operation"`
	Error         string `json:"error"`
	Description   string `json:"description"`
}

type updateRequest struct {
	commonRequestDetails

	ServiceID       string          `json:"service_id"`
	PlanID          string          `json:"plan_id"`
	RawParameters   json.RawMessage `json:"parameters"`
	RawContext      json.RawMessage `json:"context"`
	MaintenanceInfo json.RawMessage `json:"maintenance_info"`
	PreviousValues  previousValues  `json:"previous_values"`
}

func (ur *updateRequest) Validate() error {
	if len(ur.ServiceID) == 0 {
		return errors.New("service_id cannot be empty")
	}

	return nil
}

type previousValues struct {
	PlanID          string          `json:"plan_id"`
	ServiceID       string          `json:"service_id"`
	MaintenanceInfo json.RawMessage `json:"maintenance_info"`
}

type deprovisionRequest struct {
	commonRequestDetails
}

type lastInstanceOperationRequest struct {
	commonRequestDetails

	OperationData string `json:"operation"`
}

type lastBindOperationRequest struct {
	commonRequestDetails
	BindingID     string `json:"binding_id"`
	OperationData string `json:"operation"`
}

type commonOSBRequest interface {
	GetBrokerID() string
	GetInstanceID() string
	GetPlatformID() string
	GetTimestamp() time.Time
	SetBrokerID(string)
	SetInstanceID(string)
	SetPlatformID(string)
	SetTimestamp(time.Time)
}

type brokerError interface {
	GetError() string
	GetDescription() string
}

type commonRequestDetails struct {
	BrokerID   string    `json:"-"`
	InstanceID string    `json:"-"`
	PlatformID string    `json:"-"`
	Timestamp  time.Time `json:"-"`
}

func (r *commonRequestDetails) GetBrokerID() string {
	return r.BrokerID
}

func (r *commonRequestDetails) GetInstanceID() string {
	return r.InstanceID
}

func (r *commonRequestDetails) GetPlatformID() string {
	return r.PlatformID
}

func (r *commonRequestDetails) GetTimestamp() time.Time {
	return r.Timestamp
}

func (r *commonRequestDetails) SetBrokerID(brokerID string) {
	r.BrokerID = brokerID
}

func (r *commonRequestDetails) SetInstanceID(instanceID string) {
	r.InstanceID = instanceID
}

func (r *commonRequestDetails) SetPlatformID(platformID string) {
	r.PlatformID = platformID
}

func (r *commonRequestDetails) SetTimestamp(timestamp time.Time) {
	r.Timestamp = timestamp
}

// NewStorePlugin creates a plugin that stores service instances on OSB requests
func NewStorePlugin(repository storage.TransactionalRepository) *storePlugin {
	return &storePlugin{
		repository: repository,
	}
}

// StoreServiceInstancePlugin represents a plugin that stores service instances and bindings on OSB requests
type storePlugin struct {
	repository storage.TransactionalRepository
}

func (*storePlugin) Name() string {
	return OSBStorePluginName
}

func (sp *storePlugin) Bind(request *web.Request, next web.Handler) (*web.Response, error) {
	ctx := request.Context()
	bindingID, ok := request.PathParams[BindingIDPathParam]
	if !ok {
		return nil, fmt.Errorf("path parameter missing: %s", BindingIDPathParam)
	}

	requestPayload := &bindRequest{}
	responsePayload := bindResponse{}

	if err := decodeRequestBody(request, requestPayload); err != nil {
		return nil, err
	}
	requestPayload.BindingID = bindingID
	response, err := next.Handle(request)
	if err != nil {
		return nil, err
	}

	if !web.ShouldStoreBindings(ctx) {
		return response, nil
	}
	if err := json.Unmarshal(response.Body, &responsePayload); err != nil {
		log.C(ctx).Warnf("Could not unmarshal response body %s for broker with id %s", string(response.Body), requestPayload.BrokerID)
	}

	correlationID := log.CorrelationIDForRequest(request.Request)
	err = sp.repository.InTransaction(ctx, func(ctx context.Context, storage storage.Repository) error {
		return sp.createOsbEntity(
			response.StatusCode,
			func(state types.OperationState, category types.OperationCategory) error {
				return sp.storeOperation(ctx, storage, requestPayload.BindingID, requestPayload, responsePayload.OperationData, state, category, correlationID, types.ServiceBindingType)
			},
			func(ready bool) error {
				return sp.storeBinding(ctx, storage, requestPayload, &responsePayload, ready)
			})

	})
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (sp *storePlugin) Unbind(request *web.Request, next web.Handler) (*web.Response, error) {
	ctx := request.Context()
	bindingId, ok := request.PathParams[BindingIDPathParam]
	if !ok {
		return nil, fmt.Errorf("path parameter missing: %s", BindingIDPathParam)
	}
	requestPayload := &unbindRequest{}
	if err := parseRequestForm(request, requestPayload); err != nil {
		return nil, err
	}
	requestPayload.BindingID = bindingId

	response, err := next.Handle(request)
	if err != nil {
		return nil, err
	}

	if !web.ShouldStoreBindings(ctx) {
		return response, nil
	}
	resp := unbindResponse{}
	if err := json.Unmarshal(response.Body, &resp); err != nil {
		log.C(ctx).Warnf("Could not unmarshal response body %s for broker with id %s", string(response.Body), requestPayload.BrokerID)
	}

	correlationID := log.CorrelationIDForRequest(request.Request)
	err = sp.repository.InTransaction(ctx, func(ctx context.Context, storage storage.Repository) error {
		return sp.removeOsbEntity(
			response.StatusCode,
			func() error {
				byID := query.ByField(query.EqualsOperator, "id", requestPayload.BindingID)
				if err := storage.Delete(ctx, types.ServiceBindingType, byID); err != nil {
					if err != util.ErrNotFoundInStorage {
						return util.HandleStorageError(err, string(types.ServiceBindingType))
					}
				}
				return nil
			},
			func(state types.OperationState, category types.OperationCategory) error {
				return sp.storeOperation(ctx, storage, requestPayload.BindingID, requestPayload, resp.OperationData, state, category, correlationID, types.ServiceBindingType)
			})
	})

	if err != nil {
		return nil, err
	}
	return response, nil
}

func (sp *storePlugin) Provision(request *web.Request, next web.Handler) (*web.Response, error) {
	ctx := request.Context()
	requestPayload := &provisionRequest{}
	if err := decodeRequestBody(request, requestPayload); err != nil {
		return nil, err
	}

	response, err := next.Handle(request)
	if err != nil {
		return nil, err
	}

	referencedInstanceID := gjson.GetBytes(request.Body, fmt.Sprintf("parameters.%s", instance_sharing.ReferencedInstanceIDKey)).String()
	if len(referencedInstanceID) > 0 {
		requestPayload.RawParameters, err = sjson.SetBytes(requestPayload.RawParameters, instance_sharing.ReferencedInstanceIDKey, referencedInstanceID)
		if err != nil {
			return nil, err
		}
	}

	responsePayload := provisionResponse{
		InstanceUsable: true,
	}
	if err := json.Unmarshal(response.Body, &responsePayload); err != nil {
		log.C(ctx).Warnf("Could not unmarshal response body %s for broker with id %s", string(response.Body), requestPayload.BrokerID)
	}

	correlationID := log.CorrelationIDForRequest(request.Request)
	err = sp.repository.InTransaction(ctx, func(ctx context.Context, storage storage.Repository) error {
		return sp.createOsbEntity(
			response.StatusCode,
			func(state types.OperationState, category types.OperationCategory) error {
				return sp.storeOperation(ctx, storage, requestPayload.InstanceID, requestPayload, responsePayload.OperationData, state, category, correlationID, types.ServiceInstanceType)
			},
			func(ready bool) error {
				return sp.storeInstance(ctx, storage, requestPayload, &responsePayload, ready)
			})
	})

	if err != nil {
		return nil, err
	}
	return response, nil

}

func (sp *storePlugin) Deprovision(request *web.Request, next web.Handler) (*web.Response, error) {
	requestPayload := &deprovisionRequest{}
	if err := parseRequestForm(request, requestPayload); err != nil {
		return nil, err
	}
	response, err := next.Handle(request)
	if err != nil {
		return nil, err
	}
	ctx := request.Context()
	_, operationFound := opcontext.Get(ctx)
	if operationFound {
		log.C(ctx).Debug("operation found in context - Deprovision is managed by another plugin..")
		return response, nil
	}

	responsePayload := provisionResponse{
		InstanceUsable: true,
	}
	if err := json.Unmarshal(response.Body, &responsePayload); err != nil {
		log.C(ctx).Warnf("Could not unmarshal response body %s for broker with id %s", string(response.Body), requestPayload.BrokerID)
	}

	correlationID := log.CorrelationIDForRequest(request.Request)
	err = sp.repository.InTransaction(ctx, func(ctx context.Context, storage storage.Repository) error {
		err = sp.removeOsbEntity(
			response.StatusCode,
			func() error {
				byID := query.ByField(query.EqualsOperator, "id", requestPayload.InstanceID)
				if err := storage.Delete(ctx, types.ServiceInstanceType, byID); err != nil {
					if err != util.ErrNotFoundInStorage {
						return util.HandleStorageError(err, string(types.ServiceInstanceType))
					}
				}
				return nil
			},
			func(state types.OperationState, category types.OperationCategory) error {
				return sp.storeOperation(ctx, storage, requestPayload.InstanceID, requestPayload, responsePayload.OperationData, state, category, correlationID, types.ServiceInstanceType)
			})
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return response, nil
}

func (sp *storePlugin) UpdateService(request *web.Request, next web.Handler) (*web.Response, error) {
	ctx := request.Context()

	requestPayload := &updateRequest{}
	if err := decodeRequestBody(request, requestPayload); err != nil {
		return nil, err
	}
	response, err := next.Handle(request)
	if err != nil {
		return nil, err
	}

	responsePayload := provisionResponse{
		InstanceUsable: true,
	}
	if err := json.Unmarshal(response.Body, &responsePayload); err != nil {
		log.C(ctx).Warnf("Could not unmarshal response body %s for broker with id %s", string(response.Body), requestPayload.BrokerID)
	}

	correlationID := log.CorrelationIDForRequest(request.Request)
	if err := sp.repository.InTransaction(ctx, func(ctx context.Context, storage storage.Repository) error {
		var state types.OperationState
		switch response.StatusCode {
		case http.StatusOK:
			state = types.SUCCEEDED
		case http.StatusAccepted:
			state = types.IN_PROGRESS
		default:
			return nil
		}

		if err := sp.updateInstance(ctx, storage, requestPayload, &responsePayload); err != nil {
			return err
		}
		if err := sp.storeOperation(ctx, storage, requestPayload.InstanceID, requestPayload, responsePayload.OperationData, state, types.UPDATE, correlationID, types.ServiceInstanceType); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return response, nil
}

func (sp *storePlugin) PollBinding(request *web.Request, next web.Handler) (*web.Response, error) {
	ctx := request.Context()
	bindingID, ok := request.PathParams[BindingIDPathParam]
	if !ok {
		return nil, fmt.Errorf("path parameter missing: %s", BindingIDPathParam)
	}

	requestPayload := &lastBindOperationRequest{}
	if err := parseRequestForm(request, requestPayload); err != nil {
		return nil, err
	}
	requestPayload.BindingID = bindingID

	response, err := next.Handle(request)
	if err != nil {
		return nil, err
	}

	if !web.ShouldStoreBindings(ctx) {
		return response, nil
	}

	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusGone {
		return response, nil
	}

	responsePayload := lastBindOperationResponse{}
	if err := json.Unmarshal(response.Body, &responsePayload); err != nil {
		log.C(ctx).Warnf("Could not unmarshal response body %s for broker with id %s", string(response.Body), requestPayload.BrokerID)
	}
	correlationID := log.CorrelationIDForRequest(request.Request)
	if err := sp.repository.InTransaction(ctx, func(ctx context.Context, storage storage.Repository) error {

		operationFromDB, err := sp.getOperationFromDB(ctx, storage, requestPayload.BindingID, requestPayload.OperationData)
		if err != nil {
			return err
		}
		if operationFromDB == nil {
			return nil
		}
		if response.StatusCode == http.StatusGone {
			if operationFromDB.Type != types.DELETE {
				return nil
			}
			responsePayload.State = types.SUCCEEDED
		}

		var instanceOp entityOperation
		if operationFromDB.State != responsePayload.State {
			switch operationFromDB.Type {
			case types.CREATE:
				instanceOp, err = sp.handlePollCreateResponse(ctx, storage, &responsePayload, responsePayload.State, operationFromDB, correlationID)
				if err != nil {
					return err
				}
			case types.DELETE:
				instanceOp, err = sp.handlePollDeleteResponse(ctx, storage, &responsePayload, responsePayload.State, operationFromDB, correlationID)
				if err != nil {
					return err
				}
			default:
				return fmt.Errorf("unsupported operation type %s", operationFromDB.Type)
			}
		}
		switch instanceOp {
		case READY:
			if err := sp.updateEntityReady(ctx, storage, requestPayload.BindingID, types.ServiceBindingType); err != nil {
				return err
			}
		case DELETE:
			byID := query.ByField(query.EqualsOperator, "id", requestPayload.BindingID)
			if err := storage.Delete(ctx, types.ServiceBindingType, byID); err != nil {
				if err != util.ErrNotFoundInStorage {
					return util.HandleStorageError(err, string(types.ServiceInstanceType))
				}
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return response, nil
}

func (sp *storePlugin) PollInstance(request *web.Request, next web.Handler) (*web.Response, error) {
	requestPayload := &lastInstanceOperationRequest{}
	if err := parseRequestForm(request, requestPayload); err != nil {
		return nil, err
	}

	response, err := next.Handle(request)
	if err != nil {
		return nil, err
	}
	ctx := request.Context()
	_, operationFound := opcontext.Get(ctx)
	if operationFound {
		log.C(ctx).Debug("operation found in context, pollInstance is managed by another plugin..")
		return response, nil
	}
	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusGone {
		return response, nil
	}
	resp := lastInstanceOperationResponse{
		provisionResponse: provisionResponse{
			InstanceUsable: true,
		},
	}
	if err := json.Unmarshal(response.Body, &resp); err != nil {
		log.C(ctx).Warnf("Could not unmarshal response body %s for broker with id %s", string(response.Body), requestPayload.BrokerID)
	}

	correlationID := log.CorrelationIDForRequest(request.Request)
	if err := sp.repository.InTransaction(ctx, func(ctx context.Context, storage storage.Repository) error {

		operationFromDB, ex := sp.getOperationFromDB(ctx, storage, requestPayload.InstanceID, requestPayload.OperationData)
		if ex != nil {
			return ex
		}
		if operationFromDB == nil {
			return nil
		}
		if response.StatusCode == http.StatusGone {
			if operationFromDB.Type != types.DELETE {
				return nil
			}
			resp.State = types.SUCCEEDED
		}

		var instanceOp entityOperation
		if operationFromDB.State != resp.State {
			switch operationFromDB.Type {
			case types.CREATE:
				instanceOp, err = sp.handlePollCreateResponse(ctx, storage, &resp, resp.State, operationFromDB, correlationID)
				if err != nil {
					return err
				}

			case types.UPDATE:
				instanceOp, err = sp.handlePollUpdateResponse(ctx, storage, &resp, resp.State, operationFromDB, correlationID)
				if err != nil {
					return err
				}
			case types.DELETE:
				instanceOp, err = sp.handlePollDeleteResponse(ctx, storage, &resp, resp.State, operationFromDB, correlationID)
				if err != nil {
					return err
				}
			default:
				return fmt.Errorf("unsupported operation type %s", operationFromDB.Type)
			}
		}

		switch instanceOp {
		case READY:
			if err := sp.updateEntityReady(ctx, storage, requestPayload.InstanceID, types.ServiceInstanceType); err != nil {
				return err
			}
		case DELETE:
			byID := query.ByField(query.EqualsOperator, "id", requestPayload.InstanceID)
			if err := storage.Delete(ctx, types.ServiceInstanceType, byID); err != nil {
				if err != util.ErrNotFoundInStorage {
					return util.HandleStorageError(err, string(types.ServiceInstanceType))
				}
			}
		case ROLLBACK:
			if err := sp.rollbackInstance(ctx, requestPayload, storage, resp.InstanceUsable); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return response, nil
}

func (sp *storePlugin) getOperationFromDB(ctx context.Context, storage storage.Repository, id string, operation_id string) (*types.Operation, error) {
	criteria := []query.Criterion{
		query.ByField(query.EqualsOperator, "resource_id", id),
		query.OrderResultBy("paging_sequence", query.DescOrder),
	}
	if len(operation_id) != 0 {
		criteria = append(criteria, query.ByField(query.EqualsOperator, "external_id", operation_id))
	}
	op, err := storage.Get(ctx, types.OperationType, criteria...)
	if err != nil && err != util.ErrNotFoundInStorage {
		return nil, util.HandleStorageError(err, string(types.OperationType))
	}
	if op == nil {
		return nil, nil
	}

	operationFromDB := op.(*types.Operation)
	return operationFromDB, nil
}

func (sp *storePlugin) updateOperation(ctx context.Context, operation *types.Operation, storage storage.Repository, resp brokerError, state types.OperationState, correlationID string) error {
	operation.State = state
	operation.CorrelationID = correlationID
	if len(resp.GetError()) != 0 || len(resp.GetDescription()) != 0 {
		errorBytes, err := json.Marshal(&util.HTTPError{
			ErrorType:   fmt.Sprintf("BrokerError:%s", resp.GetError()),
			Description: resp.GetDescription(),
		})
		if err != nil {
			return err
		}
		operation.Errors, err = sjson.SetBytes(operation.Errors, "errors.-1", errorBytes)
		if err != nil {
			return err
		}
	}

	if _, err := storage.Update(ctx, operation, types.LabelChanges{}); err != nil {
		return util.HandleStorageError(err, string(operation.GetType()))
	}

	return nil
}

func (sp *storePlugin) storeOperation(ctx context.Context, storage storage.Repository, resourceID string, req commonOSBRequest, operationData string, state types.OperationState, category types.OperationCategory, correlationID string, objType types.ObjectType) error {

	UUID, err := uuid.NewV4()
	if err != nil {
		return fmt.Errorf("could not generate GUID for %s: %s", objType, err)
	}
	operation := &types.Operation{
		Base: types.Base{
			ID:        UUID.String(),
			CreatedAt: req.GetTimestamp(),
			UpdatedAt: req.GetTimestamp(),
			Labels:    make(map[string][]string),
			Ready:     true,
		},
		Type:          category,
		State:         state,
		ResourceID:    resourceID,
		ResourceType:  objType,
		PlatformID:    req.GetPlatformID(),
		CorrelationID: correlationID,
		ExternalID:    operationData,
	}

	if _, err := storage.Create(ctx, operation); err != nil {
		return util.HandleStorageError(err, string(operation.GetType()))
	}

	return nil
}

func (sp *storePlugin) storeInstance(ctx context.Context, storage storage.Repository, req *provisionRequest, resp *provisionResponse, ready bool) error {
	plan, err := findServicePlanByCatalogIDs(ctx, storage, req.BrokerID, req.ServiceID, req.PlanID)
	if err != nil {
		return err
	}
	planID := plan.GetID()
	instanceName := gjson.GetBytes(req.RawContext, "instance_name").String()
	if len(instanceName) == 0 {
		log.C(ctx).Debugf("Instance name missing. Defaulting to id %s", req.InstanceID)
		instanceName = req.InstanceID
	}

	referencedInstanceID := gjson.GetBytes(req.RawParameters, instance_sharing.ReferencedInstanceIDKey).String()
	instance := &types.ServiceInstance{
		Base: types.Base{
			ID:        req.InstanceID,
			CreatedAt: req.Timestamp,
			UpdatedAt: req.Timestamp,
			Labels:    make(map[string][]string),
			Ready:     ready,
		},
		Name:                 instanceName,
		ServicePlanID:        planID,
		PlatformID:           req.PlatformID,
		DashboardURL:         resp.DashboardURL,
		MaintenanceInfo:      req.RawMaintenanceInfo,
		Context:              req.RawContext,
		Usable:               true,
		ReferencedInstanceID: referencedInstanceID,
	}
	if _, err := storage.Create(ctx, instance); err != nil {
		return util.HandleStorageError(err, string(instance.GetType()))
	}
	return nil
}

func (sp *storePlugin) storeBinding(ctx context.Context, repository storage.Repository, req *bindRequest, resp *bindResponse, ready bool) error {
	bindingName := gjson.GetBytes(req.RawContext, "binding_name").String()
	if len(bindingName) == 0 {
		log.C(ctx).Debugf("Binding name missing. Defaulting to id %s", req.BindingID)
		bindingName = req.BindingID
	}
	binding := &types.ServiceBinding{
		Base: types.Base{
			ID:        req.BindingID,
			CreatedAt: req.Timestamp,
			UpdatedAt: req.Timestamp,
			Labels:    make(map[string][]string),
			Ready:     ready,
		},
		Name:              bindingName,
		ServiceInstanceID: req.InstanceID,
		SyslogDrainURL:    resp.SyslogDrainUrl,
		RouteServiceURL:   resp.RouteServiceUrl,
		VolumeMounts:      resp.VolumeMounts,
		Endpoints:         resp.Endpoints,
		Context:           req.RawContext,
		BindResource:      req.BindResource,
		Parameters:        req.Parameters,
		Credentials:       nil,
	}

	serviceInstanceObj, err := storage.GetObjectByField(ctx, repository, types.ServiceInstanceType, "id", req.InstanceID)
	if err != nil {
		return err
	}

	serviceInstance := serviceInstanceObj.(*types.ServiceInstance)

	if len(serviceInstance.ReferencedInstanceID) > 0 {
		binding.Context = serviceInstance.Context
	}

	if _, err := repository.Create(ctx, binding); err != nil {
		return util.HandleStorageError(err, string(binding.GetType()))
	}
	return nil
}

func (sp *storePlugin) updateInstance(ctx context.Context, storage storage.Repository, req *updateRequest, resp *provisionResponse) error {
	byID := query.ByField(query.EqualsOperator, "id", req.InstanceID)
	var instance types.Object
	var err error
	if instance, err = storage.Get(ctx, types.ServiceInstanceType, byID); err != nil {
		if err != util.ErrNotFoundInStorage {
			return util.HandleStorageError(err, string(types.ServiceInstanceType))
		}
	}
	if instance == nil {
		return nil
	}
	serviceInstance := instance.(*types.ServiceInstance)
	previousValuesBytes, err := json.Marshal(req.PreviousValues)
	if err != nil {
		return err
	}
	previousValuesBytes, err = sjson.SetBytes(previousValuesBytes, smServicePlanIDKey, serviceInstance.ServicePlanID)
	if err != nil {
		return err
	}
	previousValuesBytes, err = sjson.SetBytes(previousValuesBytes, smContextKey, serviceInstance.Context)
	if err != nil {
		return err
	}
	if len(req.PlanID) != 0 && req.PreviousValues.PlanID != req.PlanID {
		var err error
		plan, err := findServicePlanByCatalogIDs(ctx, storage, req.BrokerID, req.ServiceID, req.PlanID)
		if err != nil {
			return err
		}
		serviceInstance.ServicePlanID = plan.GetID()
	}
	if len(resp.DashboardURL) != 0 {
		serviceInstance.DashboardURL = resp.DashboardURL
	}
	if len(req.MaintenanceInfo) != 0 {
		serviceInstance.MaintenanceInfo = req.MaintenanceInfo
	}
	if len(req.RawContext) != 0 {
		serviceInstance.Context = req.RawContext
	}

	serviceInstance.PreviousValues = previousValuesBytes
	if _, err := storage.Update(ctx, serviceInstance, types.LabelChanges{}); err != nil {
		return util.HandleStorageError(err, string(serviceInstance.GetType()))
	}

	return nil
}

func (sp *storePlugin) rollbackInstance(ctx context.Context, req commonOSBRequest, storage storage.Repository, usable bool) error {
	byID := query.ByField(query.EqualsOperator, "id", req.GetInstanceID())
	var instance types.Object
	var err error
	if instance, err = storage.Get(ctx, types.ServiceInstanceType, byID); err != nil {
		if err != util.ErrNotFoundInStorage {
			return util.HandleStorageError(err, string(types.ServiceInstanceType))
		}
	}
	if instance == nil {
		return nil
	}
	serviceInstance := instance.(*types.ServiceInstance)
	serviceInstance.Usable = usable

	if _, ok := req.(*lastInstanceOperationRequest); ok {
		previousValues := serviceInstance.PreviousValues
		oldCatalogPlanID := gjson.GetBytes(previousValues, smServicePlanIDKey).String()
		if len(oldCatalogPlanID) != 0 {
			serviceInstance.ServicePlanID = oldCatalogPlanID
		}
		oldContext := gjson.GetBytes(previousValues, smContextKey).Raw
		if len(oldCatalogPlanID) != 0 {
			serviceInstance.Context = []byte(oldContext)
		}
		oldMaintenanceInfo := gjson.GetBytes(previousValues, "maintenance_info").Raw
		if len(oldMaintenanceInfo) != 0 {
			serviceInstance.MaintenanceInfo = []byte(oldMaintenanceInfo)
		}
	}

	if _, err := storage.Update(ctx, serviceInstance, types.LabelChanges{}); err != nil {
		return util.HandleStorageError(err, string(serviceInstance.GetType()))
	}

	return nil
}

func (sp *storePlugin) updateEntityReady(ctx context.Context, storage storage.Repository, resourceID string, objectType types.ObjectType) error {
	byID := query.ByField(query.EqualsOperator, "id", resourceID)
	var instance types.Object
	var err error
	if instance, err = storage.Get(ctx, objectType, byID); err != nil {
		if err != util.ErrNotFoundInStorage {
			return util.HandleStorageError(err, string(types.ServiceInstanceType))
		}
	}
	if instance == nil {
		return nil
	}
	instance.SetReady(true)

	if _, err := storage.Update(ctx, instance, types.LabelChanges{}); err != nil {
		return util.HandleStorageError(err, string(instance.GetType()))
	}

	return nil
}

func findServicePlanByCatalogIDs(ctx context.Context, storage storage.Repository, brokerID, catalogServiceID, catalogPlanID string) (*types.ServicePlan, error) {
	byCatalogServiceID := query.ByField(query.EqualsOperator, "catalog_id", catalogServiceID)
	byBrokerID := query.ByField(query.EqualsOperator, "broker_id", brokerID)
	serviceOffering, err := storage.Get(ctx, types.ServiceOfferingType, byBrokerID, byCatalogServiceID)
	if err != nil {
		return nil, util.HandleStorageError(err, string(types.ServiceOfferingType))
	}

	byServiceOfferingID := query.ByField(query.EqualsOperator, "service_offering_id", serviceOffering.GetID())
	byCatalogPlanID := query.ByField(query.EqualsOperator, "catalog_id", catalogPlanID)
	servicePlan, err := storage.Get(ctx, types.ServicePlanType, byServiceOfferingID, byCatalogPlanID)
	if err != nil {
		return nil, util.HandleStorageError(err, string(types.ServicePlanType))
	}

	return servicePlan.(*types.ServicePlan), nil
}

func parseRequestForm(request *web.Request, body commonOSBRequest) error {
	platform, err := ExtractPlatformFromContext(request.Context())
	if err != nil {
		return err
	}
	brokerID, ok := request.PathParams[BrokerIDPathParam]
	if !ok {
		return fmt.Errorf("path parameter missing: %s", BrokerIDPathParam)
	}
	instanceID, ok := request.PathParams[InstanceIDPathParam]
	if !ok {
		return fmt.Errorf("path parameter missing: %s", InstanceIDPathParam)
	}
	body.SetBrokerID(brokerID)
	body.SetInstanceID(instanceID)
	body.SetPlatformID(platform.ID)
	body.SetTimestamp(time.Now().UTC())

	return nil
}

func ExtractPlatformFromContext(ctx context.Context) (*types.Platform, error) {
	user, ok := web.UserFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("user details not found in request context")
	}
	platform := &types.Platform{}
	if err := user.Data(platform); err != nil {
		return nil, err
	}
	if err := platform.Validate(); err != nil {
		return nil, fmt.Errorf("invalid platform found in user context: %s", err)
	}
	return platform, nil
}

func decodeRequestBody(request *web.Request, body commonOSBRequest) error {
	if err := util.BytesToObject(request.Body, body); err != nil {
		return err
	}
	return parseRequestForm(request, body)
}

func (sp *storePlugin) handlePollDeleteResponse(ctx context.Context, storage storage.Repository, resp brokerError, state types.OperationState, operationFromDB *types.Operation, correlationID string) (entityOperation, error) {
	var entOp entityOperation
	switch state {
	case types.SUCCEEDED:
		entOp = DELETE
	case types.FAILED:
		entOp = ROLLBACK
	default:
		return NONE, nil
	}
	if err := sp.updateOperation(ctx, operationFromDB, storage, resp, state, correlationID); err != nil {
		return NONE, err
	}
	return entOp, nil
}

func (sp *storePlugin) handlePollCreateResponse(ctx context.Context, storage storage.Repository, resp brokerError, state types.OperationState, operationFromDB *types.Operation, correlationID string) (entityOperation, error) {
	if err := sp.updateOperation(ctx, operationFromDB, storage, resp, state, correlationID); err != nil {
		return NONE, err
	}
	switch state {
	case types.SUCCEEDED:
		return READY, nil
	case types.FAILED:
		return DELETE, nil
	default:
		return NONE, nil
	}
}

func (sp *storePlugin) handlePollUpdateResponse(ctx context.Context, storage storage.Repository, resp brokerError, state types.OperationState, operationFromDB *types.Operation, correlationID string) (entityOperation, error) {
	var entOp entityOperation
	switch state {
	case types.SUCCEEDED:
		entOp = NONE
	case types.FAILED:
		entOp = ROLLBACK
	default:
		return NONE, nil
	}
	if err := sp.updateOperation(ctx, operationFromDB, storage, resp, state, correlationID); err != nil {
		return NONE, err
	}
	return entOp, nil
}

func (sp *storePlugin) removeOsbEntity(resStatus int, deleteEntity func() error, storeOperation func(state types.OperationState, category types.OperationCategory) error) error {
	switch resStatus {
	case http.StatusOK:
		fallthrough
	case http.StatusGone:
		if err := deleteEntity(); err != nil {
			return err
		}
		if err := storeOperation(types.SUCCEEDED, types.DELETE); err != nil {
			return err
		}
	case http.StatusAccepted:
		if err := storeOperation(types.IN_PROGRESS, types.DELETE); err != nil {
			return err
		}
	}
	return nil
}

func (sp *storePlugin) createOsbEntity(resStatus int, storeOperation func(state types.OperationState, category types.OperationCategory) error, storeEntity func(ready bool) error) error {
	switch resStatus {
	case http.StatusCreated:
		if err := storeOperation(types.SUCCEEDED, types.CREATE); err != nil {
			return err
		}
		if err := storeEntity(true); err != nil {
			return err
		}
	case http.StatusOK:
		if err := storeEntity(true); err != nil {
			if err != util.ErrAlreadyExistsInStorage {
				return err
			}
		} else {
			if err := storeOperation(types.SUCCEEDED, types.CREATE); err != nil {
				return err
			}
		}
	case http.StatusAccepted:
		if err := storeOperation(types.IN_PROGRESS, types.CREATE); err != nil {
			return err
		}
		if err := storeEntity(false); err != nil {
			return err
		}
	}
	return nil
}
