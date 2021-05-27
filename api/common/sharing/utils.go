package sharing

import (
	"context"
	"github.com/Peripli/service-manager/pkg/instance_sharing"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
	"github.com/tidwall/gjson"
	"reflect"
)

func ExtractReferenceInstanceID(req *web.Request, repository storage.Repository, body []byte, tenantIdentifier string, getTenantId func() string, isSMAAP bool) (string, error) {
	var err error
	var selectorResult types.ObjectList
	ctx := req.Context()
	parameters := gjson.GetBytes(body, "parameters").Map()

	if len(parameters) == 0 {
		return "", util.HandleInstanceSharingError(util.ErrMissingOrInvalidReferenceParameter, instance_sharing.ReferencedInstanceIDKey)
	}
	selectorResult, err = getInstanceBySelector(ctx, repository, parameters, tenantIdentifier, getTenantId(), isSMAAP)

	if err != nil {
		return "", err
	}
	if selectorResult == nil || selectorResult.Len() == 0 {
		// todo: add a new error: not found a shared instance which meets your criteria
		return "", util.HandleInstanceSharingError(util.ErrReferencedInstanceNotFound, "")
	}
	if selectorResult.Len() > 1 {
		// there is more than one shared instance that meets your criteria
		return "", util.HandleInstanceSharingError(util.ErrMultipleReferenceSelectorResults, "")
	}
	referencedInstance := selectorResult.ItemAt(0).(*types.ServiceInstance)

	// If the selector is instanceID, we would like to inform the user if the target instance was not shared:
	if !referencedInstance.IsShared() {
		log.C(ctx).Debugf("The target instance %s is not shared.", referencedInstance.ID)
		return "", util.HandleInstanceSharingError(util.ErrReferencedInstanceNotShared, referencedInstance.ID)
	}

	return referencedInstance.ID, nil
}

func IsValidReferenceInstancePatchRequest(req *web.Request, instance *types.ServiceInstance, planIDProperty string) error {
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

func getInstanceBySelector(ctx context.Context, repository storage.Repository, parameters map[string]gjson.Result, tenantIdentifier, tenantID string, smaap bool) (types.ObjectList, error) {

	objectList, err := getObjectList(ctx, repository, tenantIdentifier, tenantID)
	if err != nil {
		return nil, util.HandleStorageError(err, types.ServiceInstanceType.String())
	}

	filteredArray := types.NewObjectArray()
	filteredArray, err = filterListToArray(ctx, repository, filteredArray, objectList, parameters, smaap)
	if err != nil {
		return nil, err
	}

	return filteredArray, nil
}

func getObjectList(ctx context.Context, repository storage.Repository, tenantIdentifier string, tenantID string) (types.ObjectList, error) {
	referencePlan, _ := types.PlanFromContext(ctx)
	queryParams := map[string]interface{}{
		"tenant_identifier": tenantIdentifier,
		"tenant_id":         tenantID,
		"offering_id":       referencePlan.ServiceOfferingID,
	}

	objectList, err := repository.QueryForList(ctx, types.ServiceInstanceType, storage.QueryForSharedInstances, queryParams)
	return objectList, err
}

func filterListToArray(ctx context.Context, repository storage.Repository, objectArray *types.ObjectArray, objectList types.ObjectList, parameters map[string]gjson.Result, smaap bool) (*types.ObjectArray, error) {
	var err error
	referencedInstanceID := parameters[instance_sharing.ReferencedInstanceIDKey].String()
	planNameSelector := parameters[instance_sharing.ReferencePlanNameSelector].String()
	instanceNameSelector := parameters[instance_sharing.ReferenceInstanceNameSelector].String()
	labelsSelector := parameters[instance_sharing.ReferenceLabelSelector].Raw

	if referencedInstanceID == "*" {
		objectArray, err = filterBySelector(objectArray, objectList, "*", "")
		if err != nil {
			return nil, err
		}
	}
	if len(referencedInstanceID) > 1 {
		objectArray, err = filterBySelector(objectArray, objectList, "id", referencedInstanceID)
		if err != nil {
			return nil, util.HandleInstanceSharingError(util.ErrReferencedInstanceNotFound, referencedInstanceID)

		}
	}
	if len(planNameSelector) > 0 {
		objectArray, err = filterByPlanSelector(ctx, repository, objectArray, objectList, planNameSelector, smaap)
		if err != nil {
			return nil, err
		}
	}
	if len(instanceNameSelector) > 0 {
		objectArray, err = filterBySelector(objectArray, objectList, "name", instanceNameSelector)
		if err != nil {
			return nil, err
		}
	}
	if len(labelsSelector) > 0 {
		objectArray, err = filterByLabelSelector(objectArray, objectList, []byte(labelsSelector))
		if err != nil {
			return nil, err
		}
	}
	return objectArray, nil
}

func filterByPlanSelector(ctx context.Context, repository storage.Repository, objectArray *types.ObjectArray, objectList types.ObjectList, selector string, smaap bool) (*types.ObjectArray, error) {
	var selectorPlanObj types.Object
	var err error
	var key string
	if smaap {
		key = "name"
	} else {
		key = "catalog_name"
	}
	selectorPlanObj, err = storage.GetObjectByField(ctx, repository, types.ServicePlanType, key, selector)
	if err != nil {
		return types.NewObjectArray(), util.HandleInstanceSharingError(util.ErrReferencedInstanceNotFound, "")
	}
	selectorPlan := selectorPlanObj.(*types.ServicePlan)
	return filterBySelector(objectArray, objectList, "service_plan_id", selectorPlan.ID)
}

func filterByLabelSelector(objectArray *types.ObjectArray, objectList types.ObjectList, labels []byte) (*types.ObjectArray, error) {
	for index := 0; index < objectList.Len(); index++ {
		isNil := objectList.ItemAt(index) == nil
		if isNil {
			return nil, nil
		}
		instance := objectList.ItemAt(index).(*types.ServiceInstance)
		var selectorLabels types.Labels
		if err := util.BytesToObject(labels, &selectorLabels); err != nil {
			return nil, err
		}
		if reflect.DeepEqual(instance.Labels, selectorLabels) {
			objectArray.Add(objectList.ItemAt(index))
		}
	}
	if objectArray.Len() == 0 {
		return objectArray, util.HandleInstanceSharingError(util.ErrReferencedInstanceNotFound, "")

	}
	return objectArray, nil
}

func filterBySelector(objectArray *types.ObjectArray, objectList types.ObjectList, key, value string) (*types.ObjectArray, error) {
	for index := 0; index < objectList.Len(); index++ {
		isNil := objectList.ItemAt(index) == nil
		instance := objectList.ItemAt(index).(*types.ServiceInstance)
		switch key {
		case "*":
			{
				objectArray.Add(objectList.ItemAt(index))
			}
		case "id":
			if !isNil && instance.ID == value {
				objectArray.Add(objectList.ItemAt(index))
			}
		case "name":
			if !isNil && instance.Name == value {
				objectArray.Add(objectList.ItemAt(index))
			}
		case "service_plan_id":
			if !isNil && instance.ServicePlanID == value {
				objectArray.Add(objectList.ItemAt(index))
			}
		}
	}
	if objectArray.Len() == 0 {
		return objectArray, util.HandleInstanceSharingError(util.ErrNoResultsForReferenceSelector, "")

	}
	return objectArray, nil
}
