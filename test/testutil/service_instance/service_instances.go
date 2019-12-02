package service_instance

import (
	"context"
	"fmt"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/test/common"
	"github.com/gofrs/uuid"
	"time"

	. "github.com/onsi/ginkgo"
)

func Prepare(ctx *common.TestContext, platformID, planID string, OSBContext string) (string, *types.ServiceInstance) {
	var brokerID string
	if planID == "" {
		cService := common.GenerateTestServiceWithPlans(common.GenerateFreeTestPlan())
		catalog := common.NewEmptySBCatalog()
		catalog.AddService(cService)
		brokerID, _, _ = ctx.RegisterBrokerWithCatalog(catalog)

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
		planID = obj.GetID()
	}

	instanceID, err := uuid.NewV4()
	if err != nil {
		Fail(fmt.Sprintf("failed to generate instance GUID: %s", err))
	}

	return brokerID, &types.ServiceInstance{
		Base: types.Base{
			ID:        instanceID.String(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Name:          "test-service-instance",
		ServicePlanID: planID,
		PlatformID:    platformID,
		Context:       []byte(OSBContext),
		Ready:         true,
		Usable:        true,
	}
}
