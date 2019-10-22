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

// CreateInterceptorChain holds a mapping of aroundTx and onTx funcs with their respective names.
// Using the ordered string slices aroundTxNames and onTxNames the funcs in the two maps can be wrapped in the correct order.
type CreateInterceptorChain struct {
	aroundTxNames []string
	aroundTxFuncs map[string]func(InterceptCreateAroundTxFunc) InterceptCreateAroundTxFunc

	onTxNames []string
	onTxFuncs map[string]func(InterceptCreateOnTxFunc) InterceptCreateOnTxFunc
}

func (c *CreateInterceptorChain) Name() string {
	return "CreateInterceptorChain"
}

// AroundTxCreate wraps the provided InterceptCreateAroundTxFunc into all the existing aroundTx funcs
func (c *CreateInterceptorChain) AroundTxCreate(f InterceptCreateAroundTxFunc) InterceptCreateAroundTxFunc {
	for i := range c.aroundTxNames {
		f = c.aroundTxFuncs[c.aroundTxNames[len(c.aroundTxNames)-1-i]](f)
	}
	return f
}

// OnTxCreate wraps the provided InterceptCreateOnTxFunc into all the existing onTx funcs
func (c *CreateInterceptorChain) OnTxCreate(f InterceptCreateOnTxFunc) InterceptCreateOnTxFunc {
	for i := range c.onTxNames {
		f = c.onTxFuncs[c.onTxNames[len(c.onTxNames)-1-i]](f)
	}
	return f
}

// newCreateInterceptorChain returns a function which spawns all create interceptors, sorts them and wraps them into one.
func newCreateInterceptorChain(providers []OrderedCreateInterceptorProvider) *CreateInterceptorChain {
	chain := &CreateInterceptorChain{}
	chain.aroundTxFuncs = make(map[string]func(InterceptCreateAroundTxFunc) InterceptCreateAroundTxFunc)
	chain.aroundTxNames = make([]string, 0, len(providers))
	chain.onTxFuncs = make(map[string]func(InterceptCreateOnTxFunc) InterceptCreateOnTxFunc)
	chain.onTxNames = make([]string, 0, len(providers))

	for _, p := range providers {
		interceptor := p.Provide()

		chain.aroundTxFuncs[p.Name()] = interceptor.AroundTxCreate
		chain.aroundTxNames = insertName(chain.aroundTxNames, p.AroundTxPosition.PositionType, p.AroundTxPosition.Name, p.Name())

		chain.onTxFuncs[p.Name()] = interceptor.OnTxCreate
		chain.onTxNames = insertName(chain.onTxNames, p.OnTxPosition.PositionType, p.OnTxPosition.Name, p.Name())
	}
	return chain
}

// CreateInterceptorProvider provides CreateInterceptors for each request
//go:generate counterfeiter . CreateInterceptorProvider
type CreateInterceptorProvider interface {
	Named
	Provide() CreateInterceptor
}

// InterceptCreateAroundTxFunc hook for entity creation outside of transaction
type InterceptCreateAroundTxFunc func(ctx context.Context, obj types.Object) (types.Object, error)

// InterceptCreateOnTxFunc hook for entity creation in transaction
type InterceptCreateOnTxFunc func(ctx context.Context, txStorage Repository, obj types.Object) (types.Object, error)

// CreateInterceptor provides hooks on entity creation
//go:generate counterfeiter . CreateInterceptor
type CreateInterceptor interface {
	AroundTxCreate(h InterceptCreateAroundTxFunc) InterceptCreateAroundTxFunc
	OnTxCreate(f InterceptCreateOnTxFunc) InterceptCreateOnTxFunc
}

// insertName inserts the given newInterceptorName into it's the expected position.
// The resulting names slice can then be used to wrap all interceptors into the right order
func insertName(names []string, positionType PositionType, name, newInterceptorName string) []string {
	if positionType == PositionNone {
		names = append(names, newInterceptorName)
		return names
	}
	pos := findName(names, name)
	if pos == -1 {
		panic(fmt.Errorf("could not find create API hook with name %s", name))
	}
	names = append(names, "")
	if positionType == PositionAfter {
		pos = pos + 1
	}
	copy(names[pos+1:], names[pos:])
	names[pos] = newInterceptorName
	return names
}

func findName(names []string, existingInterceptorName string) int {
	for i, name := range names {
		if name == existingInterceptorName {
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

// Named interface for named entities
type Named interface {
	Name() string
}
