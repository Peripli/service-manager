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

	// verify the reference plan and the target instance are of the same service offering:
	referencePlan, ok := types.PlanFromContext(ctx)
	if !ok || referencePlan == nil {
		return "", util.HandleStorageError(util.ErrNotFoundInStorage, types.ServicePlanType.String())
	}

	var referencedInstance *types.ServiceInstance
	referencedInstanceID, exists := parameters[instance_sharing.ReferencedInstanceIDKey]
	if exists && len(referencedInstanceID.String()) > 0 && referencedInstanceID.String() != "*" {
		referencedInstance, err = getInstanceByID(ctx, repository, referencedInstanceID.String(), referencePlan.ServiceOfferingID, tenantIdentifier, getTenantId())
		if err != nil {
			return "", err
		}
	} else {
		selectors := parameters[instance_sharing.SelectorsKey].Map()
		referencedInstance, err = filterInstancesBySelectors(ctx, repository, referencePlan.ServiceOfferingID, tenantIdentifier, getTenantId(), selectors, smaap)
		if err != nil {
			log.C(ctx).Errorf("Failed to filter instances by selectors: %s", err)
			return "", err
		}

	}

	return referencedInstance.ID, nil
}

func IsValidReferenceInstancePatchRequest(req *web.Request, instance *types.ServiceInstance, planIDProperty string) error {
	newPlanID := gjson.GetBytes(req.Body, planIDProperty).String()
	if instance.ServicePlanID != newPlanID {
		return util.HandleInstanceSharingError(util.ErrChangingPlanOfReferenceInstance, instance.ID)
	}
	parameters := gjson.GetBytes(req.Body, "parameters").Map()
	referencedInstanceID, exists := parameters[instance_sharing.ReferencedInstanceIDKey]

	containsSameReference := exists && referencedInstanceID.String() == instance.ReferencedInstanceID
	if len(parameters) > 0 && !containsSameReference {
		return util.HandleInstanceSharingError(util.ErrChangingParametersOfReferenceInstance, instance.ID)
	}
	return nil
}

func getInstanceByID(ctx context.Context, repository storage.Repository, instanceID, offeringID, tenantIdentifier, tenantValue string) (*types.ServiceInstance, error) {
	instanceObj, err := repository.Get(ctx, types.ServiceInstanceType,
		query.ByLabel(query.EqualsOperator, tenantIdentifier, tenantValue),
		query.ByField(query.EqualsOperator, "id", instanceID),
	)
	if err != nil {
		log.C(ctx).Errorf("failed to retrieve service instance %s: %v", instanceID, err)
		return nil, util.HandleInstanceSharingError(util.ErrReferencedInstanceNotFound, instanceID)
	}
	instance := instanceObj.(*types.ServiceInstance)

	if !instance.IsShared() {
		log.C(ctx).Debugf("The target instance %s is not shared.", instance.ID)
		return nil, util.HandleInstanceSharingError(util.ErrReferencedInstanceNotShared, instance.ID)
	}

	sameOffering, err := isSameServiceOffering(ctx, repository, offeringID, instance.ServicePlanID)
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
	/* failures:
	 * parameters with empty values
	 * combination of ID with any other selector
	 * parameters should not be empty, but the values of each parameter can be "" (empty string/object)
	 */
	if len(parameters) == 0 {
		return util.HandleInstanceSharingError(util.ErrMissingOrInvalidReferenceParameter, "")
	}
	hasAtLeastOneSelector := false
	ID, IDExists := parameters[instance_sharing.ReferencedInstanceIDKey]
	validID := IDExists && ID.Type == gjson.String && len(ID.String()) > 0

	selectors, selectorsExists := parameters[instance_sharing.SelectorsKey]
	if selectorsExists {
		selectorsMap := selectors.Map()
		name, nameExists := selectorsMap[instance_sharing.ReferenceInstanceNameSelectorKey]
		plan, planExists := selectorsMap[instance_sharing.ReferencePlanNameSelectorKey]
		labels, labelsExists := selectorsMap[instance_sharing.ReferenceLabelSelectorKey]
		validName := nameExists && name.Type == gjson.String && len(name.String()) > 0
		validPlan := planExists && plan.Type == gjson.String && len(plan.String()) > 0
		validLabels := labelsExists && labels.String() != "" && len(labels.Array()) > 0
		hasAtLeastOneSelector = validLabels || validName || validPlan
	}
	if (validID && hasAtLeastOneSelector) || (!validID && !hasAtLeastOneSelector) {
		return util.HandleInstanceSharingError(util.ErrMissingOrInvalidReferenceParameter, "")
	}
	return nil
}

func getSharedInstances(ctx context.Context, repository storage.Repository, offeringID, tenantIdentifier, tenantID string) (types.ObjectList, error) {
	queryParams := map[string]interface{}{
		"tenant_identifier": tenantIdentifier,
		"tenant_id":         tenantID,
		"offering_id":       offeringID,
	}
	objectList, err := repository.QueryForList(ctx, types.ServiceInstanceType, storage.QueryForSharedInstances, queryParams)
	return objectList, err
}

