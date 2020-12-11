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
	"math"
	"net/http"
	"time"

	"github.com/tidwall/sjson"

	"github.com/Peripli/service-manager/operations/opcontext"

	"github.com/Peripli/service-manager/pkg/log"

	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/util"

	"github.com/Peripli/service-manager/pkg/types"

	"github.com/Peripli/service-manager/storage"
	osbc "github.com/kubernetes-sigs/go-open-service-broker-client/v2"
)

const ServiceBindingCreateInterceptorProviderName = "ServiceBindingCreateInterceptorProvider"

// ServiceBindingCreateInterceptorProvider provides an interceptor that notifies the actual broker about instance creation
type ServiceBindingCreateInterceptorProvider struct {
	*BaseSMAAPInterceptorProvider
}

type bindResponseDetails struct {
	Credentials     map[string]interface{}
	SyslogDrainURL  *string
	RouteServiceURL *string
	VolumeMounts    []interface{}
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
	*BaseSMAAPInterceptorProvider
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
	repository          *storage.InterceptableTransactionalRepository
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
		smaapOperated := instance.Labels != nil && len(instance.Labels[OperatedByLabelKey]) > 0

		if instance.PlatformID != types.SMPlatform && !smaapOperated {
			log.C(ctx).Debugf("platform is not %s. Skipping interceptor %s", types.SMPlatform, ServiceBindingDeleteInterceptorProviderName)
			return f(ctx, obj)
		}
		operation, found := opcontext.Get(ctx)
		if !found {
			return nil, fmt.Errorf("operation missing from context")
		}

		osbClient, broker, service, plan, err := preparePrerequisites(ctx, i.repository, i.osbClientCreateFunc, instance)
		if err != nil {
			return nil, err
		}

		if isBindable := !i.isPlanBindable(service, plan); isBindable {
			return nil, &util.HTTPError{
				ErrorType:   "BrokerError",
				Description: fmt.Sprintf("plan %s is not bindable", plan.CatalogName),
				StatusCode:  http.StatusBadRequest,
			}
		}

		if isReady := i.isInstanceReady(instance); !isReady {
			return nil, &util.HTTPError{
				ErrorType:   "OperationInProgress",
				Description: fmt.Sprintf("creation of instance %s is still in progress or failed", instance.Name),
				StatusCode:  http.StatusUnprocessableEntity,
			}
		}

		isDeleting, err := i.isInstanceInDeletion(ctx, instance.ID)
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

		if operation.Reschedule {
			if err := i.pollServiceBinding(ctx, osbClient, binding, instance, plan, operation, broker.ID, service.CatalogID, plan.CatalogID, operation.ExternalID, true); err != nil {
				return nil, err
			}
			return binding, nil
		}

		var bindResponse *osbc.BindResponse
		if !operation.Reschedule {
			operation.Context.ServiceInstanceID = binding.ServiceInstanceID
			bindRequest, err := i.prepareBindRequest(instance, binding, service.CatalogID, plan.CatalogID, service.BindingsRetrievable)
			if err != nil {
				return nil, fmt.Errorf("failed to prepare bind request: %s", err)
			}
			contextBytes, err := json.Marshal(bindRequest.Context)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal OSB context %+v: %s", bindRequest.Context, err)
			}
			binding.Context = contextBytes

			log.C(ctx).Infof("Sending bind request %s to broker with name %s", logBindRequest(bindRequest), broker.Name)
			bindResponse, err = osbClient.Bind(bindRequest)
			if err != nil {
				brokerError := &util.HTTPError{
					ErrorType:   "BrokerError",
					Description: fmt.Sprintf("Failed bind request %s: %s", logBindRequest(bindRequest), err),
					StatusCode:  http.StatusBadGateway,
				}
				if shouldStartOrphanMitigation(err) {
					// store the instance data on the operation context so that later on we can do orphan mitigation
					operation.DeletionScheduled = time.Now()
					operation.Reschedule = false
					operation.RescheduleTimestamp = time.Time{}
					if _, err := i.repository.Update(ctx, operation, types.LabelChanges{}); err != nil {
						return nil, fmt.Errorf("failed to update operation with id %s to schedule orphan mitigation after broker error %s: %s", operation.ID, brokerError, err)
					}
				}

				if operation.IsAsyncResponse() {
					_, err := f(ctx, obj)
					if err != nil {
						return nil, err
					}
				}
				return nil, brokerError
			}

			bindResponseDetails := &bindResponseDetails{
				Credentials:     bindResponse.Credentials,
				SyslogDrainURL:  bindResponse.SyslogDrainURL,
				RouteServiceURL: bindResponse.RouteServiceURL,
				VolumeMounts:    bindResponse.VolumeMounts,
			}
			if err := i.enrichBindingWithBindingResponse(binding, bindResponseDetails); err != nil {
				return nil, fmt.Errorf("could not enrich binding details with binding response details: %s", err)
			}

			if bindResponse.Async {
				log.C(ctx).Infof("Successful asynchronous binding request %s to broker %s returned response %s",
					logBindRequest(bindRequest), broker.Name, logBindResponse(bindResponse))
				operation.Reschedule = true
				if operation.Context.IsAsyncNotDefined {
					operation.Context.Async = true
				}
				if operation.RescheduleTimestamp.IsZero() {
					operation.RescheduleTimestamp = time.Now()
				}
				if bindResponse.OperationKey != nil {
					operation.ExternalID = string(*bindResponse.OperationKey)
				}
				if _, err := i.repository.Update(ctx, operation, types.LabelChanges{}); err != nil {
					return nil, fmt.Errorf("failed to update operation with id %s to mark that next execution should be a reschedule: %s", instance.ID, err)
				}
			} else {
				log.C(ctx).Infof("Successful synchronous bind %s to broker %s returned response %s",
					logBindRequest(bindRequest), broker.Name, logBindResponse(bindResponse))
			}

			object, err := f(ctx, obj)
			if err != nil {
				return nil, err
			}
			binding = object.(*types.ServiceBinding)
		}

		if shouldStartPolling(operation) {
			if err := i.pollServiceBinding(ctx, osbClient, binding, instance, plan, operation, broker.ID, service.CatalogID, plan.CatalogID, operation.ExternalID, true); err != nil {
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

		operation, found := opcontext.Get(ctx)
		if !found {
			return fmt.Errorf("operation missing from context")
		}

		if bindings.Len() != 0 {
			binding := bindings.ItemAt(0).(*types.ServiceBinding)
			if operation.Type == types.CREATE && !operation.IsAsyncResponse() {
				if err := i.repository.RawRepository.Delete(ctx, types.ServiceBindingType, deletionCriteria...); err != nil {
					return err
				}
			}
			if err := i.deleteSingleBinding(ctx, binding, operation); err != nil {
				return err
			}
		} else if operation.InOrphanMitigationState() && operation.Context.ServiceInstanceID != "" {
			// In case we don't have a binding & use the operation context to perform orphan mitigation
			binding := types.ServiceBinding{}
			binding.ServiceInstanceID = operation.Context.ServiceInstanceID
			binding.ID = operation.ResourceID
			if err := i.deleteSingleBinding(ctx, &binding, operation); err != nil {
				return err
			}
		}

		if shouldRemoveResource(operation) {
			if err := f(ctx, deletionCriteria...); err != nil {
				return err
			}
		}

		return nil
	}
}

