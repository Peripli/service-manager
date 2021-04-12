package interceptors

import "github.com/Peripli/service-manager/pkg/types"

func GenerateReferencePlanForShareableOfferings(catalogServices []*types.ServiceOffering, catalogPlansMap map[string][]*types.ServicePlan) {
	for _, service := range catalogServices {
		for _, servicePlan := range service.Plans {
			if servicePlan.IsShareablePlan() {
				referencePlan := generateReferencePlanObject(servicePlan.ServiceOfferingID)
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
