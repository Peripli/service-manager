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

	// SM generates its own ids for the services and plans - currently for the platform we want to provide the original catalog id
	for _, service := range catalog {
		service.ID = service.CatalogID
		service.Name = service.CatalogName
		for _, plan := range service.Plans {
			plan.ID = plan.CatalogID
			plan.Name = plan.CatalogName
		}
	}
	return &types.ServiceOfferings{
		ServiceOfferings: catalog,
	}, nil
}
