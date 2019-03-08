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

package extension

import (
	"context"

	"github.com/Peripli/service-manager/pkg/query"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
)

type DeleteHookOnAPIConstructor func(InterceptDeleteOnAPI) InterceptDeleteOnAPI
type DeleteHookOnTransactionConstructor func(InterceptDeleteOnTransaction) InterceptDeleteOnTransaction

type deleteHookOnAPIHandler struct {
	DeleteHookOnAPIFuncs         []DeleteHookOnAPIConstructor
	DeleteHookOnTransactionFuncs []DeleteHookOnTransactionConstructor
}

func (c *deleteHookOnAPIHandler) OnAPI(f InterceptDeleteOnAPI) InterceptDeleteOnAPI {
	for i := range c.DeleteHookOnAPIFuncs {
		f = c.DeleteHookOnAPIFuncs[len(c.DeleteHookOnAPIFuncs)-1-i](f)
	}
	return f
}

func (c *deleteHookOnAPIHandler) OnTransaction(f InterceptDeleteOnTransaction) InterceptDeleteOnTransaction {
	for i := range c.DeleteHookOnTransactionFuncs {
		f = c.DeleteHookOnTransactionFuncs[len(c.DeleteHookOnTransactionFuncs)-1-i](f)
	}
	return f
}

func UnionDeleteInterceptor(providers ...DeleteInterceptorProvider) DeleteInterceptorProvider {
	return func() DeleteInterceptor {
		c := &deleteHookOnAPIHandler{}
		for _, h := range providers {
			hook := h()
			c.DeleteHookOnAPIFuncs = append(c.DeleteHookOnAPIFuncs, hook.OnAPI)
			c.DeleteHookOnTransactionFuncs = append(c.DeleteHookOnTransactionFuncs, hook.OnTransaction)
		}
		return c
	}
}

type DeleteInterceptorProvider func() DeleteInterceptor

type InterceptDeleteOnAPI func(ctx context.Context, deletionCriteria ...query.Criterion) error
type InterceptDeleteOnTransaction func(ctx context.Context, txStorage storage.Warehouse, deletionCriteria ...query.Criterion) (types.ObjectList, error)

type DeleteInterceptor interface {
	OnAPI(h InterceptDeleteOnAPI) InterceptDeleteOnAPI
	OnTransaction(f InterceptDeleteOnTransaction) InterceptDeleteOnTransaction
}
