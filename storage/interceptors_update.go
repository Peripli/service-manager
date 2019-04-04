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

package storage

import (
	"context"
	"fmt"

	"github.com/Peripli/service-manager/pkg/query"

	"github.com/Peripli/service-manager/pkg/types"
)

type namedUpdateAPIFunc struct {
	Name string
	Func func(InterceptUpdateAroundTxFunc) InterceptUpdateAroundTxFunc
}

type namedUpdateTxFunc struct {
	Name string
	Func func(InterceptUpdateOnTxFunc) InterceptUpdateOnTxFunc
}

type UpdateInterceptorChain struct {
	UpdateHookOnAPIFuncs         []*namedUpdateAPIFunc
	UpdateHookOnTransactionFuncs []*namedUpdateTxFunc
}

func (c *UpdateInterceptorChain) AroundTxUpdate(f InterceptUpdateAroundTxFunc) InterceptUpdateAroundTxFunc {
	for i := range c.UpdateHookOnAPIFuncs {
		f = c.UpdateHookOnAPIFuncs[len(c.UpdateHookOnAPIFuncs)-1-i].Func(f)
	}
	return f
}

func (c *UpdateInterceptorChain) OnTxUpdate(f InterceptUpdateOnTxFunc) InterceptUpdateOnTxFunc {
	for i := range c.UpdateHookOnTransactionFuncs {
		f = c.UpdateHookOnTransactionFuncs[len(c.UpdateHookOnTransactionFuncs)-1-i].Func(f)
	}
	return f
}

// NewUpdateInterceptorChain returns a function which spawns all update interceptors, sorts them and wraps them into one.
func NewUpdateInterceptorChain(providers []UpdateInterceptorProvider) *UpdateInterceptorChain {
	c := &UpdateInterceptorChain{}
	c.UpdateHookOnAPIFuncs = make([]*namedUpdateAPIFunc, 0, len(providers))
	c.UpdateHookOnTransactionFuncs = make([]*namedUpdateTxFunc, 0, len(providers))

	for _, p := range providers {
		interceptor := p.Provide()
		positionAPIType := PositionNone
		positionTxType := PositionNone
		nameAPI := ""
		nameTx := ""

		if orderedProvider, isOrdered := p.(Ordered); isOrdered {
			positionAPIType, nameAPI = orderedProvider.PositionAPI()
			positionTxType, nameTx = orderedProvider.PositionTx()
		}

		c.insertAroundTxFunc(positionAPIType, nameAPI, &namedUpdateAPIFunc{
			Name: interceptor.Name(),
			Func: interceptor.AroundTxUpdate,
		})
		c.insertTxFunc(positionTxType, nameTx, &namedUpdateTxFunc{
			Name: interceptor.Name(),
			Func: interceptor.OnTxUpdate,
		})
	}
	return c
}

// UpdateContext provides changes done by the update operation
type UpdateContext struct {
	Object        types.Object
	ObjectChanges []byte
	LabelChanges  []*query.LabelChange
}

// UpdateInterceptorProvider provides UpdateInterceptors for each request
//go:generate counterfeiter . UpdateInterceptorProvider
type UpdateInterceptorProvider interface {
	Provide() UpdateInterceptor
}

type InterceptUpdateAroundTx interface {
	InterceptUpdateAroundTx(ctx context.Context, obj types.Object, labelChanges ...*query.LabelChange) (types.Object, error)
}

// InterceptUpdateAroundTxFunc hook for entity update outside of transaction
type InterceptUpdateAroundTxFunc func(ctx context.Context, obj types.Object, labelChanges ...*query.LabelChange) (types.Object, error)

func (iuat InterceptUpdateAroundTxFunc) InterceptUpdateAroundTx(ctx context.Context, obj types.Object, labelChanges ...*query.LabelChange) (types.Object, error) {
	return iuat(ctx, obj, labelChanges...)

}

type InterceptUpdateOnTx interface {
	InterceptUpdateOnTx(ctx context.Context, txStorage Warehouse, obj types.Object, labelChanges ...*query.LabelChange) (types.Object, error)
}

// InterceptUpdateOnTxFunc hook for entity update in transaction
type InterceptUpdateOnTxFunc func(ctx context.Context, txStorage Warehouse, obj types.Object, labelChanges ...*query.LabelChange) (types.Object, error)

func (iutf InterceptUpdateOnTxFunc) InterceptUpdateOnTx(ctx context.Context, txStorage Warehouse, obj types.Object, labelChanges ...*query.LabelChange) (types.Object, error) {
	return iutf(ctx, txStorage, obj, labelChanges...)
}

// UpdateInterceptor provides hooks on entity update
//go:generate counterfeiter . UpdateInterceptor
type UpdateInterceptor interface {
	Named
	AroundTxUpdate(h InterceptUpdateAroundTxFunc) InterceptUpdateAroundTxFunc
	OnTxUpdate(f InterceptUpdateOnTxFunc) InterceptUpdateOnTxFunc
}

func (c *UpdateInterceptorChain) insertAroundTxFunc(positionType PositionType, name string, h *namedUpdateAPIFunc) {
	if positionType == PositionNone {
		c.UpdateHookOnAPIFuncs = append(c.UpdateHookOnAPIFuncs, h)
		return
	}
	pos := c.findAPIFuncPosition(c.UpdateHookOnAPIFuncs, name)
	if pos == -1 {
		panic(fmt.Errorf("could not find update API hook with name %s", name))
	}
	c.UpdateHookOnAPIFuncs = append(c.UpdateHookOnAPIFuncs, nil)
	if positionType == PositionAfter {
		pos = pos + 1
	}
	copy(c.UpdateHookOnAPIFuncs[pos+1:], c.UpdateHookOnAPIFuncs[pos:])
	c.UpdateHookOnAPIFuncs[pos] = h
}

func (c *UpdateInterceptorChain) insertTxFunc(positionType PositionType, name string, h *namedUpdateTxFunc) {
	if positionType == PositionNone {
		c.UpdateHookOnTransactionFuncs = append(c.UpdateHookOnTransactionFuncs, h)
		return
	}
	pos := c.findTxFuncPosition(c.UpdateHookOnTransactionFuncs, name)
	if pos == -1 {
		panic(fmt.Errorf("could not find update transaction hook with name %s", name))
	}
	c.UpdateHookOnTransactionFuncs = append(c.UpdateHookOnTransactionFuncs, nil)
	if positionType == PositionAfter {
		pos = pos + 1
	}
	copy(c.UpdateHookOnTransactionFuncs[pos+1:], c.UpdateHookOnTransactionFuncs[pos:])
	c.UpdateHookOnTransactionFuncs[pos] = h
}

func (c *UpdateInterceptorChain) findAPIFuncPosition(funcs []*namedUpdateAPIFunc, name string) int {
	for i, f := range funcs {
		if f.Name == name {
			return i
		}
	}

	return -1
}

func (c *UpdateInterceptorChain) findTxFuncPosition(funcs []*namedUpdateTxFunc, name string) int {
	for i, f := range funcs {
		if f.Name == name {
			return i
		}
	}

	return -1
}
