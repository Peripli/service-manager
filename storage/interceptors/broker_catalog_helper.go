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
func GenerateReferencePlanForShareableOfferings(catalogServices []*types.ServiceOffering, catalogPlansMap map[string][]*types.ServicePlan, existingReferencePlan *types.ServicePlan) error {
	for _, service := range catalogServices {
		for _, plan := range service.Plans {
			if plan.IsShareablePlan() {
				if !isPlanBindable(service, plan) {
					return util.HandleInstanceSharingError(util.ErrPlanMustBeBindable, plan.Name)
				}
				var referencePlan *types.ServicePlan
				if existingReferencePlan != nil {
					referencePlan = existingReferencePlan
				} else {
					referencePlan = generateReferencePlanObject(plan.ServiceOfferingID)
				}
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

func getExistingReferencePlan(ctx context.Context, repository storage.Repository, catalogServices []*types.ServiceOffering) (*types.ServicePlan, error) {
	for _, service := range catalogServices {
		byID := query.ByField(query.EqualsOperator, "service_offering_id", service.ID)
		byName := query.ByField(query.EqualsOperator, "name", constant.ReferencePlanName)
		var planObject types.Object
		var err error
		if planObject, err = repository.Get(ctx, types.ServicePlanType, byID, byName); err != nil {
			if err != util.ErrNotFoundInStorage {
				return nil, util.HandleStorageError(err, string(types.ServicePlanType))
			}
		}
		if planObject == nil {
			return nil, nil
		}
		plan := planObject.(*types.ServicePlan)
		return plan, nil
	}
	return nil, nil
}

func generateReferencePlanObject(serviceOfferingId string) *types.ServicePlan {
	referencePlan := new(types.ServicePlan)
	identity := constant.ReferencePlanName

	UUID, err := uuid.NewV4()
	if err != nil {
		panic(fmt.Errorf("could not generate GUID for ServicePlan: %s", err))
	}

	referencePlan.CatalogName = identity
	referencePlan.ServiceOfferingID = serviceOfferingId
	referencePlan.Name = identity
	referencePlan.Description = "Reference plan"
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
