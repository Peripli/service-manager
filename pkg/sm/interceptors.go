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

package sm

import (
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
)

type concreteBuilder interface {
	Apply(repository *storage.InterceptableRepository, interceptorType types.ObjectType, orderer storage.Ordered)
}

type interceptorBuilder struct {
	concreteBuilder

	orderer         *storage.OrderedProviderImpl
	interceptorType types.ObjectType
	repository      *storage.InterceptableRepository
}

func (creator *interceptorBuilder) Apply() {
	if creator.orderer == nil {
		creator.orderer = &storage.OrderedProviderImpl{}
	}
	creator.concreteBuilder.Apply(creator.repository, creator.interceptorType, creator.orderer)
}

func (creator *interceptorBuilder) Before(name string) *interceptorBuilder {
	return creator.TxBefore(name).APIBefore(name)
}

func (creator *interceptorBuilder) After(name string) *interceptorBuilder {
	return creator.TxAfter(name).APIAfter(name)
}

func (creator *interceptorBuilder) TxBefore(name string) *interceptorBuilder {
	if creator.orderer == nil {
		creator.orderer = &storage.OrderedProviderImpl{}
	}
	creator.orderer.NameTx = name
	creator.orderer.PositionTypeTx = storage.PositionBefore
	return creator
}

func (creator *interceptorBuilder) APIBefore(name string) *interceptorBuilder {
	if creator.orderer == nil {
		creator.orderer = &storage.OrderedProviderImpl{}
	}
	creator.orderer.NameAPI = name
	creator.orderer.PositionTypeAPI = storage.PositionBefore
	return creator
}

func (creator *interceptorBuilder) APIAfter(name string) *interceptorBuilder {
	if creator.orderer == nil {
		creator.orderer = &storage.OrderedProviderImpl{}
	}
	creator.orderer.NameAPI = name
	creator.orderer.PositionTypeAPI = storage.PositionAfter
	return creator
}

func (creator *interceptorBuilder) TxAfter(name string) *interceptorBuilder {
	if creator.orderer == nil {
		creator.orderer = &storage.OrderedProviderImpl{}
	}
	creator.orderer.NameTx = name
	creator.orderer.PositionTypeTx = storage.PositionAfter
	return creator
}

type orderedCreateInterceptorProvider struct {
	storage.CreateInterceptorProvider
	storage.Ordered
}

type createInterceptorBuilder struct {
	provider storage.CreateInterceptorProvider
}

func (cib *createInterceptorBuilder) Apply(repository *storage.InterceptableRepository, interceptsType types.ObjectType, orderer storage.Ordered) {
	repository.AddCreateInterceptorProviders(interceptsType, &orderedCreateInterceptorProvider{
		CreateInterceptorProvider: cib.provider,
		Ordered:                   orderer,
	})
}

type updateInterceptorBuilder struct {
	provider storage.UpdateInterceptorProvider
}

type orderedUpdateInterceptorProvider struct {
	storage.Ordered
	storage.UpdateInterceptorProvider
}

func (uib *updateInterceptorBuilder) Apply(repository *storage.InterceptableRepository, interceptsType types.ObjectType, orderer storage.Ordered) {
	repository.AddUpdateInterceptorProviders(interceptsType, &orderedUpdateInterceptorProvider{
		UpdateInterceptorProvider: uib.provider,
		Ordered:                   orderer,
	})
}

type deleteInterceptorBuilder struct {
	provider storage.DeleteInterceptorProvider
}

type orderedDeleteInterceptorProvider struct {
	storage.Ordered
	storage.DeleteInterceptorProvider
}

func (dib *deleteInterceptorBuilder) Apply(repository *storage.InterceptableRepository, interceptsType types.ObjectType, orderer storage.Ordered) {
	repository.AddDeleteInterceptorProviders(interceptsType, &orderedDeleteInterceptorProvider{
		DeleteInterceptorProvider: dib.provider,
		Ordered:                   orderer,
	})
}
