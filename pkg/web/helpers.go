package web

import (
	"context"
	"net/http"

	"github.com/Peripli/service-manager/pkg/util/slice"
	"github.com/gofrs/uuid"
)

type contextKey int

const (
	userKey contextKey = iota
)

var correlationIDHeaders = []string{"X-Correlation-ID", "X-CorrelationID", "X-ForRequest-ID", "X-Vcap-Request-Id"}

// UserFromContext gets the authenticated user from the context
func UserFromContext(ctx context.Context) (*User, bool) {
	userStr, ok := ctx.Value(userKey).(*User)
	return userStr, ok
}

// NewContextWithUser sets the authenticated user in the context
func NewContextWithUser(ctx context.Context, user *User) context.Context {
	return context.WithValue(ctx, userKey, user)
}

// CorrelationIDFromHeaders returns checks the http headers for any of the supported correlation id headers.
// The first that matches is taken as the correlation id. If none exists a new one is generated.
func CorrelationIDFromHeaders(header http.Header) string {
	for key, val := range header {
		if slice.StringsAnyEquals(correlationIDHeaders, key) {
			return val[0]
		}
	}
	uuids, err := uuid.NewV4()
	if err != nil {
		return "default"
	}
	return uuids.String()
}
