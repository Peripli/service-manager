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

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
)

type CreateHookOnAPIConstructor func(InterceptCreateOnAPI) InterceptCreateOnAPI
type CreateHookOnTransactionConstructor func(InterceptCreateOnTransaction) InterceptCreateOnTransaction

type createHookOnAPIHandler struct {
	CreateHookOnAPIFuncs         []CreateHookOnAPIConstructor
	CreateHookOnTransactionFuncs []CreateHookOnTransactionConstructor
}

func (c *createHookOnAPIHandler) OnAPI(f InterceptCreateOnAPI) InterceptCreateOnAPI {
	for i := range c.CreateHookOnAPIFuncs {
		f = c.CreateHookOnAPIFuncs[len(c.CreateHookOnAPIFuncs)-1-i](f)
	}
	return f
}

func (c *createHookOnAPIHandler) OnTransaction(f InterceptCreateOnTransaction) InterceptCreateOnTransaction {
	for i := range c.CreateHookOnTransactionFuncs {
		f = c.CreateHookOnTransactionFuncs[len(c.CreateHookOnTransactionFuncs)-1-i](f)
	}
	return f
}

//TODO I Dont see much a of a point for the array to array-warpper thingy (union) - just put the array in the controller
func UnionCreateInterceptor(providers []CreateInterceptorProvider) CreateInterceptorProvider {
	return func() CreateInterceptor {
		c := &createHookOnAPIHandler{}
		for _, h := range providers {
			hook := h()
			c.CreateHookOnAPIFuncs = append(c.CreateHookOnAPIFuncs, hook.OnAPI)
			c.CreateHookOnTransactionFuncs = append(c.CreateHookOnTransactionFuncs, hook.OnTransaction)
		}
		return c
	}
}

type CreateInterceptorProvider func() CreateInterceptor

type InterceptCreateOnAPI func(ctx context.Context, obj types.Object) (types.Object, error)
type InterceptCreateOnTransaction func(ctx context.Context, txStorage storage.Warehouse, newObject types.Object) error

type CreateInterceptor interface {
	OnAPI(h InterceptCreateOnAPI) InterceptCreateOnAPI
	OnTransaction(f InterceptCreateOnTransaction) InterceptCreateOnTransaction
}
