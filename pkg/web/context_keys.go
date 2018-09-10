package web

import (
	"context"

	"github.com/Peripli/service-manager/pkg/audit"
)

type contextKey int

const (
	userKey contextKey = iota
	auditKey
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

func ContextWithAuditEvent(ctx context.Context, event *audit.Event) context.Context {
	return context.WithValue(ctx, auditKey, event)
}

func AuditEventFromContext(ctx context.Context) *audit.Event {
	event, _ := ctx.Value(auditKey).(*audit.Event)
	return event
}
