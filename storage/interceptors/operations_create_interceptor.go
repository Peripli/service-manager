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
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
)

const OperationsCreateInterceptorName = "OperationsCreateInterceptor"

type OperationsCreateInsterceptorProvider struct {
	TenantIdentifier string
}

func (c *OperationsCreateInsterceptorProvider) Name() string {
	return OperationsCreateInterceptorName
}

func (c *OperationsCreateInsterceptorProvider) Provide() storage.CreateOnTxInterceptor {
	return &operationsCreateInterceptor{
		TenantIdentifier: c.TenantIdentifier,
	}
}

type operationsCreateInterceptor struct {
	TenantIdentifier string
}

func (c *operationsCreateInterceptor) OnTxCreate(h storage.InterceptCreateOnTxFunc) storage.InterceptCreateOnTxFunc {
	return func(ctx context.Context, storage storage.Repository, obj types.Object) (types.Object, error) {
		operation := obj.(*types.Operation)

		tenantID := query.RetrieveFromCriteria(c.TenantIdentifier, query.CriteriaForContext(ctx)...)[0]

		if tenantID == "" {
			log.D().Debugf("Could not add %s label to operation with id %s. Label not found in context criteria.", c.TenantIdentifier, operation.ID)
			return h(ctx, storage, operation)
		}

		labels := operation.GetLabels()
		if labels == nil {
			labels = types.Labels{}
		}
		labels[c.TenantIdentifier] = []string{tenantID}

		operation.SetLabels(labels)

		return h(ctx, storage, operation)
	}
}
