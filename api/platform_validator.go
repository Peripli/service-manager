package api

import (
	"context"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/storage"
)

// PlatformValidator is a type of ResourceValidator
type PlatformValidator struct {
	DefaultResourceValidator
}

// ValidateDelete ensures that there are no existing service instances prior to deleting of a platform
func (pv *PlatformValidator) ValidateDelete(ctx context.Context, repository storage.Repository, object types.Object) error {
	_, ok := object.(*types.Platform)
	if !ok {
		log.C(ctx).Debugf("Provided object is of type %s. Cannot validate platform deletion.", object.GetType())
		return ErrIncompatibleObjectType
	}

	log.C(ctx).Debugf("Fetching service instances for platform with ID (%s)", object.GetID())
	byPlatformIDs := query.ByField(query.EqualsOperator, "platform_id", object.GetID())
	instanceIDs, err := retrieveServiceInstanceIDsByCriteria(ctx, repository, byPlatformIDs)
	if err != nil {
		return err
	}

	if len(instanceIDs) > 0 {
		log.C(ctx).Infof("Found service instances associated with platform with ID (%s): %s", object.GetID(), instanceIDs)
		return &util.ErrForeignKeyViolation{
			Entity:             object.GetType().String(),
			ReferenceEntity:    types.ServiceInstanceType.String(),
			ReferenceEntityIDs: instanceIDs,
		}
	}

	log.C(ctx).Debugf("No service instances associated with platform with ID (%s) were found. Platform deletion can continue...", object.GetID())
	return nil
}
