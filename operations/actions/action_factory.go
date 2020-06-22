package actions

import (
	"context"
	"fmt"
	"github.com/Peripli/service-manager/operations/opcontext"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/services"
	"github.com/Peripli/service-manager/storage"
)

type Factory struct {
	BrokerService services.BrokerService
	Repository    storage.Repository
}

func (factory Factory) RunAction(ctx context.Context, entity types.Object) (types.Object, error) {
	if entity.GetType() == types.ServiceInstanceType {
		newAction := ServiceInstanceActions{
			brokerService: factory.BrokerService,
			repository:    factory.Repository,
		}

		operation, found := opcontext.Get(ctx)
		if !found {
			return nil, fmt.Errorf("operation missing from context")
		}

		return newAction.RunActionByOperation(ctx, entity, *operation)
	}

	return factory.Repository.Create(ctx, entity)
}
