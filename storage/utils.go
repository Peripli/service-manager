package storage

import (
	"context"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"

)

func GetServiceByServiceBinding(repository Repository,ctx context.Context, serviceBindingId  string) (*types.ServiceOffering, error){
	byID:=query.ByField(query.EqualsOperator, "id",serviceBindingId)
	serviceBindingObject, err := repository.Get(ctx, types.ServiceBindingType, byID)
	if err!=nil{
		return nil, util.HandleStorageError(err, types.ServiceBindingType.String())
	}
	serviceBinding:=serviceBindingObject.(*types.ServiceBinding)
	return GetServiceByServiceInstance(repository,ctx,serviceBinding.ServiceInstanceID)
}



func GetServiceByServiceInstance(repository Repository,ctx context.Context, serviceInstanceId  string) (*types.ServiceOffering, error){
	byID := query.ByField(query.EqualsOperator, "id", serviceInstanceId)
	criteria := query.CriteriaForContext(ctx)
	obj, err := repository.Get(ctx, types.ServiceInstanceType, append(criteria, byID)...)
	if err != nil {
		return nil, util.HandleStorageError(err, types.ServiceInstanceType.String())
	}
	serviceInstance := obj.(*types.ServiceInstance)
	planObject, err := repository.Get(context.TODO(), types.ServicePlanType, query.ByField(query.EqualsOperator, "id", serviceInstance.ServicePlanID))
	if err != nil {
		return nil, util.HandleStorageError(err, types.ServicePlanType.String())
	}
	plan := planObject.(*types.ServicePlan)
	serviceObject, err := repository.Get(context.TODO(), types.ServiceOfferingType, query.ByField(query.EqualsOperator, "id", plan.ServiceOfferingID))
	if err != nil {
		return nil, util.HandleStorageError(err, types.ServiceOfferingType.String())
	}
	service := serviceObject.(*types.ServiceOffering)
	return service , nil
}
