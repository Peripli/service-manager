package interceptors

import "github.com/Peripli/service-manager/pkg/types"

func GenerateReferencePlanForShareableOfferings(catalogServices []*types.ServiceOffering) {
	for _, service := range catalogServices {
		for _, servicePlan := range service.Plans {
			if servicePlan.IsShareablePlan() {
				service.Plans = append(service.Plans, generateReferencePlanObject(servicePlan.ServiceOfferingID))
				break
			}
		}
	}
}
