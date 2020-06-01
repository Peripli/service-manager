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
	"time"
)

const CascadeOperationCreateInterceptorProviderName = "CascadeOperationCreateInterceptorProvider"

type cascadeOperationCreateInterceptor struct {
	TenantIdentifier string
	utils            *operations.CascadeUtils
}

type CascadeOperationCreateInterceptorProvider struct {
	TenantIdentifier string
}

func (c *CascadeOperationCreateInterceptorProvider) Provide() storage.CreateOnTxInterceptor {
	return &cascadeOperationCreateInterceptor{
		TenantIdentifier: c.TenantIdentifier,
		utils: &operations.CascadeUtils{
			TenantIdentifier: c.TenantIdentifier,
		},
	}
}

func (c *CascadeOperationCreateInterceptorProvider) Name() string {
	return CascadeOperationCreateInterceptorProviderName
}

func (co *cascadeOperationCreateInterceptor) OnTxCreate(f storage.InterceptCreateOnTxFunc) storage.InterceptCreateOnTxFunc {
	return func(ctx context.Context, storage storage.Repository, obj types.Object) (types.Object, error) {
		operation := obj.(*types.Operation)
		if operation.CascadeRootID == "" || operation.Type != types.DELETE {
			return f(ctx, storage, operation)
		}

		// init operation properties
		operation.PlatformID = types.SMPlatform
		operation.State = types.PENDING
		operation.Base.CreatedAt = time.Now()
		operation.Base.UpdatedAt = time.Now()
		operation.Base.Ready = true

		if err := operation.Validate(); err != nil {
			return nil, err
		}
		if exists, err := doesExistCascadeOperationForResource(ctx, storage, operation); err != nil || exists != nil {
			// in case cascade operation does exists for this resource
			return exists, err
		}

		isVirtual := types.IsVirtualType(operation.ResourceType)
		var cascadeResource types.Object
		if isVirtual {
			// currently we have only one virtual object: tenant
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
			// fetching and validating object does exist
			cascadeResource, err = storage.Get(ctx, operation.ResourceType, query.ByField(query.EqualsOperator, "id", operation.ResourceID))
			if err != nil {
				return nil, err
			}
		}

		ops, err := co.utils.GetAllLevelsCascadeOperations(ctx, cascadeResource, operation, storage)
		if err != nil {
			return nil, err
		}
		if len(ops) == 0 && isVirtual {
			operation.State = types.SUCCEEDED
		}
		for _, op := range ops {
			if _, err := storage.Create(ctx, op); err != nil {
				return nil, util.HandleStorageError(err, string(op.GetType()))
			}
		}
		return f(ctx, storage, operation)
	}
}

func doesExistCascadeOperationForResource(ctx context.Context, storage storage.Repository, operation *types.Operation) (*types.Operation, error) {
	criteria := []query.Criterion{
		query.ByField(query.EqualsOperator, "resource_id", operation.ResourceID),
		query.ByField(query.InOperator, "state", string(types.IN_PROGRESS), string(types.PENDING)),
		query.ByField(query.NotEqualsOperator, "cascade_root_id", ""),
	}
	op, err := storage.Get(ctx, types.OperationType, criteria...)
	if err != nil {
		if err == util.ErrNotFoundInStorage {
			return nil, nil
		}
		return nil, err
	}
	return op.(*types.Operation), nil
}
