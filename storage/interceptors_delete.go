package storage

import (
	"context"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/query"
)

// InterceptDeleteAroundTxFunc hook for entity deletion outside of transaction
type InterceptDeleteAroundTxFunc func(ctx context.Context, deletionCriteria ...query.Criterion) error

// InterceptDeleteOnTxFunc hook for entity deletion in transaction
type InterceptDeleteOnTxFunc func(ctx context.Context, txStorage Repository, objects types.ObjectList, deletionCriteria ...query.Criterion) error

// DeleteAroundTxInterceptor provides hooks on entity deletion during AroundTx
//go:generate counterfeiter . DeleteAroundTxInterceptor
type DeleteAroundTxInterceptor interface {
	AroundTxDelete(f InterceptDeleteAroundTxFunc) InterceptDeleteAroundTxFunc
}

// DeleteOnTxInterceptor provides hooks on entity deletion during OnTx
//go:generate counterfeiter . DeleteOnTxInterceptor
type DeleteOnTxInterceptor interface {
	OnTxDelete(f InterceptDeleteOnTxFunc) InterceptDeleteOnTxFunc
}

// DeleteInterceptor provides hooks on entity deletion both during AroundTx and OnTx
//go:generate counterfeiter . DeleteInterceptor
type DeleteInterceptor interface {
	DeleteAroundTxInterceptor
	DeleteOnTxInterceptor
}

//go:generate counterfeiter . DeleteOnTxInterceptorProvider
type DeleteOnTxInterceptorProvider interface {
	web.Named
	Provide() DeleteOnTxInterceptor
}

type OrderedDeleteOnTxInterceptorProvider struct {
	InterceptorOrder
	DeleteOnTxInterceptorProvider
}

//go:generate counterfeiter . DeleteAroundTxInterceptorProvider
type DeleteAroundTxInterceptorProvider interface {
	web.Named
	Provide() DeleteAroundTxInterceptor
}

type OrderedDeleteAroundTxInterceptorProvider struct {
	InterceptorOrder
	DeleteAroundTxInterceptorProvider
}

// DeleteInterceptorProvider provides DeleteInterceptors for each request
//go:generate counterfeiter . DeleteInterceptorProvider
type DeleteInterceptorProvider interface {
	web.Named
	Provide() DeleteInterceptor
}

type OrderedDeleteInterceptorProvider struct {
	InterceptorOrder
	DeleteInterceptorProvider
}

type DeleteAroundTxInterceptorChain struct {
	aroundTxNames []string
	aroundTxFuncs map[string]DeleteAroundTxInterceptor
}

// AroundTxDelete wraps the provided InterceptDeleteAroundTxFunc into all the existing aroundTx funcs
func (c *DeleteAroundTxInterceptorChain) AroundTxDelete(f InterceptDeleteAroundTxFunc) InterceptDeleteAroundTxFunc {
	for i := range c.aroundTxNames {
		if interceptor, found := c.aroundTxFuncs[c.aroundTxNames[len(c.aroundTxNames)-1-i]]; found {
			f = interceptor.AroundTxDelete(f)
		}
	}
	return f
}

type DeleteOnTxInterceptorChain struct {
	onTxNames []string
	onTxFuncs map[string]DeleteOnTxInterceptor
}

// OnTxDelete wraps the provided InterceptDeleteOnTxFunc into all the existing onTx funcs
func (c *DeleteOnTxInterceptorChain) OnTxDelete(f InterceptDeleteOnTxFunc) InterceptDeleteOnTxFunc {
	for i := range c.onTxNames {
		if interceptor, found := c.onTxFuncs[c.onTxNames[len(c.onTxNames)-1-i]]; found {
			f = interceptor.OnTxDelete(f)
		}
	}
	return f
}

// DeleteInterceptorChain is an interceptor tha provides and chains a list of ordered interceptor providers.
type DeleteInterceptorChain struct {
	*DeleteAroundTxInterceptorChain
	*DeleteOnTxInterceptorChain
}

func (itr *InterceptableTransactionalRepository) newDeleteOnTxInterceptorChain(objectType types.ObjectType) *DeleteOnTxInterceptorChain {
	providers := itr.deleteOnTxProviders[objectType]
	onTxFuncs := make(map[string]DeleteOnTxInterceptor, len(providers))
	for _, provider := range providers {
		onTxFuncs[provider.Name()] = provider.Provide()
	}
	return &DeleteOnTxInterceptorChain{
		onTxNames: itr.orderedDeleteOnTxProvidersNames[objectType],
		onTxFuncs: onTxFuncs,
	}
}

func (itr *InterceptableTransactionalRepository) newDeleteInterceptorChain(objectType types.ObjectType) *DeleteInterceptorChain {
	aroundTxFuncs := make(map[string]DeleteAroundTxInterceptor)
	for _, p := range itr.deleteAroundTxProviders[objectType] {
		aroundTxFuncs[p.Name()] = p.Provide()
	}

	onTxFuncs := make(map[string]DeleteOnTxInterceptor)
	for _, p := range itr.deleteOnTxProviders[objectType] {
		onTxFuncs[p.Name()] = p.Provide()
	}

	for _, p := range itr.deleteProviders[objectType] {
		// Provide once to share state
		interceptor := p.Provide()
		aroundTxFuncs[p.Name()] = interceptor
		onTxFuncs[p.Name()] = interceptor

	}

	return &DeleteInterceptorChain{
		DeleteAroundTxInterceptorChain: &DeleteAroundTxInterceptorChain{
			aroundTxNames: itr.orderedDeleteAroundTxProvidersNames[objectType],
			aroundTxFuncs: aroundTxFuncs,
		},
		DeleteOnTxInterceptorChain: &DeleteOnTxInterceptorChain{
			onTxNames: itr.orderedDeleteOnTxProvidersNames[objectType],
			onTxFuncs: onTxFuncs,
		},
	}
}
