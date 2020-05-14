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
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
)

const CascadeOperationCreateInterceptorProviderName = "CascadeOperationCreateInterceptorProvider"

type cascadeOperationCreateInterceptor struct {
	Repository storage.TransactionalRepository
	TenantIdentifier string
}

type CascadeOperationCreateInterceptorProvider struct {
	Repository *storage.InterceptableTransactionalRepository
	TenantIdentifier string
}

func (c *CascadeOperationCreateInterceptorProvider) Provide() storage.CreateAroundTxInterceptor {
	return &cascadeOperationCreateInterceptor{
		Repository: c.Repository.RawRepository,
		TenantIdentifier: c.TenantIdentifier,
	}
}

func (c *CascadeOperationCreateInterceptorProvider) Name() string {
	return CascadeOperationCreateInterceptorProviderName
}

func (co *cascadeOperationCreateInterceptor) AroundTxCreate(f storage.InterceptCreateAroundTxFunc) storage.InterceptCreateAroundTxFunc {
	return func(ctx context.Context, obj types.Object) (types.Object, error) {
		operation := obj.(*types.Operation)
		if !web.IsCascadeOperation(ctx) || operation.Type != types.DELETE {
			return f(ctx, operation)
		}
		ops, err := getChildren(ctx, co.Repository, operation)
		if err != nil {
			return nil, err
		}
		err = co.Repository.InTransaction(ctx, func(ctx context.Context, storage storage.Repository) error {
			for _, operation := range ops {
				if _, err := storage.Create(ctx, operation); err != nil {
					return util.HandleStorageError(err, string(operation.GetType()))
				}
			}
			return nil
		})

		if err != nil {
			return nil, err
		}

		return operation, nil
	}
}

func getChildren(ctx context.Context, repository storage.TransactionalRepository, op *types.Operation) ([]*types.Operation, error) {
	return nil, nil
}
