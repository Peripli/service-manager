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

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types"
)

// InterceptCreateAroundTxFunc hook for entity creation outside of transaction
type InterceptCreateAroundTxFunc func(ctx context.Context, obj types.Object) (types.Object, error)

// InterceptCreateOnTxFunc hook for entity creation in transaction
type InterceptCreateOnTxFunc func(ctx context.Context, txStorage Repository, obj types.Object) (types.Object, error)

// CreateAroundTxInterceptor provides hooks on entity creation during AroundTx
//go:generate counterfeiter . CreateAroundTxInterceptor
type CreateAroundTxInterceptor interface {
	AroundTxCreate(f InterceptCreateAroundTxFunc) InterceptCreateAroundTxFunc
}

// CreateOnTxInterceptor provides hooks on entity creation during OnTx
//go:generate counterfeiter . CreateOnTxInterceptor
type CreateOnTxInterceptor interface {
	OnTxCreate(f InterceptCreateOnTxFunc) InterceptCreateOnTxFunc
}

// CreateInterceptor provides hooks on entity creation both during AroundTx and OnTx
//go:generate counterfeiter . CreateInterceptor
type CreateInterceptor interface {
	CreateAroundTxInterceptor
	CreateOnTxInterceptor
}

//go:generate counterfeiter . CreateOnTxInterceptorProvider
type CreateOnTxInterceptorProvider interface {
	web.Named
	Provide() CreateOnTxInterceptor
}

type OrderedCreateOnTxInterceptorProvider struct {
	InterceptorOrder
	CreateOnTxInterceptorProvider
}

//go:generate counterfeiter . CreateAroundTxInterceptorProvider
type CreateAroundTxInterceptorProvider interface {
	web.Named
	Provide() CreateAroundTxInterceptor
}

type OrderedCreateAroundTxInterceptorProvider struct {
	InterceptorOrder
	CreateAroundTxInterceptorProvider
}

// CreateInterceptorProvider provides CreateInterceptors for each request
//go:generate counterfeiter . CreateInterceptorProvider
type CreateInterceptorProvider interface {
	web.Named
	Provide() CreateInterceptor
}

type OrderedCreateInterceptorProvider struct {
	InterceptorOrder
	CreateInterceptorProvider
}

type CreateAroundTxInterceptorChain struct {
	aroundTxNames []string
	aroundTxFuncs map[string]CreateAroundTxInterceptor
}

// AroundTxCreate wraps the provided InterceptCreateAroundTxFunc into all the existing aroundTx funcs
func (c *CreateAroundTxInterceptorChain) AroundTxCreate(f InterceptCreateAroundTxFunc) InterceptCreateAroundTxFunc {
	for i := range c.aroundTxNames {
		if interceptor, found := c.aroundTxFuncs[c.aroundTxNames[len(c.aroundTxNames)-1-i]]; found {
			f = interceptor.AroundTxCreate(f)
		}
	}
	return f
}

type CreateOnTxInterceptorChain struct {
	onTxNames []string
	onTxFuncs map[string]CreateOnTxInterceptor
}

// OnTxCreate wraps the provided InterceptCreateOnTxFunc into all the existing onTx funcs
func (c *CreateOnTxInterceptorChain) OnTxCreate(f InterceptCreateOnTxFunc) InterceptCreateOnTxFunc {
	for i := range c.onTxNames {
		if interceptor, found := c.onTxFuncs[c.onTxNames[len(c.onTxNames)-1-i]]; found {
			f = interceptor.OnTxCreate(f)
		}
	}
	return f
}

// CreateInterceptorChain is an interceptor tha provides and chains a list of ordered interceptor providers.
type CreateInterceptorChain struct {
	*CreateAroundTxInterceptorChain
	*CreateOnTxInterceptorChain
}

func (itr *InterceptableTransactionalRepository) newCreateOnTxInterceptorChain(objectType types.ObjectType) *CreateOnTxInterceptorChain {
	providers := itr.createOnTxProviders[objectType]
	onTxFuncs := make(map[string]CreateOnTxInterceptor, len(providers))
	for _, provider := range providers {
		onTxFuncs[provider.Name()] = provider.Provide()
	}
	return &CreateOnTxInterceptorChain{
		onTxNames: itr.orderedCreateOnTxProvidersNames[objectType],
		onTxFuncs: onTxFuncs,
	}
}

func (itr *InterceptableTransactionalRepository) newCreateInterceptorChain(objectType types.ObjectType) *CreateInterceptorChain {
	aroundTxFuncs := make(map[string]CreateAroundTxInterceptor)
	for _, p := range itr.createAroundTxProviders[objectType] {
		aroundTxFuncs[p.Name()] = p.Provide()
	}

	onTxFuncs := make(map[string]CreateOnTxInterceptor)
	for _, p := range itr.createOnTxProviders[objectType] {
		onTxFuncs[p.Name()] = p.Provide()
	}

	for _, p := range itr.createProviders[objectType] {
		// Provide once to share state
		interceptor := p.Provide()
		aroundTxFuncs[p.Name()] = interceptor
		onTxFuncs[p.Name()] = interceptor

	}

	return &CreateInterceptorChain{
		CreateAroundTxInterceptorChain: &CreateAroundTxInterceptorChain{
			aroundTxNames: itr.orderedCreateAroundTxProvidersNames[objectType],
			aroundTxFuncs: aroundTxFuncs,
		},
		CreateOnTxInterceptorChain: &CreateOnTxInterceptorChain{
			onTxNames: itr.orderedCreateOnTxProvidersNames[objectType],
			onTxFuncs: onTxFuncs,
		},
	}
}
