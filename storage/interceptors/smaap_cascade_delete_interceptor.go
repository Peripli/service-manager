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
	"github.com/Peripli/service-manager/operations"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
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
		if !operation.Cascade || operation.Type != types.DELETE {
			return f(ctx, storage, operation)
		}
		// validate operation is valid
		if err := operation.Validate(); err != nil {
			return nil, err
		}
		// validate no operation for the same resource
		if err, exists := co.operationExists(ctx, storage, operation); err != nil || exists {
			return operation, err
		}

		var cascadeResource types.Object
		if operation.Virtual {
			// currently we have only one virtual object - tenant
			cascadeResource = &types.Tenant{
				VirtualType: types.VirtualType{
					Base: types.Base{
						ID: operation.ResourceID,
					},
				},
				TenantIdentifier: co.TenantIdentifier,
			}
		} else {
			var err error
			// validating object does exist
			cascadeResource, err = storage.Get(ctx, operation.ResourceType, query.ByField(query.EqualsOperator, "id", operation.ResourceID))
			if err != nil {
				return nil, err
			}
		}
		ops, err := operations.GetAllLevelsCascadeOperations(ctx, cascadeResource, operation, storage)
		if err != nil {
			return nil, err
		}
		for _, op := range ops {
			if _, err := storage.Create(ctx, op); err != nil {
				return nil, util.HandleStorageError(err, string(op.GetType()))
			}
		}
		return f(ctx, storage, operation)
	}
}

func (co *cascadeOperationCreateInterceptor) operationExists(ctx context.Context, storage storage.Repository, operation *types.Operation) (error, bool) {
	criteria := []query.Criterion{
		query.ByField(query.EqualsOperator, "resource_id", operation.ResourceID),
		query.ByField(query.EqualsOperator, "state", string(types.IN_PROGRESS)),
		query.ByField(query.EqualsOperator, "cascade", "true"),
	}
	ops, err := storage.List(ctx, types.OperationType, criteria...)
	if err != nil {
		return err, false
	} else if ops.Len() > 0 {
		return nil, true
	}
	return nil, false
}
