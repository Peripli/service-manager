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
	planSelector = "plan_selector"
	nameSelector = "name_selector"
)

func ExtractReferenceInstanceID(ctx context.Context, repository storage.Repository, body []byte, tenantIdentifier string, getTenantId func() string) (string, error) {
	parameters := gjson.GetBytes(body, "parameters").Map()

	referencedInstanceID := parameters[instance_sharing.ReferencedInstanceIDKey].String()
	planID := parameters[planSelector].String()
	instanceName := parameters[nameSelector].String()

	var criteria []query.Criterion
	criteria = append(criteria, query.ByLabel(query.EqualsOperator, tenantIdentifier, getTenantId()))
	criteria = append(criteria, query.CriteriaForContext(ctx)...)

	if len(referencedInstanceID) > 0 {
		criteria = append(criteria, query.ByField(query.EqualsOperator, "id", referencedInstanceID))
	} else if len(planID) > 0 {
		//criteria = append(criteria, query.ByField(query.EqualsOperator, "shared", "true"))
		criteria = append(criteria, query.ByField(query.EqualsOperator, "service_plan_id", planID))
	} else if len(instanceName) > 0 {
		//criteria = append(criteria, query.ByField(query.EqualsOperator, "shared", "true"))
		criteria = append(criteria, query.ByField(query.EqualsOperator, "name", instanceName))
	} else {
		return "", util.HandleInstanceSharingError(util.ErrMissingOrInvalidReferenceParameter, instance_sharing.ReferencedInstanceIDKey)
	}

	referencedInstanceObj, err := repository.List(ctx, types.ServiceInstanceType, criteria...)
	if err != nil {
		return "", err
	}
	if referencedInstanceObj.Len() != 0 {
		return "", util.HandleInstanceSharingError(util.ErrReferencedInstanceNotFound, "")
	}
	if referencedInstanceObj.Len() != 1 {
		return "", util.HandleInstanceSharingError(util.ErrMultipleReferenceSelectorResults, "")
	}
	referencedInstance := referencedInstanceObj.ItemAt(0).(*types.ServiceInstance)
	//referencedInstanceObj, err := storage.GetObjectByField(ctx, repository, types.ServiceInstanceType, "id", referencedInstanceID.String(), byLabel)
	//if err != nil {
	//	log.C(ctx).Errorf("Failed to retrieve the instance %s by the caller %s", referencedInstanceID, getTenantId())
	//	return "", util.HandleInstanceSharingError(util.ErrReferencedInstanceNotFound, referencedInstanceID.String())
	//}
	//referencedInstance := referencedInstanceObj.(*types.ServiceInstance)

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
