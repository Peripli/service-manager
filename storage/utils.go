package storage

import (
	"context"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
)

func GetServiceOfferingByServiceInstanceId(repository Repository, ctx context.Context, serviceInstanceId string) (*types.ServiceOffering, error) {
	byID := query.ByField(query.EqualsOperator, "id", serviceInstanceId)
	criteria := query.CriteriaForContext(ctx)
	obj, err := repository.Get(ctx, types.ServiceInstanceType, append(criteria, byID)...)
	if err != nil {
		return nil, util.HandleStorageError(err, types.ServiceInstanceType.String())
	}
	serviceInstance := obj.(*types.ServiceInstance)
	planObject, err := repository.Get(ctx, types.ServicePlanType, query.ByField(query.EqualsOperator, "id", serviceInstance.ServicePlanID))
	if err != nil {
		return nil, util.HandleStorageError(err, types.ServicePlanType.String())
	}
	plan := planObject.(*types.ServicePlan)
	serviceObject, err := repository.Get(ctx, types.ServiceOfferingType, query.ByField(query.EqualsOperator, "id", plan.ServiceOfferingID))
	if err != nil {
		return nil, util.HandleStorageError(err, types.ServiceOfferingType.String())
	}
	service := serviceObject.(*types.ServiceOffering)
	return service, nil
}

func AttachLastOperations(ctx context.Context, objectType types.ObjectType, resources types.ObjectList, repository Repository) error {
	lastOperationsMap, err := GetLastOperations(ctx, objectType, getResourceIds(resources), repository)

	if err != nil {
		return err
	}

	for i := 0; i < resources.Len(); i++ {
		resource := resources.ItemAt(i)
		if LastOp, ok := lastOperationsMap[resource.GetID()]; ok {
			LastOp.TransitiveResources = nil
			resource.SetLastOperation(LastOp)
		}
	}

	return nil
}

func getResourceIds(resources types.ObjectList) []string {
	var resourceIds []string
	for i := 0; i < resources.Len(); i++ {
		resource := resources.ItemAt(i)
		resourceIds = append(resourceIds, resource.GetID())
	}
	return resourceIds
}

func AttachLastOperation(ctx context.Context, objectID string, object types.Object, repository Repository) error {
	ops, err := GetLastOperations(ctx, object.GetType(), []string{objectID}, repository)

	if err != nil {
		return err
	}

	if lastOperation, ok := ops[objectID]; ok {
		lastOperation.TransitiveResources = nil
		object.SetLastOperation(lastOperation)
	}

	return nil
}

func GetLastOperations(ctx context.Context, resourceType types.ObjectType, resourceIDs []string, repository Repository) (map[string]*types.Operation, error) {
	if len(resourceIDs) == 0 {
		return nil, nil
	}

	queryParams := map[string]interface{}{
		"id_list":       resourceIDs,
		"resource_type": resourceType,
	}

	resourceLastOps, err := repository.QueryForList(
		ctx,
		types.OperationType,
		QueryForLastOperationsPerResource,
		queryParams)

	if err != nil {
		return nil, util.HandleStorageError(err, types.OperationType.String())
	}

	instanceLastOpsMap := make(map[string]*types.Operation)

	for i := 0; i < resourceLastOps.Len(); i++ {
		lastOp := resourceLastOps.ItemAt(i).(*types.Operation)
		instanceLastOpsMap[lastOp.ResourceID] = lastOp
	}

	return instanceLastOpsMap, nil
}
