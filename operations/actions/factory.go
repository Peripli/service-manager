package actions

import (
	"context"
	"fmt"
	"github.com/Peripli/service-manager/operations/opcontext"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/storage"
)

type StorageAction func(ctx context.Context, repository storage.Repository) (types.Object, error)

type ScheduledActions interface {
	RunActionByOperation(ctx context.Context, entity types.Object, operation types.Operation) (types.Object, error)
	WithRepository(repository storage.Repository) ServiceInstanceActions
}

type ScheduledActionsProvider struct {
	SupportedActions map[types.ObjectType]ScheduledActions
	EventBus         *SyncEventBus
}

type RunnableAction interface {
	isRunnable() bool
}

func (factory ScheduledActionsProvider) GetContextWithEventBus(ctx context.Context) context.Context {
	return context.WithValue(ctx, Notification{}, factory.EventBus)
}

func (factory ScheduledActionsProvider) GetAction(ctx context.Context, entity types.Object, action StorageAction) StorageAction {
	return func(ctx context.Context, repository storage.Repository) (types.Object, error) {
		if entityActions, ok := factory.SupportedActions[entity.GetType()]; ok {

			span, ctx := util.CreateChildSpan(ctx,fmt.Sprintf("Getting action for entity of type-%s",entity.GetType()));
			defer span.FinishSpan()

			operation, found := opcontext.Get(ctx)
			if !found {
				return nil, fmt.Errorf("operation missing from context")
			}

			return entityActions.WithRepository(repository).RunActionByOperation(ctx, entity, *operation)
		}

		return action(ctx, repository);
	}
}
