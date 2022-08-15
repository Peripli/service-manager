package common

import (
	"context"
	"fmt"
	"time"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/storage"

	"github.com/gofrs/uuid"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/query"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types"
)

func DeleteBinding(ctx *TestContext, bindingID, instanceID string) error {
	instanceObject, err := ctx.SMRepository.Get(context.TODO(), types.ServiceInstanceType, query.ByField(query.EqualsOperator, "id", instanceID))
	if err != nil {
		return err
	}
	instance := instanceObject.(*types.ServiceInstance)

	planObject, err := ctx.SMRepository.Get(context.TODO(), types.ServicePlanType, query.ByField(query.EqualsOperator, "id", instance.ServicePlanID))
	if err != nil {
		return err
	}
	plan := planObject.(*types.ServicePlan)

	serviceObject, err := ctx.SMRepository.Get(context.TODO(), types.ServiceOfferingType, query.ByField(query.EqualsOperator, "id", plan.ServiceOfferingID))
	if err != nil {
		return err
	}
	service := serviceObject.(*types.ServiceOffering)

	brokerObject, err := ctx.SMRepository.Get(context.TODO(), types.ServiceBrokerType, query.ByField(query.EqualsOperator, "id", service.BrokerID))
	if err != nil {
		return err
	}
	broker := brokerObject.(*types.ServiceBroker)

	if _, foundServer := ctx.Servers[BrokerServerPrefix+broker.ID]; !foundServer {
		brokerServer := NewBrokerServerWithCatalog(SBCatalog(broker.Catalog))
		broker.BrokerURL = brokerServer.URL()
		UUID, err := uuid.NewV4()
		if err != nil {
			return fmt.Errorf("could not generate GUID: %s", err)
		}
		if _, err := ctx.SMScheduler.ScheduleSyncStorageAction(context.TODO(), &types.Operation{
			Base: types.Base{
				ID:        UUID.String(),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				Ready:     true,
			},
			Type:          types.UPDATE,
			State:         types.IN_PROGRESS,
			ResourceID:    broker.ID,
			ResourceType:  types.ServiceBrokerType,
			CorrelationID: "-",
		}, func(ctx context.Context, repository storage.Repository) (object types.Object, e error) {
			return repository.Update(ctx, broker, types.LabelChanges{})
		}); err != nil {
			return err
		}
	}

	UUID, err := uuid.NewV4()
	if err != nil {
		return fmt.Errorf("could not generate GUID: %s", err)
	}
	if _, err := ctx.SMScheduler.ScheduleSyncStorageAction(context.TODO(), &types.Operation{
		Base: types.Base{
			ID:        UUID.String(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Ready:     true,
		},
		Type:          types.DELETE,
		State:         types.IN_PROGRESS,
		ResourceID:    instance.ID,
		ResourceType:  types.ServiceBindingType,
		CorrelationID: "-",
	}, func(ctx context.Context, repository storage.Repository) (types.Object, error) {
		byID := query.ByField(query.EqualsOperator, "id", bindingID)
		if err := repository.Delete(ctx, types.ServiceBindingType, byID); err != nil {
			return nil, err
		}
		return nil, nil
	}); err != nil {
		return err
	}

	return nil
}
