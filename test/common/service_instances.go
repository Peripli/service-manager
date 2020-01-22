package common

import (
	"context"
	"fmt"
	"time"

	"github.com/Peripli/service-manager/storage"

	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/gofrs/uuid"

	. "github.com/onsi/ginkgo"
)

func Prepare(ctx *TestContext, planID string) (string, Object) {
	var brokerID string
	if planID == "" {
		brokerID, planID = preparePlan(ctx)
	}

	instanceID, err := uuid.NewV4()
	if err != nil {
		Fail(fmt.Sprintf("failed to generate instance GUID: %s", err))
	}

	return brokerID, Object{
		"id":              instanceID.String(),
		"name":            "test-service-instance",
		"service_plan_id": planID,
		"dashboard_url":   "http://test-service.com/dashboard",
	}
}

func CreateInPlatform(ctx *TestContext, platformID string) (string, *types.ServiceInstance) {
	brokerID, planID := preparePlan(ctx)
	return brokerID, CreateInPlatformForPlan(ctx, platformID, planID)
}

func CreateInPlatformForPlan(ctx *TestContext, platformID, planID string) *types.ServiceInstance {
	operationID, err := uuid.NewV4()
	if err != nil {
		Fail(fmt.Sprintf("failed to generate instance GUID: %s", err))
	}
	instanceID, err := uuid.NewV4()
	if err != nil {
		Fail(fmt.Sprintf("failed to generate instance GUID: %s", err))
	}
	operation := &types.Operation{
		Base: types.Base{
			ID:        operationID.String(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Type:         types.CREATE,
		State:        types.IN_PROGRESS,
		ResourceID:   instanceID.String(),
		ResourceType: types.ServiceInstanceType,
	}

	instance := &types.ServiceInstance{
		Base: types.Base{
			ID:        instanceID.String(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Ready:     true,
		},
		Name:          "test-service-instance",
		ServicePlanID: planID,
		PlatformID:    platformID,
		DashboardURL:  "http://testurl.com",
	}

	if _, err := ctx.SMScheduler.ScheduleSyncStorageAction(context.TODO(), operation, func(ctx context.Context, repository storage.Repository) (object types.Object, e error) {
		return repository.Create(ctx, instance)
	}); err != nil {
		Fail(fmt.Sprintf("failed to create instance with name %s", instance.Name))
	}

	return instance
}

func Delete(ctx *TestContext, instance *types.ServiceInstance) error {
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
			return repository.Update(ctx, broker, query.LabelChanges{})
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
		ResourceType:  types.ServiceInstanceType,
		CorrelationID: "-",
	}, func(ctx context.Context, repository storage.Repository) (types.Object, error) {
		byID := query.ByField(query.EqualsOperator, "id", instance.ID)
		if err := repository.Delete(ctx, types.ServiceInstanceType, byID); err != nil {
			return nil, err
		}
		return nil, nil
	}); err != nil {
		return err
	}

	return nil
}

func preparePlan(ctx *TestContext) (string, string) {
	cService := GenerateTestServiceWithPlans(GenerateFreeTestPlan())
	catalog := NewEmptySBCatalog()
	catalog.AddService(cService)
	brokerID, _, brokerServer := ctx.RegisterBrokerWithCatalog(catalog)
	ctx.Servers[BrokerServerPrefix+brokerID] = brokerServer

	byBrokerID := query.ByField(query.EqualsOperator, "broker_id", brokerID)
	obj, err := ctx.SMRepository.Get(context.Background(), types.ServiceOfferingType, byBrokerID)
	if err != nil {
		Fail(fmt.Sprintf("unable to fetch service offering: %s", err))
	}

	byServiceOfferingID := query.ByField(query.EqualsOperator, "service_offering_id", obj.GetID())
	obj, err = ctx.SMRepository.Get(context.Background(), types.ServicePlanType, byServiceOfferingID)
	if err != nil {
		Fail(fmt.Sprintf("unable to service plan: %s", err))
	}

	return brokerID, obj.GetID()
}
