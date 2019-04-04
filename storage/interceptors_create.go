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

	"github.com/Peripli/service-manager/pkg/types"
)

type namedCreateAPIFunc struct {
	Name string
	Func func(InterceptCreateAroundTxFunc) InterceptCreateAroundTxFunc
}

type namedCreateTxFunc struct {
	Name string
	Func func(InterceptCreateOnTxFunc) InterceptCreateOnTxFunc
}

type CreateInterceptorChain struct {
	CreateHookOnAPIFuncs         []*namedCreateAPIFunc
	CreateHookOnTransactionFuncs []*namedCreateTxFunc
}

func (c *CreateInterceptorChain) AroundTxCreate(f InterceptCreateAroundTxFunc) InterceptCreateAroundTxFunc {
	for i := range c.CreateHookOnAPIFuncs {
		f = c.CreateHookOnAPIFuncs[len(c.CreateHookOnAPIFuncs)-1-i].Func(f)
	}
	return f
}

func (c *CreateInterceptorChain) OnTxCreate(f InterceptCreateOnTxFunc) InterceptCreateOnTxFunc {
	for i := range c.CreateHookOnTransactionFuncs {
		f = c.CreateHookOnTransactionFuncs[len(c.CreateHookOnTransactionFuncs)-1-i].Func(f)
	}
	return f
}

// NewCreateInterceptorChain returns a function which spawns all create interceptors, sorts them and wraps them into one.
func NewCreateInterceptorChain(providers []CreateInterceptorProvider) *CreateInterceptorChain {
	c := &CreateInterceptorChain{}
	c.CreateHookOnAPIFuncs = make([]*namedCreateAPIFunc, 0, len(providers))
	c.CreateHookOnTransactionFuncs = make([]*namedCreateTxFunc, 0, len(providers))

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

		c.insertAPIFunc(positionAPIType, nameAPI, &namedCreateAPIFunc{
			Name: interceptor.Name(),
			Func: interceptor.AroundTxCreate,
		})
		c.insertTxFunc(positionTxType, nameTx, &namedCreateTxFunc{
			Name: interceptor.Name(),
			Func: interceptor.OnTxCreate,
		})
	}
	return c
}

// CreateInterceptorProvider provides CreateInterceptors for each request
//go:generate counterfeiter . CreateInterceptorProvider
type CreateInterceptorProvider interface {
	Provide() CreateInterceptor
}

type InterceptCreateAroundTx interface {
	InterceptCreateAroundTx(ctx context.Context, obj types.Object) (types.Object, error)
}

// InterceptCreateAroundTxFunc hook for entity creation outside of transaction
type InterceptCreateAroundTxFunc func(ctx context.Context, obj types.Object) (types.Object, error)

func (ica InterceptCreateAroundTxFunc) InterceptCreateAroundTx(ctx context.Context, obj types.Object) (types.Object, error) {
	return ica(ctx, obj)
}

type InterceptCreateOnTx interface {
	InterceptCreateOnTx(ctx context.Context, txStorage Warehouse, newObject types.Object) error
}

// InterceptCreateOnTxFunc hook for entity creation in transaction
type InterceptCreateOnTxFunc func(ctx context.Context, txStorage Warehouse, newObject types.Object) error

func (ico InterceptCreateOnTxFunc) InterceptCreateOnTx(ctx context.Context, txStorage Warehouse, newObject types.Object) error {
	return ico(ctx, txStorage, newObject)
}

// CreateInterceptor provides hooks on entity creation
//go:generate counterfeiter . CreateInterceptor
type CreateInterceptor interface {
	Named
	AroundTxCreate(h InterceptCreateAroundTxFunc) InterceptCreateAroundTxFunc
	OnTxCreate(f InterceptCreateOnTxFunc) InterceptCreateOnTxFunc
}

func (c *CreateInterceptorChain) insertAPIFunc(positionType PositionType, name string, h *namedCreateAPIFunc) {
	if positionType == PositionNone {
		c.CreateHookOnAPIFuncs = append(c.CreateHookOnAPIFuncs, h)
		return
	}
	pos := c.findAPIFuncPosition(c.CreateHookOnAPIFuncs, name)
	if pos == -1 {
		panic(fmt.Errorf("could not find create API hook with name %s", name))
	}
	c.CreateHookOnAPIFuncs = append(c.CreateHookOnAPIFuncs, nil)
	if positionType == PositionAfter {
		pos = pos + 1
	}
	copy(c.CreateHookOnAPIFuncs[pos+1:], c.CreateHookOnAPIFuncs[pos:])
	c.CreateHookOnAPIFuncs[pos] = h
}

func (c *CreateInterceptorChain) insertTxFunc(positionType PositionType, name string, h *namedCreateTxFunc) {
	if positionType == PositionNone {
		c.CreateHookOnTransactionFuncs = append(c.CreateHookOnTransactionFuncs, h)
		return
	}
	pos := c.findTxFuncPosition(c.CreateHookOnTransactionFuncs, name)
	if pos == -1 {
		panic(fmt.Errorf("could not find create transaction hook with name %s", name))
	}
	c.CreateHookOnTransactionFuncs = append(c.CreateHookOnTransactionFuncs, nil)
	if positionType == PositionAfter {
		pos = pos + 1
	}
	copy(c.CreateHookOnTransactionFuncs[pos+1:], c.CreateHookOnTransactionFuncs[pos:])
	c.CreateHookOnTransactionFuncs[pos] = h
}

func (c *CreateInterceptorChain) findAPIFuncPosition(funcs []*namedCreateAPIFunc, name string) int {
	for i, f := range funcs {
		if f.Name == name {
			return i
		}
	}

	return -1
}

func (c *CreateInterceptorChain) findTxFuncPosition(funcs []*namedCreateTxFunc, name string) int {
	for i, f := range funcs {
		if f.Name == name {
			return i
		}
	}

	return -1
}

// PositionType could be "before", "after" or "none"
type PositionType string

const (
	// PositionNone states that a position is not set and the item will be appended
	PositionNone PositionType = "none"

	// PositionBefore states that a position should be calculated before another position
	PositionBefore PositionType = "before"

	// PositionAfter states that a position should be calculated after another position
	PositionAfter PositionType = "after"
)

// OrderedProviderImpl is an implementation of Ordered interface used for ordering interceptors
type OrderedProviderImpl struct {
	PositionTypeTx  PositionType
	PositionTypeAPI PositionType
	NameTx          string
	NameAPI         string
}

// PositionTx returns the position of the interceptor transaction hook
func (opi *OrderedProviderImpl) PositionTx() (PositionType, string) {
	if opi.NameTx == "" {
		return PositionNone, ""
	}
	return opi.PositionTypeTx, opi.NameTx
}

// PositionAPI returns the position of the interceptor out of transaction hook
func (opi *OrderedProviderImpl) PositionAPI() (PositionType, string) {
	if opi.NameAPI == "" {
		return PositionNone, ""
	}
	return opi.PositionTypeAPI, opi.NameAPI
}

// Ordered interface for positioning interceptors
type Ordered interface {
	PositionTx() (PositionType, string)
	PositionAPI() (PositionType, string)
}

// Named interface for named entities
type Named interface {
	Name() string
}
