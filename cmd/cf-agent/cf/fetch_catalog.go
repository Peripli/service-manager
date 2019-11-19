package cf

import (
	"context"

	"github.com/Peripli/service-manager/pkg/agent/platform"
)

// Fetch implements pkg/agent/platform/CatalogFetcher.Fetch and provides logic for triggering refetching
// of the broker's catalog
func (pc *PlatformClient) Fetch(ctx context.Context, broker *platform.ServiceBroker) error {
	_, err := pc.UpdateBroker(ctx, &platform.UpdateServiceBrokerRequest{
		GUID:      broker.GUID,
		Name:      broker.Name,
		BrokerURL: broker.BrokerURL,
	})

	return err
}
