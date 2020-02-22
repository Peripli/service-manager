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
	// StoreServiceInstancePluginName is the plugin name
	StoreServiceInstancePluginName = "StoreServiceInstancePlugin"
	smServicePlanIDKey             = "sm_service_plan_id"
	smContextKey                   = "sm_context_key"
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

type lastOperationRequest struct {
	commonRequestDetails

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

type Response struct {
	DashboardURL   string `json:"dashboard_url"`
	OperationData  string `json:"operation"`
	Error          string `json:"error"`
	Description    string `json:"description"`
	InstanceUsable bool   `json:"instance_usable"`
}

type lastOperationResponse struct {
	Response

	State types.OperationState `json:"state"`
}

// NewStoreServiceInstancesPlugin creates a plugin that stores service instances on OSB requests
func NewStoreServiceInstancesPlugin(repository storage.TransactionalRepository) *StoreServiceInstancePlugin {
	return &StoreServiceInstancePlugin{
		Repository: repository,
	}
}

// StoreServiceInstancePlugin represents a plugin that stores service instances on OSB requests
type StoreServiceInstancePlugin struct {
	Repository storage.TransactionalRepository
}

func (*StoreServiceInstancePlugin) Name() string {
	return StoreServiceInstancePluginName
}

func (ssi *StoreServiceInstancePlugin) Provision(request *web.Request, next web.Handler) (*web.Response, error) {
	ctx := request.Context()

	requestPayload := &provisionRequest{}
	if err := decodeRequestBody(request, requestPayload); err != nil {
		return nil, err
	}

	response, err := next.Handle(request)
	if err != nil {
		return nil, err
	}

	resp := Response{
		InstanceUsable: true,
	}

	if err := json.Unmarshal(response.Body, &resp); err != nil {
		log.C(ctx).Warnf("Could not unmarshal response body %s for broker with id %s", string(response.Body), requestPayload.BrokerID)
	}

	correlationID := log.CorrelationIDForRequest(request.Request)
	if err := ssi.Repository.InTransaction(ctx, func(ctx context.Context, storage storage.Repository) error {
		switch response.StatusCode {
		case http.StatusCreated:
			if err := ssi.storeOperation(ctx, storage, requestPayload, &resp, types.SUCCEEDED, types.CREATE, correlationID); err != nil {
				return err
			}
			if err := ssi.storeInstance(ctx, storage, requestPayload, &resp, true); err != nil {
				return err
			}
		case http.StatusOK:
			if err := ssi.storeInstance(ctx, storage, requestPayload, &resp, true); err != nil {
				if err != util.ErrAlreadyExistsInStorage {
					return err
				}
			} else {
				if err := ssi.storeOperation(ctx, storage, requestPayload, &resp, types.SUCCEEDED, types.CREATE, correlationID); err != nil {
					return err
				}
			}
		case http.StatusAccepted:
			if err := ssi.storeOperation(ctx, storage, requestPayload, &resp, types.IN_PROGRESS, types.CREATE, correlationID); err != nil {
				return err
			}
			if err := ssi.storeInstance(ctx, storage, requestPayload, &resp, false); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return response, nil
}

func (ssi *StoreServiceInstancePlugin) Deprovision(request *web.Request, next web.Handler) (*web.Response, error) {
	ctx := request.Context()

	requestPayload := &deprovisionRequest{}
	if err := parseRequestForm(request, requestPayload); err != nil {
		return nil, err
	}

	response, err := next.Handle(request)
	if err != nil {
		return nil, err
	}

	resp := Response{
		InstanceUsable: true,
	}
	if err := json.Unmarshal(response.Body, &resp); err != nil {
		log.C(ctx).Warnf("Could not unmarshal response body %s for broker with id %s", string(response.Body), requestPayload.BrokerID)
	}

	correlationID := log.CorrelationIDForRequest(request.Request)
	if err := ssi.Repository.InTransaction(ctx, func(ctx context.Context, storage storage.Repository) error {
		switch response.StatusCode {
		case http.StatusOK:
			fallthrough
		case http.StatusGone:
			byID := query.ByField(query.EqualsOperator, "id", requestPayload.InstanceID)
			if err := storage.Delete(ctx, types.ServiceInstanceType, byID); err != nil {
				if err != util.ErrNotFoundInStorage {
					return util.HandleStorageError(err, string(types.ServiceInstanceType))

				}
			}
			if err := ssi.storeOperation(ctx, storage, requestPayload, &resp, types.SUCCEEDED, types.DELETE, correlationID); err != nil {
				return err
			}
		case http.StatusAccepted:
			if err := ssi.storeOperation(ctx, storage, requestPayload, &resp, types.IN_PROGRESS, types.DELETE, correlationID); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return response, nil
}

func (ssi *StoreServiceInstancePlugin) UpdateService(request *web.Request, next web.Handler) (*web.Response, error) {
	ctx := request.Context()

	requestPayload := &updateRequest{}
	if err := decodeRequestBody(request, requestPayload); err != nil {
		return nil, err
	}

	response, err := next.Handle(request)
	if err != nil {
		return nil, err
	}

	resp := Response{
		InstanceUsable: true,
	}
	if err := json.Unmarshal(response.Body, &resp); err != nil {
		log.C(ctx).Warnf("Could not unmarshal response body %s for broker with id %s", string(response.Body), requestPayload.BrokerID)
	}

	correlationID := log.CorrelationIDForRequest(request.Request)
	if err := ssi.Repository.InTransaction(ctx, func(ctx context.Context, storage storage.Repository) error {
		switch response.StatusCode {
		case http.StatusOK:
			if err := ssi.updateInstance(ctx, storage, requestPayload, &resp, true); err != nil {
				return err
			}
			if err := ssi.storeOperation(ctx, storage, requestPayload, &resp, types.SUCCEEDED, types.UPDATE, correlationID); err != nil {
				return err
			}
		case http.StatusAccepted:
			if err := ssi.updateInstance(ctx, storage, requestPayload, &resp, true); err != nil {
				return err
			}
			if err := ssi.storeOperation(ctx, storage, requestPayload, &resp, types.IN_PROGRESS, types.UPDATE, correlationID); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return response, nil
}

func (ssi *StoreServiceInstancePlugin) PollInstance(request *web.Request, next web.Handler) (*web.Response, error) {
	ctx := request.Context()

	requestPayload := &lastOperationRequest{}
	if err := parseRequestForm(request, requestPayload); err != nil {
		return nil, err
	}

	response, err := next.Handle(request)
	if err != nil {
		return nil, err
	}

	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusGone {
		return response, nil
	}

	resp := lastOperationResponse{
		Response: Response{
			InstanceUsable: true,
		},
	}
	if err := json.Unmarshal(response.Body, &resp); err != nil {
		log.C(ctx).Warnf("Could not unmarshal response body %s for broker with id %s", string(response.Body), requestPayload.BrokerID)
	}

	correlationID := log.CorrelationIDForRequest(request.Request)
	if err := ssi.Repository.InTransaction(ctx, func(ctx context.Context, storage storage.Repository) error {
		criteria := []query.Criterion{
			query.ByField(query.EqualsOperator, "resource_id", requestPayload.InstanceID),
			query.OrderResultBy("paging_sequence", query.DescOrder),
		}
		if len(requestPayload.OperationData) != 0 {
			criteria = append(criteria, query.ByField(query.EqualsOperator, "external_id", requestPayload.OperationData))
		}
		op, err := storage.Get(ctx, types.OperationType, criteria...)
		if err != nil && err != util.ErrNotFoundInStorage {
			return util.HandleStorageError(err, string(types.OperationType))
		}
		if op == nil {
			return nil
		}

		operationFromDB := op.(*types.Operation)
		if response.StatusCode == http.StatusGone {
			if operationFromDB.Type != types.DELETE {
				return nil
			}
			resp.State = types.SUCCEEDED
		}

		if operationFromDB.State != resp.State {
			switch operationFromDB.Type {
			case types.CREATE:
				switch resp.State {
				case types.SUCCEEDED:
					if err := ssi.updateInstanceReady(ctx, storage, requestPayload.InstanceID); err != nil {
						return err
					}
					if err := ssi.updateOperation(ctx, operationFromDB, storage, requestPayload, &resp.Response, types.SUCCEEDED, correlationID); err != nil {
						return err
					}
				case types.FAILED:
					byID := query.ByField(query.EqualsOperator, "id", requestPayload.InstanceID)
					if err := storage.Delete(ctx, types.ServiceInstanceType, byID); err != nil {
						if err != util.ErrNotFoundInStorage {
							return util.HandleStorageError(err, string(types.ServiceInstanceType))
						}
					}
					if err := ssi.updateOperation(ctx, operationFromDB, storage, requestPayload, &resp.Response, types.FAILED, correlationID); err != nil {
						return err
					}
				}
			case types.UPDATE:
				switch resp.State {
				case types.SUCCEEDED:
					if err := ssi.updateOperation(ctx, operationFromDB, storage, requestPayload, &resp.Response, types.SUCCEEDED, correlationID); err != nil {
						return err
					}
				case types.FAILED:
					if err := ssi.rollbackInstance(ctx, requestPayload, storage, resp.InstanceUsable); err != nil {
						return err
					}
					if err := ssi.updateOperation(ctx, operationFromDB, storage, requestPayload, &resp.Response, types.FAILED, correlationID); err != nil {
						return err
					}
				}
			case types.DELETE:
				switch resp.State {
				case types.SUCCEEDED:
					byID := query.ByField(query.EqualsOperator, "id", requestPayload.InstanceID)
					if err := storage.Delete(ctx, types.ServiceInstanceType, byID); err != nil {
						if err != util.ErrNotFoundInStorage {
							return util.HandleStorageError(err, string(types.ServiceInstanceType))
						}
					}
					if err := ssi.updateOperation(ctx, operationFromDB, storage, requestPayload, &resp.Response, types.SUCCEEDED, correlationID); err != nil {
						return err
					}
				case types.FAILED:
					if err := ssi.rollbackInstance(ctx, requestPayload, storage, resp.InstanceUsable); err != nil {
						return err
					}
					if err := ssi.updateOperation(ctx, operationFromDB, storage, requestPayload, &resp.Response, types.FAILED, correlationID); err != nil {
						return err
					}
				}
			default:
				return fmt.Errorf("unsupported operation type %s", operationFromDB.Type)
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return response, nil
}

func (ssi *StoreServiceInstancePlugin) updateOperation(ctx context.Context, operation *types.Operation, storage storage.Repository, req commonOSBRequest, resp *Response, state types.OperationState, correlationID string) error {
	operation.State = state
	operation.CorrelationID = correlationID
	if len(resp.Error) != 0 || len(resp.Description) != 0 {
		errorBytes, err := json.Marshal(&util.HTTPError{
			ErrorType:   fmt.Sprintf("BrokerError:%s", resp.Error),
			Description: resp.Description,
		})
		if err != nil {
			return err
		}
		operation.Errors, err = sjson.SetBytes(operation.Errors, "errors.-1", errorBytes)
		if err != nil {
			return err
		}
	}

	if _, err := storage.Update(ctx, operation, query.LabelChanges{}); err != nil {
		return util.HandleStorageError(err, string(operation.GetType()))
	}

	return nil
}

func (ssi *StoreServiceInstancePlugin) storeOperation(ctx context.Context, storage storage.Repository, req commonOSBRequest, resp *Response, state types.OperationState, category types.OperationCategory, correlationID string) error {
	UUID, err := uuid.NewV4()
	if err != nil {
		return fmt.Errorf("could not generate GUID for %s: %s", "/v1/service_instances", err)
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
		ResourceID:    req.GetInstanceID(),
		ResourceType:  "/v1/service_instances",
		PlatformID:    req.GetPlatformID(),
		CorrelationID: correlationID,
		ExternalID:    resp.OperationData,
	}

	if _, err := storage.Create(ctx, operation); err != nil {
		return util.HandleStorageError(err, string(operation.GetType()))
	}

	return nil
}

func (ssi *StoreServiceInstancePlugin) storeInstance(ctx context.Context, storage storage.Repository, req *provisionRequest, resp *Response, ready bool) error {
	planID, err := findServicePlanIDByCatalogIDs(ctx, storage, req.BrokerID, req.ServiceID, req.PlanID)
	if err != nil {
		return err
	}
	instanceName := gjson.GetBytes(req.RawContext, "instance_name").String()
	if len(instanceName) == 0 {
		log.C(ctx).Debugf("Instance name missing. Defaulting to id %s", req.InstanceID)
		instanceName = req.InstanceID
	}
	instance := &types.ServiceInstance{
		Base: types.Base{
			ID:        req.InstanceID,
			CreatedAt: req.Timestamp,
			UpdatedAt: req.Timestamp,
			Labels:    make(map[string][]string),
			Ready:     ready,
		},
		Name:            instanceName,
		ServicePlanID:   planID,
		PlatformID:      req.PlatformID,
		DashboardURL:    resp.DashboardURL,
		MaintenanceInfo: req.RawMaintenanceInfo,
		Context:         req.RawContext,
		Usable:          true,
	}
	if _, err := storage.Create(ctx, instance); err != nil {
		return util.HandleStorageError(err, string(instance.GetType()))
	}
	return nil
}

func (ssi *StoreServiceInstancePlugin) updateInstance(ctx context.Context, storage storage.Repository, req *updateRequest, resp *Response, usable bool) error {
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
		serviceInstance.ServicePlanID, err = findServicePlanIDByCatalogIDs(ctx, storage, req.BrokerID, req.ServiceID, req.PlanID)
		if err != nil {
			return err
		}
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
	if _, err := storage.Update(ctx, serviceInstance, query.LabelChanges{}); err != nil {
		return util.HandleStorageError(err, string(serviceInstance.GetType()))
	}

	return nil
}

func (ssi *StoreServiceInstancePlugin) rollbackInstance(ctx context.Context, req commonOSBRequest, storage storage.Repository, usable bool) error {
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

	if _, ok := req.(*lastOperationRequest); ok {
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

	if _, err := storage.Update(ctx, serviceInstance, query.LabelChanges{}); err != nil {
		return util.HandleStorageError(err, string(serviceInstance.GetType()))
	}

	return nil
}

func (ssi *StoreServiceInstancePlugin) updateInstanceReady(ctx context.Context, storage storage.Repository, instanceID string) error {
	byID := query.ByField(query.EqualsOperator, "id", instanceID)
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
	serviceInstance.Ready = true

	if _, err := storage.Update(ctx, serviceInstance, query.LabelChanges{}); err != nil {
		return util.HandleStorageError(err, string(serviceInstance.GetType()))
	}

	return nil
}

func findServicePlanIDByCatalogIDs(ctx context.Context, storage storage.Repository, brokerID, catalogServiceID, catalogPlanID string) (string, error) {
	byCatalogServiceID := query.ByField(query.EqualsOperator, "catalog_id", catalogServiceID)
	byBrokerID := query.ByField(query.EqualsOperator, "broker_id", brokerID)
	serviceOffering, err := storage.Get(ctx, types.ServiceOfferingType, byBrokerID, byCatalogServiceID)
	if err != nil {
		return "", util.HandleStorageError(err, string(types.ServiceOfferingType))
	}

	byServiceOfferingID := query.ByField(query.EqualsOperator, "service_offering_id", serviceOffering.GetID())
	byCatalogPlanID := query.ByField(query.EqualsOperator, "catalog_id", catalogPlanID)
	servicePlan, err := storage.Get(ctx, types.ServicePlanType, byServiceOfferingID, byCatalogPlanID)
	if err != nil {
		return "", util.HandleStorageError(err, string(types.ServicePlanType))
	}

	return servicePlan.GetID(), nil
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
