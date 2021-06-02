package sharing

import (
	"bytes"
	"context"
	"fmt"
	"github.com/Peripli/service-manager/pkg/instance_sharing"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
	"github.com/tidwall/gjson"
	"reflect"
)

func ExtractReferencedInstanceID(req *web.Request, repository storage.Repository, body []byte, tenantIdentifier string, getTenantId func() string, smaap bool) (string, error) {
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

	var referencedInstance *types.ServiceInstance
	referencedInstanceID := parameters[instance_sharing.ReferencedInstanceIDKey].String()
	if len(referencedInstanceID) > 1 {
		referencedInstance, err = filterInstancesByID(ctx, repository, referencedInstanceID, tenantIdentifier, getTenantId())
		if err != nil {
			return "", err
		}
	} else {
		filteredInstancesMap := map[string]*types.ServiceInstance{}
		filteredInstancesMap, err = filterInstancesBySelectors(ctx, repository, sharedInstancesMap, parameters, smaap)
		err = validateSelectorResults(filteredInstancesMap, parameters)
		if err != nil {
			return "", err
		}

		referencedInstance, err = retrieveInstanceFromMap(sharedInstancesMap)
		if err != nil {
			return "", err
		}
	}

	return referencedInstance.ID, nil
}

func filterInstancesByID(ctx context.Context, repository storage.Repository, referencedInstanceID string, tenantIdentifier string, tenantValue string) (*types.ServiceInstance, error) {
	referencedInstanceObj, err := repository.Get(ctx, types.ServiceInstanceType,
		query.ByLabel(query.EqualsOperator, tenantIdentifier, tenantValue),
		query.ByField(query.EqualsOperator, "id", referencedInstanceID),
	)
	if err != nil {
		log.C(ctx).Errorf("failed retrieving service instance %s: %v", referencedInstanceID, err)
		return nil, util.HandleInstanceSharingError(util.ErrReferencedInstanceNotFound, referencedInstanceID)
	}
	referencedInstance := referencedInstanceObj.(*types.ServiceInstance)

	referencePlan, ok := types.PlanFromContext(ctx)
	if !ok || referencePlan == nil {
		return nil, util.HandleStorageError(util.ErrNotFoundInStorage, types.ServicePlanType.String())
	}
	// verify the reference plan and the target instance are of the same service offering:
	sameOffering, err := isSameServiceOffering(ctx, repository, referencePlan.ServiceOfferingID, referencedInstance.ServicePlanID)
	if err != nil {
		return nil, err
	}
	if !sameOffering {
		log.C(ctx).Debugf("The target instance %s is not of the same service offering.", referencedInstance.ID)
		return nil, util.HandleInstanceSharingError(util.ErrReferenceWithWrongServiceOffering, referencedInstance.ID)
	}

	if !referencedInstance.IsShared() {
		log.C(ctx).Debugf("The target instance %s is not shared.", referencedInstance.ID)
		return nil, util.HandleInstanceSharingError(util.ErrReferencedInstanceNotShared, referencedInstance.ID)
	}

	return referencedInstance, nil
}

func validateSelectorResults(results map[string]*types.ServiceInstance, parameters map[string]gjson.Result) error {
	if results == nil || len(results) == 0 {
		referencedInstanceID := parameters[instance_sharing.ReferencedInstanceIDKey].String()
		if len(referencedInstanceID) > 1 {
			return util.HandleInstanceSharingError(util.ErrReferencedInstanceNotFound, referencedInstanceID)
		}
		return util.HandleInstanceSharingError(util.ErrNoResultsForReferenceSelector, "")
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

func retrieveInstanceFromMap(results map[string]*types.ServiceInstance) (*types.ServiceInstance, error) {
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

		// "*" for any shared instance of the tenant
		if exists && val.String() == "*" {
			selectors["id"] = true
		}

		val, exists = parameters[instance_sharing.ReferenceInstanceNameSelector]
		if exists {
			selectors["name"] = instance.Name == val.String()
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
	return instance.ServicePlanID == selectorPlan.ID, nil
}

func filterByLabelSelector(instance *types.ServiceInstance, labels []byte) (bool, error) {
	var selectorLabels types.Labels
	if err := util.BytesToObject(labels, &selectorLabels); err != nil {
		return false, err
	}
	return reflect.DeepEqual(instance.Labels, selectorLabels), nil
}

func isSameServiceOffering(ctx context.Context, repository storage.Repository, offeringID string, planID string) (bool, error) {
	byID := query.ByField(query.EqualsOperator, "id", planID)
	servicePlanObj, err := repository.Get(ctx, types.ServicePlanType, byID)
	if err != nil {
		return false, util.HandleStorageError(err, types.ServicePlanType.String())
	}
	servicePlan := servicePlanObj.(*types.ServicePlan)
	return offeringID == servicePlan.ServiceOfferingID, nil
}
