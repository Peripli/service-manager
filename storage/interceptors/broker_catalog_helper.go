package interceptors

import (
	"fmt"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/gofrs/uuid"
)

func GenerateReferencePlanForShareableOfferings(catalogServices []*types.ServiceOffering, catalogPlansMap map[string][]*types.ServicePlan) {
	for _, service := range catalogServices {
		for _, plan := range service.Plans {
			if plan.IsShareablePlan() {
				referencePlan := generateReferencePlanObject(plan.ServiceOfferingID)
				service.Plans = append(service.Plans, referencePlan)
				// When not on "update catalog" flow:
				if catalogPlansMap != nil {
					catalogPlansMap[service.CatalogID] = append(catalogPlansMap[service.CatalogID], referencePlan)
				}
				break
			}
		}
	}
}

func generateReferencePlanObject(serviceOfferingId string) *types.ServicePlan {
	referencePlan := new(types.ServicePlan)
	identity := "reference-plan"
	referencePlan.CatalogName = identity
	referencePlan.ServiceOfferingID = serviceOfferingId
	referencePlan.Name = identity
	referencePlan.Description = "Reference plan"
	UUID, err := uuid.NewV4()
	if err != nil {
		panic(fmt.Errorf("could not generate GUID for ServicePlan: %s", err))
	}
	referencePlan.ID = UUID.String()
	referencePlan.CatalogID = UUID.String()

	return referencePlan
}
