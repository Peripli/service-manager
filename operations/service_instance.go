package operations

import (
	"context"
	"fmt"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/services"
	"github.com/Peripli/service-manager/storage"
	"time"
)

type ServiceInstanceActions struct {
	brokerService services.BrokerService
	repository    storage.Repository
}

func NewServiceInstanceActions(brokerService services.BrokerService, repository storage.Repository) InstanceActions {
	return ServiceInstanceActions{
		brokerService: brokerService,
		repository:    repository,
	}
}
func (si ServiceInstanceActions) RunActionByOperation(ctx context.Context, entity types.Object, operation types.Operation) (types.Object, error) {
	switch operation.Type {
	case types.CREATE:
		return si.createHandler(ctx, entity, operation)
	}
	return nil, nil
}

func (si ServiceInstanceActions) createHandler(ctx context.Context, entity types.Object, operation types.Operation) (types.Object, error) {
	instance := entity.(*types.ServiceInstance)

	if operation.Reschedule {
		return si.pollServiceInstance(ctx, *instance, operation)
	} else {
		resAsInstance, operation, err := si.createServiceInstance(ctx, instance, operation)

		if err != nil {
			return nil, err
		}

		instanceRes := resAsInstance.(*types.ServiceInstance)
		if operation.Reschedule && ctx.Value("async_mode") != "" {
			return si.pollServiceInstance(ctx, *instanceRes, *operation)
		} else {
			return instanceRes, err
		}
	}
}

func (si ServiceInstanceActions) deleteServiceInstance(ctx context.Context, obj types.Object, operation types.Operation) (types.Object, error) {
	return nil, nil
}

func (si ServiceInstanceActions) pollServiceInstance(ctx context.Context, serviceInstance types.ServiceInstance, operation types.Operation) (types.Object, error) {
	hasCompleted, err := si.brokerService.PollServiceInstance(serviceInstance, ctx, operation.ExternalID, true, operation.RescheduleTimestamp, operation.Type,true);
	if err != nil {
		return nil, err
	}

	if !hasCompleted {
		return nil, nil
	}

	operation.Reschedule = false
	operation.RescheduleTimestamp = time.Time{}
	if _, err := si.repository.Update(ctx, &operation, types.LabelChanges{}); err != nil {
		return nil, fmt.Errorf("failed to update operation with id %s to mark that next execution should be a reschedule: %s", operation.ID, err)
	}

	return &serviceInstance, nil

}

func (si ServiceInstanceActions) createServiceInstance(ctx context.Context, obj types.Object, operation types.Operation) (types.Object, *types.Operation, error) {
	instance := obj.(*types.ServiceInstance)
	instance.Usable = false

	provisionResponse, err := si.brokerService.ProvisionServiceInstance(*instance, ctx)

	if err != nil {
		return nil, nil, err
	}

	instance.DashboardURL = provisionResponse.DashboardURL

	if provisionResponse.Async {
		operation.Reschedule = true
		//set the operation as async, based on the broker response.
		operation.IsAsync = true
		if operation.RescheduleTimestamp.IsZero() {
			operation.RescheduleTimestamp = time.Now()
		}

		operation.ExternalID = provisionResponse.OperationKey

		if _, err := si.repository.Update(ctx, &operation, types.LabelChanges{}); err != nil {
			return nil, nil, fmt.Errorf("failed to update operation with id %s to mark that next execution should be a reschedule: %s", instance.ID, err)
		}
	} else {

	}

	if _, err := si.repository.Create(ctx, instance); err != nil {
		return nil, nil, fmt.Errorf("failed to create servie instance")
	}

	return instance, &operation, nil
}
