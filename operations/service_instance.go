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
	eventBus      *SyncEventBus
}

func NewServiceInstanceActions(brokerService services.BrokerService, repository storage.Repository, eventBus *SyncEventBus) InstanceActions {
	return ServiceInstanceActions{
		brokerService: brokerService,
		repository:    repository,
		eventBus:      eventBus,
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

	isDeleteRescheduleRequired := operation.InOrphanMitigationState() &&
		time.Now().UTC().Before(operation.DeletionScheduled.Add(time.Hour*12)) &&
		operation.State != types.SUCCEEDED

	if isDeleteRescheduleRequired {
		return si.deleteServiceInstance(ctx, *instance, operation)
	}

	if operation.Reschedule {
		return si.pollServiceInstance(ctx, *instance, operation)
	} else {
		resAsInstance, _, err := si.createServiceInstance(ctx, instance, operation)
		if err != nil {
			return nil, err
		}
		return resAsInstance, err
	}
}

func (si ServiceInstanceActions) deleteServiceInstance(ctx context.Context, serviceInstance types.ServiceInstance, operation types.Operation) (types.Object, error) {
	_, err := si.brokerService.DeleteServiceInstance(serviceInstance, ctx);
	if err != nil {
		return nil, err
	}
	operation.Reschedule = false
	operation.RescheduleTimestamp = time.Time{}
	if _, err := si.repository.Update(ctx, &operation, types.LabelChanges{}); err != nil {
		return nil, fmt.Errorf("failed to update operation with id %s to mark that next execution should be a reschedule: %s", operation.ID, err)
	}
	return &serviceInstance, nil
}

func (si ServiceInstanceActions) pollServiceInstance(ctx context.Context, serviceInstance types.ServiceInstance, operation types.Operation) (types.Object, error) {
	hasCompleted, err := si.brokerService.PollServiceInstance(serviceInstance, ctx, operation.ExternalID, true, operation.RescheduleTimestamp, operation.Type, true);
	if err != nil {
		return nil, err
	}

	if !hasCompleted {
		return &serviceInstance, nil
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
	provisionResponse := si.brokerService.ProvisionServiceInstance(*instance, ctx)

	if provisionResponse.Error != nil {

		if provisionResponse.OrphanMitigation {
			operation.DeletionScheduled = time.Now().UTC()
			operation.RescheduleTimestamp = time.Time{}
			if _, err := si.repository.Update(ctx, &operation, types.LabelChanges{}); err != nil {
				return nil, nil, fmt.Errorf("failed to update operation with id %s to mark that next execution should be a reschedule: %s", instance.ID, err)
			}
		}

		if _, err := si.repository.Create(ctx, instance); err != nil {
			return nil, nil, err
		}

		return nil, nil, provisionResponse.Error
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
	}


	if _, err := si.repository.Create(ctx, instance); err != nil {
		return nil, nil, err
	}

	return instance, &operation, nil
}
