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

func ExtractReferencedInstanceID(ctx context.Context, repository storage.Repository, body []byte, tenantIdentifier string, getTenantId func() string) (string, error) {
	parameters := gjson.GetBytes(body, "parameters").Map()
	referencedInstanceID, exists := parameters[instance_sharing.ReferencedInstanceIDKey]

	if !exists {
		return "", util.HandleInstanceSharingError(util.ErrMissingOrInvalidReferenceParameter, instance_sharing.ReferencedInstanceIDKey)
	}

	referencedInstanceObj, err := repository.Get(ctx, types.ServiceInstanceType,
		query.ByLabel(query.EqualsOperator, tenantIdentifier, getTenantId()),
		query.ByField(query.EqualsOperator, "id", referencedInstanceID.String()),
	)
	if err != nil {
		log.C(ctx).Errorf("failed retrieving service instance %s: %v", referencedInstanceID.String(), err)
		return "", util.HandleInstanceSharingError(util.ErrReferencedInstanceNotFound, referencedInstanceID.String())
	}
	referencedInstance := referencedInstanceObj.(*types.ServiceInstance)

	referencePlan, ok := types.PlanFromContext(ctx)
	if !ok || referencePlan == nil {
		return "", util.HandleStorageError(util.ErrNotFoundInStorage, types.ServicePlanType.String())
	}
	// verify the reference plan and the target instance are of the same service offering:
	sameOffering, err := isSameServiceOffering(ctx, repository, referencePlan.ServiceOfferingID, referencedInstance.ServicePlanID)
	if err != nil {
		return "", err
	}
	if !sameOffering {
		log.C(ctx).Debugf("The target instance %s is not of the same service offering.", referencedInstance.ID)
		return "", util.HandleInstanceSharingError(util.ErrReferenceWithWrongServiceOffering, referencedInstance.ID)
	}

	if !referencedInstance.IsShared() {
		log.C(ctx).Debugf("The target instance %s is not shared.", referencedInstance.ID)
		return "", util.HandleInstanceSharingError(util.ErrReferencedInstanceNotShared, referencedInstance.ID)
	}

	return referencedInstanceID.String(), nil
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
