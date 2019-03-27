package osb

import (
	"context"

	"github.com/Peripli/service-manager/storage"
	"github.com/Peripli/service-manager/storage/catalog"

	"github.com/Peripli/service-manager/pkg/types"
)

// StorageCatalogFetcher fetches the broker's catalog from SM DB
type StorageCatalogFetcher struct {
	Repository storage.Repository
}

// FetchCatalog implements osb.CatalogFetcher and fetches the catalog for the broker with the specified broker id from SM DB
func (scf *StorageCatalogFetcher) FetchCatalog(ctx context.Context, brokerID string) (*types.ServiceOfferings, error) {
	result, err := catalog.Load(ctx, brokerID, scf.Repository)
	if err != nil {
		return nil, err
	}
	// SM generates its own ids for the services and plans - currently for the platform we want to provide the original catalog id
	for _, service := range result.ServiceOfferings {
		service.ID = service.CatalogID
		service.Name = service.CatalogName
		for _, plan := range service.Plans {
			plan.ID = plan.CatalogID
			plan.Name = plan.CatalogName
		}
		result.Add(service)
	}
	return result, nil
}
