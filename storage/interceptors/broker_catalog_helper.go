package interceptors

import (
	"context"
	"fmt"
	"github.com/Peripli/service-manager/constant"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
	"github.com/gofrs/uuid"
	"strconv"
	"time"
)

func convertReferencePlanObjectToOSBPlan(plan *types.ServicePlan) interface{} {
	return map[string]interface{}{
		"id":          plan.ID,
		"name":        plan.Name,
		"description": plan.Description,
		"bindable":    plan.Bindable,
	}
}

func generateReferencePlanObject(serviceOfferingId string) *types.ServicePlan {
	referencePlan := new(types.ServicePlan)

	UUID, err := uuid.NewV4()
	if err != nil {
		panic(fmt.Errorf("could not generate GUID for ServicePlan: %s", err))
	}

	referencePlan.ID = UUID.String()
	referencePlan.CatalogID = UUID.String()
	referencePlan.CatalogName = constant.ReferencePlanName
	referencePlan.Name = constant.ReferencePlanName
	referencePlan.Description = constant.ReferencePlanDescription
	referencePlan.ServiceOfferingID = serviceOfferingId
	referencePlan.Bindable = newTrue()
	referencePlan.Ready = true
	referencePlan.CreatedAt = time.Now()
	referencePlan.UpdatedAt = time.Now()

	return referencePlan
}

func isBindablePlan(service *types.ServiceOffering, plan *types.ServicePlan) bool {
	if plan.Bindable != nil {
		return *plan.Bindable
	}

	return service.Bindable
}

func servicePlanUsesReservedNameForReferencePlan(servicePlan *types.ServicePlan) bool {
	return servicePlan.Name == constant.ReferencePlanName || servicePlan.CatalogName == constant.ReferencePlanName
}

func planHasSharedInstances(storage storage.Repository, ctx context.Context, planID string) (bool, error) {
	byServicePlanID := query.ByField(query.EqualsOperator, "service_plan_id", planID)
	bySharedValue := query.ByField(query.EqualsOperator, "shared", strconv.FormatBool(true))
	listOfSharedInstances, err := storage.List(ctx, types.ServiceInstanceType, byServicePlanID, bySharedValue)
	if err != nil {
		return false, err
	}
	if listOfSharedInstances.Len() > 0 {
		return true, nil
	}
	return false, nil
}

func newTrue() *bool {
	b := true
	return &b
}
