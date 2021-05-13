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

func ExtractReferenceInstanceID(ctx context.Context, repository storage.Repository, body []byte, tenantIdentifier string, getTenantId func() string) (string, error) {
	parameters := gjson.GetBytes(body, "parameters").Map()
	referencedInstanceID, exists := parameters[instance_sharing.ReferencedInstanceIDKey]

	if !exists {
		return "", util.HandleInstanceSharingError(util.ErrMissingOrInvalidReferenceParameter, instance_sharing.ReferencedInstanceIDKey)
	}

	byLabel := query.ByLabel(query.EqualsOperator, tenantIdentifier, getTenantId())
	referencedInstanceObj, err := storage.GetObjectByField(ctx, repository, types.ServiceInstanceType, "id", referencedInstanceID.String(), byLabel)
	if err != nil {
		log.C(ctx).Errorf("Failed to retrieve the instance %s by the caller %s", referencedInstanceID, getTenantId())
		return "", util.HandleInstanceSharingError(util.ErrReferencedInstanceNotFound, referencedInstanceID.String())
	}
	referencedInstance := referencedInstanceObj.(*types.ServiceInstance)

	if !referencedInstance.IsShared() {
		log.C(ctx).Debugf("The target instance %s is not shared.", referencedInstance.ID)
		return "", util.HandleInstanceSharingError(util.ErrReferencedInstanceNotShared, referencedInstance.ID)
	}

	return referencedInstanceID.String(), nil
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
