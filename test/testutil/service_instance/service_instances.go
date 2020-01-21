package service_instance

import (
	"context"
	"fmt"

	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/test/common"
	"github.com/gofrs/uuid"

	. "github.com/onsi/ginkgo"
)

func Prepare(ctx *common.TestContext, planID string) (string, common.Object) {
	var brokerID string
	var brokerServer *common.BrokerServer
	if planID == "" {
		cService := common.GenerateTestServiceWithPlans(common.GenerateFreeTestPlan())
		catalog := common.NewEmptySBCatalog()
		catalog.AddService(cService)
		brokerID, _, brokerServer = ctx.RegisterBrokerWithCatalog(catalog)
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
		planID = obj.GetID()
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
