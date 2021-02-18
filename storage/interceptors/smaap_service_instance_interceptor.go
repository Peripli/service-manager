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
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/tidwall/sjson"

	"github.com/Peripli/service-manager/operations/opcontext"
	"github.com/Peripli/service-manager/pkg/util"

	"github.com/Peripli/service-manager/pkg/log"

	"github.com/Peripli/service-manager/pkg/query"

	"github.com/Peripli/service-manager/pkg/types"
	osbc "github.com/kubernetes-sigs/go-open-service-broker-client/v2"

	"github.com/Peripli/service-manager/storage"
)

const (
	ServiceInstanceCreateInterceptorProviderName = "ServiceInstanceCreateInterceptorProvider"
	OperatedByLabelKey                           = "operated_by"
)

type BaseSMAAPInterceptorProvider struct {
	OSBClientCreateFunc osbc.CreateFunc
	Repository          *storage.InterceptableTransactionalRepository
	TenantKey           string
	PollingInterval     time.Duration
}

// ServiceInstanceCreateInterceptorProvider provides an interceptor that notifies the actual broker about instance creation
type ServiceInstanceCreateInterceptorProvider struct {
	*BaseSMAAPInterceptorProvider
}

func (p *ServiceInstanceCreateInterceptorProvider) Provide() storage.CreateAroundTxInterceptor {
	return &ServiceInstanceInterceptor{
		osbClientCreateFunc: p.OSBClientCreateFunc,
		repository:          p.Repository,
		tenantKey:           p.TenantKey,
		pollingInterval:     p.PollingInterval,
	}
}

func (c *ServiceInstanceCreateInterceptorProvider) Name() string {
	return ServiceInstanceCreateInterceptorProviderName
}

const ServiceInstanceUpdateInterceptorProviderName = "ServiceInstanceUpdateInterceptorProvider"

// ServiceInstanceUpdateInterceptorProvider provides an interceptor that notifies the actual broker about instance updates
type ServiceInstanceUpdateInterceptorProvider struct {
	*BaseSMAAPInterceptorProvider
}

func (p *ServiceInstanceUpdateInterceptorProvider) Provide() storage.UpdateAroundTxInterceptor {
	return &ServiceInstanceInterceptor{
		osbClientCreateFunc: p.OSBClientCreateFunc,
		repository:          p.Repository,
		tenantKey:           p.TenantKey,
		pollingInterval:     p.PollingInterval,
	}
}

func (c *ServiceInstanceUpdateInterceptorProvider) Name() string {
	return ServiceInstanceUpdateInterceptorProviderName
}

const ServiceInstanceDeleteInterceptorProviderName = "ServiceInstanceDeleteInterceptorProvider"

// ServiceInstanceDeleteInterceptorProvider provides an interceptor that notifies the actual broker about instance deletion
type ServiceInstanceDeleteInterceptorProvider struct {
	*BaseSMAAPInterceptorProvider
}

func (p *ServiceInstanceDeleteInterceptorProvider) Provide() storage.DeleteAroundTxInterceptor {
	return &ServiceInstanceInterceptor{
		osbClientCreateFunc: p.OSBClientCreateFunc,
		repository:          p.Repository,
		tenantKey:           p.TenantKey,
		pollingInterval:     p.PollingInterval,
	}
}

func (c *ServiceInstanceDeleteInterceptorProvider) Name() string {
	return ServiceInstanceDeleteInterceptorProviderName
}

type ServiceInstanceInterceptor struct {
	osbClientCreateFunc osbc.CreateFunc
	repository          *storage.InterceptableTransactionalRepository
	tenantKey           string
	pollingInterval     time.Duration
}

