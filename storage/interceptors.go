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

type namedCreateAPIFunc struct {
	Name string
	Func func(InterceptCreateAroundTx) InterceptCreateAroundTx
}

type namedCreateTxFunc struct {
	Name string
	Func func(InterceptCreateOnTx) InterceptCreateOnTx
}

type createHookOnAPIHandler struct {
	CreateHookOnAPIFuncs         []*namedCreateAPIFunc
	CreateHookOnTransactionFuncs []*namedCreateTxFunc
}

func (c *createHookOnAPIHandler) AroundTxCreate(f InterceptCreateAroundTx) InterceptCreateAroundTx {
	for i := range c.CreateHookOnAPIFuncs {
		f = c.CreateHookOnAPIFuncs[len(c.CreateHookOnAPIFuncs)-1-i].Func(f)
	}
	return f
}

func (c *createHookOnAPIHandler) OnTxCreate(f InterceptCreateOnTx) InterceptCreateOnTx {
	for i := range c.CreateHookOnTransactionFuncs {
		f = c.CreateHookOnTransactionFuncs[len(c.CreateHookOnTransactionFuncs)-1-i].Func(f)
	}
	return f
}

// UnionCreateInterceptor returns a function which spawns all create interceptors, sorts them and wraps them into one.
func UnionCreateInterceptor(providers []CreateInterceptorProvider) func() CreateInterceptor {
	return func() CreateInterceptor {
		c := &createHookOnAPIHandler{}
		c.CreateHookOnAPIFuncs = make([]*namedCreateAPIFunc, 0, len(providers))
		c.CreateHookOnTransactionFuncs = make([]*namedCreateTxFunc, 0, len(providers))

		for _, p := range providers {
			hook := p.Provide()
			positionAPIType := PositionNone
			positionTxType := PositionNone
			nameAPI := ""
			nameTx := ""

			if orderedProvider, isOrdered := p.(Ordered); isOrdered {
				positionAPIType, nameAPI = orderedProvider.PositionAPI()
				positionTxType, nameTx = orderedProvider.PositionTx()
			}

			c.insertAPIFunc(positionAPIType, nameAPI, &namedCreateAPIFunc{
				Name: p.Name(),
				Func: hook.AroundTxCreate,
			})
			c.insertTxFunc(positionTxType, nameTx, &namedCreateTxFunc{
				Name: p.Name(),
				Func: hook.OnTxCreate,
			})
		}
		return c
	}
}

// CreateInterceptorProvider provides CreateInterceptors for each request
//go:generate counterfeiter . CreateInterceptorProvider
type CreateInterceptorProvider interface {
	Named
	Provide() CreateInterceptor
}

// InterceptCreateAroundTx hook for entity creation outside of transaction
type InterceptCreateAroundTx func(ctx context.Context, obj types.Object) (types.Object, error)

// InterceptCreateOnTx hook for entity creation in transaction
type InterceptCreateOnTx func(ctx context.Context, txStorage Warehouse, newObject types.Object) error

// CreateInterceptor provides hooks on entity creation
//go:generate counterfeiter . CreateInterceptor
type CreateInterceptor interface {
	AroundTxCreate(h InterceptCreateAroundTx) InterceptCreateAroundTx
	OnTxCreate(f InterceptCreateOnTx) InterceptCreateOnTx
}

