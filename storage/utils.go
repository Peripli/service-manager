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
	"net/http"
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

func IsReferencePlan(ctx context.Context, repository Repository, objectType, byKey, servicePlanID string) (bool, error) {
	dbPlanObject, err := GetObjectByField(ctx, repository, types.ObjectType(objectType), byKey, servicePlanID)
	if err != nil {
		return false, err
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
		return nil, err
	}
	return references, nil
}

func IsReferencedShared(ctx context.Context, repository Repository, referencedInstanceID string) (bool, error) {
	dbReferencedObject, err := GetObjectByField(ctx, repository, types.ServiceInstanceType, "id", referencedInstanceID)
	if err != nil {
		return false, util.HandleStorageError(err, types.ServiceInstanceType.String())
	}
	referencedInstance := dbReferencedObject.(*types.ServiceInstance)

	if !referencedInstance.IsShared() {
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
		return err
	}
	instance := dbReferencedObject.(*types.ServiceInstance)
	sharedInstanceTenantID := instance.Labels[tenantIdentifier][0]

	if sharedInstanceTenantID != callerTenantID {
		log.C(ctx).Errorf("Instance owner %s is not the same as the caller %s", sharedInstanceTenantID, callerTenantID)
		return &util.HTTPError{
			ErrorType:   "NotFound",
			Description: "could not find such service instance",
			StatusCode:  http.StatusNotFound,
		}
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
