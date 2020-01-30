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

package interceptors

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Peripli/service-manager/pkg/log"

	"github.com/Peripli/service-manager/operations"

	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/util"

	"github.com/Peripli/service-manager/pkg/types"

	"github.com/Peripli/service-manager/storage"
	osbc "github.com/kubernetes-sigs/go-open-service-broker-client/v2"
)

const ServiceBindingCreateInterceptorProviderName = "ServiceBindingCreateInterceptorProvider"

// ServiceBindingCreateInterceptorProvider provides an interceptor that notifies the actual broker about instance creation
type ServiceBindingCreateInterceptorProvider struct {
	OSBClientCreateFunc osbc.CreateFunc
	Repository          storage.TransactionalRepository
	TenantKey           string
	PollingInterval     time.Duration
}

func (p *ServiceBindingCreateInterceptorProvider) Provide() storage.CreateAroundTxInterceptor {
	return &ServiceBindingInterceptor{
		osbClientCreateFunc: p.OSBClientCreateFunc,
		repository:          p.Repository,
		tenantKey:           p.TenantKey,
		pollingInterval:     p.PollingInterval,
	}
}

func (c *ServiceBindingCreateInterceptorProvider) Name() string {
	return ServiceBindingCreateInterceptorProviderName
}

const ServiceBindingDeleteInterceptorProviderName = "ServiceBindingDeleteInterceptorProvider"

// ServiceBindingDeleteInterceptorProvider provides an interceptor that notifies the actual broker about instance deletion
type ServiceBindingDeleteInterceptorProvider struct {
	OSBClientCreateFunc osbc.CreateFunc
	Repository          storage.TransactionalRepository
	TenantKey           string
	PollingInterval     time.Duration
}

func (p *ServiceBindingDeleteInterceptorProvider) Provide() storage.DeleteAroundTxInterceptor {
	return &ServiceBindingInterceptor{
		osbClientCreateFunc: p.OSBClientCreateFunc,
		repository:          p.Repository,
		tenantKey:           p.TenantKey,
		pollingInterval:     p.PollingInterval,
	}
}

func (c *ServiceBindingDeleteInterceptorProvider) Name() string {
	return ServiceBindingDeleteInterceptorProviderName
}

type ServiceBindingInterceptor struct {
	osbClientCreateFunc osbc.CreateFunc
	repository          storage.TransactionalRepository
	tenantKey           string
	pollingInterval     time.Duration
}

