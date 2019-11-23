package api

import (
	"context"
	"errors"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/storage"
	"net/http"
)

// ResourceValidator allows plugging custom validation logic prior to executing CUD API requests
type ResourceValidator interface {
	ValidateCreate(ctx context.Context, repository storage.Repository, object types.Object) error
	ValidateUpdate(ctx context.Context, repository storage.Repository, object types.Object) error
	ValidateDelete(ctx context.Context, repository storage.Repository, object types.Object) error
}

var ErrIncompatibleObjectType = errors.New("incompatible object type provided")

type DefaultResourceValidator struct{}

func (drv *DefaultResourceValidator) ValidateCreate(_ context.Context, _ storage.Repository, object types.Object) error {
	if err := object.Validate(); err != nil {
		return &util.HTTPError{
			ErrorType:   "BadRequest",
			Description: err.Error(),
			StatusCode:  http.StatusBadRequest,
		}
	}
	return nil
}

func (drv *DefaultResourceValidator) ValidateUpdate(_ context.Context, _ storage.Repository, object types.Object) error {
	if err := object.Validate(); err != nil {
		return &util.HTTPError{
			ErrorType:   "BadRequest",
			Description: err.Error(),
			StatusCode:  http.StatusBadRequest,
		}
	}
	return nil
}

func (drv *DefaultResourceValidator) ValidateDelete(_ context.Context, _ storage.Repository, object types.Object) error {
	return nil
}

func retrieveServiceInstanceIDsByCriteria(ctx context.Context, repository storage.Repository, criteria ...query.Criterion) ([]string, error) {
	objectList, err := repository.List(ctx, types.ServiceInstanceType, criteria...)
	if err != nil {
		return nil, util.HandleStorageError(err, string(types.ServiceInstanceType))
	}

	instanceIDs := make([]string, 0)
	for i := 0; i < objectList.Len(); i++ {
		instanceIDs = append(instanceIDs, objectList.ItemAt(i).GetID())
	}

	return instanceIDs, nil
}
