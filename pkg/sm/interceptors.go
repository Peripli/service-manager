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

//
//type concreteBuilder interface {
//	Apply(interceptables []storage.Interceptable, interceptorType types.ObjectType, orderer storage.Ordered)
//}
//
//type interceptorBuilder struct {
//	concreteBuilder
//
//	orderer         *storage.OrderedProviderImpl
//	interceptorType types.ObjectType
//	interceptables  []storage.Interceptable
//}
//
//func (creator *interceptorBuilder) Apply() {
//	if creator.orderer == nil {
//		creator.orderer = &storage.OrderedProviderImpl{}
//	}
//	creator.concreteBuilder.Apply(creator.interceptables, creator.interceptorType, creator.orderer)
//}
//
//func (creator *interceptorBuilder) Before(name string) *interceptorBuilder {
//	return creator.TxBefore(name).APIBefore(name)
//}
//
//func (creator *interceptorBuilder) After(name string) *interceptorBuilder {
//	return creator.TxAfter(name).APIAfter(name)
//}
//
//func (creator *interceptorBuilder) TxBefore(name string) *interceptorBuilder {
//	if creator.orderer == nil {
//		creator.orderer = &storage.OrderedProviderImpl{}
//	}
//	creator.orderer.NameTx = name
//	creator.orderer.PositionTypeTx = storage.PositionBefore
//	return creator
//}
//
//func (creator *interceptorBuilder) APIBefore(name string) *interceptorBuilder {
//	if creator.orderer == nil {
//		creator.orderer = &storage.OrderedProviderImpl{}
//	}
//	creator.orderer.NameAPI = name
//	creator.orderer.PositionTypeAPI = storage.PositionBefore
//	return creator
//}
//
//func (creator *interceptorBuilder) APIAfter(name string) *interceptorBuilder {
//	if creator.orderer == nil {
//		creator.orderer = &storage.OrderedProviderImpl{}
//	}
//	creator.orderer.NameAPI = name
//	creator.orderer.PositionTypeAPI = storage.PositionAfter
//	return creator
//}
//
//func (creator *interceptorBuilder) TxAfter(name string) *interceptorBuilder {
//	if creator.orderer == nil {
//		creator.orderer = &storage.OrderedProviderImpl{}
//	}
//	creator.orderer.NameTx = name
//	creator.orderer.PositionTypeTx = storage.PositionAfter
//	return creator
//}
//
//type updateInterceptorBuilder struct {
//	provider storage.UpdateInterceptorProvider
//}
//
//type orderedUpdateInterceptorProvider struct {
//	storage.Ordered
//	storage.UpdateInterceptorProvider
//}
//
//func apply(interceptables []storage.Interceptable, interceptsType types.ObjectType, f func(interceptable storage.Interceptable)) {
//	isApplied := false
//	for _, interceptable := range interceptables {
//		if interceptsType == interceptable.InterceptsType() {
//			isApplied = true
//			f(interceptable)
//		}
//	}
//	if !isApplied {
//		log.D().Panicf("interceptor could not be applied to %s", string(interceptsType))
//	}
//}
//
//func (uib *updateInterceptorBuilder) Apply(interceptables []storage.InterceptableRepository, interceptsType types.ObjectType, orderer storage.Ordered) {
//	apply(interceptables, interceptsType, func(interceptable storage.Interceptable) {
//		interceptable.AddUpdateInterceptorProviders(&orderedUpdateInterceptorProvider{
//			UpdateInterceptorProvider: uib.provider,
//			Ordered:                   orderer,
//		})
//	})
//}
//
//type orderedCreateInterceptorProvider struct {
//	storage.CreateInterceptorProvider
//	storage.Ordered
//}
//
//type createInterceptorBuilder struct {
//	provider storage.CreateInterceptorProvider
//}
//
//func (cib *createInterceptorBuilder) Apply(interceptables []storage.Interceptable, interceptsType types.ObjectType, orderer storage.Ordered) {
//	apply(interceptables, interceptsType, func(interceptable storage.Interceptable) {
//		interceptable.AddCreateInterceptorProviders(&orderedCreateInterceptorProvider{
//			CreateInterceptorProvider: cib.provider,
//			Ordered:                   orderer,
//		})
//	})
//}
//
//type deleteInterceptorBuilder struct {
//	provider storage.DeleteInterceptorProvider
//}
//
//type orderedDeleteInterceptorProvider struct {
//	storage.Ordered
//	storage.DeleteInterceptorProvider
//}
//
//func (dib *deleteInterceptorBuilder) Apply(interceptables []storage.Interceptable, interceptsType types.ObjectType, orderer storage.Ordered) {
//	apply(interceptables, interceptsType, func(interceptable storage.Interceptable) {
//		interceptable.AddDeleteInterceptorProviders(&orderedDeleteInterceptorProvider{
//			DeleteInterceptorProvider: dib.provider,
//			Ordered:                   orderer,
//		})
//	})
//}
