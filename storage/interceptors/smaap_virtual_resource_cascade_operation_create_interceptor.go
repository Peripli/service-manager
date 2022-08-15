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
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/operations"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/util"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/storage"
	"time"
)

const VirtualResourceCascadeOperationCreateInterceptorProviderName = "VirtualResourceCascadeOperationCreateInterceptorProvider"

type VirtualResourceCascadeOperationCreateInterceptor struct {
	TenantIdentifier string
}

type VirtualResourceCascadeOperationCreateInterceptorProvider struct {
	TenantIdentifier string
}

func (c *VirtualResourceCascadeOperationCreateInterceptorProvider) Provide() storage.CreateOnTxInterceptor {
	return &VirtualResourceCascadeOperationCreateInterceptor{
		TenantIdentifier: c.TenantIdentifier,
	}
}

func (c *VirtualResourceCascadeOperationCreateInterceptorProvider) Name() string {
	return VirtualResourceCascadeOperationCreateInterceptorProviderName
}

func (co *VirtualResourceCascadeOperationCreateInterceptor) OnTxCreate(f storage.InterceptCreateOnTxFunc) storage.InterceptCreateOnTxFunc {
	return func(ctx context.Context, storage storage.Repository, obj types.Object) (types.Object, error) {
		operation := obj.(*types.Operation)
		// currently we have only one virtual object: tenant
		isVirtual := types.IsVirtualType(operation.ResourceType)
		if !isVirtual || operation.CascadeRootID == "" || operation.Type != types.DELETE {
			return f(ctx, storage, operation)
		}

		if duplicate, err := operations.FindCascadeOperationForResource(ctx, storage, operation.ResourceID); err != nil || duplicate != nil {
			// in case cascade operation does exists for this resource
			return duplicate, err
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

		cascadeResource := types.NewTenant(operation.ResourceID, co.TenantIdentifier)
		ops, err := operations.GetAllLevelsCascadeOperations(ctx, cascadeResource, operation, storage)

		if err != nil {
			return nil, err
		}

		if len(ops) == 0 {
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
