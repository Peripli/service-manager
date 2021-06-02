package sharing

import (
	"bytes"
	"context"
	"fmt"
	"github.com/Peripli/service-manager/pkg/instance_sharing"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
	"github.com/tidwall/gjson"
	"reflect"
)

func ExtractReferenceInstanceID(req *web.Request, repository storage.Repository, body []byte, tenantIdentifier string, getTenantId func() string, smaap bool) (string, error) {
	var err error
	ctx := req.Context()
	parameters := gjson.GetBytes(body, "parameters").Map()

	err = validateParameters(parameters)
	if err != nil {
		return "", err
	}

	sharedInstancesMap := map[string]*types.ServiceInstance{}
	sharedInstancesMap, err = getSharedInstancesByTenant(ctx, repository, tenantIdentifier, getTenantId())
	if err != nil {
		return "", err
	}

	filteredInstancesMap, err := filterInstancesBySelectors(ctx, repository, sharedInstancesMap, parameters, smaap)
	if err != nil {
		return "", err
	}

	err = validateSelectorResults(filteredInstancesMap, parameters)
	if err != nil {
		return "", err
	}

	referencedInstance, err := getInstanceFromResult(sharedInstancesMap)
	if err != nil {
		return "", err
	}

	// If the selector is instanceID, we would like to inform the user if the target instance was not shared:
	if !referencedInstance.IsShared() {
		log.C(ctx).Debugf("The target instance %s is not shared.", referencedInstance.ID)
		return "", util.HandleInstanceSharingError(util.ErrReferencedInstanceNotShared, referencedInstance.ID)
	}

	return referencedInstance.ID, nil
}

func validateSelectorResults(results map[string]*types.ServiceInstance, parameters map[string]gjson.Result) error {
	if results == nil || len(results) == 0 {
		// todo: add a new error: not found a shared instance which meets your criteria
		return util.HandleInstanceSharingError(util.ErrReferencedInstanceNotFound, createKeyValuePairs(parameters))
	}
	if len(results) > 1 {
		// there is more than one shared instance that meets your criteria
		return util.HandleInstanceSharingError(util.ErrMultipleReferenceSelectorResults, "")
	}
	return nil
}
func createKeyValuePairs(m map[string]gjson.Result) string {
	b := new(bytes.Buffer)
	for key, value := range m {
		_, err := fmt.Fprintf(b, "%s=\"%s\"\n", key, value)
		if err != nil {
			return ""
		}
	}
	return b.String()
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

func getInstanceFromResult(results map[string]*types.ServiceInstance) (*types.ServiceInstance, error) {
	var instance *types.ServiceInstance
	for key := range results {
		instance = results[key]
	}
	return instance, nil
}

func getSharedInstancesByTenant(ctx context.Context, repository storage.Repository, tenantIdentifier, tenantID string) (map[string]*types.ServiceInstance, error) {
	objectsList, err := getObjectList(ctx, repository, tenantIdentifier, tenantID)
	if err != nil {
		return nil, util.HandleStorageError(err, types.ServiceInstanceType.String())
	}
	return convertObjectListToInstancesMap(objectsList), nil
}

func validateParameters(parameters map[string]gjson.Result) error {
	if len(parameters) == 0 {
		return util.HandleInstanceSharingError(util.ErrMissingOrInvalidReferenceParameter, instance_sharing.ReferencedInstanceIDKey)
	}

	_, byID := parameters[instance_sharing.ReferencedInstanceIDKey]
	if byID && len(parameters) > 1 {
		return util.HandleInstanceSharingError(util.ErrInvalidReferenceSelectors, "")
	}

	return nil
}

func convertObjectListToInstancesMap(objectList types.ObjectList) map[string]*types.ServiceInstance {
	instancesMap := map[string]*types.ServiceInstance{}
	if objectList == nil {
		return instancesMap
	}
	for index := 0; index < objectList.Len(); index++ {
		instance := objectList.ItemAt(index).(*types.ServiceInstance)
		instancesMap[instance.ID] = instance
	}
	return instancesMap
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

func filterInstancesBySelectors(ctx context.Context, repository storage.Repository, instances map[string]*types.ServiceInstance, parameters map[string]gjson.Result, smaap bool) (map[string]*types.ServiceInstance, error) {
	var err error
	for _, instance := range instances {
		// selectors: true or false -> the entire map should be true in order to pass
		selectors := make(map[string]interface{})

		val, exists := parameters[instance_sharing.ReferencedInstanceIDKey]
		if exists {
			selectors["id"] = filterBySelector(instance, "id", val.String())
		}

		val, exists = parameters[instance_sharing.ReferenceInstanceNameSelector]
		if exists {
			selectors["name"] = filterBySelector(instance, "name", val.String())
		}

		val, exists = parameters[instance_sharing.ReferencePlanNameSelector]
		if exists {
			selectors["service_plan_id"], err = filterByPlanSelector(ctx, repository, instance, val.String(), smaap)
			if err != nil {
				return nil, err
			}
		}

		val, exists = parameters[instance_sharing.ReferenceLabelSelector]
		if exists {
			selectors["label"], err = filterByLabelSelector(instance, []byte(val.Raw))
			if err != nil {
				return nil, err
			}
		}

		// if one of the selectors is false, delete the instance from the list
		for _, isValid := range selectors {
			_, ok := instances[instance.ID]
			if ok && isValid != nil && isValid == false {
				delete(instances, instance.ID)
			}
		}

	}
	return instances, nil
}

func filterByPlanSelector(ctx context.Context, repository storage.Repository, instance *types.ServiceInstance, selector string, smaap bool) (bool, error) {
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
		return false, util.HandleInstanceSharingError(util.ErrReferencedInstanceNotFound, "")
	}
	selectorPlan := selectorPlanObj.(*types.ServicePlan)
	return filterBySelector(instance, "service_plan_id", selectorPlan.ID), nil
}

func filterByLabelSelector(instance *types.ServiceInstance, labels []byte) (bool, error) {
	var selectorLabels types.Labels
	if err := util.BytesToObject(labels, &selectorLabels); err != nil {
		return false, err
	}
	return reflect.DeepEqual(instance.Labels, selectorLabels), nil
}

func filterBySelector(instance *types.ServiceInstance, key, value string) bool {
	switch key {
	case "id":
		// "*" for any shared instance of the tenant
		if value == "*" {
			return true
		}
		return instance.ID == value
	case "name":
		return instance.Name == value
	case "service_plan_id":
		// plan name should be converted to planID by this point, so we support OSB and SMAAP flows
		return instance.ServicePlanID == value
	}
	return false
}
