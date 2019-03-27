package web

import (
	"fmt"

	"github.com/Peripli/service-manager/pkg/extension"
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
	if creator.orderer != nil {
		if creator.orderer.NameAPI == "" || creator.orderer.NameTx == "" {
			panic(fmt.Errorf("interceptor positional names are not provided"))
		}
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

func (creator *updateInterceptorBuilder) Apply(interceptables []extension.Interceptable, interceptorType types.ObjectType, orderer extension.Ordered) {
	for _, interceptable := range interceptables {
		if interceptorType == interceptable.InterceptsType() {
			if orderer == nil {
				interceptable.AddUpdateInterceptorProviders(creator.provider)
			} else {
				interceptable.AddUpdateInterceptorProviders(&orderedUpdateInterceptorProvider{
					UpdateInterceptorProvider: creator.provider,
					Ordered:                   orderer,
				})
			}
		}
	}
}

type orderedCreateInterceptorProvider struct {
	extension.CreateInterceptorProvider
	extension.Ordered
}

type createInterceptorBuilder struct {
	provider extension.CreateInterceptorProvider
}

func (builder *createInterceptorBuilder) Apply(interceptables []extension.Interceptable, interceptorType types.ObjectType, orderer extension.Ordered) {
	for _, interceptable := range interceptables {
		if interceptorType == interceptable.InterceptsType() {
			if orderer == nil {
				interceptable.AddCreateInterceptorProviders(builder.provider)
			} else {
				interceptable.AddCreateInterceptorProviders(&orderedCreateInterceptorProvider{
					CreateInterceptorProvider: builder.provider,
					Ordered:                   orderer,
				})
			}
		}
	}
}

type deleteInterceptorBuilder struct {
	provider extension.DeleteInterceptorProvider
}

type orderedDeleteInterceptorProvider struct {
	extension.Ordered
	extension.DeleteInterceptorProvider
}

func (creator *deleteInterceptorBuilder) Apply(interceptables []extension.Interceptable, interceptorType types.ObjectType, orderer extension.Ordered) {
	for _, interceptable := range interceptables {
		if interceptorType == interceptable.InterceptsType() {
			if orderer == nil {
				interceptable.AddDeleteInterceptorProviders(creator.provider)
			} else {
				interceptable.AddDeleteInterceptorProviders(&orderedDeleteInterceptorProvider{
					DeleteInterceptorProvider: creator.provider,
					Ordered:                   orderer,
				})
			}
		}
	}
}
