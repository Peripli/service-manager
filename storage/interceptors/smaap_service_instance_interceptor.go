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
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/Peripli/service-manager/operations"

	"github.com/Peripli/service-manager/pkg/util"

	"github.com/Peripli/service-manager/pkg/log"

	"github.com/Peripli/service-manager/pkg/query"

	"github.com/Peripli/service-manager/pkg/types"
	osbc "github.com/kubernetes-sigs/go-open-service-broker-client/v2"

	"github.com/Peripli/service-manager/storage"
)

const ServiceInstanceCreateInterceptorProviderName = "ServiceInstanceCreateInterceptorProvider"

// ServiceInstanceCreateInterceptorProvider provides an interceptor that notifies the actual broker about instance creation
type ServiceInstanceCreateInterceptorProvider struct {
	OSBClientCreateFunc osbc.CreateFunc
	Repository          storage.TransactionalRepository
	TenantKey           string
	PollingInterval     time.Duration
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
	OSBClientCreateFunc osbc.CreateFunc
	Repository          storage.TransactionalRepository
	TenantKey           string
	PollingInterval     time.Duration
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
	OSBClientCreateFunc osbc.CreateFunc
	Repository          storage.TransactionalRepository
	TenantKey           string
	PollingInterval     time.Duration
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
	repository          storage.TransactionalRepository
	tenantKey           string
	pollingInterval     time.Duration
}

