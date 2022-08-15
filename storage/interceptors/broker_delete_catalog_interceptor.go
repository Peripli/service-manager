package interceptors

import (
	"context"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/query"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/storage"
)

const BrokerDeleteCatalogInterceptorName = "BrokerDeleteCatalogInterceptor"

// BrokerDeleteCatalogInterceptorProvider provides a broker interceptor for delete operations
type BrokerDeleteCatalogInterceptorProvider struct {
	CatalogLoader func(ctx context.Context, brokerID string, repository storage.Repository) (*types.ServiceOfferings, error)
}

func (c *BrokerDeleteCatalogInterceptorProvider) Provide() storage.DeleteInterceptor {
	return &brokerDeleteCatalogInterceptor{
		CatalogLoader: c.CatalogLoader,
	}
}

func (c *BrokerDeleteCatalogInterceptorProvider) Name() string {
	return BrokerDeleteCatalogInterceptorName
}

type brokerDeleteCatalogInterceptor struct {
	CatalogLoader func(ctx context.Context, brokerID string, repository storage.Repository) (*types.ServiceOfferings, error)
}

func (b *brokerDeleteCatalogInterceptor) AroundTxDelete(h storage.InterceptDeleteAroundTxFunc) storage.InterceptDeleteAroundTxFunc {
	return h
}

// OnTxDelete loads the broker catalog. Currently the catalog is required so that the additional data to the delete broker notifications can be attached.
func (b *brokerDeleteCatalogInterceptor) OnTxDelete(h storage.InterceptDeleteOnTxFunc) storage.InterceptDeleteOnTxFunc {
	return func(ctx context.Context, txStorage storage.Repository, objects types.ObjectList, deletionCriteria ...query.Criterion) error {
		brokers := objects.(*types.ServiceBrokers)
		for _, broker := range brokers.ServiceBrokers {
			serviceOfferings, err := b.CatalogLoader(ctx, broker.GetID(), txStorage)
			if err != nil {
				return err
			}

			broker.Services = serviceOfferings.ServiceOfferings
		}

		return h(ctx, txStorage, objects, deletionCriteria...)
	}
}
