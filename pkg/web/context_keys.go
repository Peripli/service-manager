package web

import (
	"context"
	"fmt"
	"github.com/Peripli/service-manager/pkg/types"
)

type contextKey int

const (
	userKey contextKey = iota
	isAuthorizedKey
	authenticationErrorKey
	authorizationErrorKey
	shouldStoreBindingsKey
	instanceKey
)

// ShouldStoreBindings returns whether the request has to store bindings
func ShouldStoreBindings(ctx context.Context) bool {
	shouldStoreBindings := ctx.Value(shouldStoreBindingsKey)
	return shouldStoreBindings == nil || shouldStoreBindings.(bool)
}

// ContextWithStoreBindings sets the shouldStoreBindings flag in the context
func ContextWithStoreBindingsFlag(ctx context.Context, shouldStoreBindings bool) context.Context {
	return context.WithValue(ctx, shouldStoreBindingsKey, shouldStoreBindings)
}

// UserFromContext gets the authenticated user from the context
func UserFromContext(ctx context.Context) (*UserContext, bool) {
	userCtx, ok := ctx.Value(userKey).(*UserContext)
	return userCtx, ok && userCtx != nil
}

// ContextWithUser sets the authenticated user in the context
func ContextWithUser(ctx context.Context, user *UserContext) context.Context {
	return context.WithValue(ctx, userKey, user)
}

// IsAuthorized returns whether the request has been authorized
func IsAuthorized(ctx context.Context) bool {
	_, ok := ctx.Value(isAuthorizedKey).(bool)
	return ok
}

// ContextWithAuthorization sets the boolean flag isAuthorized in the request context
func ContextWithAuthorization(ctx context.Context) context.Context {
	return context.WithValue(ctx, isAuthorizedKey, true)
}

func ContextWithAuthenticationError(ctx context.Context, authNError error) context.Context {
	return context.WithValue(ctx, authenticationErrorKey, authNError)
}

func AuthenticationErrorFromContext(ctx context.Context) (bool, error) {
	authnError, ok := ctx.Value(authenticationErrorKey).(error)
	return ok && authnError != nil, authnError
}

func ContextWithAuthorizationError(ctx context.Context, authZError error) context.Context {
	found, currentAuthZError := AuthorizationErrorFromContext(ctx)
	if found {
		authZError = fmt.Errorf("%s or %s", currentAuthZError, authZError)
	}
	return context.WithValue(ctx, authorizationErrorKey, authZError)
}

func AuthorizationErrorFromContext(ctx context.Context) (bool, error) {
	authzError, ok := ctx.Value(authorizationErrorKey).(error)
	return ok && authzError != nil, authzError
}

func ContextWithInstance(ctx context.Context, serviceInstance *types.ServiceInstance) context.Context {
	return context.WithValue(ctx, instanceKey, serviceInstance)
}

func InstanceFromContext(ctx context.Context) (*types.ServiceInstance, bool) {
	instance, ok := ctx.Value(instanceKey).(*types.ServiceInstance)
	return instance, ok && instance != nil
}
