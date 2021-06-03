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
	if exists && referencedInstanceID.String() != "*" {
		referencedInstance, err = getInstanceByID(ctx, repository, referencedInstanceID.String(), referencePlan.ServiceOfferingID, tenantIdentifier, getTenantId())
		if err != nil {
			log.C(ctx).Errorf("Failed to retrieve the instance %s, error: %s", referencedInstanceID.String(), err)
			return "", err
		}
	} else {
		sharedInstances, err := getSharedInstances(ctx, repository, referencePlan.ServiceOfferingID, tenantIdentifier, getTenantId())
		if err != nil {
			log.C(ctx).Errorf("Failed to retrieve shared instances: %s", err)
			return "", err
		}

		filteredInstancesList, err := filterInstancesBySelectors(ctx, repository, sharedInstances, parameters, smaap, referencePlan.ServiceOfferingID)
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

func getInstanceByID(ctx context.Context, repository storage.Repository, instanceID, offeringID, tenantIdentifier, tenantValue string) (*types.ServiceInstance, error) {
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
	if len(parameters) == 0 {
		return util.HandleInstanceSharingError(util.ErrMissingOrInvalidReferenceParameter, instance_sharing.ReferencedInstanceIDKey)
	}
	// don't allow combination of id with selectors.
	// parameters should not be empty, but the values of each parameter can be ""
	ID, IDExists := parameters[instance_sharing.ReferencedInstanceIDKey]
	name, nameExists := parameters[instance_sharing.ReferenceInstanceNameSelector]
	plan, planExists := parameters[instance_sharing.ReferencePlanNameSelector]
	labels, labelsExists := parameters[instance_sharing.ReferenceLabelSelector]
	var selectorLabels types.Labels
	if err := util.BytesToObject([]byte(labels.Raw), &selectorLabels); labelsExists && err != nil {
		return err
	}
	hasAtLeastOneSelector := selectorLabels != nil && len(selectorLabels) > 0 || nameExists && len(name.String()) > 0 || planExists && len(plan.String()) > 0
	if IDExists && len(ID.String()) > 0 && hasAtLeastOneSelector {
		return util.HandleInstanceSharingError(util.ErrInvalidReferenceSelectors, "")
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

func filterInstancesBySelectors(ctx context.Context, repository storage.Repository, instances types.ObjectList, parameters map[string]gjson.Result, smaap bool, offeringID string) (types.ServiceInstances, error) {
	var err error
	var filteredInstances types.ServiceInstances

	planSelectorVal, planSelectorExists := parameters[instance_sharing.ReferencePlanNameSelector]
	var selectorPlan *types.ServicePlan
	if planSelectorExists {
		selectorPlan, err = getSelectorPlan(ctx, repository, smaap, offeringID, planSelectorVal.String())
		if err != nil {
			log.C(ctx).Errorf("Failed to retrieve the plan %s with the offering %s via the selector provided: %s", planSelectorVal.String(), offeringID, err)
			return types.ServiceInstances{}, err
		}
	}

	for i := 0; i < instances.Len(); i++ {

		// "*" for any shared instance of the tenant, if parameter not passed, set true for any shared instance
		selectorVal, exists := parameters[instance_sharing.ReferencedInstanceIDKey]
		if exists && selectorVal.String() != "*" {
			continue
		}

		instance := instances.ItemAt(i).(*types.ServiceInstance)

		selectorVal, exists = parameters[instance_sharing.ReferenceInstanceNameSelector]
		if exists && len(selectorVal.String()) > 0 && !(instance.Name == selectorVal.String()) {
			continue
		}

		if planSelectorExists && (selectorPlan == nil || instance.ServicePlanID != selectorPlan.ID) {
			continue
		}

		selectorVal, exists = parameters[instance_sharing.ReferenceLabelSelector]
		if exists && len(selectorVal.String()) > 0 {
			match, err := matchLabels(instance.Labels, []byte(selectorVal.Raw))
			if err != nil {
				return types.ServiceInstances{}, err
			}
			if !match {
				continue
			}
		}

		filteredInstances.Add(instance)
		// only single result is accepted for selectors:
		if filteredInstances.Len() > 1 {
			log.C(ctx).Errorf("%s", util.ErrMultipleReferenceSelectorResults)
			return types.ServiceInstances{}, util.HandleInstanceSharingError(util.ErrMultipleReferenceSelectorResults, "")
		}
	}
	if filteredInstances.Len() == 0 {
		log.C(ctx).Errorf("%s", util.ErrNoResultsForReferenceSelector)
		return types.ServiceInstances{}, util.HandleInstanceSharingError(util.ErrNoResultsForReferenceSelector, "")
	}
	return filteredInstances, nil
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
		return nil, util.HandleInstanceSharingError(util.ErrReferencedInstanceNotFound, "")
	}
	selectorPlan := planObject.(*types.ServicePlan)
	return selectorPlan, nil
}

func matchLabels(instanceLabels types.Labels, labels []byte) (bool, error) {
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

	for selectorLabelKey, selectorLabelArray := range selectorLabels {
		instanceLabelVal, exists := instanceLabels[selectorLabelKey]
		if exists && instanceLabels != nil {
			match := true
			for _, selectorLabelVal := range selectorLabelArray {
				match = match && contains(instanceLabelVal, selectorLabelVal)
				if !match {
					return false, nil
				}
			}
		} else {
			return false, nil
		}
	}

	return true, nil
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
