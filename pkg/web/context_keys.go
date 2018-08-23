package web

import (
	"context"
)

type contextKey int

const (
	userKey contextKey = iota
)

// UserFromContext gets the authenticated user from the context
func UserFromContext(ctx context.Context) (*User, bool) {
	userStr, ok := ctx.Value(userKey).(*User)
	return userStr, ok
}

// NewContextWithUser sets the authenticated user in the context
func NewContextWithUser(ctx context.Context, user *User) context.Context {
	return context.WithValue(ctx, userKey, user)
}