func (c *createHookOnAPIHandler) insertAPIFunc(positionType PositionType, name string, h *namedCreateAPIFunc) {
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

func (c *createHookOnAPIHandler) insertTxFunc(positionType PositionType, name string, h *namedCreateTxFunc) {
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

func (c *createHookOnAPIHandler) findAPIFuncPosition(funcs []*namedCreateAPIFunc, name string) int {
	for i, f := range funcs {
		if f.Name == name {
			return i
		}
	}

	return -1
}

func (c *createHookOnAPIHandler) findTxFuncPosition(funcs []*namedCreateTxFunc, name string) int {
	for i, f := range funcs {
		if f.Name == name {
			return i
		}
	}

	return -1
}

type namedDeleteAPIFunc struct {
	Name string
	Func func(InterceptDeleteOnAPI) InterceptDeleteOnAPI
}

type namedDeleteTxFunc struct {
	Name string
	Func func(InterceptDeleteOnTx) InterceptDeleteOnTx
}

type deleteHookOnAPIHandler struct {
	DeleteHookOnAPIFuncs []*namedDeleteAPIFunc
	DeleteHookOnTxFuncs  []*namedDeleteTxFunc
}

func (c *deleteHookOnAPIHandler) OnAPIDelete(f InterceptDeleteOnAPI) InterceptDeleteOnAPI {
	for i := range c.DeleteHookOnAPIFuncs {
		f = c.DeleteHookOnAPIFuncs[len(c.DeleteHookOnAPIFuncs)-1-i].Func(f)
	}
	return f
}

func (c *deleteHookOnAPIHandler) OnTxDelete(f InterceptDeleteOnTx) InterceptDeleteOnTx {
	for i := range c.DeleteHookOnTxFuncs {
		f = c.DeleteHookOnTxFuncs[len(c.DeleteHookOnTxFuncs)-1-i].Func(f)
	}
	return f
}

// UnionDeleteInterceptor returns a function which spawns all delete interceptors, sorts them and wraps them into one.
func UnionDeleteInterceptor(providers []DeleteInterceptorProvider) func() DeleteInterceptor {
	return func() DeleteInterceptor {
		c := &deleteHookOnAPIHandler{}
		c.DeleteHookOnAPIFuncs = make([]*namedDeleteAPIFunc, 0, len(providers))
		c.DeleteHookOnTxFuncs = make([]*namedDeleteTxFunc, 0, len(providers))

		for _, p := range providers {
			hook := p.Provide()
			positionAPIType := PositionNone
			positionTxType := PositionNone
			nameAPI := ""
			nameTx := ""

			if orderedProvider, isOrdered := p.(Ordered); isOrdered {
				positionAPIType, nameAPI = orderedProvider.PositionAPI()
				positionTxType, nameTx = orderedProvider.PositionTx()
			}

			c.insertAPIFunc(positionAPIType, nameAPI, &namedDeleteAPIFunc{
				Name: p.Name(),
				Func: hook.OnAPIDelete,
			})
			c.insertTxFunc(positionTxType, nameTx, &namedDeleteTxFunc{
				Name: p.Name(),
				Func: hook.OnTxDelete,
			})
		}
		return c
	}
}

// DeleteInterceptorProvider provides DeleteInterceptors for each request
//go:generate counterfeiter . DeleteInterceptorProvider
type DeleteInterceptorProvider interface {
	Named
	Provide() DeleteInterceptor
}

// InterceptDeleteOnAPI hook for entity deletion outside of transaction
type InterceptDeleteOnAPI func(ctx context.Context, deletionCriteria ...query.Criterion) (types.ObjectList, error)

// InterceptDeleteOnTx hook for entity deletion in transaction
type InterceptDeleteOnTx func(ctx context.Context, txStorage Warehouse, deletionCriteria ...query.Criterion) (types.ObjectList, error)

// DeleteInterceptor provides hooks on entity deletion
//go:generate counterfeiter . DeleteInterceptor
type DeleteInterceptor interface {
	OnAPIDelete(h InterceptDeleteOnAPI) InterceptDeleteOnAPI
	OnTxDelete(f InterceptDeleteOnTx) InterceptDeleteOnTx
}

func (c *deleteHookOnAPIHandler) insertAPIFunc(positionType PositionType, name string, h *namedDeleteAPIFunc) {
	if positionType == PositionNone {
		c.DeleteHookOnAPIFuncs = append(c.DeleteHookOnAPIFuncs, h)
		return
	}
	pos := c.findAPIFuncPosition(c.DeleteHookOnAPIFuncs, name)
	if pos == -1 {
		panic(fmt.Errorf("could not find delete API hook with name %s", name))
	}
	c.DeleteHookOnAPIFuncs = append(c.DeleteHookOnAPIFuncs, nil)
	if positionType == PositionAfter {
		pos = pos + 1
	}
	copy(c.DeleteHookOnAPIFuncs[pos+1:], c.DeleteHookOnAPIFuncs[pos:])
	c.DeleteHookOnAPIFuncs[pos] = h
}

func (c *deleteHookOnAPIHandler) insertTxFunc(positionType PositionType, name string, h *namedDeleteTxFunc) {
	if positionType == PositionNone {
		c.DeleteHookOnTxFuncs = append(c.DeleteHookOnTxFuncs, h)
		return
	}
	pos := c.findTxFuncPosition(c.DeleteHookOnTxFuncs, name)
	if pos == -1 {
		panic(fmt.Errorf("could not find delete transaction hook with name %s", name))
	}
	c.DeleteHookOnTxFuncs = append(c.DeleteHookOnTxFuncs, nil)
	if positionType == PositionAfter {
		pos = pos + 1
	}
	copy(c.DeleteHookOnTxFuncs[pos+1:], c.DeleteHookOnTxFuncs[pos:])
	c.DeleteHookOnTxFuncs[pos] = h
}

func (c *deleteHookOnAPIHandler) findAPIFuncPosition(funcs []*namedDeleteAPIFunc, name string) int {
	for i, f := range funcs {
		if f.Name == name {
			return i
		}
	}

	return -1
}

func (c *deleteHookOnAPIHandler) findTxFuncPosition(funcs []*namedDeleteTxFunc, name string) int {
	for i, f := range funcs {
		if f.Name == name {
			return i
		}
	}

	return -1
}

type namedUpdateAPIFunc struct {
	Name string
	Func func(InterceptUpdateOnAPI) InterceptUpdateOnAPI
}

type namedUpdateTxFunc struct {
	Name string
	Func func(InterceptUpdateOnTx) InterceptUpdateOnTx
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

func (c *updateHookOnAPIHandler) OnTxUpdate(f InterceptUpdateOnTx) InterceptUpdateOnTx {
	for i := range c.UpdateHookOnTransactionFuncs {
		f = c.UpdateHookOnTransactionFuncs[len(c.UpdateHookOnTransactionFuncs)-1-i].Func(f)
	}
	return f
}

// UnionUpdateInterceptor returns a function which spawns all update interceptors, sorts them and wraps them into one.
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
				positionTxType, nameTx = orderedProvider.PositionTx()
			}

			c.insertAPIFunc(positionAPIType, nameAPI, &namedUpdateAPIFunc{
				Name: p.Name(),
				Func: hook.OnAPIUpdate,
			})
			c.insertTxFunc(positionTxType, nameTx, &namedUpdateTxFunc{
				Name: p.Name(),
				Func: hook.OnTxUpdate,
			})
		}
		return c
	}
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
	Named
	Provide() UpdateInterceptor
}

// InterceptUpdateOnAPI hook for entity update outside of transaction
type InterceptUpdateOnAPI func(ctx context.Context, obj types.Object, labelChanges ...*query.LabelChange) (types.Object, error)

// InterceptUpdateOnTx hook for entity update in transaction
type InterceptUpdateOnTx func(ctx context.Context, txStorage Warehouse, obj types.Object, labelChanges ...*query.LabelChange) (types.Object, error)

// UpdateInterceptor provides hooks on entity update
//go:generate counterfeiter . UpdateInterceptor
type UpdateInterceptor interface {
	OnAPIUpdate(h InterceptUpdateOnAPI) InterceptUpdateOnAPI
	OnTxUpdate(f InterceptUpdateOnTx) InterceptUpdateOnTx
}

func (c *updateHookOnAPIHandler) insertAPIFunc(positionType PositionType, name string, h *namedUpdateAPIFunc) {
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

func (c *updateHookOnAPIHandler) insertTxFunc(positionType PositionType, name string, h *namedUpdateTxFunc) {
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
//go:generate counterfeiter . Named
type Named interface {
	Name() string
}
