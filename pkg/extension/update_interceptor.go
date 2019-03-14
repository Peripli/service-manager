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

type UpdateHookOnAPIConstructor func(InterceptUpdateOnAPI) InterceptUpdateOnAPI
type UpdateHookOnTransactionConstructor func(InterceptUpdateOnTransaction) InterceptUpdateOnTransaction

type updateHookOnAPIHandler struct {
	UpdateHookOnAPIFuncs         []UpdateHookOnAPIConstructor
	UpdateHookOnTransactionFuncs []UpdateHookOnTransactionConstructor
}

func (c *updateHookOnAPIHandler) OnAPIUpdate(f InterceptUpdateOnAPI) InterceptUpdateOnAPI {
	for i := range c.UpdateHookOnAPIFuncs {
		f = c.UpdateHookOnAPIFuncs[len(c.UpdateHookOnAPIFuncs)-1-i](f)
	}
	return f
}

func (c *updateHookOnAPIHandler) OnTransactionUpdate(f InterceptUpdateOnTransaction) InterceptUpdateOnTransaction {
	for i := range c.UpdateHookOnTransactionFuncs {
		f = c.UpdateHookOnTransactionFuncs[len(c.UpdateHookOnTransactionFuncs)-1-i](f)
	}
	return f
}

func UnionUpdateInterceptor(providers []UpdateInterceptorProvider) UpdateInterceptorProvider {
	return func() UpdateInterceptor {
		c := &updateHookOnAPIHandler{}
		for _, h := range providers {
			hook := h()
			c.UpdateHookOnAPIFuncs = append(c.UpdateHookOnAPIFuncs, hook.OnAPIUpdate)
			c.UpdateHookOnTransactionFuncs = append(c.UpdateHookOnTransactionFuncs, hook.OnTransactionUpdate)
		}
		return c
	}
}

type UpdateContext struct {
	ObjectID      string
	ObjectChanges []byte
	LabelChanges  []*query.LabelChange
}

type UpdateInterceptorProvider func() UpdateInterceptor

type InterceptUpdateOnAPI func(ctx context.Context, changes UpdateContext) (types.Object, error)
type InterceptUpdateOnTransaction func(ctx context.Context, txStorage storage.Warehouse, ojb types.Object, changes UpdateContext) (types.Object, error)

type UpdateInterceptor interface {
	OnAPIUpdate(h InterceptUpdateOnAPI) InterceptUpdateOnAPI
	OnTransactionUpdate(f InterceptUpdateOnTransaction) InterceptUpdateOnTransaction
}
