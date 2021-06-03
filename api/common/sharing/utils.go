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
		log.C(ctx).Errorf("Failed to validate parameters: %s", err)
		return "", err
	}

	var referencedInstance *types.ServiceInstance
	referencedInstanceID, exists := parameters[instance_sharing.ReferencedInstanceIDKey]
	if exists && referencedInstanceID.String() != "*" {
		referencedInstance, err = getInstanceByID(ctx, repository, referencedInstanceID.String(), tenantIdentifier, getTenantId())
		if err != nil {
			log.C(ctx).Errorf("Failed to retrieve the instance %s, error: %s", referencedInstanceID.String(), err)
			return "", err
		}
	} else {
		sharedInstances, err := getSharedInstances(ctx, repository, tenantIdentifier, getTenantId())
		if err != nil {
			log.C(ctx).Errorf("Failed to retrieve shared instances: %s", err)
			return "", err
		}

		filteredInstancesList, err := filterInstancesBySelectors(ctx, repository, sharedInstances, parameters, smaap)
		if err != nil {
			log.C(ctx).Errorf("Failed to filter instances by selectors: %s", err)
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

func getInstanceByID(ctx context.Context, repository storage.Repository, instanceID string, tenantIdentifier string, tenantValue string) (*types.ServiceInstance, error) {
	instanceObj, err := repository.Get(ctx, types.ServiceInstanceType,
		query.ByLabel(query.EqualsOperator, tenantIdentifier, tenantValue),
		query.ByField(query.EqualsOperator, "id", instanceID),
	)
	if err != nil {
		log.C(ctx).Errorf("failed retrieving service instance %s: %v", instanceID, err)
		return nil, util.HandleInstanceSharingError(util.ErrReferencedInstanceNotFound, instanceID)
	}
	instance := instanceObj.(*types.ServiceInstance)

	if !instance.IsShared() {
		log.C(ctx).Debugf("The target instance %s is not shared.", instance.ID)
		return nil, util.HandleInstanceSharingError(util.ErrReferencedInstanceNotShared, instance.ID)
	}

	// verify the reference plan and the target instance are of the same service offering:
	referencePlan, ok := types.PlanFromContext(ctx)
	if !ok || referencePlan == nil {
		return nil, util.HandleStorageError(util.ErrNotFoundInStorage, types.ServicePlanType.String())
	}
	sameOffering, err := isSameServiceOffering(ctx, repository, referencePlan.ServiceOfferingID, instance.ServicePlanID)
	if err != nil {
		return nil, err
	}
	if !sameOffering {
		log.C(ctx).Debugf("The target instance %s is not of the same service offering.", instance.ID)
		return nil, util.HandleInstanceSharingError(util.ErrReferenceWithWrongServiceOffering, instance.ID)
	}

	return instance, nil
}

func validateParameters(parameters map[string]gjson.Result) error {
	if len(parameters) == 0 {
		return util.HandleInstanceSharingError(util.ErrMissingOrInvalidReferenceParameter, instance_sharing.ReferencedInstanceIDKey)
	}
	// don't allow combination of id with selectors.
	// parameters should not be empty, but the values of each parameter can be ""
	id := parameters[instance_sharing.ReferencedInstanceIDKey].String()
	label := parameters[instance_sharing.ReferenceLabelSelector].Raw
	name := parameters[instance_sharing.ReferenceInstanceNameSelector].String()
	plan := parameters[instance_sharing.ReferencePlanNameSelector].String()
	hasAtLeastOneSelector := len(label) > 0 || len(name) > 0 || len(plan) > 0
	if len(id) > 0 && hasAtLeastOneSelector {
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
		// only single result is accepted for selectors:
		if filteredInstances.Len() > 1 {
			log.C(ctx).Errorf("%s", util.ErrMultipleReferenceSelectorResults)
			return types.ServiceInstances{}, util.HandleInstanceSharingError(util.ErrMultipleReferenceSelectorResults, "")
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
			selectors["service_plan_id"], err = compareInstancePlanWithSelector(ctx, repository, instance.ServicePlanID, selectorVal.String(), smaap)
			if err != nil {
				return types.ServiceInstances{}, err
			}
		}

		selectorVal, exists = parameters[instance_sharing.ReferenceLabelSelector]
		if exists {
			selectors["label"], err = compareInstanceLabelsWithSelector(instance.Labels, []byte(selectorVal.Raw))
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
	if filteredInstances.Len() == 0 {
		log.C(ctx).Errorf("%s", util.ErrNoResultsForReferenceSelector)
		return types.ServiceInstances{}, util.HandleInstanceSharingError(util.ErrNoResultsForReferenceSelector, "")
	}
	return filteredInstances, nil
}

func compareInstancePlanWithSelector(ctx context.Context, repository storage.Repository, planID string, selector string, smaap bool) (bool, error) {
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
	return planID == selectorPlan.ID, nil
}

func compareInstanceLabelsWithSelector(instanceLabels types.Labels, labels []byte) (bool, error) {
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
		instanceLabelVal, exists := instanceLabels[selectorLabelKey]
		if exists && instanceLabels != nil {
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
