package interceptors

import (
	"fmt"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/gofrs/uuid"
	"net/http"
)

func GenerateReferencePlanForShareableOfferings(catalogServices []*types.ServiceOffering, catalogPlansMap map[string][]*types.ServicePlan) error {
	for _, service := range catalogServices {
		for _, plan := range service.Plans {
			if plan.IsShareablePlan() {
				if !isPlanBindable(service, plan) {
					return &util.HTTPError{
						ErrorType:   "BadRequest",
						Description: fmt.Sprintf("plan %s must be bindable in order to support instance sharing.", plan.Name),
						StatusCode:  http.StatusBadRequest,
					}
				}
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
	return nil
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
	referencePlan.Bindable = newTrue()

	return referencePlan
}

func isPlanBindable(service *types.ServiceOffering, plan *types.ServicePlan) bool {
	if plan.Bindable != nil {
		return *plan.Bindable
	}

	return service.Bindable
}

func newTrue() *bool {
	b := true
	return &b
}