func filterInstancesBySelectors(ctx context.Context, repository storage.Repository, offeringID, tenantIdentifier, tenantValue string, selectors map[string]gjson.Result, smaap bool) (*types.ServiceInstance, error) {
	var err error
	var filteredInstances types.ServiceInstances

	var sharedInstances types.ObjectList

	// Filter by label query:
	labelsSelector, labelExists := selectors[instance_sharing.ReferenceLabelSelectorKey]
	labelsSelectorArray := labelsSelector.Array()
	hasLabelsSelectors := labelExists && len(labelsSelectorArray) > 0 && labelsSelectorArray[0].String() != ""
	if hasLabelsSelectors {
		var criteria []query.Criterion
		for index := 0; index < len(labelsSelectorArray); index++ {
			queryToParse := labelsSelectorArray[index].String()
			criterion, err := query.Parse(query.LabelQuery, queryToParse)
			if err != nil {
				return nil, err
			}
			criteria = append(criteria, criterion...)
		}
		criteria = append(criteria, query.ByLabel(query.EqualsOperator, tenantIdentifier, tenantValue))
		criteria = append(criteria, query.ByField(query.EqualsOperator, "shared", "true"))
		sharedInstancesByLabels, err := getInstanceByLabelSelector(ctx, repository, criteria)
		if err != nil {
			return nil, err
		}
		allowedPlans, err := getPlansOfServiceOffering(ctx, repository, offeringID)
		if err != nil {
			return nil, err
		}
		sharedInstances = filterInstancesByAllowedPlans(sharedInstancesByLabels, allowedPlans)
	} else {
		sharedInstances, err = getSharedInstances(ctx, repository, offeringID, tenantIdentifier, tenantValue)
		if err != nil {
			log.C(ctx).Errorf("Failed to retrieve shared instances: %s", err)
			return nil, err
		}
	}

	// Filter by plan name:
	planSelectorVal, planSelectorExists := selectors[instance_sharing.ReferencePlanNameSelectorKey]
	var selectorPlan *types.ServicePlan
	if planSelectorExists && len(planSelectorVal.String()) > 0 {
		selectorPlan, err = getSelectorPlan(ctx, repository, smaap, offeringID, planSelectorVal.String())
		if err != nil {
			return nil, err
		}
	}

	for i := 0; i < sharedInstances.Len(); i++ {
		instance := sharedInstances.ItemAt(i).(*types.ServiceInstance)

		selectorVal, exists := selectors[instance_sharing.ReferenceInstanceNameSelectorKey]
		if exists && len(selectorVal.String()) > 0 && !(instance.Name == selectorVal.String()) {
			continue
		}

		if selectorPlan != nil && instance.ServicePlanID != selectorPlan.ID {
			continue
		}

		filteredInstances.Add(instance)
		// only single result is accepted for selectors:
		if filteredInstances.Len() > 1 {
			return nil, util.HandleInstanceSharingError(util.ErrMultipleReferenceSelectorResults, "")
		}
	}
	if filteredInstances.Len() == 0 {
		return nil, util.HandleInstanceSharingError(util.ErrNoResultsForReferenceSelector, "")
	}
	return filteredInstances.ItemAt(0).(*types.ServiceInstance), nil
}

func filterInstancesByAllowedPlans(instances types.ObjectList, plans types.ObjectList) types.ObjectList {
	filteredInstances := &types.ServiceInstances{
		ServiceInstances: make([]*types.ServiceInstance, 0),
	}
	for i := 0; i < instances.Len(); i++ {
		instance := instances.ItemAt(i).(*types.ServiceInstance)
		if !isInAllowedPlans(instance, plans) {
			continue
		}
		filteredInstances.Add(instance)
	}
	return filteredInstances
}

func isInAllowedPlans(instance *types.ServiceInstance, plans types.ObjectList) bool {
	for i := 0; i < plans.Len(); i++ {
		plan := plans.ItemAt(i).(*types.ServicePlan)
		if instance.ServicePlanID == plan.ID {
			return true
		}
	}
	return false
}

func getPlansOfServiceOffering(ctx context.Context, repository storage.Repository, offeringID string) (types.ObjectList, error) {
	plans, err := repository.List(ctx, types.ServicePlanType,
		query.ByField(query.EqualsOperator, "service_offering_id", offeringID),
	)
	if err != nil {
		return nil, util.HandleStorageError(err, types.ServicePlanType.String())
	}
	return plans, nil
}

func getSelectorPlan(ctx context.Context, repository storage.Repository, smaap bool, offeringID, selectorValue string) (*types.ServicePlan, error) {
	var err error
	var key string
	if smaap {
		key = "name"
	} else {
		key = "catalog_name"
	}
	planObject, err := repository.Get(ctx, types.ServicePlanType,
		query.ByField(query.EqualsOperator, key, selectorValue),
		query.ByField(query.EqualsOperator, "service_offering_id", offeringID),
	)
	if err != nil {
		return nil, util.HandleInstanceSharingError(util.ErrNoResultsForReferenceSelector, "")
	}
	selectorPlan := planObject.(*types.ServicePlan)
	return selectorPlan, nil
}

func getInstanceByLabelSelector(ctx context.Context, repository storage.Repository, criteria []query.Criterion) (types.ObjectList, error) {
	var err error

	instancesList, err := repository.List(ctx, types.ServiceInstanceType, criteria...)
	if err != nil {
		log.C(ctx).Errorf("Failed to retrieve instance by label selector: %s", err)
		return nil, util.HandleInstanceSharingError(util.ErrNoResultsForReferenceSelector, "")
	}
	if instancesList == nil || instancesList.Len() == 0 {
		return nil, util.HandleInstanceSharingError(util.ErrNoResultsForReferenceSelector, "")
	}
	return instancesList, nil
}

func isSameServiceOffering(ctx context.Context, repository storage.Repository, offeringID string, planID string) (bool, error) {
	count, err := repository.Count(ctx, types.ServicePlanType,
		query.ByField(query.EqualsOperator, "service_offering_id", offeringID),
		query.ByField(query.EqualsOperator, "id", planID),
	)
	if err != nil {
		return false, util.HandleStorageError(err, types.ServicePlanType.String())
	}
	return count > 0, nil
}