func (i *ServiceBindingInterceptor) AroundTxCreate(f storage.InterceptCreateAroundTxFunc) storage.InterceptCreateAroundTxFunc {
	return func(ctx context.Context, obj types.Object) (types.Object, error) {
		binding := obj.(*types.ServiceBinding)

		instance, err := getInstanceByID(ctx, binding.ServiceInstanceID, i.repository)
		if err != nil {
			return nil, err
		}

		if instance.PlatformID != types.SMPlatform {
			log.C(ctx).Debugf("platform is not %s. Skipping interceptor %s", types.SMPlatform, ServiceBindingDeleteInterceptorProviderName)
			return f(ctx, obj)
		}
		operation, found := operations.GetFromContext(ctx)
		if !found {
			return nil, fmt.Errorf("operation missing from context")
		}

		osbClient, broker, service, plan, err := prepare(ctx, i.repository, i.osbClientCreateFunc, instance)
		if err != nil {
			return nil, err
		}

		if isBindable := !i.isPlanBindable(service, plan); isBindable {
			return nil, &util.HTTPError{
				ErrorType:   "BrokerError",
				Description: fmt.Sprintf("plan %s is not bindable", plan.CatalogName),
				StatusCode:  http.StatusBadGateway,
			}
		}

		if isReady := i.isInstanceReady(instance); !isReady {
			return nil, &util.HTTPError{
				ErrorType:   "OperationInProgress",
				Description: fmt.Sprintf("creation of instance %s is still in progress or failed", instance.Name),
				StatusCode:  http.StatusUnprocessableEntity,
			}
		}

		isDeleting, err := i.isInstanceDeleting(ctx, instance.ID)
		if err != nil {
			return nil, fmt.Errorf("could not determine instance state: %s", err)
		}

		if isDeleting {
			return nil, &util.HTTPError{
				ErrorType:   "OperationInProgress",
				Description: fmt.Sprintf("instance %s is in state of delteion", instance.Name),
				StatusCode:  http.StatusUnprocessableEntity,
			}
		}

		var bindResponse *osbc.BindResponse
		if !operation.Reschedule {
			bindRequest := i.prepareBindRequest(instance, binding, service.CatalogID, plan.CatalogID)
			contextBytes, err := json.Marshal(bindRequest.Context)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal OSB context %+v: %s", bindRequest.Context, err)
			}
			binding.Context = contextBytes

			log.C(ctx).Infof("Sending bind request %+v to broker with name %s", bindRequest, broker.Name)
			bindResponse, err = osbClient.Bind(bindRequest)
			if err != nil {
				brokerError := &util.HTTPError{
					ErrorType:   "BrokerError",
					Description: fmt.Sprintf("Failed bind request %+v: %s", bindRequest, err),
					StatusCode:  http.StatusBadGateway,
				}
				if shouldStartOrphanMitigation(err) {
					// store the instance so that later on we can do orphan mitigation
					_, err := f(ctx, obj)
					if err != nil {
						return nil, fmt.Errorf("broker error %s caused orphan mitigation which required storing the resource which failed with: %s", brokerError, err)
					}

					// mark the operation as deletion scheduled meaning orphan mitigation is required
					operation.DeletionScheduled = time.Now()
					operation.Reschedule = false
					if _, err := i.repository.Update(ctx, operation, query.LabelChanges{}); err != nil {
						return nil, fmt.Errorf("failed to update operation with id %s to schedule orphan mitigation after broker error %s: %s", operation.ID, brokerError, err)
					}
				}
				return nil, brokerError
			}

			if err := i.enrichBindingWithBindingResponse(binding, bindResponse); err != nil {
				return nil, fmt.Errorf("could not enrich binding details with binding response details: %s", err)
			}

			if bindResponse.Async {
				log.C(ctx).Infof("Successful asynchronous binding request %+v to broker %s returned response %+v",
					bindRequest, broker.Name, bindResponse)
				operation.Reschedule = true
				if bindResponse.OperationKey != nil {
					operation.ExternalID = string(*bindResponse.OperationKey)
				}
				if _, err := i.repository.Update(ctx, operation, query.LabelChanges{}); err != nil {
					return nil, fmt.Errorf("failed to update operation with id %s to mark that next execution should be a reschedule", instance.ID)
				}
			} else {
				log.C(ctx).Infof("Successful synchronous bind %+v to broker %s returned response %+v",
					bindRequest, broker.Name, bindResponse)
			}
		}

		object, err := f(ctx, obj)
		if err != nil {
			return nil, err
		}
		binding = object.(*types.ServiceBinding)

		if operation.Reschedule {
			if err := i.pollServiceBinding(ctx, osbClient, binding, operation, broker.ID, service.CatalogID, plan.CatalogID, operation.ExternalID, true); err != nil {
				return nil, err
			}
		}
		return binding, nil
	}
}

func (i *ServiceBindingInterceptor) AroundTxDelete(f storage.InterceptDeleteAroundTxFunc) storage.InterceptDeleteAroundTxFunc {
	return func(ctx context.Context, deletionCriteria ...query.Criterion) error {
		bindings, err := i.repository.List(ctx, types.ServiceBindingType, deletionCriteria...)
		if err != nil {
			return fmt.Errorf("failed to get bindings for deletion: %s", err)
		}

		if bindings.Len() > 1 {
			return fmt.Errorf("deletion of multiple bindings is not supported")
		}

		if bindings.Len() != 0 {
			binding := bindings.ItemAt(0).(*types.ServiceBinding)

			operation, found := operations.GetFromContext(ctx)
			if !found {
				return fmt.Errorf("operation missing from context")
			}

			if err := i.deleteSingleBinding(ctx, binding, operation); err != nil {
				return err
			}
		}

		if err := f(ctx, deletionCriteria...); err != nil {
			return err
		}

		return nil
	}
}

