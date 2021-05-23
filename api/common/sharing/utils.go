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

const ()

func ExtractReferenceInstanceID(req *web.Request, repository storage.Repository, body []byte, tenantIdentifier string, getTenantId func() string, isSMAAP bool) (string, error) {
	var err error
	ctx := req.Context()
	referencePlan, _ := types.PlanFromContext(ctx)
	parameters := gjson.GetBytes(body, "parameters").Map()

	selectorResult, err := getInstanceBySelector(ctx, repository, parameters, isSMAAP, tenantIdentifier, getTenantId(), referencePlan.ServiceOfferingID)

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

func getInstanceBySelector(ctx context.Context, repository storage.Repository, parameters map[string]gjson.Result, isSMAAP bool, tenantIdentifier, tenantID, offeringID string) (types.ObjectList, error) {
	var objectList types.ObjectList
	var err error
	params := map[string]interface{}{
		"tenant_identifier": tenantIdentifier,
		"tenant_id":         tenantID,
		"offering_id":       offeringID,
	}
	var namedQuery storage.NamedQuery

	labelsSelector := parameters[instance_sharing.ReferenceLabelSelector]

	referencedInstanceID := parameters[instance_sharing.ReferencedInstanceIDKey].String()
	planNameSelector := parameters[instance_sharing.ReferencePlanNameSelector].String()
	instanceNameSelector := parameters[instance_sharing.ReferenceInstanceNameSelector].String()

	if referencedInstanceID == "*" {
		namedQuery = storage.QueryForReferenceBySharedInstanceSelector
	} else if len(referencedInstanceID) > 1 {
		params["selector_value"] = referencedInstanceID
		namedQuery = storage.QueryForReferenceByInstanceID
	} else if len(planNameSelector) > 0 {
		params["selector_value"] = planNameSelector
		if isSMAAP {
			namedQuery = storage.QueryForSMAAPReferenceByPlanSelector
		} else {
			namedQuery = storage.QueryForOSBReferenceByPlanSelector
		}
	} else if len(instanceNameSelector) > 0 {
		params["selector_value"] = instanceNameSelector
		namedQuery = storage.QueryForReferenceByNameSelector
	} else if len(labelsSelector.Raw) > 0 {
		var criteria []query.Criterion
		for labelKey, labelObject := range labelsSelector.Map() {
			for _, labelValue := range labelObject.Array() {
				criteria = append(criteria, query.ByLabel(query.EqualsOperator, labelKey, labelValue.String()))
			}
		}
	}

	objectList, err = repository.QueryForList(ctx, types.ServiceInstanceType, namedQuery, params)
	if err != nil {
		return nil, util.HandleStorageError(err, types.ServiceInstanceType.String())
	}

	return objectList, nil
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