func shouldRemoveResource(operation *types.Operation) bool {
	if operation.State == types.FAILED {
		if operation.Type == types.CREATE && operation.IsAsyncResponse() {
			return false
		}

		if operation.Type == types.DELETE && !operation.InOrphanMitigationState() {
			return false
		}
	}
	// Keep the instance in case the operation type is "DELETE" and the operation is rescheduled
	if operation.Type == types.DELETE && operation.Reschedule {
		return false
	}
	return true
}

func (i *ServiceBindingInterceptor) deleteSingleBinding(ctx context.Context, binding *types.ServiceBinding, operation *types.Operation) error {
	instance, err := getInstanceByID(ctx, binding.ServiceInstanceID, i.repository)
	if err != nil {
		return err
	}

	osbClient, broker, service, plan, err := preparePrerequisites(ctx, i.repository, i.osbClientCreateFunc, instance)
	if err != nil {
		return err
	}

	if operation.Reschedule {
		if err := i.pollServiceBinding(ctx, osbClient, binding, instance, plan, operation, broker.ID, service.CatalogID, plan.CatalogID, operation.ExternalID, true); err != nil {
			return err
		}
		return nil
	}

	var unbindResponse *osbc.UnbindResponse
	if !operation.Reschedule {
		unbindRequest := prepareUnbindRequest(instance, binding, service.CatalogID, plan.CatalogID, service.BindingsRetrievable)

		log.C(ctx).Infof("Sending unbind request %s to broker with name %s", logUnbindRequest(unbindRequest), broker.Name)
		unbindResponse, err = osbClient.Unbind(unbindRequest)
		if err != nil {
			if osbc.IsGoneError(err) {
				log.C(ctx).Infof("Synchronous unbind %s to broker %s returned 410 GONE and is considered success",
					logUnbindRequest(unbindRequest), broker.Name)
				return nil
			}
			brokerError := &util.HTTPError{
				ErrorType:   "BrokerError",
				Description: fmt.Sprintf("Failed unbind request %s: %s", logUnbindRequest(unbindRequest), err),
				StatusCode:  http.StatusBadGateway,
			}
			if shouldStartOrphanMitigation(err) {
				operation.DeletionScheduled = time.Now()
				operation.Reschedule = false
				operation.RescheduleTimestamp = time.Time{}
				if _, err := i.repository.Update(ctx, operation, types.LabelChanges{}); err != nil {
					return fmt.Errorf("failed to update operation with id %s to schedule orphan mitigation after broker error %s: %s", operation.ID, brokerError, err)
				}
			}
			return brokerError
		}

		if unbindResponse.Async {
			log.C(ctx).Infof("Successful asynchronous unbind request %s to broker %s returned response %s",
				logUnbindRequest(unbindRequest), broker.Name, logUnbindResponse(unbindResponse))
			operation.Reschedule = true
			if operation.Context.IsAsyncNotDefined {
				operation.Context.Async = true
			}
			if operation.RescheduleTimestamp.IsZero() {
				operation.RescheduleTimestamp = time.Now()
			}

			if unbindResponse.OperationKey != nil {
				operation.ExternalID = string(*unbindResponse.OperationKey)
			}
			if _, err := i.repository.Update(ctx, operation, types.LabelChanges{}); err != nil {
				return fmt.Errorf("failed to update operation with id %s to mark that rescheduling is possible: %s", operation.ID, err)
			}
		} else {
			log.C(ctx).Infof("Successful synchronous unbind %s to broker %s returned response %s",
				logUnbindRequest(unbindRequest), broker.Name, logUnbindResponse(unbindResponse))
		}
	}

	if shouldStartPolling(operation) {
		if err := i.pollServiceBinding(ctx, osbClient, binding, instance, plan, operation, broker.ID, service.CatalogID, plan.CatalogID, operation.ExternalID, true); err != nil {
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

func (i *ServiceBindingInterceptor) isInstanceInDeletion(ctx context.Context, instanceID string) (bool, error) {
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
	lastOperationObject, err := repository.Get(ctx, types.OperationType, byResourceID, orderDesc)
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
				Description: util.HandleStorageError(err, types.ServiceInstanceType.String()).Error(),
				StatusCode:  http.StatusNotFound,
			}
		}
		return nil, fmt.Errorf("could not fetch instance with id %s from db: %s", instanceID, util.HandleStorageError(err, types.ServiceInstanceType.String()))
	}

	return instanceObject.(*types.ServiceInstance), nil
}

