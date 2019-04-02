package web

import (
	"github.com/Peripli/service-manager/pkg/extension"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/types"
)

type concreteBuilder interface {
	Apply(interceptables []extension.Interceptable, interceptorType types.ObjectType, orderer extension.Ordered)
}

type interceptorBuilder struct {
	concreteBuilder

	orderer         *extension.OrderedProviderImpl
	interceptorType types.ObjectType
	interceptables  []extension.Interceptable
}

func (creator *interceptorBuilder) Apply() {
	if creator.orderer == nil {
		creator.orderer = &extension.OrderedProviderImpl{}
	}
	creator.concreteBuilder.Apply(creator.interceptables, creator.interceptorType, creator.orderer)
}

func (creator *interceptorBuilder) Before(name string) *interceptorBuilder {
	return creator.TxBefore(name).APIBefore(name)
}

func (creator *interceptorBuilder) After(name string) *interceptorBuilder {
	return creator.TxAfter(name).APIAfter(name)
}

func (creator *interceptorBuilder) TxBefore(name string) *interceptorBuilder {
	if creator.orderer == nil {
		creator.orderer = &extension.OrderedProviderImpl{}
	}
	creator.orderer.NameTx = name
	creator.orderer.PositionTypeTx = extension.PositionBefore
	return creator
}

func (creator *interceptorBuilder) APIBefore(name string) *interceptorBuilder {
	if creator.orderer == nil {
		creator.orderer = &extension.OrderedProviderImpl{}
	}
	creator.orderer.NameAPI = name
	creator.orderer.PositionTypeAPI = extension.PositionBefore
	return creator
}

func (creator *interceptorBuilder) APIAfter(name string) *interceptorBuilder {
	if creator.orderer == nil {
		creator.orderer = &extension.OrderedProviderImpl{}
	}
	creator.orderer.NameAPI = name
	creator.orderer.PositionTypeAPI = extension.PositionAfter
	return creator
}

func (creator *interceptorBuilder) TxAfter(name string) *interceptorBuilder {
	if creator.orderer == nil {
		creator.orderer = &extension.OrderedProviderImpl{}
	}
	creator.orderer.NameTx = name
	creator.orderer.PositionTypeTx = extension.PositionAfter
	return creator
}

type updateInterceptorBuilder struct {
	provider extension.UpdateInterceptorProvider
}

type orderedUpdateInterceptorProvider struct {
	extension.Ordered
	extension.UpdateInterceptorProvider
}

func apply(interceptables []extension.Interceptable, interceptsType types.ObjectType, f func(interceptable extension.Interceptable)) {
	isApplied := false
	for _, interceptable := range interceptables {
		if interceptsType == interceptable.InterceptsType() {
			isApplied = true
			f(interceptable)
		}
	}
	if !isApplied {
		log.D().Panicf("interceptor could not be applied to %s", string(interceptsType))
	}
}

func (uib *updateInterceptorBuilder) Apply(interceptables []extension.Interceptable, interceptsType types.ObjectType, orderer extension.Ordered) {
	apply(interceptables, interceptsType, func(interceptable extension.Interceptable) {
		interceptable.AddUpdateInterceptorProviders(&orderedUpdateInterceptorProvider{
			UpdateInterceptorProvider: uib.provider,
			Ordered:                   orderer,
		})
	})
}

type orderedCreateInterceptorProvider struct {
	extension.CreateInterceptorProvider
	extension.Ordered
}

type createInterceptorBuilder struct {
	provider extension.CreateInterceptorProvider
}

func (cib *createInterceptorBuilder) Apply(interceptables []extension.Interceptable, interceptsType types.ObjectType, orderer extension.Ordered) {
	apply(interceptables, interceptsType, func(interceptable extension.Interceptable) {
		interceptable.AddCreateInterceptorProviders(&orderedCreateInterceptorProvider{
			CreateInterceptorProvider: cib.provider,
			Ordered:                   orderer,
		})
	})
}

type deleteInterceptorBuilder struct {
	provider extension.DeleteInterceptorProvider
}

type orderedDeleteInterceptorProvider struct {
	extension.Ordered
	extension.DeleteInterceptorProvider
}

func (dib *deleteInterceptorBuilder) Apply(interceptables []extension.Interceptable, interceptsType types.ObjectType, orderer extension.Ordered) {
	apply(interceptables, interceptsType, func(interceptable extension.Interceptable) {
		interceptable.AddDeleteInterceptorProviders(&orderedDeleteInterceptorProvider{
			DeleteInterceptorProvider: dib.provider,
			Ordered:                   orderer,
		})
	})
}
