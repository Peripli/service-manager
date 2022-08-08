package blocked_clients

import (
	"context"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
	"time"
)

type BlockedClientsManager struct {
	repository storage.Repository
	ctx        context.Context
	Cache      *storage.Cache
}

func Init(ctx context.Context, repository storage.Repository) *BlockedClientsManager {
	b := &BlockedClientsManager{ctx: ctx, repository: repository}
	b.Cache = storage.NewCache(time.Minute*5, b.getBlockClients)
	return b
}

func (b *BlockedClientsManager) getBlockClients() error {
	blockedClientsList, err := b.repository.List(b.ctx, types.BlockedClientsType)
	if err != nil {
		return err
	}
	b.Cache.Flush()
	for i := 0; i < blockedClientsList.Len(); i++ {
		blockedClient := blockedClientsList.ItemAt(i).(*types.BlockedClient)
		b.Cache.Add(blockedClient.ClientID, blockedClient)
	}
	return nil
}
