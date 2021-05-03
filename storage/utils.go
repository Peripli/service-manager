package storage

import (
	"context"
	"fmt"
	"github.com/Peripli/service-manager/pkg/instance_sharing"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/tidwall/gjson"
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
		return nil, util.HandleStorageError(util.ErrNotFoundInStorage, objectType.String())
	}
	return dbObject, nil
}

func IsReferencePlan(ctx context.Context, repository Repository, objectType, byKey, servicePlanID string) (bool, error) {
	dbPlanObject, err := GetObjectByField(ctx, repository, types.ObjectType(objectType), byKey, servicePlanID)
	if err != nil {
		return false, err // err retrieved from GetObjectByField is "not found" type.
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

func IsReferencedShared(ctx context.Context, repository Repository, referencedInstanceID string) (bool, error) {
	dbReferencedObject, err := GetObjectByField(ctx, repository, types.ServiceInstanceType, "id", referencedInstanceID)
	if err != nil {
		return false, util.HandleStorageError(util.ErrNotFoundInStorage, types.ServiceInstanceType.String())
	}
	referencedInstance := dbReferencedObject.(*types.ServiceInstance)

	if !referencedInstance.IsShared() {
		log.C(ctx).Debugf("IsReferencedShared failed. The target instance %s is not shared", referencedInstanceID)
		return false, util.HandleInstanceSharingError(util.ErrReferencedInstanceNotShared, referencedInstanceID)
	}
	return true, nil
}

func ValidateOwnership(repository Repository, tenantIdentifier string, req *web.Request, callerTenantID string) error {
	ctx := req.Context()
	path := fmt.Sprintf("parameters.%s", instance_sharing.ReferencedInstanceIDKey)
	referencedInstanceID := gjson.GetBytes(req.Body, path).String()
	dbReferencedObject, err := GetObjectByField(ctx, repository, types.ServiceInstanceType, "id", referencedInstanceID)
	if err != nil {
		log.C(ctx).Errorf("Failed retrieving the reference-instance by the ID: %s", referencedInstanceID)
		return util.HandleStorageError(util.ErrNotFoundInStorage, types.ServiceInstanceType.String())
	}
	instance := dbReferencedObject.(*types.ServiceInstance)
	sharedInstanceTenantID := instance.Labels[tenantIdentifier][0]

	if sharedInstanceTenantID != callerTenantID {
		log.C(ctx).Errorf("Instance owner %s is not the same as the caller %s", sharedInstanceTenantID, callerTenantID)
		return util.HandleStorageError(util.ErrNotFoundInStorage, types.ServiceInstanceType.String())
	}
	return nil
}

func IsValidReferenceInstancePatchRequest(req *web.Request, instance *types.ServiceInstance, planIDProperty string) error {
	// epsilontal todo: How can we update labels and do we want to allow the change?
	newPlanID := gjson.GetBytes(req.Body, planIDProperty).String()
	if instance.ServicePlanID != newPlanID {
		return util.HandleInstanceSharingError(util.ErrChangingPlanOfReferenceInstance, instance.ID)
	}

	parametersRaw := gjson.GetBytes(req.Body, "parameters").Raw
	if parametersRaw != "" {
		return util.HandleInstanceSharingError(util.ErrChangingParametersOfReferenceInstance, instance.ID)
	}

	return nil
}
