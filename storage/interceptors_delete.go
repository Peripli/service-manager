package storage

import (
	"context"
	"fmt"

	"github.com/Peripli/service-manager/pkg/types"

	"github.com/Peripli/service-manager/pkg/query"
)

type namedDeleteAPIFunc struct {
	Name string
	Func func(InterceptDeleteAroundTxFunc) InterceptDeleteAroundTxFunc
}

type namedDeleteTxFunc struct {
	Name string
	Func func(InterceptDeleteOnTxFunc) InterceptDeleteOnTxFunc
}

// TODO this has ONAPI in the name and has an array of TX Funcs that are structs. Naming / abstraction needs improvements
type DeleteInterceptorChain struct {
	DeleteHookOnAPIFuncs []*namedDeleteAPIFunc
	DeleteHookOnTxFuncs  []*namedDeleteTxFunc
}

func (c *DeleteInterceptorChain) AroundTxDelete(f InterceptDeleteAroundTxFunc) InterceptDeleteAroundTxFunc {
	for i := range c.DeleteHookOnAPIFuncs {
		f = c.DeleteHookOnAPIFuncs[len(c.DeleteHookOnAPIFuncs)-1-i].Func(f)
	}
	return f
}

func (c *DeleteInterceptorChain) OnTxDelete(f InterceptDeleteOnTxFunc) InterceptDeleteOnTxFunc {
	for i := range c.DeleteHookOnTxFuncs {
		f = c.DeleteHookOnTxFuncs[len(c.DeleteHookOnTxFuncs)-1-i].Func(f)
	}
	return f
}

// NewDeleteInterceptorChain returns a function which chains all delete interceptors, sorts them and wraps them into one.
//TODO this doesnt need to return a func
func NewDeleteInterceptorChain(providers []DeleteInterceptorProvider) *DeleteInterceptorChain {
	c := &DeleteInterceptorChain{}
	c.DeleteHookOnAPIFuncs = make([]*namedDeleteAPIFunc, 0, len(providers))
	c.DeleteHookOnTxFuncs = make([]*namedDeleteTxFunc, 0, len(providers))

	for _, p := range providers {
		interceptor := p.Provide()
		positionAPIType := PositionNone
		positionTxType := PositionNone
		nameAPI := ""
		nameTx := ""

		if orderedInterceptor, isOrdered := p.(Ordered); isOrdered {
			positionAPIType, nameAPI = orderedInterceptor.PositionAPI()
			positionTxType, nameTx = orderedInterceptor.PositionTx()
		}

		c.insertAPIFunc(positionAPIType, nameAPI, &namedDeleteAPIFunc{
			Name: interceptor.Name(),
			Func: interceptor.AroundTxDelete,
		})
		c.insertTxFunc(positionTxType, nameTx, &namedDeleteTxFunc{
			Name: interceptor.Name(),
			Func: interceptor.OnTxDelete,
		})
	}

	return c
}

// DeleteInterceptorProvider provides DeleteInterceptorChain for each request
//go:generate counterfeiter . DeleteInterceptorProvider
type DeleteInterceptorProvider interface {
	Provide() DeleteInterceptor
}

//TODO this needs a better name
type InterceptDeleteAroundTx interface {
	InterceptDeleteAroundTx(context.Context, ...query.Criterion) (types.ObjectList, error)
}

//TODO this needs a better name

// InterceptDeleteAroundTxFunc hook for entity deletion outside of transaction
type InterceptDeleteAroundTxFunc func(ctx context.Context, deletionCriteria ...query.Criterion) (types.ObjectList, error)

func (idf InterceptDeleteAroundTxFunc) InterceptDeleteAroundTx(ctx context.Context, criteria ...query.Criterion) (types.ObjectList, error) {
	return idf(ctx, criteria...)
}

type InterceptDeleteOnTx interface {
	InterceptDeleteOnTx(context.Context, Warehouse, ...query.Criterion) (types.ObjectList, error)
}

// InterceptDeleteOnTxFunc hook for entity deletion in transaction
type InterceptDeleteOnTxFunc func(ctx context.Context, txStorage Warehouse, deletionCriteria ...query.Criterion) (types.ObjectList, error)

// DeleteInterceptor provides hooks on entity deletion
//go:generate counterfeiter . DeleteInterceptor
type DeleteInterceptor interface {
	Named
	AroundTxDelete(h InterceptDeleteAroundTxFunc) InterceptDeleteAroundTxFunc
	OnTxDelete(f InterceptDeleteOnTxFunc) InterceptDeleteOnTxFunc
}

//TODO fix reciever names
func (c *DeleteInterceptorChain) insertAPIFunc(positionType PositionType, name string, h *namedDeleteAPIFunc) {
	if positionType == PositionNone {
		c.DeleteHookOnAPIFuncs = append(c.DeleteHookOnAPIFuncs, h)
		return
	}
	pos := c.findAPIFuncPosition(c.DeleteHookOnAPIFuncs, name)
	if pos == -1 {
		panic(fmt.Errorf("could not find delete API hook with name %s", name))
	}
	c.DeleteHookOnAPIFuncs = append(c.DeleteHookOnAPIFuncs, nil)
	if positionType == PositionAfter {
		pos = pos + 1
	}
	copy(c.DeleteHookOnAPIFuncs[pos+1:], c.DeleteHookOnAPIFuncs[pos:])
	c.DeleteHookOnAPIFuncs[pos] = h
}

func (c *DeleteInterceptorChain) insertTxFunc(positionType PositionType, name string, h *namedDeleteTxFunc) {
	if positionType == PositionNone {
		c.DeleteHookOnTxFuncs = append(c.DeleteHookOnTxFuncs, h)
		return
	}
	pos := c.findTxFuncPosition(c.DeleteHookOnTxFuncs, name)
	if pos == -1 {
		panic(fmt.Errorf("could not find delete transaction hook with name %s", name))
	}
	c.DeleteHookOnTxFuncs = append(c.DeleteHookOnTxFuncs, nil)
	if positionType == PositionAfter {
		pos = pos + 1
	}
	copy(c.DeleteHookOnTxFuncs[pos+1:], c.DeleteHookOnTxFuncs[pos:])
	c.DeleteHookOnTxFuncs[pos] = h
}

func (c *DeleteInterceptorChain) findAPIFuncPosition(funcs []*namedDeleteAPIFunc, name string) int {
	for i, f := range funcs {
		if f.Name == name {
			return i
		}
	}

	return -1
}

func (c *DeleteInterceptorChain) findTxFuncPosition(funcs []*namedDeleteTxFunc, name string) int {
	for i, f := range funcs {
		if f.Name == name {
			return i
		}
	}

	return -1
}
