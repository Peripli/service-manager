package sharing

import (
	"context"
	"github.com/Peripli/service-manager/pkg/instance_sharing"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
	"github.com/tidwall/gjson"
)

func ExtractReferencedInstanceID(req *web.Request, repository storage.Repository, body []byte, tenantIdentifier string, getTenantId func() string, smaap bool) (string, error) {
	var err error
	ctx := req.Context()
	parameters := gjson.GetBytes(body, "parameters").Map()

	err = validateParameters(parameters)
	if err != nil {
		return "", err
	}

	sharedInstances, err := getSharedInstancesByTenant(ctx, repository, tenantIdentifier, getTenantId())
	if err != nil {
		return "", err
	}

	var referencedInstance *types.ServiceInstance
	referencedInstanceID, exists := parameters[instance_sharing.ReferencedInstanceIDKey]
	if exists && referencedInstanceID.String() != "*" {
		referencedInstance, err = getInstanceByID(ctx, repository, referencedInstanceID.String(), tenantIdentifier, getTenantId())
		if err != nil {
			return "", err
		}
	} else {
		filteredInstancesList, err := filterInstancesBySelectors(ctx, repository, sharedInstances, parameters, smaap)
		err = validateSingleResult(filteredInstancesList, parameters)
		if err != nil {
			return "", err
		}
		referencedInstance = filteredInstancesList.ItemAt(0).(*types.ServiceInstance)
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

func getInstanceByID(ctx context.Context, repository storage.Repository, referencedInstanceID string, tenantIdentifier string, tenantValue string) (*types.ServiceInstance, error) {
	referencedInstanceObj, err := repository.Get(ctx, types.ServiceInstanceType,
		query.ByLabel(query.EqualsOperator, tenantIdentifier, tenantValue),
		query.ByField(query.EqualsOperator, "id", referencedInstanceID),
	)
	if err != nil {
		log.C(ctx).Errorf("failed retrieving service instance %s: %v", referencedInstanceID, err)
		return nil, util.HandleInstanceSharingError(util.ErrReferencedInstanceNotFound, referencedInstanceID)
	}
	referencedInstance := referencedInstanceObj.(*types.ServiceInstance)

	if !referencedInstance.IsShared() {
		log.C(ctx).Debugf("The target instance %s is not shared.", referencedInstance.ID)
		return nil, util.HandleInstanceSharingError(util.ErrReferencedInstanceNotShared, referencedInstance.ID)
	}

	// verify the reference plan and the target instance are of the same service offering:
	referencePlan, ok := types.PlanFromContext(ctx)
	if !ok || referencePlan == nil {
		return nil, util.HandleStorageError(util.ErrNotFoundInStorage, types.ServicePlanType.String())
	}
	sameOffering, err := isSameServiceOffering(ctx, repository, referencePlan.ServiceOfferingID, referencedInstance.ServicePlanID)
	if err != nil {
		return nil, err
	}
	if !sameOffering {
		log.C(ctx).Debugf("The target instance %s is not of the same service offering.", referencedInstance.ID)
		return nil, util.HandleInstanceSharingError(util.ErrReferenceWithWrongServiceOffering, referencedInstance.ID)
	}

	return referencedInstance, nil
}

func validateSingleResult(results types.ServiceInstances, parameters map[string]gjson.Result) error {
	if results.Len() == 0 {
		referencedInstanceID, exists := parameters[instance_sharing.ReferencedInstanceIDKey]
		if exists && referencedInstanceID.String() != "*" {
			return util.HandleInstanceSharingError(util.ErrReferencedInstanceNotFound, referencedInstanceID.String())
		}
		return util.HandleInstanceSharingError(util.ErrNoResultsForReferenceSelector, "")
	} else if results.Len() > 1 && len(parameters) == 0 {
		return util.HandleInstanceSharingError(util.ErrMissingOrInvalidReferenceParameter, instance_sharing.ReferencedInstanceIDKey)
	} else if results.Len() > 1 {
		// there is more than one shared instance that meets your criteria
		return util.HandleInstanceSharingError(util.ErrMultipleReferenceSelectorResults, "")
	}
	return nil
}

func getSharedInstancesByTenant(ctx context.Context, repository storage.Repository, tenantIdentifier, tenantID string) (types.ObjectList, error) {
	sharedInstances, err := getSharedInstances(ctx, repository, tenantIdentifier, tenantID)
	if err != nil {
		return nil, util.HandleStorageError(err, types.ServiceInstanceType.String())
	}
	return sharedInstances, nil
}

func validateParameters(parameters map[string]gjson.Result) error {
	// when provisioning reference by instance id, we only allow one parameter:
	_, byID := parameters[instance_sharing.ReferencedInstanceIDKey]
	if byID && len(parameters) > 1 {
		return util.HandleInstanceSharingError(util.ErrInvalidReferenceSelectors, "")
	}
	return nil
}

func getSharedInstances(ctx context.Context, repository storage.Repository, tenantIdentifier string, tenantID string) (types.ObjectList, error) {
	referencePlan, _ := types.PlanFromContext(ctx)
	queryParams := map[string]interface{}{
		"tenant_identifier": tenantIdentifier,
		"tenant_id":         tenantID,
		"offering_id":       referencePlan.ServiceOfferingID,
	}

	objectList, err := repository.QueryForList(ctx, types.ServiceInstanceType, storage.QueryForSharedInstances, queryParams)
	return objectList, err
}

func filterInstancesBySelectors(ctx context.Context, repository storage.Repository, instances types.ObjectList, parameters map[string]gjson.Result, smaap bool) (types.ServiceInstances, error) {
	var err error
	var filteredInstances types.ServiceInstances
	for i := 0; i < instances.Len(); i++ {
		if filteredInstances.Len() > 1 {
			break
		}
		// selectors: true or false -> the entire map should be true in order to pass
		selectors := make(map[string]bool)
		selectorVal, exists := parameters[instance_sharing.ReferencedInstanceIDKey]

		// "*" for any shared instance of the tenant, if parameter not passed, set true for any shared instance
		if !exists || selectorVal.String() == "*" {
			selectors["id"] = true
		}

		instance := instances.ItemAt(i).(*types.ServiceInstance)

		selectorVal, exists = parameters[instance_sharing.ReferenceInstanceNameSelector]
		if exists {
			selectors["name"] = instance.Name == selectorVal.String()
		}

		selectorVal, exists = parameters[instance_sharing.ReferencePlanNameSelector]
		if exists {
			selectors["service_plan_id"], err = foundSharedInstanceWithPlan(ctx, repository, instance, selectorVal.String(), smaap)
			if err != nil {
				return types.ServiceInstances{}, err
			}
		}

		selectorVal, exists = parameters[instance_sharing.ReferenceLabelSelector]
		if exists {
			selectors["label"], err = foundSharedInstanceWithLabels(instance, []byte(selectorVal.Raw))
			if err != nil {
				return types.ServiceInstances{}, err
			}
		}

		// add the instance if the selectors are true:
		shouldAdd := true
		for _, isValid := range selectors {
			shouldAdd = shouldAdd && isValid
		}
		if shouldAdd {
			filteredInstances.Add(instance)
		}
	}
	return filteredInstances, nil
}

func foundSharedInstanceWithPlan(ctx context.Context, repository storage.Repository, instance *types.ServiceInstance, selector string, smaap bool) (bool, error) {
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

func foundSharedInstanceWithLabels(instance *types.ServiceInstance, labels []byte) (bool, error) {
	var selectorLabels types.Labels
	if err := util.BytesToObject(labels, &selectorLabels); err != nil {
		return false, err
	}
	/*
		todo: remove after review
		should accept this label selector:
			selectorLabels: {
				"key": ["x"],
				"region": ["us", "jp"]
			}

			instanceLabels: {
				"key": ["x", "y"],
				"region": ["us", "eu", "jp"]
			}
	*/
	selectorValidation := make(map[string]bool)
	for selectorLabelKey, selectorLabelArray := range selectorLabels {
		instanceLabelVal, exists := instance.Labels[selectorLabelKey]
		if exists && instance.Labels != nil {
			validLabel := true
			for _, selectorLabelVal := range selectorLabelArray {
				validLabel = validLabel && contains(instanceLabelVal, selectorLabelVal)
				selectorValidation[selectorLabelKey] = validLabel
			}
		} else {
			selectorValidation[selectorLabelKey] = false
		}
	}

	foundSharedInstance := true
	for _, isValid := range selectorValidation {
		foundSharedInstance = foundSharedInstance && isValid
	}

	return foundSharedInstance, nil
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

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
