package instcontext

import (
	"context"
	"github.com/Peripli/service-manager/pkg/types"
)

// instanceCtxKey allows putting the current instance is the context
// This way it can be reused and not fetched again from the DB
type instanceCtxKey struct{}

func Get(ctx context.Context) (*types.ServiceInstance, bool) {
	currentInstance := ctx.Value(instanceCtxKey{})
	if currentInstance == nil {
		return nil, false
	}
	return currentInstance.(*types.ServiceInstance), true
}

func Set(ctx context.Context, serviceInstance *types.ServiceInstance) context.Context {
	return context.WithValue(ctx, instanceCtxKey{}, serviceInstance)
}
