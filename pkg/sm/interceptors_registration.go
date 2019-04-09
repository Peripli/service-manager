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

type registrator interface {
	Apply(repository *storage.InterceptableTransactionalRepository, interceptorType types.ObjectType, orderer storage.Ordered)
}

type interceptorRegistrationBuilder struct {
	registrator

	orderer         *storage.OrderedByName
	interceptorType types.ObjectType
	repository      *storage.InterceptableTransactionalRepository
}

func (creator *interceptorRegistrationBuilder) Apply() {
	if creator.orderer == nil {
		creator.orderer = &storage.OrderedByName{}
	}
	creator.registrator.Apply(creator.repository, creator.interceptorType, creator.orderer)
}

func (creator *interceptorRegistrationBuilder) Before(name string) *interceptorRegistrationBuilder {
	return creator.TxBefore(name).AroundTxBefore(name)
}

func (creator *interceptorRegistrationBuilder) After(name string) *interceptorRegistrationBuilder {
	return creator.TxAfter(name).AroundTxAfter(name)
}

func (creator *interceptorRegistrationBuilder) TxBefore(name string) *interceptorRegistrationBuilder {
	if creator.orderer == nil {
		creator.orderer = &storage.OrderedByName{}
	}
	creator.orderer.NameTx = name
	creator.orderer.PositionTxType = storage.PositionBefore
	return creator
}

func (creator *interceptorRegistrationBuilder) AroundTxBefore(name string) *interceptorRegistrationBuilder {
	if creator.orderer == nil {
		creator.orderer = &storage.OrderedByName{}
	}
	creator.orderer.NameAroundTx = name
	creator.orderer.PositionAroundTxType = storage.PositionBefore
	return creator
}

func (creator *interceptorRegistrationBuilder) AroundTxAfter(name string) *interceptorRegistrationBuilder {
	if creator.orderer == nil {
		creator.orderer = &storage.OrderedByName{}
	}
	creator.orderer.NameAroundTx = name
	creator.orderer.PositionAroundTxType = storage.PositionAfter
	return creator
}

func (creator *interceptorRegistrationBuilder) TxAfter(name string) *interceptorRegistrationBuilder {
	if creator.orderer == nil {
		creator.orderer = &storage.OrderedByName{}
	}
	creator.orderer.NameTx = name
	creator.orderer.PositionTxType = storage.PositionAfter
	return creator
}

type orderedCreateInterceptorProvider struct {
	storage.CreateInterceptorProvider
	storage.Ordered
}

type createInterceptorRegistration struct {
	provider storage.CreateInterceptorProvider
}

func (cib *createInterceptorRegistration) Apply(repository *storage.InterceptableTransactionalRepository, interceptsType types.ObjectType, orderer storage.Ordered) {
	repository.AddCreateInterceptorProviders(interceptsType, &orderedCreateInterceptorProvider{
		CreateInterceptorProvider: cib.provider,
		Ordered:                   orderer,
	})
}

type updateInterceptorRegistration struct {
	provider storage.UpdateInterceptorProvider
}

type orderedUpdateInterceptorProvider struct {
	storage.Ordered
	storage.UpdateInterceptorProvider
}

func (uib *updateInterceptorRegistration) Apply(repository *storage.InterceptableTransactionalRepository, interceptsType types.ObjectType, orderer storage.Ordered) {
	repository.AddUpdateInterceptorProviders(interceptsType, &orderedUpdateInterceptorProvider{
		UpdateInterceptorProvider: uib.provider,
		Ordered:                   orderer,
	})
}

type deleteInterceptorRegistration struct {
	provider storage.DeleteInterceptorProvider
}

type orderedDeleteInterceptorProvider struct {
	storage.Ordered
	storage.DeleteInterceptorProvider
}

func (dib *deleteInterceptorRegistration) Apply(repository *storage.InterceptableTransactionalRepository, interceptsType types.ObjectType, orderer storage.Ordered) {
	repository.AddDeleteInterceptorProviders(interceptsType, &orderedDeleteInterceptorProvider{
		DeleteInterceptorProvider: dib.provider,
		Ordered:                   orderer,
	})
}
