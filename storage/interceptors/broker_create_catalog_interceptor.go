/*
 * Copyright 2018 The Service Manager Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package interceptors

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/storage"
	"github.com/gofrs/uuid"
)

const BrokerCreateCatalogInterceptorName = "BrokerCreateCatalogInterceptor"

type BrokerCreateCatalogInterceptorProvider struct {
	CatalogFetcher func(ctx context.Context, broker *types.ServiceBroker) ([]byte, error)
}

func (c *BrokerCreateCatalogInterceptorProvider) Name() string {
	return BrokerCreateCatalogInterceptorName
}

func (c *BrokerCreateCatalogInterceptorProvider) Provide() storage.CreateInterceptor {
	return &brokerCreateCatalogInterceptor{
		CatalogFetcher: c.CatalogFetcher,
	}

}

type brokerCreateCatalogInterceptor struct {
	CatalogFetcher func(ctx context.Context, broker *types.ServiceBroker) ([]byte, error)
}

func (c *brokerCreateCatalogInterceptor) AroundTxCreate(h storage.InterceptCreateAroundTxFunc) storage.InterceptCreateAroundTxFunc {
	return func(ctx context.Context, obj types.Object) (types.Object, error) {
		broker := obj.(*types.ServiceBroker)
		if err := brokerCatalogAroundTx(ctx, broker, c.CatalogFetcher); err != nil {
			return nil, err
		}

		return h(ctx, broker)
	}
}

func (c *brokerCreateCatalogInterceptor) OnTxCreate(f storage.InterceptCreateOnTxFunc) storage.InterceptCreateOnTxFunc {
	return func(ctx context.Context, storage storage.Repository, obj types.Object) (types.Object, error) {
		var createdObj types.Object
		var err error
		if createdObj, err = f(ctx, storage, obj); err != nil {
			return nil, err
		}
		broker := obj.(*types.ServiceBroker)

		for _, service := range broker.Services {
			if _, err := storage.Create(ctx, service); err != nil {
				return nil, err
			}
			for _, servicePlan := range service.Plans {
				if _, err := storage.Create(ctx, servicePlan); err != nil {
					return nil, err
				}
			}
		}

		return createdObj, nil
	}
}

func brokerCatalogAroundTx(ctx context.Context, broker *types.ServiceBroker, fetcher func(ctx context.Context, broker *types.ServiceBroker) ([]byte, error)) error {
	catalogBytes, err := fetcher(ctx, broker)
	if err != nil {
		return err
	}
	broker.Catalog = catalogBytes

	catalogResponse := struct {
		Services []*types.ServiceOffering `json:"services"`
	}{}
	if err := util.BytesToObject(catalogBytes, &catalogResponse); err != nil {
		return err
	}

	for _, service := range catalogResponse.Services {
		service.CatalogID = service.ID
		service.CatalogName = service.Name
		service.BrokerID = broker.ID
		service.CreatedAt = broker.UpdatedAt
		service.UpdatedAt = broker.UpdatedAt
		UUID, err := uuid.NewV4()
		if err != nil {
			return err
		}
		service.ID = UUID.String()
		if err := service.Validate(); err != nil {
			return &util.HTTPError{
				ErrorType:   "BadRequest",
				Description: fmt.Sprintf("service offering constructed during catalog insertion for broker with name %s is invalid: %s", broker.Name, err),
				StatusCode:  http.StatusBadRequest,
			}
		}
		for _, servicePlan := range service.Plans {
			servicePlan.CatalogID = servicePlan.ID
			servicePlan.CatalogName = servicePlan.Name
			servicePlan.ServiceOfferingID = service.ID
			servicePlan.CreatedAt = broker.UpdatedAt
			servicePlan.UpdatedAt = broker.UpdatedAt
			UUID, err := uuid.NewV4()
			if err != nil {
				return err
			}
			servicePlan.ID = UUID.String()
			if err := servicePlan.Validate(); err != nil {
				return &util.HTTPError{
					ErrorType:   "BadRequest",
					Description: fmt.Sprintf("service plan constructed during catalog insertion for broker with name %s is invalid: %s", broker.Name, err),
					StatusCode:  http.StatusBadRequest,
				}
			}
		}
	}
	broker.Services = catalogResponse.Services

	return nil
}
