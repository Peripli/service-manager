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
	"fmt"

	"github.com/Peripli/service-manager/pkg/query"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
)

type UpdateHookOnAPIConstructor func(InterceptUpdateOnAPI) InterceptUpdateOnAPI
type UpdateHookOnTransactionConstructor func(InterceptUpdateOnTransaction) InterceptUpdateOnTransaction

type namedUpdateAPIFunc struct {
	Name string
	Func UpdateHookOnAPIConstructor
}

type namedUpdateTxFunc struct {
	Name string
	Func UpdateHookOnTransactionConstructor
}

type updateHookOnAPIHandler struct {
	UpdateHookOnAPIFuncs         []*namedUpdateAPIFunc
	UpdateHookOnTransactionFuncs []*namedUpdateTxFunc
}

func (c *updateHookOnAPIHandler) OnAPIUpdate(f InterceptUpdateOnAPI) InterceptUpdateOnAPI {
	for i := range c.UpdateHookOnAPIFuncs {
		f = c.UpdateHookOnAPIFuncs[len(c.UpdateHookOnAPIFuncs)-1-i].Func(f)
	}
	return f
}

func (c *updateHookOnAPIHandler) OnTransactionUpdate(f InterceptUpdateOnTransaction) InterceptUpdateOnTransaction {
	for i := range c.UpdateHookOnTransactionFuncs {
		f = c.UpdateHookOnTransactionFuncs[len(c.UpdateHookOnTransactionFuncs)-1-i].Func(f)
	}
	return f
}

func UnionUpdateInterceptor(providers []UpdateInterceptorProvider) func() UpdateInterceptor {
	return func() UpdateInterceptor {
		c := &updateHookOnAPIHandler{}
		c.UpdateHookOnAPIFuncs = make([]*namedUpdateAPIFunc, 0, len(providers))
		c.UpdateHookOnTransactionFuncs = make([]*namedUpdateTxFunc, 0, len(providers))

		for _, p := range providers {
			hook := p.Provide()
			positionAPIType := PositionNone
			positionTxType := PositionNone
			nameAPI := ""
			nameTx := ""

			if orderedProvider, isOrdered := p.(Ordered); isOrdered {
				positionAPIType, nameAPI = orderedProvider.PositionAPI()
				positionTxType, nameTx = orderedProvider.PositionTransaction()
			}

			c.insertAPIFunc(positionAPIType, nameAPI, &namedUpdateAPIFunc{
				Name: p.Name(),
				Func: hook.OnAPIUpdate,
			})
			c.insertTxFunc(positionTxType, nameTx, &namedUpdateTxFunc{
				Name: p.Name(),
				Func: hook.OnTransactionUpdate,
			})
		}
		return c
	}
}

type UpdateContext struct {
	Object        types.Object
	ObjectChanges []byte
	LabelChanges  []*query.LabelChange
}

type UpdateInterceptorProvider interface {
	Named
	Provide() UpdateInterceptor
}

type InterceptUpdateOnAPI func(ctx context.Context, changes *UpdateContext) (types.Object, error)
type InterceptUpdateOnTransaction func(ctx context.Context, txStorage storage.Warehouse, oldObject types.Object, changes *UpdateContext) (types.Object, error)

type UpdateInterceptor interface {
	OnAPIUpdate(h InterceptUpdateOnAPI) InterceptUpdateOnAPI
	OnTransactionUpdate(f InterceptUpdateOnTransaction) InterceptUpdateOnTransaction
}

func (c *updateHookOnAPIHandler) insertAPIFunc(positionType PositionType, name string, h *namedUpdateAPIFunc) {
	if positionType == PositionNone {
		c.UpdateHookOnAPIFuncs = append(c.UpdateHookOnAPIFuncs, h)
		return
	}
	pos := c.findAPIFuncPosition(c.UpdateHookOnAPIFuncs, name)
	if pos == -1 {
		// TODO: Must validate on bootstrap
		panic(fmt.Errorf("could not find update API hook with name %s", name))
	}
	c.UpdateHookOnAPIFuncs = append(c.UpdateHookOnAPIFuncs, nil)
	if positionType == PositionAfter {
		pos = pos + 1
	}
	copy(c.UpdateHookOnAPIFuncs[pos+1:], c.UpdateHookOnAPIFuncs[pos:])
	c.UpdateHookOnAPIFuncs[pos] = h
}

func (c *updateHookOnAPIHandler) insertTxFunc(positionType PositionType, name string, h *namedUpdateTxFunc) {
	if positionType == PositionNone {
		c.UpdateHookOnTransactionFuncs = append(c.UpdateHookOnTransactionFuncs, h)
		return
	}
	pos := c.findTxFuncPosition(c.UpdateHookOnTransactionFuncs, name)
	if pos == -1 {
		// TODO: Must validate on bootstrap
		panic(fmt.Errorf("could not find update transaction hook with name %s", name))
	}
	c.UpdateHookOnTransactionFuncs = append(c.UpdateHookOnTransactionFuncs, nil)
	if positionType == PositionAfter {
		pos = pos + 1
	}
	copy(c.UpdateHookOnTransactionFuncs[pos+1:], c.UpdateHookOnTransactionFuncs[pos:])
	c.UpdateHookOnTransactionFuncs[pos] = h
}

func (c *updateHookOnAPIHandler) findAPIFuncPosition(funcs []*namedUpdateAPIFunc, name string) int {
	for i, f := range funcs {
		if f.Name == name {
			return i
		}
	}

	return -1
}

func (c *updateHookOnAPIHandler) findTxFuncPosition(funcs []*namedUpdateTxFunc, name string) int {
	for i, f := range funcs {
		if f.Name == name {
			return i
		}
	}

	return -1
}
