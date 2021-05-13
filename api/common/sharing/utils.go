package sharing

import (
	"context"
	"github.com/Peripli/service-manager/pkg/instance_sharing"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
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
		log.C(ctx).Errorf("Failed retrieving the reference-instance by the ID: %s", referencedInstanceObj)
		return "", util.HandleStorageError(util.ErrNotFoundInStorage, types.ServiceInstanceType.String())
	}
	referencedInstance := referencedInstanceObj.(*types.ServiceInstance)

	if referencedInstance == nil {
		log.C(ctx).Errorf("Failed to retrieve the instance %s by the caller %s", referencedInstanceID, getTenantId())
		return "", util.HandleStorageError(util.ErrNotFoundInStorage, types.ServiceInstanceType.String())
	}

	if !referencedInstance.IsShared() {
		log.C(ctx).Debugf("The target instance %s is not shared.", referencedInstance.ID)
		return "", util.HandleInstanceSharingError(util.ErrReferencedInstanceNotShared, referencedInstance.ID)
	}

	return referencedInstanceID.String(), nil
}
