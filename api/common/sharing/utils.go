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

const (
	planSelector  = "plan_name_selector"
	nameSelector  = "instance_name_selector"
	labelSelector = "instance_labels_selector"
)

func ExtractReferenceInstanceID(req *web.Request, repository storage.Repository, body []byte, tenantIdentifier string, getTenantId func() string, isSMAAP bool) (string, error) {
	ctx := req.Context()
	referencePlan, _ := types.PlanFromContext(ctx)
	// todo: which flow we run (osb / smaap) which handles the plan_selector
	parameters := gjson.GetBytes(body, "parameters").Map()

	referencedInstanceID := parameters[instance_sharing.ReferencedInstanceIDKey].String()
	planNameSelector := parameters[planSelector].String()
	instanceNameSelector := parameters[nameSelector].String()

	var criteria []query.Criterion
	criteria = append(criteria, query.ByLabel(query.EqualsOperator, tenantIdentifier, getTenantId()))
	criteria = append(criteria, query.CriteriaForContext(ctx)...)

	// by service offering
	if referencedInstanceID == "*" {
		criteria = append(criteria, query.ByField(query.EqualsOperator, "shared", "true"))
		selectorPlan, err := retrievePlanBySelector(ctx, repository, referencePlan.ServiceOfferingID, planNameSelector, isSMAAP)
		if err != nil {
			return "", err
		}
		criteria = append(criteria, query.ByField(query.EqualsOperator, "service_plan_id", selectorPlan.ID))
	} else if len(referencedInstanceID) > 0 {
		criteria = append(criteria, query.ByField(query.EqualsOperator, "id", referencedInstanceID))
	} else if len(planNameSelector) > 0 {
		selectorPlan, err := retrievePlanBySelector(ctx, repository, referencePlan.ServiceOfferingID, planNameSelector, isSMAAP)
		if err != nil {
			return "", err
		}
		criteria = append(criteria, query.ByField(query.EqualsOperator, "shared", "true"))
		criteria = append(criteria, query.ByField(query.EqualsOperator, "service_plan_id", selectorPlan.ID))
	} else if len(instanceNameSelector) > 0 {
		criteria = append(criteria, query.ByField(query.EqualsOperator, "shared", "true"))
		criteria = append(criteria, query.ByField(query.EqualsOperator, "service_offering_id", referencePlan.ServiceOfferingID))
		criteria = append(criteria, query.ByField(query.EqualsOperator, "name", instanceNameSelector))
	} else {
		return "", util.HandleInstanceSharingError(util.ErrMissingOrInvalidReferenceParameter, instance_sharing.ReferencedInstanceIDKey)
	}

	referencedInstanceObj, err := repository.List(ctx, types.ServiceInstanceType, criteria...)
	if err != nil {
		return "", err
	}
	if referencedInstanceObj.Len() == 0 {
		// todo: add a new error: not found a shared instance which meets your criteria
		return "", util.HandleInstanceSharingError(util.ErrReferencedInstanceNotFound, "")
	}
	if referencedInstanceObj.Len() != 1 {
		// there is more than one shared instance that meets your criteria
		return "", util.HandleInstanceSharingError(util.ErrMultipleReferenceSelectorResults, "")
	}
	referencedInstance := referencedInstanceObj.ItemAt(0).(*types.ServiceInstance)

	// If the selector is instanceID, we would like to inform the user if the target instance was not shared:
	if !referencedInstance.IsShared() {
		log.C(ctx).Debugf("The target instance %s is not shared.", referencedInstance.ID)
		return "", util.HandleInstanceSharingError(util.ErrReferencedInstanceNotShared, referencedInstance.ID)
	}

	return referencedInstance.ID, nil
}

func retrievePlanBySelector(ctx context.Context, repository storage.Repository, offeringID string, planName string, smaap bool) (*types.ServicePlan, error) {
	planNameKey := "name"
	if !smaap {
		planNameKey = "catalog_name"
	}
	var criteria []query.Criterion
	criteria = append(criteria, query.ByField(query.EqualsOperator, "service_offering_id", offeringID))
	criteria = append(criteria, query.ByField(query.EqualsOperator, planNameKey, planName))

	plansObj, err := repository.List(ctx, types.ServiceInstanceType, criteria...)
	if err != nil {
		return nil, err
	}
	if plansObj.Len() == 0 {
		// todo: add a new error: not found a shared instance which meets your criteria
		return nil, util.HandleInstanceSharingError(util.ErrReferencedInstanceNotFound, "")
	}
	if plansObj.Len() > 1 {
		// there is more than one shared instance that meets your criteria
		return nil, util.HandleInstanceSharingError(util.ErrMultipleReferenceSelectorResults, "")
	}
	return plansObj.ItemAt(0).(*types.ServicePlan), nil
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
