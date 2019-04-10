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

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/storage"
	osbc "github.com/pmorie/go-open-service-broker-client/v2"
)

const CreateBrokerInterceptorName = "CreateBrokerCatalogInterceptor"

type BrokerCreateInterceptorProvider struct {
	OsbClientCreateFunc osbc.CreateFunc
}

func (c *BrokerCreateInterceptorProvider) Name() string {
	return CreateBrokerInterceptorName
}

func (c *BrokerCreateInterceptorProvider) Provide() storage.CreateInterceptor {
	return &CreateBrokerInterceptor{
		OSBClientCreateFunc: c.OsbClientCreateFunc,
	}
}

type CreateBrokerInterceptor struct {
	OSBClientCreateFunc osbc.CreateFunc
}

func (c *CreateBrokerInterceptor) AroundTxCreate(h storage.InterceptCreateAroundTxFunc) storage.InterceptCreateAroundTxFunc {
	return func(ctx context.Context, obj types.Object) (types.Object, error) {
		broker := obj.(*types.ServiceBroker)
		catalog, err := getBrokerCatalog(ctx, c.OSBClientCreateFunc, broker)
		if err != nil {
			return nil, err
		}
		if broker.Services, err = osbCatalogToOfferings(catalog, broker.ID); err != nil {
			return nil, err
		}

		return h(ctx, broker)
	}
}

func (c *CreateBrokerInterceptor) OnTxCreate(f storage.InterceptCreateOnTxFunc) storage.InterceptCreateOnTxFunc {
	return func(ctx context.Context, storage storage.Repository, obj types.Object) error {
		if err := f(ctx, storage, obj); err != nil {
			return err
		}
		broker := obj.(*types.ServiceBroker)

		for serviceIndex := range broker.Services {
			service := broker.Services[serviceIndex]
			if _, err := storage.Create(ctx, service); err != nil {
				return util.HandleStorageError(err, "service_offering")
			}
			for planIndex := range service.Plans {
				servicePlan := service.Plans[planIndex]
				if _, err := storage.Create(ctx, servicePlan); err != nil {
					return util.HandleStorageError(err, "service_plan")
				}
			}
		}
		return nil
	}
}
