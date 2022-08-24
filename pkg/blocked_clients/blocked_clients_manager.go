package blocked_clients

import (
	"context"
	"encoding/json"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
	"github.com/Peripli/service-manager/storage/events"
)

type BlockedClientsManager struct {
	repository             storage.Repository
	smCtx                  context.Context
	Cache                  *storage.Cache
	callbacks              map[string]func(*events.Message) error
	postgresEventsListener *events.PostgresEventListener
}

func Init(ctx context.Context, repository storage.Repository, storageURI string) *BlockedClientsManager {
	b := &BlockedClientsManager{smCtx: ctx, repository: repository}
	b.Cache = storage.NewCache(-1, nil, b.getBlockClients)
	b.getBlockClients()
	b.callbacks = map[string]func(*events.Message) error{
		"blocked_clients-INSERT": func(envelope *events.Message) error {
			var blockedClient types.BlockedClient

			if err := json.Unmarshal(envelope.Data, &blockedClient); err != nil {
				log.C(ctx).Debugf("error unmarshalling new blocked client")
				return err
			}

			if err := b.Cache.AddL(blockedClient.ClientID, blockedClient); err != nil {
				log.C(ctx).Debugf("error adding a blocked client in casche %s", blockedClient.ClientID)
			}
			return nil
		},
		"blocked_clients-DELETE": func(envelope *events.Message) error {
			var blockedClient types.BlockedClient
			if err := json.Unmarshal(envelope.Data, &blockedClient); err != nil {
				log.C(ctx).Debugf("error unmarshalling new blocked client")
				return err
			}

			b.Cache.DeleteL(blockedClient.ClientID)
			return nil
		},
	}

	b.postgresEventsListener = events.NewPostgresEventListener(ctx, storageURI, b.callbacks)
	return b
}

func (b *BlockedClientsManager) getBlockClients() error {
	blockedClientsList, err := b.repository.List(b.smCtx, types.BlockedClientsType)
	if err != nil {
		log.C(b.smCtx).Info("error retrieving blocked clients", err)
		return err
	}
	for i := 0; i < blockedClientsList.Len(); i++ {
		blockedClient := blockedClientsList.ItemAt(i).(*types.BlockedClient)
		b.Cache.Add(blockedClient.ClientID, blockedClient)
	}
	return nil
}