func (i *ServiceBindingInterceptor) deleteSingleBinding(ctx context.Context, binding *types.ServiceBinding, operation *types.Operation) error {
	instance, err := getInstanceByID(ctx, binding.ServiceInstanceID, i.repository)
	if err != nil {
		return err
	}

	osbClient, broker, service, plan, err := prepare(ctx, i.repository, i.osbClientCreateFunc, instance)
	if err != nil {
		return err
	}

	var unbindResponse *osbc.UnbindResponse
	if !operation.Reschedule {
		unbindRequest := prepareUnbindRequest(instance, binding, service.CatalogID, plan.CatalogID)

		log.C(ctx).Infof("Sending unbind request %+v to broker with name %s", unbindRequest, broker.Name)
		unbindResponse, err = osbClient.Unbind(unbindRequest)
		if err != nil {
			if osbc.IsGoneError(err) {
				log.C(ctx).Infof("Synchronous unbind %+v to broker %s returned 410 GONE and is considered success",
					unbindRequest, broker.Name)
				return nil
			}
			brokerError := &util.HTTPError{
				ErrorType:   "BrokerError",
				Description: fmt.Sprintf("Failed unbind request %+v: %s", unbindRequest, err),
				StatusCode:  http.StatusBadGateway,
			}
			if shouldStartOrphanMitigation(err) {
				operation.DeletionScheduled = time.Now()
				operation.Reschedule = false
				if _, err := i.repository.Update(ctx, operation, query.LabelChanges{}); err != nil {
					return fmt.Errorf("failed to update operation with id %s to schedule orphan mitigation after broker error %s: %s", operation.ID, brokerError, err)
				}
			}
			return brokerError
		}

		if unbindResponse.Async {
			log.C(ctx).Infof("Successful asynchronous unbind request %+v to broker %s returned response %+v",
				unbindRequest, broker.Name, unbindResponse)
			operation.Reschedule = true

			if unbindResponse.OperationKey != nil {
				operation.ExternalID = string(*unbindResponse.OperationKey)
			}
			if _, err := i.repository.Update(ctx, operation, query.LabelChanges{}); err != nil {
				return fmt.Errorf("failed to update operation with id %s to mark that rescheduling is possible", operation.ID)
			}
		} else {
			log.C(ctx).Infof("Successful synchronous unbind %+v to broker %s returned response %+v",
				unbindRequest, broker.Name, unbindResponse)
		}
	}

	if operation.Reschedule {
		if err := i.pollServiceBinding(ctx, osbClient, binding, operation, broker.ID, service.CatalogID, plan.CatalogID, operation.ExternalID, true); err != nil {
			return err
		}
	}

	return nil
}

func (i *ServiceBindingInterceptor) isPlanBindable(service *types.ServiceOffering, plan *types.ServicePlan) bool {
	if plan.Bindable != nil {
		return *plan.Bindable
	}

	return service.Bindable
}

func (i *ServiceBindingInterceptor) isInstanceReady(instance *types.ServiceInstance) bool {
	return instance.Ready
}

func (i *ServiceBindingInterceptor) isInstanceDeleting(ctx context.Context, instanceID string) (bool, error) {
	lastOperation, found, err := getLastOperationByResourceID(ctx, instanceID, i.repository)
	if err != nil {
		return false, fmt.Errorf("could not determine in instance is in process of deletion: %s", err)
	}
	if !found {
		return false, nil
	}

	deletionScheduled := !lastOperation.DeletionScheduled.IsZero()
	deletionInProgress := lastOperation.Type == types.DELETE && lastOperation.State == types.IN_PROGRESS
	return deletionScheduled || deletionInProgress, nil
}

