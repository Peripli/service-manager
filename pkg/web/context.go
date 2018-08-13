package web

import (
	"context"

	"github.com/Peripli/service-manager/pkg/sec"
)

type contextKey int

const userKey contextKey = 0

// UserFromContext gets the authenticated user from the context
func UserFromContext(ctx context.Context) (*sec.User, bool) {
	userStr, ok := ctx.Value(userKey).(*sec.User)
	return userStr, ok
}

// NewContextWithUser sets the authenticated user in the context
func NewContextWithUser(ctx context.Context, user *sec.User) context.Context {
	return context.WithValue(ctx, userKey, user)
}
