package web

import (
	"context"
	"fmt"
)

type contextKey int

const (
	userKey contextKey = iota
	isAuthorizedKey
	authenticationErrorKey
	authorizationErrorKey
)

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

func AuthenticationErrorFromContext(ctx context.Context) (error, bool) {
	authnError, ok := ctx.Value(authenticationErrorKey).(error)
	return authnError, ok && authnError != nil
}

func ContextWithAuthorizationError(ctx context.Context, authZError error) context.Context {
	currentAuthZError, found := AuthorizationErrorFromContext(ctx)
	if found {
		authZError = fmt.Errorf("%s or %s", currentAuthZError, authZError)
	}
	return context.WithValue(ctx, authorizationErrorKey, authZError)
}

func AuthorizationErrorFromContext(ctx context.Context) (error, bool) {
	authzError, ok := ctx.Value(authorizationErrorKey).(error)
	return authzError, ok && authzError != nil
}