func (i *ServiceBindingInterceptor) prepareBindRequest(instance *types.ServiceInstance, binding *types.ServiceBinding, serviceCatalogID, planCatalogID string, bindingRetrievable bool) (*osbc.BindRequest, error) {
	context := make(map[string]interface{})
	if len(binding.Context) != 0 {
		var err error
		binding.Context, err = sjson.SetBytes(binding.Context, "instance_name", instance.Name)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(binding.Context, &context); err != nil {
			return nil, fmt.Errorf("failed to unmarshal already present OSB context: %s", err)
		}
	} else {
		context = map[string]interface{}{
			"platform":      types.SMPlatform,
			"instance_name": instance.Name,
		}

		if len(i.tenantKey) != 0 {
			if tenantValue, ok := binding.GetLabels()[i.tenantKey]; ok {
				context[i.tenantKey] = tenantValue[0]
			}
		}

		contextBytes, err := json.Marshal(context)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal OSB context %+v: %s", context, err)
		}
		binding.Context = contextBytes
	}

	bindRequest := &osbc.BindRequest{
		BindingID:         binding.ID,
		InstanceID:        instance.ID,
		AcceptsIncomplete: bindingRetrievable,
		ServiceID:         serviceCatalogID,
		PlanID:            planCatalogID,
		Parameters:        binding.Parameters,
		Context:           context,
		//TODO no OI for SM platform yet
		OriginatingIdentity: nil,
	}

	return bindRequest, nil
}

