package storage

import (
	"context"
	"github.com/Peripli/service-manager/pkg/instance_sharing"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
)

func GetServiceOfferingByServiceInstanceId(repository Repository, ctx context.Context, serviceInstanceId string) (*types.ServiceOffering, error) {
	byID := query.ByField(query.EqualsOperator, "id", serviceInstanceId)
	criteria := query.CriteriaForContext(ctx)
	obj, err := repository.Get(ctx, types.ServiceInstanceType, append(criteria, byID)...)
	if err != nil {
		log.C(ctx).Errorf("Failed retrieving the service instance by ID: %s", serviceInstanceId)
		return nil, util.HandleStorageError(util.ErrNotFoundInStorage, types.ServiceInstanceType.String())
	}
	serviceInstance := obj.(*types.ServiceInstance)
	planObject, err := repository.Get(ctx, types.ServicePlanType, query.ByField(query.EqualsOperator, "id", serviceInstance.ServicePlanID))
	if err != nil {
		log.C(ctx).Errorf("Failed retrieving the service plan by ID: %s", serviceInstance.ServicePlanID)
		return nil, util.HandleStorageError(util.ErrNotFoundInStorage, types.ServiceInstanceType.String())
	}
	plan := planObject.(*types.ServicePlan)
	serviceObject, err := repository.Get(ctx, types.ServiceOfferingType, query.ByField(query.EqualsOperator, "id", plan.ServiceOfferingID))
	if err != nil {
		log.C(ctx).Errorf("Failed retrieving the service offering by ID: %s", plan.ServiceOfferingID)
		return nil, util.HandleStorageError(util.ErrNotFoundInStorage, types.ServiceInstanceType.String())
	}
	service := serviceObject.(*types.ServiceOffering)
	return service, nil
}

func GetObjectByField(ctx context.Context, repository Repository, objectType types.ObjectType, byKey, byValue string) (types.Object, error) {
	byID := query.ByField(query.EqualsOperator, byKey, byValue)
	dbObject, err := repository.Get(ctx, objectType, byID)
	if err != nil {
		log.C(ctx).Errorf("GetObjectByField failed retrieving the %s by %s: %s", objectType.String(), byKey, byValue)
		return nil, err
	}
	return dbObject, nil
}

func IsReferencePlan(ctx context.Context, repository Repository, objectType, byKey, servicePlanID string) (bool, error) {
	dbPlanObject, err := GetObjectByField(ctx, repository, types.ObjectType(objectType), byKey, servicePlanID)
	if err != nil {
		return false, util.HandleStorageError(util.ErrNotFoundInStorage, objectType)
	}
	plan := dbPlanObject.(*types.ServicePlan)
	return plan.Name == instance_sharing.ReferencePlanName, nil
}

func GetInstanceReferencesByID(ctx context.Context, repository Repository, instanceID string) (types.ObjectList, error) {
	references, err := repository.List(
		ctx,
		types.ServiceInstanceType,
		query.ByField(query.EqualsOperator, instance_sharing.ReferencedInstanceIDKey, instanceID))
	if err != nil {
		log.C(ctx).Errorf("Failed retrieving the references of the instance: %s", instanceID)
		return nil, util.HandleStorageError(util.ErrNotFoundInStorage, types.ServiceInstanceType.String())
	}
	return references, nil
}
