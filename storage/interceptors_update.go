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

// InterceptUpdateAroundTxFunc hook for entity update outside of transaction
type InterceptUpdateAroundTxFunc func(ctx context.Context, newObj types.Object, labelChanges ...*types.LabelChange) (types.Object, error)

// InterceptUpdateOnTxFunc hook for entity update in transaction
type InterceptUpdateOnTxFunc func(ctx context.Context, txStorage Repository, oldObj, newObj types.Object, labelChanges ...*types.LabelChange) (types.Object, error)

// UpdateAroundTxInterceptor provides hooks on entity update during AroundTx
//go:generate counterfeiter . UpdateAroundTxInterceptor
type UpdateAroundTxInterceptor interface {
	AroundTxUpdate(h InterceptUpdateAroundTxFunc) InterceptUpdateAroundTxFunc
}

// UpdateOnTxInterceptor provides hooks on entity update during OnTx
//go:generate counterfeiter . UpdateOnTxInterceptor
type UpdateOnTxInterceptor interface {
	OnTxUpdate(f InterceptUpdateOnTxFunc) InterceptUpdateOnTxFunc
}

// UpdateInterceptor provides hooks on entity update
//go:generate counterfeiter . UpdateInterceptor
type UpdateInterceptor interface {
	UpdateAroundTxInterceptor
	UpdateOnTxInterceptor
}

//go:generate counterfeiter . UpdateOnTxInterceptorProvider
type UpdateOnTxInterceptorProvider interface {
	web.Named
	Provide() UpdateOnTxInterceptor
}

type OrderedUpdateOnTxInterceptorProvider struct {
	InterceptorOrder
	UpdateOnTxInterceptorProvider
}

//go:generate counterfeiter . UpdateAroundTxInterceptorProvider
type UpdateAroundTxInterceptorProvider interface {
	web.Named
	Provide() UpdateAroundTxInterceptor
}

type OrderedUpdateAroundTxInterceptorProvider struct {
	InterceptorOrder
	UpdateAroundTxInterceptorProvider
}

// UpdateInterceptorProvider provides UpdateInterceptors for each request
//go:generate counterfeiter . UpdateInterceptorProvider
type UpdateInterceptorProvider interface {
	web.Named
	Provide() UpdateInterceptor
}

type OrderedUpdateInterceptorProvider struct {
	InterceptorOrder
	UpdateInterceptorProvider
}

type UpdateAroundTxInterceptorChain struct {
	aroundTxNames []string
	aroundTxFuncs map[string]UpdateAroundTxInterceptor
}

// AroundTxUpdate wraps the provided InterceptUpdateAroundTxFunc into all the existing aroundTx funcs
func (c *UpdateAroundTxInterceptorChain) AroundTxUpdate(f InterceptUpdateAroundTxFunc) InterceptUpdateAroundTxFunc {
	for i := range c.aroundTxNames {
		if interceptor, found := c.aroundTxFuncs[c.aroundTxNames[len(c.aroundTxNames)-1-i]]; found {
			f = interceptor.AroundTxUpdate(f)
		}
	}
	return f
}

type UpdateOnTxInterceptorChain struct {
	onTxNames []string
	onTxFuncs map[string]UpdateOnTxInterceptor
}

// OnTxUpdate wraps the provided InterceptUpdateOnTxFunc into all the existing onTx funcs
func (c *UpdateOnTxInterceptorChain) OnTxUpdate(f InterceptUpdateOnTxFunc) InterceptUpdateOnTxFunc {
	for i := range c.onTxNames {
		if interceptor, found := c.onTxFuncs[c.onTxNames[len(c.onTxNames)-1-i]]; found {
			f = interceptor.OnTxUpdate(f)
		}
	}
	return f
}

// UpdateInterceptorChain is an interceptor tha provides and chains a list of ordered interceptor providers.
type UpdateInterceptorChain struct {
	*UpdateAroundTxInterceptorChain
	*UpdateOnTxInterceptorChain
}

func (itr *InterceptableTransactionalRepository) newUpdateOnTxInterceptorChain(objectType types.ObjectType) *UpdateOnTxInterceptorChain {
	providers := itr.updateOnTxProviders[objectType]
	onTxFuncs := make(map[string]UpdateOnTxInterceptor, len(providers))
	for _, provider := range providers {
		onTxFuncs[provider.Name()] = provider.Provide()
	}
	return &UpdateOnTxInterceptorChain{
		onTxNames: itr.orderedUpdateOnTxProvidersNames[objectType],
		onTxFuncs: onTxFuncs,
	}
}

func (itr *InterceptableTransactionalRepository) newUpdateInterceptorChain(objectType types.ObjectType) *UpdateInterceptorChain {
	aroundTxFuncs := make(map[string]UpdateAroundTxInterceptor)
	for _, p := range itr.updateAroundTxProviders[objectType] {
		aroundTxFuncs[p.Name()] = p.Provide()
	}

	onTxFuncs := make(map[string]UpdateOnTxInterceptor)
	for _, p := range itr.updateOnTxProviders[objectType] {
		onTxFuncs[p.Name()] = p.Provide()
	}

	for _, p := range itr.updateProviders[objectType] {
		// Provide once to share state
		interceptor := p.Provide()
		aroundTxFuncs[p.Name()] = interceptor
		onTxFuncs[p.Name()] = interceptor

	}

	return &UpdateInterceptorChain{
		UpdateAroundTxInterceptorChain: &UpdateAroundTxInterceptorChain{
			aroundTxNames: itr.orderedUpdateAroundTxProvidersNames[objectType],
			aroundTxFuncs: aroundTxFuncs,
		},
		UpdateOnTxInterceptorChain: &UpdateOnTxInterceptorChain{
			onTxNames: itr.orderedUpdateOnTxProvidersNames[objectType],
			onTxFuncs: onTxFuncs,
		},
	}
}