func prepareUnbindRequest(instance *types.ServiceInstance, binding *types.ServiceBinding, serviceCatalogID, planCatalogID string, bindingRetrievable bool) *osbc.UnbindRequest {
	unbindRequest := &osbc.UnbindRequest{
		BindingID:         binding.ID,
		InstanceID:        instance.ID,
		AcceptsIncomplete: bindingRetrievable,
		ServiceID:         serviceCatalogID,
		PlanID:            planCatalogID,
		//TODO no OI for SM platform yet
		OriginatingIdentity: nil,
	}

	return unbindRequest
}

func (i *ServiceBindingInterceptor) pollServiceBinding(ctx context.Context, osbClient osbc.Client, binding *types.ServiceBinding, instance *types.ServiceInstance, plan *types.ServicePlan, operation *types.Operation, brokerID, serviceCatalogID, planCatalogID, operationKey string, enableOrphanMitigation bool) error {
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

	planMaxPollingDuration := time.Duration(plan.MaximumPollingDuration) * time.Second
	leftPollingDuration := time.Duration(math.MaxInt64) // Never tick if plan has not specified max_polling_duration

	if planMaxPollingDuration > 0 {
		// MaximumPollingDuration can span multiple reschedules
		leftPollingDuration = planMaxPollingDuration - (time.Since(operation.RescheduleTimestamp))
		if leftPollingDuration <= 0 { // The Maximum Polling Duration elapsed before this polling start
			return i.processMaxPollingDurationElapsed(ctx, binding, instance, plan, operation, enableOrphanMitigation)
		}
	}

	maxPollingDurationTicker := time.NewTicker(leftPollingDuration)
	defer maxPollingDurationTicker.Stop()

	ticker := time.NewTicker(i.pollingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.C(ctx).Errorf("Terminating poll last operation for binding with id %s and name %s due to context done event", binding.ID, binding.Name)
			// The context is done, either because SM crashed/exited or because action timeout elapsed. In this case the operation should be kept in progress.
			// This way the operation would be rescheduled and the polling will span multiple reschedules, but no more than max_polling_interval if provided in the plan.
			return nil
		case <-maxPollingDurationTicker.C:
			return i.processMaxPollingDurationElapsed(ctx, binding, instance, plan, operation, enableOrphanMitigation)
		case <-ticker.C:
			log.C(ctx).Infof("Sending poll last operation request %s for binding with id %s and name %s",
				logPollBindingRequest(pollingRequest), binding.ID, binding.Name)
			pollingResponse, err := osbClient.PollBindingLastOperation(pollingRequest)
			if err != nil {
				if osbc.IsGoneError(err) && operation.Type == types.DELETE {
					log.C(ctx).Infof("Successfully finished polling operation for binding with id %s and name %s", binding.ID, binding.Name)

					operation.Reschedule = false
					operation.RescheduleTimestamp = time.Time{}
					if _, err := i.repository.Update(ctx, operation, types.LabelChanges{}); err != nil {
						return fmt.Errorf("failed to update operation with id %s to mark that next execution should be a reschedule: %s", operation.ID, err)
					}
					return nil
				}

				return &util.HTTPError{
					ErrorType: "BrokerError",
					Description: fmt.Sprintf("Failed poll last operation request %s for binding with id %s and name %s: %s",
						logPollBindingRequest(pollingRequest), binding.ID, binding.Name, err),
					StatusCode: http.StatusBadGateway,
				}
			}

			switch pollingResponse.State {
			case osbc.StateInProgress:
				log.C(ctx).Infof("Polling of binding still in progress. Rescheduling polling last operation request %s for binding of instance with id %s and name %s...",
					logPollBindingRequest(pollingRequest), binding.ID, binding.Name)

			case osbc.StateSucceeded:
				log.C(ctx).Infof("Successfully finished polling operation for binding with id %s and name %s", binding.ID, binding.Name)

				operation.Reschedule = false
				operation.RescheduleTimestamp = time.Time{}
				if _, err := i.repository.Update(ctx, operation, types.LabelChanges{}); err != nil {
					return fmt.Errorf("failed to update operation with id %s to mark that next execution should be a reschedule: %s", operation.ID, err)
				}

				// for async creation of bindings, an extra fetching of the binding is required to get the credentials
				if operation.Type == types.CREATE {
					bindingDetails, err := i.getBindingDetailsFromBroker(ctx, binding, operation, brokerID, osbClient)
					if err != nil {
						return err
					}
					if err := i.enrichBindingWithBindingResponse(binding, bindingDetails); err != nil {
						return fmt.Errorf("could not enrich binding details with binding response details: %s", err)
					}
				}

				return nil
			case osbc.StateFailed:
				log.C(ctx).Infof("Failed polling operation for binding with id %s and name %s with response %s",
					binding.ID, binding.Name, logPollBindingResponse(pollingResponse))
				operation.Reschedule = false
				operation.RescheduleTimestamp = time.Time{}
				if enableOrphanMitigation {
					operation.DeletionScheduled = time.Now()
				}
				if _, err := i.repository.Update(ctx, operation, types.LabelChanges{}); err != nil {
					return fmt.Errorf("failed to update operation with id %s after failed of last operation for binding with id %s: %s", operation.ID, binding.ID, err)
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

func (i *ServiceBindingInterceptor) processMaxPollingDurationElapsed(ctx context.Context, binding *types.ServiceBinding, instance *types.ServiceInstance, plan *types.ServicePlan, operation *types.Operation, enableOrphanMitigation bool) error {
	log.C(ctx).Errorf("Terminating poll last operation for binding with id %s and name %s for instance with id %s and name %s due to maximum_polling_duration %ds for it's plan %s is reached", binding.ID, binding.Name, instance.ID, instance.Name, plan.MaximumPollingDuration, plan.Name)
	operation.Reschedule = false
	operation.RescheduleTimestamp = time.Time{}
	if enableOrphanMitigation {
		operation.DeletionScheduled = time.Now()
	}
	if _, err := i.repository.Update(ctx, operation, types.LabelChanges{}); err != nil {
		return fmt.Errorf("failed to update operation with id %s after failed of last operation for binding with id %s: %s", operation.ID, binding.ID, err)
	}
	return &util.HTTPError{
		ErrorType:   "BrokerError",
		Description: fmt.Sprintf("failed polling operation for binding with id %s and name %s for instance with id %s and name %s due to maximum_polling_duration %ds for it's plan %s is reached", binding.ID, binding.Name, instance.ID, instance.Name, plan.MaximumPollingDuration, plan.Name),
		StatusCode:  http.StatusBadGateway,
	}
}

func (i *ServiceBindingInterceptor) getBindingDetailsFromBroker(ctx context.Context, binding *types.ServiceBinding, operation *types.Operation, brokerID string, osbClient osbc.Client) (*bindResponseDetails, error) {
	getBindingRequest := &osbc.GetBindingRequest{
		InstanceID: binding.ServiceInstanceID,
		BindingID:  binding.ID,
	}
	log.C(ctx).Infof("Sending get binding request %s to broker with id %s", logGetBindingRequest(getBindingRequest), brokerID)
	bindingResponse, err := osbClient.GetBinding(getBindingRequest)
	if err != nil {
		brokerError := &util.HTTPError{
			ErrorType:   "BrokerError",
			Description: fmt.Sprintf("Failed get bind request %s after successfully finished polling: %s", logGetBindingRequest(getBindingRequest), err),
			StatusCode:  http.StatusBadGateway,
		}
		if shouldStartOrphanMitigation(err) {
			// mark the operation as deletion scheduled meaning orphan mitigation is required
			operation.DeletionScheduled = time.Now()
			operation.Reschedule = false
			operation.RescheduleTimestamp = time.Time{}
			if _, err := i.repository.Update(ctx, operation, types.LabelChanges{}); err != nil {
				return nil, fmt.Errorf("failed to update operation with id %s to schedule orphan mitigation after broker error %s: %s",
					operation.ID, brokerError, err)
			}
		}
		return nil, brokerError
	}

	log.C(ctx).Infof("broker with id %s returned successful get binding response %s", brokerID, logGetBindingResponse(bindingResponse))
	bindResponseDetails := &bindResponseDetails{
		Credentials:     bindingResponse.Credentials,
		SyslogDrainURL:  bindingResponse.SyslogDrainURL,
		RouteServiceURL: bindingResponse.RouteServiceURL,
		VolumeMounts:    bindingResponse.VolumeMounts,
	}

	return bindResponseDetails, nil
}

func (i *ServiceBindingInterceptor) enrichBindingWithBindingResponse(binding *types.ServiceBinding, response *bindResponseDetails) error {
	if len(response.Credentials) != 0 {
		credentialBytes, err := json.Marshal(response.Credentials)
		if err != nil {
			return fmt.Errorf("could not marshal binding credentials: %s", err)
		}

		binding.Credentials = credentialBytes
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

func logBindRequest(request *osbc.BindRequest) string {
	return fmt.Sprintf("context: %+v, bindingID: %s, instanceID: %s, planID: %s, serviceID: %s, acceptsIncomplete: %t",
		request.Context, request.BindingID, request.InstanceID, request.PlanID, request.ServiceID, request.AcceptsIncomplete)
}

func logBindResponse(response *osbc.BindResponse) string {
	return fmt.Sprintf("async: %t, operationKey: %s", response.Async, opKeyPtrToStr(response.OperationKey))
}

func logUnbindRequest(request *osbc.UnbindRequest) string {
	return fmt.Sprintf("bindingID: %s, instanceID: %s, planID: %s, serviceID: %s, acceptsIncomplete: %t",
		request.BindingID, request.InstanceID, request.PlanID, request.ServiceID, request.AcceptsIncomplete)
}

func logUnbindResponse(response *osbc.UnbindResponse) string {
	return fmt.Sprintf("async: %t, operationKey: %s", response.Async, opKeyPtrToStr(response.OperationKey))
}

func logPollBindingRequest(request *osbc.BindingLastOperationRequest) string {
	return fmt.Sprintf("bindingID: %s, instanceID: %s, planID: %s, serviceID: %s, operationKey: %s",
		request.BindingID, request.InstanceID, strPtrToStr(request.PlanID), strPtrToStr(request.ServiceID), opKeyPtrToStr(request.OperationKey))
}

func logPollBindingResponse(response *osbc.LastOperationResponse) string {
	return fmt.Sprintf("state: %s, description: %s", response.State, strPtrToStr(response.Description))
}

func logGetBindingRequest(request *osbc.GetBindingRequest) string {
	return fmt.Sprintf("bindingID: %s, instanceID: %s", request.BindingID, request.InstanceID)
}

func logGetBindingResponse(response *osbc.GetBindingResponse) string {
	return fmt.Sprintf("response redacted: <credentials present: %t>", len(response.Credentials) != 0)
}
