package osb

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/storage"

	"github.com/Peripli/service-manager/pkg/web"
)

type StorageCatalogFetcher struct {
	CatalogStorage storage.ServiceOffering
}

func (scf *StorageCatalogFetcher) FetchCatalog(ctx context.Context, brokerID string) (*web.Response, error) {
	catalog, err := scf.CatalogStorage.ListWithServicePlansByBrokerID(ctx, brokerID)
	if err != nil {
		return nil, fmt.Errorf("could not fetch catalog for broker with id %s from SMDB: %s", brokerID, err)
	}
	return util.NewJSONResponse(http.StatusOK, catalog)
}
