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

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
)

type namedCreateAPIFunc struct {
	Name string
	Func func(InterceptCreateOnAPI) InterceptCreateOnAPI
}

type namedCreateTxFunc struct {
	Name string
	Func func(InterceptCreateOnTx) InterceptCreateOnTx
}

type createHookOnAPIHandler struct {
	CreateHookOnAPIFuncs         []*namedCreateAPIFunc
	CreateHookOnTransactionFuncs []*namedCreateTxFunc
}

func (c *createHookOnAPIHandler) OnAPICreate(f InterceptCreateOnAPI) InterceptCreateOnAPI {
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
				Func: hook.OnAPICreate,
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

// InterceptCreateOnAPI hook for entity creation outside of transaction
type InterceptCreateOnAPI func(ctx context.Context, obj types.Object) (types.Object, error)

// InterceptCreateOnTx hook for entity creation in transaction
type InterceptCreateOnTx func(ctx context.Context, txStorage storage.Warehouse, newObject types.Object) error

// CreateInterceptor provides hooks on entity creation
//go:generate counterfeiter . CreateInterceptor
type CreateInterceptor interface {
	OnAPICreate(h InterceptCreateOnAPI) InterceptCreateOnAPI
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