func (i *ServiceInstanceInterceptor) AroundTxCreate(f storage.InterceptCreateAroundTxFunc) storage.InterceptCreateAroundTxFunc {
	return func(ctx context.Context, obj types.Object) (types.Object, error) {
		instance := obj.(*types.ServiceInstance)
		instance.Usable = false
		smaapOperated := instance.Labels != nil && len(instance.Labels[OperatedByLabelKey]) > 0

		if instance.PlatformID != types.SMPlatform && !smaapOperated {
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

		if operation.Reschedule {
			if err := i.pollServiceInstance(ctx, osbClient, instance, plan, operation, service.CatalogID, plan.CatalogID, true); err != nil {
				return nil, err
			}

			return instance, nil
		}

		var provisionResponse *osbc.ProvisionResponse
		if !operation.Reschedule {
			operation.Context.ServicePlanID = instance.ServicePlanID
			provisionRequest, err := i.prepareProvisionRequest(instance, service.CatalogID, plan.CatalogID)
			if err != nil {
				return nil, fmt.Errorf("failed to prepare provision request: %s", err)
			}
			log.C(ctx).Infof("Sending provision request %s to broker with name %s", logProvisionRequest(provisionRequest), broker.Name)
			provisionResponse, err = osbClient.ProvisionInstance(provisionRequest)
			if err != nil {
				brokerError := &util.HTTPError{
					ErrorType:   "BrokerError",
					Description: fmt.Sprintf("Failed provisioning request %s: %s", logProvisionRequest(provisionRequest), err),
					StatusCode:  http.StatusBadGateway,
				}

				if shouldStartOrphanMitigation(err) {
					// mark the operation as deletion scheduled meaning orphan mitigation is required
					operation.DeletionScheduled = time.Now().UTC()
					operation.Reschedule = false
					operation.RescheduleTimestamp = time.Time{}
					if _, err := i.repository.Update(ctx, operation, types.LabelChanges{}); err != nil {
						return nil, fmt.Errorf("failed to update operation with id %s to schedule orphan mitigation after broker error %s: %s", operation.ID, brokerError, err)
					}
				}

				//save the instance in case of an async client call
				if operation.IsAsyncResponse() {
					_, err := f(ctx, obj)
					if err != nil {
						return nil, err
					}
				}

				return nil, brokerError
			}

			if provisionResponse.DashboardURL != nil {
				dashboardURL := *provisionResponse.DashboardURL
				instance.DashboardURL = dashboardURL
			}

			if provisionResponse.Async {
				log.C(ctx).Infof("Successful asynchronous provisioning request %s to broker %s returned response %s",
					logProvisionRequest(provisionRequest), broker.Name, logProvisionResponse(provisionResponse))
				operation.Reschedule = true
				if operation.Context.IsAsyncNotDefined {
					operation.Context.Async = true
				}
				if operation.RescheduleTimestamp.IsZero() {
					operation.RescheduleTimestamp = time.Now()
				}
				if provisionResponse.OperationKey != nil {
					operation.ExternalID = string(*provisionResponse.OperationKey)
				}
				if _, err := i.repository.Update(ctx, operation, types.LabelChanges{}); err != nil {
					return nil, fmt.Errorf("failed to update operation with id %s to mark that next execution should be a reschedule: %s", instance.ID, err)
				}
			} else {
				log.C(ctx).Infof("Successful synchronous provisioning %s to broker %s returned response %s",
					logProvisionRequest(provisionRequest), broker.Name, logProvisionResponse(provisionResponse))

			}

			object, err := f(ctx, obj)
			if err != nil {
				return nil, err
			}
			instance = object.(*types.ServiceInstance)
		}

		if shouldStartPolling(operation) {
			if err := i.pollServiceInstance(ctx, osbClient, instance, plan, operation, service.CatalogID, plan.CatalogID, true); err != nil {
				return nil, err
			}
		}

		return instance, nil
	}
}

func (i *ServiceInstanceInterceptor) AroundTxUpdate(f storage.InterceptUpdateAroundTxFunc) storage.InterceptUpdateAroundTxFunc {
	return func(ctx context.Context, updatedObj types.Object, labelChanges ...*types.LabelChange) (object types.Object, err error) {
		updatedInstance := updatedObj.(*types.ServiceInstance)
		smaapOperated := updatedInstance.Labels != nil && len(updatedInstance.Labels[OperatedByLabelKey]) > 0

		if updatedInstance.PlatformID != types.SMPlatform && !smaapOperated {
			return f(ctx, updatedObj, labelChanges...)
		}

		operation, found := opcontext.Get(ctx)
		if !found {
			return nil, fmt.Errorf("operation missing from context")
		}

		osbClient, broker, service, plan, err := preparePrerequisites(ctx, i.repository, i.osbClientCreateFunc, updatedInstance)
		if err != nil {
			return nil, err
		}

		if operation.Reschedule {
			if err := i.pollServiceInstance(ctx, osbClient, updatedInstance, plan, operation, service.CatalogID, plan.CatalogID, false); err != nil {
				updatedInstance.UpdateValues = types.InstanceUpdateValues{}
				_, updateErr := i.repository.RawRepository.Update(ctx, updatedInstance, types.LabelChanges{})
				if updateErr != nil {
					return nil, updateErr
				}
				return nil, err
			}
			return updatedInstance, nil
		}

		var instance *types.ServiceInstance
		if !operation.Reschedule {
			instanceObjBeforeUpdate, err := i.repository.Get(ctx, types.ServiceInstanceType, query.Criterion{
				LeftOp:   "id",
				Operator: query.EqualsOperator,
				RightOp:  []string{updatedInstance.ID},
				Type:     query.FieldQuery,
			})
			if err != nil {
				log.C(ctx).WithError(err).Errorf("could not get instance with id '%s'", updatedInstance.ID)
				return nil, err
			}
			instance = instanceObjBeforeUpdate.(*types.ServiceInstance)

			oldServicePlanObj, err := i.repository.Get(ctx, types.ServicePlanType, query.Criterion{
				LeftOp:   "id",
				Operator: query.EqualsOperator,
				RightOp:  []string{instance.ServicePlanID},
				Type:     query.FieldQuery,
			})
			if err != nil {
				return nil, &util.HTTPError{
					ErrorType:   "NotFound",
					Description: fmt.Sprintf("current service plan with id %s for instance %s no longer exists and instance updates are not allowed", instance.ServicePlanID, instance.Name),
					StatusCode:  http.StatusBadRequest,
				}
			}
			oldServicePlan := oldServicePlanObj.(*types.ServicePlan)
			var updateInstanceResponse *osbc.UpdateInstanceResponse
			updateInstanceRequest, err := i.prepareUpdateInstanceRequest(updatedInstance, service.CatalogID, plan.CatalogID, oldServicePlan.CatalogID)
			if err != nil {
				return nil, fmt.Errorf("faied to prepare update instance request: %s", err)
			}
			log.C(ctx).Infof("Sending update instance request %s to broker with name %s", logUpdateInstanceRequest(updateInstanceRequest), broker.Name)
			updateInstanceResponse, err = osbClient.UpdateInstance(updateInstanceRequest)
			if err != nil {
				brokerError := &util.HTTPError{
					ErrorType:   "BrokerError",
					Description: fmt.Sprintf("Failed update instance request %s: %s", logUpdateInstanceRequest(updateInstanceRequest), err),
					StatusCode:  http.StatusBadGateway,
				}

				return nil, brokerError
			}

			// SM should not not store parameters
			updatedInstance.Parameters = nil
			// if broker returned a dashboard URL, store it in SM
			if updateInstanceResponse.DashboardURL != nil {
				dashboardURL := *updateInstanceResponse.DashboardURL
				updatedInstance.DashboardURL = dashboardURL
			}
			instance.UpdateValues = types.InstanceUpdateValues{
				ServiceInstance: updatedInstance,
				LabelChanges:    labelChanges,
			}
			// use repository with no interceptors to attach the update details to the instance
			instanceObjAfterRawUpdate, err := i.repository.RawRepository.Update(ctx, instance, types.LabelChanges{})
			if err != nil {
				return nil, err
			}
			instance = instanceObjAfterRawUpdate.(*types.ServiceInstance)
			if updateInstanceResponse.Async {
				log.C(ctx).Infof("Successful asynchronous update instance request %s to broker %s returned response %s",
					logUpdateInstanceRequest(updateInstanceRequest), broker.Name, logUpdateInstanceResponse(updateInstanceResponse))
				operation.Reschedule = true
				if operation.Context.IsAsyncNotDefined {
					operation.Context.Async = true
				}
				if operation.RescheduleTimestamp.IsZero() {
					operation.RescheduleTimestamp = time.Now()
				}
				if updateInstanceResponse.OperationKey != nil {
					operation.ExternalID = string(*updateInstanceResponse.OperationKey)
				}

				if _, err := i.repository.Update(ctx, operation, types.LabelChanges{}); err != nil {
					return nil, fmt.Errorf("failed to update operation with id %s to mark that next execution should be a reschedule: %s", updatedInstance.ID, err)
				}
			} else {
				log.C(ctx).Infof("Successful synchronous update instance %s to broker %s returned response %s",
					logUpdateInstanceRequest(updateInstanceRequest), broker.Name, logUpdateInstanceResponse(updateInstanceResponse))
			}
		} else {
			instance = updatedInstance
		}

		if shouldStartPolling(operation) {
			if err := i.pollServiceInstance(ctx, osbClient, updatedInstance, plan, operation, service.CatalogID, plan.CatalogID, false); err != nil {
				instance.UpdateValues = types.InstanceUpdateValues{}
				_, updateErr := i.repository.RawRepository.Update(ctx, instance, types.LabelChanges{})
				if updateErr != nil {
					return nil, updateErr
				}
				return nil, err
			}
		}
		// continue further down the interceptor chain with the updated instance

		//if there are post around tx interceptors registered on the flow and they fail, we might result in a situation where we have
		// successfully updated instance and an UPDATE operation that is marked as failed!
		return f(ctx, instance.UpdateValues.ServiceInstance, instance.UpdateValues.LabelChanges...)
	}
}

func (i *ServiceInstanceInterceptor) AroundTxDelete(f storage.InterceptDeleteAroundTxFunc) storage.InterceptDeleteAroundTxFunc {
	return func(ctx context.Context, deletionCriteria ...query.Criterion) error {
		instances, err := i.repository.List(ctx, types.ServiceInstanceType, deletionCriteria...)
		if err != nil {
			return fmt.Errorf("failed to get instances for deletion: %s", err)
		}

		if instances.Len() > 1 {
			return fmt.Errorf("deletion of multiple instances is not supported")
		}

		operation, found := opcontext.Get(ctx)
		if !found {
			return fmt.Errorf("operation missing from context")
		}

		if instances.Len() != 0 {
			instance := instances.ItemAt(0).(*types.ServiceInstance)
			//When sm is async polling we have a binding (in case user expect a sync response it should be deleted)
			if operation.Type == types.CREATE && !operation.IsAsyncResponse() {
				if err := i.repository.RawRepository.Delete(ctx, types.ServiceInstanceType, deletionCriteria...); err != nil {
					return err
				}
			}
			if err := i.deleteSingleInstance(ctx, instance, operation); err != nil {
				return err
			}
		} else if operation.InOrphanMitigationState() && operation.Context.ServicePlanID != "" {
			// In case we don't have instance & use the operation context to perform orphan mitigation
			instance := types.ServiceInstance{}
			instance.ServicePlanID = operation.Context.ServicePlanID
			instance.ID = operation.ResourceID
			if err := i.deleteSingleInstance(ctx, &instance, operation); err != nil {
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

func (i *ServiceInstanceInterceptor) deleteSingleInstance(ctx context.Context, instance *types.ServiceInstance, operation *types.Operation) error {
	byServiceInstanceID := query.ByField(query.EqualsOperator, "service_instance_id", instance.ID)
	var bindingsCount int
	var err error
	if bindingsCount, err = i.repository.Count(ctx, types.ServiceBindingType, byServiceInstanceID); err != nil {
		return fmt.Errorf("could not fetch bindings for instance with id %s", instance.ID)
	}
	if bindingsCount > 0 {
		return &util.HTTPError{
			ErrorType:   "BadRequest",
			Description: fmt.Sprintf("could not delete instance due to %d existing bindings", bindingsCount),
			StatusCode:  http.StatusBadRequest,
		}
	}

	osbClient, broker, service, plan, err := preparePrerequisites(ctx, i.repository, i.osbClientCreateFunc, instance)
	if err != nil {
		return err
	}

	// if deletion scheduled is true this means that either a delete or create operation failed and orphan mitigation was required
	if operation.InOrphanMitigationState() {
		log.C(ctx).Infof("Orphan mitigation in progress for instance with id %s and name %s triggered due to failure in operation %s", instance.ID, instance.Name, operation.Type)
	}

	if operation.Reschedule {
		if err := i.pollServiceInstance(ctx, osbClient, instance, plan, operation, service.CatalogID, plan.CatalogID, true); err != nil {
			return err
		}
		return nil
	}

	var deprovisionResponse *osbc.DeprovisionResponse
	if !operation.Reschedule {
		deprovisionRequest := prepareDeprovisionRequest(instance, service.CatalogID, plan.CatalogID)

		log.C(ctx).Infof("Sending deprovision request %s to broker with name %s", logDeprovisionRequest(deprovisionRequest), broker.Name)
		deprovisionResponse, err = osbClient.DeprovisionInstance(deprovisionRequest)
		if err != nil {
			if osbc.IsGoneError(err) {
				log.C(ctx).Infof("Synchronous deprovisioning %s to broker %s returned 410 GONE and is considered success",
					logDeprovisionRequest(deprovisionRequest), broker.Name)
				return nil
			}
			brokerError := &util.HTTPError{
				ErrorType:   "BrokerError",
				Description: fmt.Sprintf("Failed deprovisioning request %s: %s", logDeprovisionRequest(deprovisionRequest), err),
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

		if deprovisionResponse.Async {
			log.C(ctx).Infof("Successful asynchronous deprovisioning request %s to broker %s returned response %s",
				logDeprovisionRequest(deprovisionRequest), broker.Name, logDeprovisionResponse(deprovisionResponse))
			operation.Reschedule = true
			if operation.Context.IsAsyncNotDefined {
				operation.Context.Async = true
			}
			if operation.RescheduleTimestamp.IsZero() {
				operation.RescheduleTimestamp = time.Now()
			}

			if deprovisionResponse.OperationKey != nil {
				operation.ExternalID = string(*deprovisionResponse.OperationKey)
			}
			if _, err := i.repository.Update(ctx, operation, types.LabelChanges{}); err != nil {
				return fmt.Errorf("failed to update operation with id %s to mark that rescheduling is possible: %s", operation.ID, err)
			}
		} else {
			log.C(ctx).Infof("Successful synchronous deprovisioning %s to broker %s returned response %s",
				logDeprovisionRequest(deprovisionRequest), broker.Name, logDeprovisionResponse(deprovisionResponse))
		}
	}

	if shouldStartPolling(operation) {
		if err := i.pollServiceInstance(ctx, osbClient, instance, plan, operation, service.CatalogID, plan.CatalogID, true); err != nil {
			return err
		}
	}

	return nil
}

func (i *ServiceInstanceInterceptor) pollServiceInstance(ctx context.Context, osbClient osbc.Client, instance *types.ServiceInstance, plan *types.ServicePlan, operation *types.Operation, serviceCatalogID, planCatalogID string, enableOrphanMitigation bool) error {
	var key *osbc.OperationKey
	if len(operation.ExternalID) != 0 {
		opKey := osbc.OperationKey(operation.ExternalID)
		key = &opKey
	}

	pollingRequest := &osbc.LastOperationRequest{
		InstanceID:   instance.ID,
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
			return i.processMaxPollingDurationElapsed(ctx, instance, plan, operation, enableOrphanMitigation)
		}
	}

	maxPollingDurationTicker := time.NewTicker(leftPollingDuration)
	defer maxPollingDurationTicker.Stop()

	ticker := time.NewTicker(i.pollingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.C(ctx).Errorf("Terminating poll last operation for instance with id %s and name %s due to context done event", instance.ID, instance.Name)
			// The context is done, either because SM crashed/exited or because action timeout elapsed. In this case the operation should be kept in progress.
			// This way the operation would be rescheduled and the polling will span multiple reschedules, but no more than max_polling_interval if provided in the plan.
			return nil
		case <-maxPollingDurationTicker.C:
			return i.processMaxPollingDurationElapsed(ctx, instance, plan, operation, enableOrphanMitigation)
		case <-ticker.C:
			log.C(ctx).Infof("Sending poll last operation request %s for instance with id %s and name %s", logPollInstanceRequest(pollingRequest), instance.ID, instance.Name)
			pollingResponse, err := osbClient.PollLastOperation(pollingRequest)
			if err != nil {
				if osbc.IsGoneError(err) && operation.Type == types.DELETE {
					log.C(ctx).Infof("Successfully finished polling operation for instance with id %s and name %s", instance.ID, instance.Name)

					operation.Reschedule = false
					operation.RescheduleTimestamp = time.Time{}
					if _, err := i.repository.Update(ctx, operation, types.LabelChanges{}); err != nil {
						return fmt.Errorf("failed to update operation with id %s to mark that next execution should not be reschedulable", operation.ID)
					}
					return nil
				} else if isUnreachableBroker(err) {
					log.C(ctx).Errorf("Broker temporarily unreachable. Rescheduling polling last operation request %s to for provisioning of instance with id %s and name %s...",
						logPollInstanceRequest(pollingRequest), instance.ID, instance.Name)
				} else {
					return &util.HTTPError{
						ErrorType: "BrokerError",
						Description: fmt.Sprintf("Failed poll last operation request %s for instance with id %s and name %s: %s",
							logPollInstanceRequest(pollingRequest), instance.ID, instance.Name, err),
						StatusCode: http.StatusBadGateway,
					}
				}
			} else {
				switch pollingResponse.State {
				case osbc.StateInProgress:
					log.C(ctx).Infof("Polling of instance still in progress. Rescheduling polling last operation request %s to for provisioning of instance with id %s and name %s...",
						logPollInstanceRequest(pollingRequest), instance.ID, instance.Name)

				case osbc.StateSucceeded:
					log.C(ctx).Infof("Successfully finished polling operation for instance with id %s and name %s", instance.ID, instance.Name)

					operation.Reschedule = false
					operation.RescheduleTimestamp = time.Time{}
					if _, err := i.repository.Update(ctx, operation, types.LabelChanges{}); err != nil {
						return fmt.Errorf("failed to update operation with id %s to mark that next execution should be a reschedule: %s", operation.ID, err)
					}

					return nil
				case osbc.StateFailed:
					log.C(ctx).Infof("Failed polling operation for instance with id %s and name %s with response %s", instance.ID, instance.Name, logPollInstanceResponse(pollingResponse))
					operation.Reschedule = false
					operation.RescheduleTimestamp = time.Time{}
					if enableOrphanMitigation {
						operation.DeletionScheduled = time.Now()
					}
					if _, err := i.repository.Update(ctx, operation, types.LabelChanges{}); err != nil {
						return fmt.Errorf("failed to update operation with id %s after failed of last operation for instance with id %s: %s", operation.ID, instance.ID, err)
					}

					errDescription := ""
					if pollingResponse.Description != nil {
						errDescription = *pollingResponse.Description
					} else {
						errDescription = "no description provided by broker"
					}
					return &util.HTTPError{
						ErrorType:   "BrokerError",
						Description: fmt.Sprintf("failed polling operation for instance with id %s and name %s due to polling last operation error: %s", instance.ID, instance.Name, errDescription),
						StatusCode:  http.StatusBadGateway,
					}
				default:
					log.C(ctx).Errorf("invalid state during poll last operation for instance with id %s and name %s: %s. Continuing polling...", instance.ID, instance.Name, pollingResponse.State)
				}
			}
		}
	}
}

func isUnreachableBroker(err error) bool {
	if timeOutError, ok := err.(net.Error); ok && timeOutError.Timeout() {
		return true
	}
	httpError, ok := osbc.IsHTTPError(err)
	if !ok {
		return false
	}
	return (httpError.StatusCode == http.StatusServiceUnavailable || httpError.StatusCode == http.StatusNotFound)
}

func shouldStartPolling(operation *types.Operation) bool {

	// In case the operation not rescheduled, don't start polling
	if !operation.Reschedule {
		return false
	}

	if operation.Context != nil {
		// The polling should start if the operation is rescheduled and (orphan mitigation state is in true state or async mode is defined)
		return !operation.Context.IsAsyncNotDefined || operation.InOrphanMitigationState()
	}
	return true
}

func (i *ServiceInstanceInterceptor) processMaxPollingDurationElapsed(ctx context.Context, instance *types.ServiceInstance, plan *types.ServicePlan, operation *types.Operation, enableOrphanMitigation bool) error {
	log.C(ctx).Errorf("Terminating poll last operation for instance with id %s and name %s due to maximum_polling_duration %ds for it's plan %s is reached", instance.ID, instance.Name, plan.MaximumPollingDuration, plan.Name)
	operation.Reschedule = false
	operation.RescheduleTimestamp = time.Time{}
	if enableOrphanMitigation {
		operation.DeletionScheduled = time.Now()
	}
	if _, err := i.repository.Update(ctx, operation, types.LabelChanges{}); err != nil {
		return fmt.Errorf("failed to update operation with id %s after failed of last operation for instance with id %s: %s", operation.ID, instance.ID, err)
	}
	return &util.HTTPError{
		ErrorType:   "BrokerError",
		Description: fmt.Sprintf("failed polling operation for instance with id %s and name %s due to maximum_polling_duration %ds for it's plan %s is reached", instance.ID, instance.Name, plan.MaximumPollingDuration, plan.Name),
		StatusCode:  http.StatusBadGateway,
	}
}

func preparePrerequisites(ctx context.Context, repository storage.Repository, osbClientFunc osbc.CreateFunc, instance *types.ServiceInstance) (osbc.Client, *types.ServiceBroker, *types.ServiceOffering, *types.ServicePlan, error) {
	planObject, err := repository.Get(ctx, types.ServicePlanType, query.ByField(query.EqualsOperator, "id", instance.ServicePlanID))
	if err != nil {
		return nil, nil, nil, nil, util.HandleStorageError(err, types.ServicePlanType.String())
	}
	plan := planObject.(*types.ServicePlan)

	serviceObject, err := repository.Get(ctx, types.ServiceOfferingType, query.ByField(query.EqualsOperator, "id", plan.ServiceOfferingID))
	if err != nil {
		return nil, nil, nil, nil, util.HandleStorageError(err, types.ServiceOfferingType.String())
	}
	service := serviceObject.(*types.ServiceOffering)

	brokerObject, err := repository.Get(ctx, types.ServiceBrokerType, query.ByField(query.EqualsOperator, "id", service.BrokerID))
	if err != nil {
		return nil, nil, nil, nil, util.HandleStorageError(err, types.ServiceBrokerType.String())
	}
	broker := brokerObject.(*types.ServiceBroker)

	tlsConfig, err := broker.GetTLSConfig()
	if err != nil {
		return nil, nil, nil, nil, err
	}

	osbClientConfig := &osbc.ClientConfiguration{
		Name:                broker.Name + " broker client",
		EnableAlphaFeatures: true,
		URL:                 broker.BrokerURL,
		APIVersion:          osbc.LatestAPIVersion(),
	}

	if broker.Credentials.Basic != nil {
		osbClientConfig.AuthConfig = &osbc.AuthConfig{
			BasicAuthConfig: &osbc.BasicAuthConfig{
				Username: broker.Credentials.Basic.Username,
				Password: broker.Credentials.Basic.Password,
			},
		}
	}

	if tlsConfig != nil {
		osbClientConfig.TLSConfig = tlsConfig
	}

	osbClient, err := osbClientFunc(osbClientConfig)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	return osbClient, broker, service, plan, nil
}

func (i *ServiceInstanceInterceptor) prepareProvisionRequest(instance *types.ServiceInstance, serviceCatalogID, planCatalogID string) (*osbc.ProvisionRequest, error) {
	instanceContext := make(map[string]interface{})
	if len(instance.Context) != 0 {
		var err error
		instance.Context, err = sjson.SetBytes(instance.Context, "instance_name", instance.Name)
		if err != nil {
			return nil, err
		}

		if err = json.Unmarshal(instance.Context, &instanceContext); err != nil {
			return nil, fmt.Errorf("failed to unmarshal already present OSB context: %s", err)
		}
	} else {
		instanceContext = map[string]interface{}{
			"platform":      types.SMPlatform,
			"instance_name": instance.Name,
		}

		if len(i.tenantKey) != 0 {
			if tenantValue, ok := instance.GetLabels()[i.tenantKey]; ok {
				instanceContext[i.tenantKey] = tenantValue[0]
			}
		}

		contextBytes, err := json.Marshal(instanceContext)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal OSB context %+v: %s", instanceContext, err)
		}
		instance.Context = contextBytes
	}

	provisionRequest := &osbc.ProvisionRequest{
		InstanceID:        instance.GetID(),
		AcceptsIncomplete: true,
		ServiceID:         serviceCatalogID,
		PlanID:            planCatalogID,
		OrganizationGUID:  "-",
		SpaceGUID:         "-",
		Parameters:        instance.Parameters,
		Context:           instanceContext,
		//TODO no OI for SM platform yet
		OriginatingIdentity: nil,
	}

	return provisionRequest, nil
}

func (i *ServiceInstanceInterceptor) prepareUpdateInstanceRequest(instance *types.ServiceInstance, serviceCatalogID, planCatalogID, oldCatalogPlanID string) (*osbc.UpdateInstanceRequest, error) {
	instanceContext := make(map[string]interface{})
	if len(instance.Context) != 0 {
		if err := json.Unmarshal(instance.Context, &instanceContext); err != nil {
			return nil, fmt.Errorf("failed to unmarshal already present OSB context: %s", err)
		}
	} else {
		instanceContext = map[string]interface{}{
			"platform":      types.SMPlatform,
			"instance_name": instance.Name,
		}

		if len(i.tenantKey) != 0 {
			if tenantValue, ok := instance.GetLabels()[i.tenantKey]; ok {
				instanceContext[i.tenantKey] = tenantValue[0]
			}
		}

		contextBytes, err := json.Marshal(instanceContext)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal OSB context %+v: %s", instanceContext, err)
		}
		instance.Context = contextBytes
	}

	return &osbc.UpdateInstanceRequest{
		InstanceID:        instance.GetID(),
		AcceptsIncomplete: true,
		ServiceID:         serviceCatalogID,
		PlanID:            &planCatalogID,
		Parameters:        instance.Parameters,
		Context:           instanceContext,
		PreviousValues: &osbc.PreviousValues{
			PlanID: oldCatalogPlanID,
		},
		//TODO no OI for SM platform yet
		OriginatingIdentity: nil,
	}, nil
}

func prepareDeprovisionRequest(instance *types.ServiceInstance, serviceCatalogID, planCatalogID string) *osbc.DeprovisionRequest {
	return &osbc.DeprovisionRequest{
		InstanceID:        instance.ID,
		AcceptsIncomplete: true,
		ServiceID:         serviceCatalogID,
		PlanID:            planCatalogID,
		//TODO no OI for SM platform yet
		OriginatingIdentity: nil,
	}
}

func shouldStartOrphanMitigation(err error) bool {
	if httpError, ok := osbc.IsHTTPError(err); ok {
		statusCode := httpError.StatusCode
		is2XX := statusCode >= 200 && statusCode < 300
		is5XX := statusCode >= 500 && statusCode < 600
		return (is2XX && statusCode != http.StatusOK) ||
			statusCode == http.StatusRequestTimeout ||
			is5XX
	}

	if urlErr, ok := err.(*url.Error); ok && urlErr.Timeout() {
		return true
	}

	return false
}

func logUpdateInstanceRequest(request *osbc.UpdateInstanceRequest) string {
	servicePlanString := ""
	if request.PlanID != nil {
		servicePlanString = "planID: " + (*request.PlanID)
	}
	return fmt.Sprintf("context: %+v, instanceID: %s, %s, serviceID: %s, acceptsIncomplete: %t",
		request.Context, request.InstanceID, servicePlanString, request.ServiceID, request.AcceptsIncomplete)
}

func logProvisionRequest(request *osbc.ProvisionRequest) string {
	return fmt.Sprintf("context: %+v, instanceID: %s, planID: %s, serviceID: %s, acceptsIncomplete: %t",
		request.Context, request.InstanceID, request.PlanID, request.ServiceID, request.AcceptsIncomplete)
}

func logProvisionResponse(response *osbc.ProvisionResponse) string {
	return fmt.Sprintf("async: %t, dashboardURL: %s, operationKey: %s", response.Async, strPtrToStr(response.DashboardURL), opKeyPtrToStr(response.OperationKey))
}

func logUpdateInstanceResponse(response *osbc.UpdateInstanceResponse) string {
	return fmt.Sprintf("async: %t, dashboardURL: %s, operationKey: %s", response.Async, strPtrToStr(response.DashboardURL), opKeyPtrToStr(response.OperationKey))
}

func logDeprovisionRequest(request *osbc.DeprovisionRequest) string {
	return fmt.Sprintf("instanceID: %s, planID: %s, serviceID: %s, acceptsIncomplete: %t",
		request.InstanceID, request.PlanID, request.ServiceID, request.AcceptsIncomplete)
}

func logDeprovisionResponse(response *osbc.DeprovisionResponse) string {
	return fmt.Sprintf("async: %t, operationKey: %s", response.Async, opKeyPtrToStr(response.OperationKey))
}

func logPollInstanceRequest(request *osbc.LastOperationRequest) string {
	return fmt.Sprintf("instanceID: %s, planID: %s, serviceID: %s, operationKey: %s",
		request.InstanceID, strPtrToStr(request.PlanID), strPtrToStr(request.ServiceID), opKeyPtrToStr(request.OperationKey))
}

func logPollInstanceResponse(response *osbc.LastOperationResponse) string {
	return fmt.Sprintf("state: %s, description: %s", response.State, strPtrToStr(response.Description))
}

func strPtrToStr(sPtr *string) string {
	if sPtr == nil {
		return ""
	}

	return *sPtr
}

func opKeyPtrToStr(opKey *osbc.OperationKey) string {
	if opKey == nil {
		return ""
	}

	return string(*opKey)
}
