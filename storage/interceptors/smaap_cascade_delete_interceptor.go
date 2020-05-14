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
	TenantIdentifier string
}

type CascadeOperationCreateInterceptorProvider struct {
	TenantIdentifier string
}

func (c *CascadeOperationCreateInterceptorProvider) Provide() storage.CreateOnTxInterceptor {
	return &cascadeOperationCreateInterceptor{
		TenantIdentifier: c.TenantIdentifier,
	}
}

func (c *CascadeOperationCreateInterceptorProvider) Name() string {
	return CascadeOperationCreateInterceptorProviderName
}

func (co *cascadeOperationCreateInterceptor) OnTxCreate(f storage.InterceptCreateOnTxFunc) storage.InterceptCreateOnTxFunc {
	return func(ctx context.Context, storage storage.Repository, obj types.Object) (types.Object, error) {
		operation := obj.(*types.Operation)
		if !web.IsCascadeOperation(ctx) || operation.Type != types.DELETE {
			return f(ctx, storage, operation)
		}
		ops, err := getChildren(ctx, storage, operation)
		if err != nil {
			return nil, err
		}

		for _, operation := range ops {
			if _, err := storage.Create(ctx, operation); err != nil {
				return nil, util.HandleStorageError(err, string(operation.GetType()))
			}
		}
		return operation, nil
	}
}

func getChildren(ctx context.Context, repository storage.Repository, op *types.Operation) ([]*types.Operation, error) {
	return nil, nil
}
