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

	"github.com/Peripli/service-manager/pkg/query"

	"github.com/Peripli/service-manager/pkg/types"
)

type UpdateInterceptorChain struct {
	aroundTxNames []string
	aroundTxFuncs map[string]func(InterceptUpdateAroundTxFunc) InterceptUpdateAroundTxFunc

	onTxNames []string
	onTxFuncs map[string]func(InterceptUpdateOnTxFunc) InterceptUpdateOnTxFunc
}

func (u *UpdateInterceptorChain) Name() string {
	return "UpdateInterceptorChain"
}

func (u *UpdateInterceptorChain) AroundTxUpdate(f InterceptUpdateAroundTxFunc) InterceptUpdateAroundTxFunc {
	for i := range u.aroundTxNames {
		f = u.aroundTxFuncs[u.aroundTxNames[len(u.aroundTxNames)-1-i]](f)
	}
	return f
}

func (u *UpdateInterceptorChain) OnTxUpdate(f InterceptUpdateOnTxFunc) InterceptUpdateOnTxFunc {
	for i := range u.onTxNames {
		f = u.onTxFuncs[u.onTxNames[len(u.onTxNames)-1-i]](f)
	}
	return f
}

// newUpdateInterceptorChain returns a function which spawns all update interceptors, sorts them and wraps them into one.
func newUpdateInterceptorChain(providers []OrderedUpdateInterceptorProvider) *UpdateInterceptorChain {
	chain := &UpdateInterceptorChain{}
	chain.aroundTxFuncs = make(map[string]func(InterceptUpdateAroundTxFunc) InterceptUpdateAroundTxFunc)
	chain.aroundTxNames = make([]string, 0, len(providers))
	chain.onTxFuncs = make(map[string]func(InterceptUpdateOnTxFunc) InterceptUpdateOnTxFunc)
	chain.onTxNames = make([]string, 0, len(providers))

	for _, p := range providers {
		interceptor := p.Provide()

		chain.aroundTxFuncs[p.Name()] = interceptor.AroundTxUpdate
		chain.aroundTxNames = insertName(chain.aroundTxNames, p.AroundTxPosition.PositionType, p.AroundTxPosition.Name, p.Name())

		chain.onTxFuncs[p.Name()] = interceptor.OnTxUpdate
		chain.onTxNames = insertName(chain.onTxNames, p.OnTxPosition.PositionType, p.OnTxPosition.Name, p.Name())
	}
	return chain
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

// InterceptUpdateAroundTxFunc hook for entity update outside of transaction
type InterceptUpdateAroundTxFunc func(ctx context.Context, newObj types.Object, labelChanges ...*query.LabelChange) (types.Object, error)

// InterceptUpdateOnTxFunc hook for entity update in transaction
type InterceptUpdateOnTxFunc func(ctx context.Context, txStorage Repository, oldObj, newObj types.Object, labelChanges ...*query.LabelChange) (types.Object, error)

// UpdateInterceptor provides hooks on entity update
//go:generate counterfeiter . UpdateInterceptor
type UpdateInterceptor interface {
	AroundTxUpdate(h InterceptUpdateAroundTxFunc) InterceptUpdateAroundTxFunc
	OnTxUpdate(f InterceptUpdateOnTxFunc) InterceptUpdateOnTxFunc
}
