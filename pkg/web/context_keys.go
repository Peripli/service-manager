package web

import (
	"context"
)

type contextKey int

const (
	userKey contextKey = iota
	isAuthorizedKey
	limitKey
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

// ContextWithPageLimit sets the page size for list requests
func ContextWithPageLimit(ctx context.Context, limit int) context.Context {
	return context.WithValue(ctx, limitKey, limit)
}

// PageLimitFromContext retrieves the page size from the context
func PageLimitFromContext(ctx context.Context) int {
	limit, _ := ctx.Value(limitKey).(int)
	return limit
}
