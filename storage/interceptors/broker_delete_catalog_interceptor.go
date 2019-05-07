package interceptors

import (
	"context"

	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
	"github.com/Peripli/service-manager/storage/catalog"
	osbc "github.com/pmorie/go-open-service-broker-client/v2"
)

const BrokerDeleteCatalogInterceptorName = "BrokerDeleteCatalogInterceptor"

// BrokerDeleteCatalogInterceptorProvider provides a broker interceptor for update operations
type BrokerDeleteCatalogInterceptorProvider struct {
	OsbClientCreateFunc osbc.CreateFunc
}

func (c *BrokerDeleteCatalogInterceptorProvider) Provide() storage.DeleteInterceptor {
	return &brokerDeleteCatalogInterceptor{
		OSBClientCreateFunc: c.OsbClientCreateFunc,
	}
}

func (c *BrokerDeleteCatalogInterceptorProvider) Name() string {
	return BrokerDeleteCatalogInterceptorName
}

type brokerDeleteCatalogInterceptor struct {
	OSBClientCreateFunc osbc.CreateFunc
}

func (b *brokerDeleteCatalogInterceptor) AroundTxDelete(h storage.InterceptDeleteAroundTxFunc) storage.InterceptDeleteAroundTxFunc {
	return h
}

func (b *brokerDeleteCatalogInterceptor) OnTxDelete(h storage.InterceptDeleteOnTxFunc) storage.InterceptDeleteOnTxFunc {
	return func(ctx context.Context, txStorage storage.Repository, objects types.ObjectList, deletionCriteria ...query.Criterion) (types.ObjectList, error) {
		brokers := objects.(*types.ServiceBrokers)
		for _, broker := range brokers.ServiceBrokers {
			serviceOfferings, err := catalog.Load(ctx, broker.GetID(), txStorage)
			if err != nil {
				return nil, err
			}

			broker.Services = serviceOfferings.ServiceOfferings
		}

		return h(ctx, txStorage, objects, deletionCriteria...)
	}
}
