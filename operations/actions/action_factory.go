package actions

import (
	"context"
	"fmt"
	"github.com/Peripli/service-manager/operations"
	"github.com/Peripli/service-manager/operations/opcontext"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/services"
	"github.com/Peripli/service-manager/storage"
)

type Factory struct {
	BrokerService services.BrokerService
	Repository    storage.Repository
}

type RunnableAction interface {
	isRunnable() bool
}

func (factory Factory) GetAction(ctx context.Context, entity types.Object, action operations.StorageAction) operations.StorageAction {

	return func(ctx context.Context, repository storage.Repository) (types.Object, error) {
		if _, ok := entity.(RunnableAction); ok {
			operation, found := opcontext.Get(ctx)
			if !found {
				return nil, fmt.Errorf("operation missing from context")
			}

			return ServiceInstanceActions{
				brokerService: factory.BrokerService,
				repository:    factory.Repository,
			}.RunActionByOperation(ctx, entity, *operation)

		}

		return action(ctx, repository);
	}
}
