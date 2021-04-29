package storage

import (
	"context"
	"github.com/Peripli/service-manager/pkg/instance_sharing"
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

func GetObjectByField(ctx context.Context, repository Repository, objectType types.ObjectType, byKey, byValue string) (types.Object, error) {
	byID := query.ByField(query.EqualsOperator, byKey, byValue)
	dbObject, err := repository.Get(ctx, objectType, byID)
	if err != nil {
		return nil, err
	}
	return dbObject, nil
}

func IsReferencePlan(ctx context.Context, repository TransactionalRepository, objectType, byKey, servicePlanID string) (bool, error) {
	dbPlanObject, err := GetObjectByField(ctx, repository, types.ObjectType(objectType), byKey, servicePlanID)
	if err != nil {
		return false, err
	}
	plan := dbPlanObject.(*types.ServicePlan)
	return plan.Name == instance_sharing.ReferencePlanName, nil
}