func getLastOperationByResourceID(ctx context.Context, resourceID string, repository storage.Repository) (*types.Operation, bool, error) {
	byResourceID := query.ByField(query.EqualsOperator, "resource_id", resourceID)
	orderDesc := query.OrderResultBy("paging_sequence", query.DescOrder)
	limitToOne := query.LimitResultBy(1)
	lastOperationObject, err := repository.Get(ctx, types.OperationType, byResourceID, orderDesc, limitToOne)
	if err != nil {
		if err == util.ErrNotFoundInStorage {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("could not fetch last operation for resource with id %s: %s", resourceID, util.HandleStorageError(err, types.OperationType.String()))
	}

	return lastOperationObject.(*types.Operation), true, nil
}

func getInstanceByID(ctx context.Context, instanceID string, repository storage.Repository) (*types.ServiceInstance, error) {
	byID := query.ByField(query.EqualsOperator, "id", instanceID)
	instanceObject, err := repository.Get(ctx, types.ServiceInstanceType, byID)
	if err != nil {
		if err == util.ErrNotFoundInStorage {
			return nil, &util.HTTPError{
				ErrorType:   "NotFound",
				Description: err.Error(),
				StatusCode:  http.StatusNotFound,
			}
		}
		return nil, fmt.Errorf("could not fetch instance with id %s from db: %s", instanceID, util.HandleStorageError(err, types.ServiceInstanceType.String()))
	}

	return instanceObject.(*types.ServiceInstance), nil
}

func (i *ServiceBindingInterceptor) prepareBindRequest(instance *types.ServiceInstance, binding *types.ServiceBinding, serviceCatalogID, planCatalogID string) *osbc.BindRequest {
	bindRequest := &osbc.BindRequest{
		BindingID:         binding.ID,
		InstanceID:        instance.ID,
		AcceptsIncomplete: true,
		ServiceID:         serviceCatalogID,
		PlanID:            planCatalogID,
		Parameters:        binding.Parameters,
		Context: map[string]interface{}{
			"platform":      types.SMPlatform,
			"instance_name": instance.Name,
		},
		//TODO no OI for SM platform yet
		OriginatingIdentity: nil,
	}
	if len(i.tenantKey) != 0 {
		if tenantValue, ok := binding.GetLabels()[i.tenantKey]; ok {
			bindRequest.Context[i.tenantKey] = tenantValue[0]
		}
	}

	return bindRequest
}

func prepareUnbindRequest(instance *types.ServiceInstance, binding *types.ServiceBinding, serviceCatalogID, planCatalogID string) *osbc.UnbindRequest {
	unbindRequest := &osbc.UnbindRequest{
		BindingID:         binding.ID,
		InstanceID:        instance.ID,
		AcceptsIncomplete: true,
		ServiceID:         serviceCatalogID,
		PlanID:            planCatalogID,
		//TODO no OI for SM platform yet
		OriginatingIdentity: nil,
	}

	return unbindRequest
}

func (i *ServiceBindingInterceptor) pollServiceBinding(ctx context.Context, osbClient osbc.Client, binding *types.ServiceBinding, operation *types.Operation, brokerID, serviceCatalogID, planCatalogID, operationKey string, enableOrphanMitigation bool) error {
	var key *osbc.OperationKey
	if len(operation.ExternalID) != 0 {
		opKey := osbc.OperationKey(operation.ExternalID)
		key = &opKey
	}

	pollingRequest := &osbc.BindingLastOperationRequest{
		InstanceID:   binding.ServiceInstanceID,
		BindingID:    binding.ID,
		ServiceID:    &serviceCatalogID,
		PlanID:       &planCatalogID,
		OperationKey: key,
		//TODO no OI for SM platform yet
		OriginatingIdentity: nil,
	}

	ticker := time.NewTicker(i.pollingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.C(ctx).Errorf("Terminating poll last operation for binding with id %s and name %s due to context done event", binding.ID, binding.Name)
			//operation should be kept in progress in this case
			return nil
		case <-ticker.C:
			log.C(ctx).Infof("Sending poll last operation request %+v for binding with id %s and name %s", pollingRequest, binding.ID, binding.Name)
			pollingResponse, err := osbClient.PollBindingLastOperation(pollingRequest)
			if err != nil {
				if osbc.IsGoneError(err) && operation.Type == types.DELETE {
					log.C(ctx).Infof("Successfully finished polling operation for binding with id %s and name %s", binding.ID, binding.Name)

					operation.Reschedule = false
					if _, err := i.repository.Update(ctx, operation, query.LabelChanges{}); err != nil {
						return fmt.Errorf("failed to update operation with id %s to mark that next execution should be a reschedule", operation.ID)
					}
					return nil
				}

				return &util.HTTPError{
					ErrorType: "BrokerError",
					Description: fmt.Sprintf("Failed poll last operation request %+v for binding with id %s and name %s: %s",
						pollingRequest, binding.ID, binding.Name, err),
					StatusCode: http.StatusBadGateway,
				}
			}

			switch pollingResponse.State {
			case osbc.StateInProgress:
				log.C(ctx).Infof("Polling of binding still in progress. Rescheduling polling last operation request %+v for binding of instance with id %s and name %s...", pollingRequest, binding.ID, binding.Name)

			case osbc.StateSucceeded:
				log.C(ctx).Infof("Successfully finished polling operation for binding with id %s and name %s", binding.ID, binding.Name)

				operation.Reschedule = false
				if _, err := i.repository.Update(ctx, operation, query.LabelChanges{}); err != nil {
					return fmt.Errorf("failed to update operation with id %s to mark that next execution should be a reschedule", operation.ID)
				}

				return nil
			case osbc.StateFailed:
				log.C(ctx).Infof("Failed polling operation for binding with id %s and name %s", binding.ID, binding.Name)
				operation.Reschedule = false
				if enableOrphanMitigation {
					operation.DeletionScheduled = time.Now()
				}
				if _, err := i.repository.Update(ctx, operation, query.LabelChanges{}); err != nil {
					return fmt.Errorf("failed to update operation with id %s after failed of last operation for binding with id %s", operation.ID, binding.ID)
				}

				errDescription := ""
				if pollingResponse.Description != nil {
					errDescription = *pollingResponse.Description
				} else {
					errDescription = "no description provided by broker"
				}
				return &util.HTTPError{
					ErrorType:   "BrokerError",
					Description: fmt.Sprintf("failed polling operation for binding with id %s and name %s due to polling last operation error: %s", binding.ID, binding.Name, errDescription),
					StatusCode:  http.StatusBadGateway,
				}
			default:
				log.C(ctx).Errorf("invalid state during poll last operation for binding with id %s and name %s: %s", binding.ID, binding.Name, pollingResponse.State)
			}
		}
	}
}

func (i *ServiceBindingInterceptor) enrichBindingWithBindingResponse(binding *types.ServiceBinding, response *osbc.BindResponse) error {
	if len(response.Credentials) != 0 {
		credentialBytes, err := json.Marshal(response.Credentials)
		if err != nil {
			return fmt.Errorf("could not marshal binding credentials: %s", err)
		}

		binding.Credentials = string(credentialBytes)
	}

	if response.RouteServiceURL != nil {
		binding.RouteServiceURL = *response.RouteServiceURL
	}

	if response.SyslogDrainURL != nil {
		binding.SyslogDrainURL = *response.SyslogDrainURL
	}

	if len(response.VolumeMounts) != 0 {
		volumeMountBytes, err := json.Marshal(response.VolumeMounts)
		if err != nil {
			return fmt.Errorf("could not marshal volume mounts: %s", err)
		}

		binding.VolumeMounts = volumeMountBytes
	}

	return nil
}
