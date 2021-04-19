package interceptors

import (
	"context"
	"fmt"
	"github.com/Peripli/service-manager/constant"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/storage"
	"github.com/gofrs/uuid"
)

func VerifyCatalogDoesNotUseReferencePlan(catalogServices []*types.ServiceOffering) error {
	for _, service := range catalogServices {
		for _, plan := range service.Plans {
			if plan.Name == constant.ReferencePlanName || plan.CatalogName == constant.ReferencePlanName {
				return util.HandleInstanceSharingError(util.ErrCatalogUsesReservedPlanName, constant.ReferencePlanName)
			}
		}
	}
	return nil
}
func GenerateReferencePlanForShareableOfferings(ctx context.Context, repository storage.Repository, catalogServices []*types.ServiceOffering, catalogPlansMap map[string][]*types.ServicePlan) error {
	for _, service := range catalogServices {
		existingReferencePlanID, err := getExistingReferencePlanID(ctx, repository, service.ID)
		if err != nil {
			return err
		}
		for _, plan := range service.Plans {
			if plan.IsShareablePlan() {
				if !isPlanBindable(service, plan) {
					return util.HandleInstanceSharingError(util.ErrPlanMustBeBindable, plan.Name)
				}
				referencePlan := generateReferencePlanObject(plan.ServiceOfferingID, existingReferencePlanID)
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

func getExistingReferencePlanID(ctx context.Context, repository storage.Repository, serviceID string) (string, error) {
	byID := query.ByField(query.EqualsOperator, "id", serviceID)
	var plan types.Object
	var err error
	if plan, err = repository.Get(ctx, types.ServicePlanType, byID); err != nil {
		if err != util.ErrNotFoundInStorage {
			return "", util.HandleStorageError(err, string(types.ServicePlanType))
		}
	}
	if plan == nil {
		return "", nil
	}
	return plan.GetID(), nil
}

func generateReferencePlanObject(serviceOfferingId string, existingPlanID string) *types.ServicePlan {
	referencePlan := new(types.ServicePlan)
	var uuidString string
	identity := constant.ReferencePlanName

	referencePlan.CatalogName = identity
	referencePlan.ServiceOfferingID = serviceOfferingId
	referencePlan.Name = identity
	referencePlan.Description = "Reference plan"
	if existingPlanID == "" {
		UUID, err := uuid.NewV4()
		if err != nil {
			panic(fmt.Errorf("could not generate GUID for ServicePlan: %s", err))
		}
		uuidString = UUID.String()
	} else {
		uuidString = existingPlanID
	}
	referencePlan.ID = uuidString
	referencePlan.CatalogID = uuidString
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
