package storage

import (
	"context"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/instance_sharing"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/log"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/query"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/util"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web"
)

func GetServiceOfferingAndPlanByServiceInstanceId(repository Repository, ctx context.Context, serviceInstanceId string) (*types.ServiceOffering, *types.ServicePlan, error) {
	byID := query.ByField(query.EqualsOperator, "id", serviceInstanceId)
	criteria := query.CriteriaForContext(ctx)
	obj, err := repository.Get(ctx, types.ServiceInstanceType, append(criteria, byID)...)
	if err != nil {
		log.C(ctx).Errorf("Failed retrieving the service instance by ID: %s", serviceInstanceId)
		return nil, nil, util.HandleStorageError(util.ErrNotFoundInStorage, types.ServiceInstanceType.String())
	}
	serviceInstance := obj.(*types.ServiceInstance)
	planObject, err := repository.Get(ctx, types.ServicePlanType, query.ByField(query.EqualsOperator, "id", serviceInstance.ServicePlanID))
	if err != nil {
		log.C(ctx).Errorf("Failed retrieving the service plan by ID: %s", serviceInstance.ServicePlanID)
		return nil, nil, util.HandleStorageError(util.ErrNotFoundInStorage, types.ServiceInstanceType.String())
	}
	plan := planObject.(*types.ServicePlan)
	serviceObject, err := repository.Get(ctx, types.ServiceOfferingType, query.ByField(query.EqualsOperator, "id", plan.ServiceOfferingID))
	if err != nil {
		log.C(ctx).Errorf("Failed retrieving the service offering by ID: %s", plan.ServiceOfferingID)
		return nil, nil, util.HandleStorageError(util.ErrNotFoundInStorage, types.ServiceInstanceType.String())
	}
	service := serviceObject.(*types.ServiceOffering)
	return service, plan, nil
}

func GetObjectByField(ctx context.Context, repository Repository, objectType types.ObjectType, byKey, byValue string, additionalQueries ...query.Criterion) (types.Object, error) {
	var criteria []query.Criterion
	byField := query.ByField(query.EqualsOperator, byKey, byValue)
	criteria = append(criteria, byField)
	if objectType == types.ServiceInstanceType {
		criteria = append(criteria, query.CriteriaForContext(ctx)...)
	}
	if len(additionalQueries) > 0 {
		criteria = append(criteria, additionalQueries...)
	}
	dbObject, err := repository.Get(ctx, objectType, criteria...)
	if err != nil {
		log.C(ctx).Errorf("GetObjectByField failed retrieving the %s by %s: %s", objectType.String(), byKey, byValue)
		return nil, err
	}
	return dbObject, nil
}

func IsReferencePlan(req *web.Request, repository Repository, objectType, byKey, servicePlanID string) (bool, error) {
	ctx := req.Context()
	dbPlanObject, err := GetObjectByField(ctx, repository, types.ObjectType(objectType), byKey, servicePlanID)
	if err != nil {
		return false, util.HandleStorageError(util.ErrNotFoundInStorage, objectType)
	}
	plan := dbPlanObject.(*types.ServicePlan)
	req.Request = req.WithContext(types.ContextWithPlan(req.Context(), plan))
	return plan.Name == instance_sharing.ReferencePlanName, nil
}

func GetInstanceReferencesByID(ctx context.Context, repository Repository, instanceID string) (types.ObjectList, error) {
	// if has references - the list of references will be returned to the client in the error description
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
