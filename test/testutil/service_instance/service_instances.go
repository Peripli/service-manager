package service_instance

import (
	"context"
	"fmt"
	"time"

	"github.com/Peripli/service-manager/storage"

	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/test/common"
	"github.com/gofrs/uuid"

	. "github.com/onsi/ginkgo"
)

func Prepare(ctx *common.TestContext, planID string) (string, common.Object) {
	var brokerID string
	if planID == "" {
		brokerID, planID = preparePlan(ctx)
	}

	instanceID, err := uuid.NewV4()
	if err != nil {
		Fail(fmt.Sprintf("failed to generate instance GUID: %s", err))
	}

	return brokerID, common.Object{
		"id":              instanceID.String(),
		"name":            "test-service-instance",
		"service_plan_id": planID,
		"dashboard_url":   "http://test-service.com/dashboard",
	}
}

func CreateInPlatform(ctx *common.TestContext, platformID string) *types.ServiceInstance {
	_, planID := preparePlan(ctx)

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

	return nil
}

func preparePlan(ctx *common.TestContext) (string, string) {
	cService := common.GenerateTestServiceWithPlans(common.GenerateFreeTestPlan())
	catalog := common.NewEmptySBCatalog()
	catalog.AddService(cService)
	brokerID, _, brokerServer := ctx.RegisterBrokerWithCatalog(catalog)
	ctx.Servers[common.BrokerServerPrefix+brokerID] = brokerServer

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
