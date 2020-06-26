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
	"github.com/Peripli/service-manager/services"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/Peripli/service-manager/operations/opcontext"
	"github.com/Peripli/service-manager/pkg/util"

	"github.com/Peripli/service-manager/pkg/log"

	"github.com/Peripli/service-manager/pkg/query"

	"github.com/Peripli/service-manager/pkg/types"
	osbc "github.com/kubernetes-sigs/go-open-service-broker-client/v2"

	"github.com/Peripli/service-manager/storage"
)

const ServiceInstanceCreateInterceptorProviderName = "ServiceInstanceCreateInterceptorProvider"

type BaseSMAAPInterceptorProvider struct {
	OSBClientCreateFunc osbc.CreateFunc
	Repository          *storage.InterceptableTransactionalRepository
	TenantKey           string
	PollingInterval     time.Duration
	BrokerService       services.BrokerService
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
		brokerService:       p.BrokerService,
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
	brokerService       services.BrokerService
}

func (i *ServiceInstanceInterceptor) AroundTxCreate(f storage.InterceptCreateAroundTxFunc) storage.InterceptCreateAroundTxFunc {
	return func(ctx context.Context, obj types.Object) (types.Object, error) {
		instance := obj.(*types.ServiceInstance)
		instance.Usable = false
		return f(ctx, obj)
	}
}

func (i *ServiceInstanceInterceptor) AroundTxUpdate(f storage.InterceptUpdateAroundTxFunc) storage.InterceptUpdateAroundTxFunc {
	return func(ctx context.Context, updatedObj types.Object, labelChanges ...*types.LabelChange) (object types.Object, err error) {
		return f(ctx, updatedObj, labelChanges...)
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

		if instances.Len() != 0 {
			instance := instances.ItemAt(0).(*types.ServiceInstance)
			operation, found := opcontext.Get(ctx)
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

	return nil
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
