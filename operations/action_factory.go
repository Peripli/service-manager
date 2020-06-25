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
type SyncBusKey struct {}
type SyncEventBus struct {
	scheduledOperations map[string][]chan types.Object
}

func (se *SyncEventBus) AddListener(id string, objectsChan chan types.Object) {

	if se.scheduledOperations == nil {
		se.scheduledOperations = make(map[string][]chan types.Object)
	}

	if _, ok := se.scheduledOperations[id]; ok {
		se.scheduledOperations[id] = append(se.scheduledOperations[id], objectsChan)
	} else {
		se.scheduledOperations[id] = []chan types.Object{objectsChan}
	}

	print (se.scheduledOperations[id])
}

func (se *SyncEventBus) NotifyCompleted(id string,object types.Object) {
	if _, ok := se.scheduledOperations[id]; ok {
		for _, handler := range se.scheduledOperations[id] {
			go func(handler chan types.Object) {
				handler <- object
			}(handler)
		}
	}
}

type Factory struct {
	SupportedActions map[types.ObjectType]InstanceActions
	EventBus *SyncEventBus
}

type RunnableAction interface {
	isRunnable() bool
}

func (factory Factory) WithSyncActions(ctx context.Context) context.Context{
	return context.WithValue(ctx, SyncBusKey{}, factory.EventBus)
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
