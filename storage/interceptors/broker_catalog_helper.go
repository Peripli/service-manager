package interceptors

import (
	"fmt"
	"github.com/Peripli/service-manager/constant"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/gofrs/uuid"
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

func newTrue() *bool {
	b := true
	return &b
}
