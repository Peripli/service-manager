package interceptors

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Peripli/service-manager/pkg/instance_sharing"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	schemas2 "github.com/Peripli/service-manager/schemas"
	"github.com/Peripli/service-manager/storage"
	"github.com/gofrs/uuid"

	"strconv"
	"time"

)

const referencePlanPath = "schemas/reference_plan.json"
func convertReferencePlanObjectToOSBPlan(plan *types.ServicePlan) interface{} {
	return map[string]interface{}{
		"id":          plan.ID,
		"name":        plan.Name,
		"description": plan.Description,
		"bindable":    plan.Bindable,
		"metadata":    plan.Metadata,
		"schemas":     plan.Schemas,

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
	referencePlan.CatalogName = instance_sharing.ReferencePlanName
	referencePlan.Name = instance_sharing.ReferencePlanName
	referencePlan.Description = instance_sharing.ReferencePlanDescription
	referencePlan.ServiceOfferingID = serviceOfferingId
	referencePlan.Bindable = newTrue()
	referencePlan.Ready = true
	referencePlan.CreatedAt = time.Now()
	referencePlan.UpdatedAt = time.Now()
	schemas, err := schemas2.SchemasLoader("reference_plan.json")
	if err == nil {
		var planSchema map[string]json.RawMessage
		err = json.Unmarshal(schemas, &planSchema)
		if err == nil {
			referencePlan.Schemas = planSchema["schemas"]
			referencePlan.Metadata = planSchema["metadata"]
		}
	}
	return referencePlan
}


func isBindablePlan(service *types.ServiceOffering, plan *types.ServicePlan) bool {
	if plan.Bindable != nil {
		return *plan.Bindable
	}
	return service.Bindable
}

func servicePlanUsesReservedNameForReferencePlan(servicePlan *types.ServicePlan) bool {
	return servicePlan.Name == instance_sharing.ReferencePlanName || servicePlan.CatalogName == instance_sharing.ReferencePlanName
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
