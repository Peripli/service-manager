package storage

import (
	"context"

	"github.com/Peripli/service-manager/pkg/types"

	"github.com/Peripli/service-manager/pkg/query"
)

type DeleteInterceptorChain struct {
	aroundTxNames []string
	aroundTxFuncs map[string]func(InterceptDeleteAroundTxFunc) InterceptDeleteAroundTxFunc

	onTxNames []string
	onTxFuncs map[string]func(InterceptDeleteOnTxFunc) InterceptDeleteOnTxFunc
}

func (d *DeleteInterceptorChain) Name() string {
	return "DeleteInterceptorChain"
}

func (d *DeleteInterceptorChain) AroundTxDelete(f InterceptDeleteAroundTxFunc) InterceptDeleteAroundTxFunc {
	for i := range d.aroundTxNames {
		f = d.aroundTxFuncs[d.aroundTxNames[len(d.aroundTxNames)-1-i]](f)
	}
	return f
}

func (d *DeleteInterceptorChain) OnTxDelete(f InterceptDeleteOnTxFunc) InterceptDeleteOnTxFunc {
	for i := range d.onTxNames {
		f = d.onTxFuncs[d.onTxNames[len(d.onTxNames)-1-i]](f)
	}
	return f
}

// newDeleteInterceptorChain returns a function which chains all delete interceptors, sorts them and wraps them into one.
func newDeleteInterceptorChain(providers []OrderedDeleteInterceptorProvider) *DeleteInterceptorChain {
	chain := &DeleteInterceptorChain{}

	chain.aroundTxFuncs = make(map[string]func(InterceptDeleteAroundTxFunc) InterceptDeleteAroundTxFunc)
	chain.aroundTxNames = make([]string, 0, len(providers))
	chain.onTxFuncs = make(map[string]func(InterceptDeleteOnTxFunc) InterceptDeleteOnTxFunc)
	chain.onTxNames = make([]string, 0, len(providers))

	for _, p := range providers {
		interceptor := p.Provide()

		chain.aroundTxFuncs[p.Name()] = interceptor.AroundTxDelete
		chain.aroundTxNames = insertName(chain.aroundTxNames, p.AroundTxPosition.PositionType, p.AroundTxPosition.Name, p.Name())

		chain.onTxFuncs[p.Name()] = interceptor.OnTxDelete
		chain.onTxNames = insertName(chain.onTxNames, p.OnTxPosition.PositionType, p.OnTxPosition.Name, p.Name())
	}

	return chain
}

// DeleteInterceptorProvider provides DeleteInterceptorChain for each request
//go:generate counterfeiter . DeleteInterceptorProvider
type DeleteInterceptorProvider interface {
	Named
	Provide() DeleteInterceptor
}

// InterceptDeleteAroundTxFunc hook for entity deletion outside of transaction
type InterceptDeleteAroundTxFunc func(ctx context.Context, deletionCriteria ...query.Criterion) (types.ObjectList, error)

// InterceptDeleteOnTxFunc hook for entity deletion in transaction
type InterceptDeleteOnTxFunc func(ctx context.Context, txStorage Repository, objects types.ObjectList, deletionCriteria ...query.Criterion) (types.ObjectList, error)

// DeleteInterceptor provides hooks on entity deletion
//go:generate counterfeiter . DeleteInterceptor
type DeleteInterceptor interface {
	AroundTxDelete(h InterceptDeleteAroundTxFunc) InterceptDeleteAroundTxFunc
	OnTxDelete(f InterceptDeleteOnTxFunc) InterceptDeleteOnTxFunc
}
