package operations

import (
	"context"
	"fmt"
	"github.com/Peripli/service-manager/operations/opcontext"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
)

type InstanceActions interface {
	RunActionByOperation(ctx context.Context, entity types.Object, operation types.Operation) (types.Object, error)
}

type Factory struct {
	SupportedActions map[types.ObjectType]InstanceActions
}

type RunnableAction interface {
	isRunnable() bool
}


func (factory Factory) GetAction(ctx context.Context, entity types.Object, action StorageAction) StorageAction {
	return func(ctx context.Context, repository storage.Repository) (types.Object, error) {
		if entityActions, ok := factory.SupportedActions[entity.GetType()]; ok {
			operation, found := opcontext.Get(ctx)
			if !found {
				return nil, fmt.Errorf("operation missing from context")
			}

			return entityActions.RunActionByOperation(ctx, entity, *operation)
		}

		return action(ctx, repository);
	}
}
