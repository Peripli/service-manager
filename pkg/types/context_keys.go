package types

import (
	"context"
)

type contextKey int

const (
	instanceCtxKey       contextKey = iota
	sharedInstanceCtxKey contextKey = iota
	planCtxKey           contextKey = iota
)

// InstanceFromContext gets the service instance from the context
func InstanceFromContext(ctx context.Context) (*ServiceInstance, bool) {
	instanceCtx, ok := ctx.Value(instanceCtxKey).(*ServiceInstance)
	return instanceCtx, ok && instanceCtx != nil
}

// InstanceFromContext gets the service instance from the context
func SharedInstanceFromContext(ctx context.Context) (*ServiceInstance, bool) {
	instanceCtx, ok := ctx.Value(sharedInstanceCtxKey).(*ServiceInstance)
	return instanceCtx, ok && instanceCtx != nil
}

// PlanFromContext gets the service plan from the context
func PlanFromContext(ctx context.Context) (*ServicePlan, bool) {
	planCtx, ok := ctx.Value(planCtxKey).(*ServicePlan)
	return planCtx, ok && planCtx != nil
}

// ContextWithInstance sets the service instance in the context
func ContextWithInstance(ctx context.Context, instance *ServiceInstance) context.Context {
	return context.WithValue(ctx, instanceCtxKey, instance)
}

// ContextWithInstance sets the service instance in the context
func ContextWithSharedInstance(ctx context.Context, instance *ServiceInstance) context.Context {
	return context.WithValue(ctx, sharedInstanceCtxKey, instance)
}

// ContextWithPlan sets the service plan in the context
func ContextWithPlan(ctx context.Context, plan *ServicePlan) context.Context {
	return context.WithValue(ctx, planCtxKey, plan)
}