func (i *ServiceInstanceInterceptor) AroundTxCreate(f storage.InterceptCreateAroundTxFunc) storage.InterceptCreateAroundTxFunc {
	return func(ctx context.Context, obj types.Object) (types.Object, error) {
		instance := obj.(*types.ServiceInstance)
		if instance.PlatformID != types.SMPlatform {
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

		var provisionResponse *osbc.ProvisionResponse
		if !operation.Reschedule {
			provisionRequest := i.prepareProvisionRequest(instance, service.CatalogID, plan.CatalogID)

			log.C(ctx).Infof("Sending provision request %+v to broker with name %s", provisionRequest, broker.Name)
			provisionResponse, err = osbClient.ProvisionInstance(provisionRequest)
			if err != nil {
				brokerError := &util.HTTPError{
					ErrorType:   "BrokerError",
					Description: fmt.Sprintf("Failed provisioning request %+v: %s", provisionRequest, err),
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

			if provisionResponse.DashboardURL != nil {
				dashboardURL := *provisionResponse.DashboardURL
				instance.DashboardURL = dashboardURL
			}

			if provisionResponse.Async {
				log.C(ctx).Infof("Successful asynchronous provisioning request %+v to broker %s returned response %+v",
					provisionRequest, broker.Name, provisionResponse)
				operation.Reschedule = true
				if provisionResponse.OperationKey != nil {
					operation.ExternalID = string(*provisionResponse.OperationKey)
				}

				if _, err := i.repository.Update(ctx, operation, query.LabelChanges{}); err != nil {
					return nil, fmt.Errorf("failed to update operation with id %s to mark that next execution should be a reschedule", instance.ID)
				}
			} else {
				log.C(ctx).Infof("Successful synchronous provisioning %+v to broker %s returned response %+v",
					provisionRequest, broker.Name, provisionResponse)

			}
		}

		object, err := f(ctx, obj)
		if err != nil {
			return nil, err
		}
		instance = object.(*types.ServiceInstance)

		if operation.Reschedule {
			if err := i.pollServiceInstance(ctx, osbClient, instance, operation, broker.ID, service.CatalogID, plan.CatalogID, operation.ExternalID, true); err != nil {
				return nil, err
			}
		}

		return instance, nil
	}
}

// TODO Update of instances in SM is not yet implemented
func (i *ServiceInstanceInterceptor) AroundTxUpdate(h storage.InterceptUpdateAroundTxFunc) storage.InterceptUpdateAroundTxFunc {
	return h
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

		if instances.Len() != 0 {
			instance := instances.ItemAt(0).(*types.ServiceInstance)
			if instance.PlatformID != types.SMPlatform {
				return f(ctx, deletionCriteria...)
			}

			operation, found := operations.GetFromContext(ctx)
			if !found {
				return fmt.Errorf("operation missing from context")
			}

			if err := i.deleteSingleInstance(ctx, instance, operation); err != nil {
				return err
			}
		}

		if err := f(ctx, deletionCriteria...); err != nil {
			return err
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

	osbClient, broker, service, plan, err := prepare(ctx, i.repository, i.osbClientCreateFunc, instance)
	if err != nil {
		return err
	}

	if !operation.DeletionScheduled.IsZero() {
		log.C(ctx).Infof("Orphan mitigation in progress for instance with id %s and name %s triggered due to failure in operation %s", instance.ID, instance.Name, operation.Type)
	}

	var deprovisionResponse *osbc.DeprovisionResponse
	if !operation.Reschedule {
		deprovisionRequest := prepareDeprovisionRequest(instance, service.CatalogID, plan.CatalogID)

		log.C(ctx).Infof("Sending deprovision request %+v to broker with name %s", deprovisionRequest, broker.Name)
		deprovisionResponse, err = osbClient.DeprovisionInstance(deprovisionRequest)
		if err != nil {
			if osbc.IsGoneError(err) {
				log.C(ctx).Infof("Synchronous deprovisioning %+v to broker %s returned 410 GONE and is considered success",
					deprovisionRequest, broker.Name)
				return nil
			}
			brokerError := &util.HTTPError{
				ErrorType:   "BrokerError",
				Description: fmt.Sprintf("Failed deprovisioning request %+v: %s", deprovisionRequest, err),
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

		if deprovisionResponse.Async {
			log.C(ctx).Infof("Successful asynchronous deprovisioning request %+v to broker %s returned response %+v",
				deprovisionRequest, broker.Name, deprovisionResponse)
			operation.Reschedule = true

			if deprovisionResponse.OperationKey != nil {
				operation.ExternalID = string(*deprovisionResponse.OperationKey)
			}
			if _, err := i.repository.Update(ctx, operation, query.LabelChanges{}); err != nil {
				return fmt.Errorf("failed to update operation with id %s to mark that rescheduling is possible", operation.ID)
			}
		} else {
			log.C(ctx).Infof("Successful synchronous deprovisioning %+v to broker %s returned response %+v",
				deprovisionRequest, broker.Name, deprovisionResponse)
		}
	}

	if operation.Reschedule {
		if err := i.pollServiceInstance(ctx, osbClient, instance, operation, broker.ID, service.CatalogID, plan.CatalogID, operation.ExternalID, true); err != nil {
			return err
		}
	}

	return nil
}

func (i *ServiceInstanceInterceptor) pollServiceInstance(ctx context.Context, osbClient osbc.Client, instance *types.ServiceInstance, operation *types.Operation, brokerID, serviceCatalogID, planCatalogID, operationKey string, enableOrphanMitigation bool) error {
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

	ticker := time.NewTicker(i.pollingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.C(ctx).Errorf("Terminating poll last operation for instance with id %s and name %s due to context done event", instance.ID, instance.Name)
			//operation should be kept in progress in this case
			return nil
		case <-ticker.C:
			log.C(ctx).Infof("Sending poll last operation request %+v for instance with id %s and name %s", pollingRequest, instance.ID, instance.Name)
			pollingResponse, err := osbClient.PollLastOperation(pollingRequest)
			if err != nil {
				if osbc.IsGoneError(err) && operation.Type == types.DELETE {
					log.C(ctx).Infof("Successfully finished polling operation for instance with id %s and name %s", instance.ID, instance.Name)

					operation.Reschedule = false
					if _, err := i.repository.Update(ctx, operation, query.LabelChanges{}); err != nil {
						return fmt.Errorf("failed to update operation with id %s to mark that next execution should be a reschedule", operation.ID)
					}
					return nil
				}

				return &util.HTTPError{
					ErrorType: "BrokerError",
					Description: fmt.Sprintf("Failed poll last operation request %+v for instance with id %s and name %s: %s",
						pollingRequest, instance.ID, instance.Name, err),
					StatusCode: http.StatusBadGateway,
				}
			}
			switch pollingResponse.State {
			case osbc.StateInProgress:
				log.C(ctx).Infof("Polling of instance still in progress. Rescheduling polling last operation request %+v to for provisioning of instance with id %s and name %s...", pollingRequest, instance.ID, instance.Name)

			case osbc.StateSucceeded:
				log.C(ctx).Infof("Successfully finished polling operation for instance with id %s and name %s", instance.ID, instance.Name)

				operation.Reschedule = false
				if _, err := i.repository.Update(ctx, operation, query.LabelChanges{}); err != nil {
					return fmt.Errorf("failed to update operation with id %s to mark that next execution should be a reschedule", operation.ID)
				}

				return nil
			case osbc.StateFailed:
				log.C(ctx).Infof("Failed polling operation for instance with id %s and name %s", instance.ID, instance.Name)
				operation.Reschedule = false
				if enableOrphanMitigation {
					operation.DeletionScheduled = time.Now()
				}
				if _, err := i.repository.Update(ctx, operation, query.LabelChanges{}); err != nil {
					return fmt.Errorf("failed to update operation with id %s after failed of last operation for instance with id %s", operation.ID, instance.ID)
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

func prepare(ctx context.Context, repository storage.Repository, osbClientFunc osbc.CreateFunc, instance *types.ServiceInstance) (osbc.Client, *types.ServiceBroker, *types.ServiceOffering, *types.ServicePlan, error) {
	planObject, err := repository.Get(ctx, types.ServicePlanType, query.ByField(query.EqualsOperator, "id", instance.ServicePlanID))
	if err != nil {
		return nil, nil, nil, nil, util.HandleStorageError(err, types.ServicePlanType.String())
	}
	plan := planObject.(*types.ServicePlan)

	serviceObject, err := repository.Get(ctx, types.ServiceOfferingType, query.ByField(query.EqualsOperator, "id", plan.ServiceOfferingID))
	if err != nil {
		return nil, nil, nil, nil, util.HandleStorageError(err, types.ServicePlanType.String())
	}
	service := serviceObject.(*types.ServiceOffering)

	brokerObject, err := repository.Get(ctx, types.ServiceBrokerType, query.ByField(query.EqualsOperator, "id", service.BrokerID))
	if err != nil {
		return nil, nil, nil, nil, util.HandleStorageError(err, types.ServicePlanType.String())
	}
	broker := brokerObject.(*types.ServiceBroker)
	osbClient, err := osbClientFunc(&osbc.ClientConfiguration{
		Name:                broker.Name + " broker client",
		EnableAlphaFeatures: true,
		URL:                 broker.BrokerURL,
		APIVersion:          osbc.LatestAPIVersion(),
		AuthConfig: &osbc.AuthConfig{
			BasicAuthConfig: &osbc.BasicAuthConfig{
				Username: broker.Credentials.Basic.Username,
				Password: broker.Credentials.Basic.Password,
			},
		},
	})
	if err != nil {
		return nil, nil, nil, nil, err
	}

	return osbClient, broker, service, plan, nil
}

func (i *ServiceInstanceInterceptor) prepareProvisionRequest(instance *types.ServiceInstance, serviceCatalogID, planCatalogID string) *osbc.ProvisionRequest {
	provisionRequest := &osbc.ProvisionRequest{
		InstanceID:        instance.GetID(),
		AcceptsIncomplete: true,
		ServiceID:         serviceCatalogID,
		PlanID:            planCatalogID,
		OrganizationGUID:  "-",
		SpaceGUID:         "-",
		Parameters:        instance.Parameters,
		Context: map[string]interface{}{
			"platform":      types.SMPlatform,
			"instance_name": instance.Name,
		},
		//TODO no OI for SM platform yet
		OriginatingIdentity: nil,
	}
	if len(i.tenantKey) != 0 {
		if tenantValue, ok := instance.GetLabels()[i.tenantKey]; ok {
			provisionRequest.Context[i.tenantKey] = tenantValue[0]
		}
	}

	return provisionRequest
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
