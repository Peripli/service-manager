package operations

import (
	"context"
	"errors"
	"fmt"
	"github.com/Peripli/service-manager/operations/opcontext"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
	"time"
)

type InstanceActions interface {
	RunActionByOperation(ctx context.Context, entity types.Object, operation types.Operation) (types.Object, error)
}
type SyncBus struct {
	Entity types.Object
	Err    error
}

type ChanItem struct {
	Channel     chan SyncBus
	Duration    time.Duration
	ChanContext context.Context
}

type SyncEventBus struct {
	scheduledOperations map[string][]ChanItem
}

func (se *SyncEventBus) removeFromEventBus(id string, chanHolder ChanItem) {
	if _, ok := se.scheduledOperations[id]; ok {
		for i := range se.scheduledOperations[id] {
			if se.scheduledOperations[id][i] == chanHolder {
				se.scheduledOperations[id] = append(se.scheduledOperations[id][:i], se.scheduledOperations[id][i+1:]...)
				break
			}
		}
	}
}

func (se *SyncEventBus) AddListener(id string, objectsChan chan SyncBus, ctx context.Context) {

	if se.scheduledOperations == nil {
		se.scheduledOperations = make(map[string][]ChanItem)
	}

	chanItem := ChanItem{
		Channel:     objectsChan,
		Duration:    30 * time.Minute,
		ChanContext: nil,
	}

	go se.withChannelWatch(id, chanItem, ctx)

	if _, ok := se.scheduledOperations[id]; ok {
		se.scheduledOperations[id] = append(se.scheduledOperations[id], chanItem)
	} else {
		se.scheduledOperations[id] = []ChanItem{chanItem}
	}

	print(se.scheduledOperations[id])
}

func (se *SyncEventBus) withChannelWatch(indexId string, chanItem ChanItem, ctx context.Context) {
	maxExecutionTime := time.NewTicker(chanItem.Duration)
	defer maxExecutionTime.Stop()
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			se.NotifyCompleted(indexId, SyncBus{
				Entity: nil,
				Err:    errors.New("the context is done, either because SM crashed/exited or because action timeout elapsed"),
			})
			se.removeFromEventBus(indexId, chanItem)
			return
		case <-maxExecutionTime.C:
			se.NotifyCompleted(indexId, SyncBus{
				Entity: nil,
				Err:    errors.New("the maximum execution time for this even has been reached"),
			})
			se.removeFromEventBus(indexId, chanItem)
			return
		}
	}
}

func (se *SyncEventBus) NotifyCompleted(id string, object SyncBus) {
	if _, ok := se.scheduledOperations[id]; ok {
		for _, handler := range se.scheduledOperations[id] {
			go func(handler chan SyncBus) {
				handler <- object
			}(handler.Channel)
		}
	}
}

type Factory struct {
	SupportedActions map[types.ObjectType]InstanceActions
	EventBus         *SyncEventBus
}

type RunnableAction interface {
	isRunnable() bool
}

func (factory Factory) WithSyncActions(ctx context.Context) context.Context {
	return context.WithValue(ctx, SyncBus{}, factory.EventBus)
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
