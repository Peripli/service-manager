package osb

import (
	"context"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
)

// StorageCatalogFetcher fetches the broker's catalog from SM DB
type StorageCatalogFetcher struct {
	CatalogStorage storage.ServiceOffering
}

// FetchCatalog implements osb.CatalogFetcher and fetches the catalog for the broker with the specified broker id from SM DB
func (scf *StorageCatalogFetcher) FetchCatalog(ctx context.Context, brokerID string) (*types.ServiceOfferings, error) {
	catalog, err := scf.CatalogStorage.ListWithServicePlansByBrokerID(ctx, brokerID)
	if err != nil {
		return nil, err
	}
	return &types.ServiceOfferings{
		ServiceOfferings: catalog,
	}, nil
}
