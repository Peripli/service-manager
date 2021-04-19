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
func GenerateReferencePlanForShareableOfferings(ctx context.Context, repository storage.Repository, catalogServices []*types.ServiceOffering, catalogPlansMap map[string][]*types.ServicePlan) (bool, error) {
	var existingReferencePlan *types.ServicePlan
	generateExecutedForCatalog := false
	for _, service := range catalogServices {
		serviceOffering, err := getServiceOfferingByCatalogID(ctx, repository, service.CatalogID)
		if err != nil {
			return false, err
		}
		if serviceOffering != nil {
			existingReferencePlan, err = getExistingReferencePlan(ctx, repository, serviceOffering.ID)
			if err != nil {
				return false, err
			}
		}
		generateExecutedForServiceOffering := false
		for _, plan := range service.Plans {
			if plan.IsShareablePlan() && isBindablePlan(service, plan) && !generateExecutedForServiceOffering {
				generateExecutedForServiceOffering = executeGenerationOnValidInstanceSharingPlan(existingReferencePlan, generateExecutedForServiceOffering, plan, service, catalogPlansMap)
				generateExecutedForCatalog = true
			} else if plan.IsShareablePlan() && !isBindablePlan(service, plan) {
				return false, util.HandleInstanceSharingError(util.ErrPlanMustBeBindable, plan.Name)
			}
		}
	}
	return generateExecutedForCatalog, nil
}

func executeGenerationOnValidInstanceSharingPlan(existingReferencePlan *types.ServicePlan, generateExecutedForServiceOffering bool, plan *types.ServicePlan, service *types.ServiceOffering, catalogPlansMap map[string][]*types.ServicePlan) bool {
	var referencePlan *types.ServicePlan
	if existingReferencePlan != nil {
		referencePlan = existingReferencePlan
	} else if !generateExecutedForServiceOffering {
		referencePlan = generateReferencePlanObject(plan.ServiceOfferingID)
		generateExecutedForServiceOffering = true
	}
	service.Plans = append(service.Plans, referencePlan)
	// When not on "update catalog" flow:
	if catalogPlansMap != nil {
		catalogPlansMap[service.CatalogID] = append(catalogPlansMap[service.CatalogID], referencePlan)
	}
	return generateExecutedForServiceOffering
}

func getExistingReferencePlan(ctx context.Context, repository storage.Repository, serviceID string) (*types.ServicePlan, error) {
	byID := query.ByField(query.EqualsOperator, "service_offering_id", serviceID)
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

func getServiceOfferingByCatalogID(ctx context.Context, repository storage.Repository, serviceID string) (*types.ServiceOffering, error) {
	byID := query.ByField(query.EqualsOperator, "catalog_id", serviceID)
	var serviceObject types.Object
	var err error
	if serviceObject, err = repository.Get(ctx, types.ServiceOfferingType, byID); err != nil {
		if err != util.ErrNotFoundInStorage {
			return nil, util.HandleStorageError(err, string(types.ServicePlanType))
		}
	}
	if serviceObject == nil {
		return nil, nil
	}
	service := serviceObject.(*types.ServiceOffering)
	return service, nil
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
	referencePlan.Description = constant.ReferencePlanDescription
	referencePlan.ID = UUID.String()
	referencePlan.CatalogID = UUID.String()
	referencePlan.Bindable = newTrue()

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
