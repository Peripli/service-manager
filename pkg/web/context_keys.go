package web

import (
	"context"
)

type contextKey int

const (
	userKey contextKey = iota
	isAuthorizedKey
)

// UserFromContext gets the authenticated user from the context
func UserFromContext(ctx context.Context) (*User, bool) {
	userStr, ok := ctx.Value(userKey).(*User)
	return userStr, ok
}

// ContextWithUser sets the authenticated user in the context
func ContextWithUser(ctx context.Context, user *User) context.Context {
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
