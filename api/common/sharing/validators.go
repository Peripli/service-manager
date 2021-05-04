package sharing

import (
	"context"
	"github.com/Peripli/service-manager/pkg/instance_sharing"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
	"github.com/tidwall/gjson"
)

func ValidateReferencedInstance(body []byte, tenantIdentifier string, repository storage.Repository, ctx context.Context, getTenantId func() string) (string, error) {
	parameters := gjson.GetBytes(body, "parameters").Map()
	referencedInstanceID, exists := parameters[instance_sharing.ReferencedInstanceIDKey]

	if !exists {
		return "", util.HandleInstanceSharingError(util.ErrMissingOrInvalidReferenceParameter, instance_sharing.ReferencedInstanceIDKey)
	}

	referencedInstanceObj, err := storage.GetObjectByField(ctx, repository, types.ServiceInstanceType, "id", referencedInstanceID.String())
	if err != nil {
		log.C(ctx).Errorf("Failed retrieving the reference-instance by the ID: %s", referencedInstanceObj)
		return "", util.HandleStorageError(util.ErrNotFoundInStorage, types.ServiceInstanceType.String())
	}
	referencedInstance := referencedInstanceObj.(*types.ServiceInstance)

	//validate Ownership in case of a multi tenant flow
	if tenantIdentifier != "" {
		targetInstanceTenantID := referencedInstance.Labels[tenantIdentifier][0]
		if targetInstanceTenantID != getTenantId() {
			log.C(ctx).Errorf("Instance owner %s is not the same as the caller %s", targetInstanceTenantID, getTenantId())
			return "", util.HandleStorageError(util.ErrNotFoundInStorage, types.ServiceInstanceType.String())
		}
	}

	if !referencedInstance.IsShared() {
		log.C(ctx).Debugf("The target instance %s is not shared.", referencedInstance.ID)
		return "", util.HandleInstanceSharingError(util.ErrReferencedInstanceNotShared, referencedInstance.ID)
	}

	return referencedInstanceID.String(), nil
}

func IsValidReferenceInstancePatchRequest(req *web.Request, instance *types.ServiceInstance, planIDProperty string) error {
	// epsilontal todo: How can we update labels and do we want to allow the change?
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
