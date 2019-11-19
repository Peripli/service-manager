package notifications

import (
	"context"
	"encoding/json"

	"github.com/Peripli/service-manager/pkg/log"

	"github.com/Peripli/service-manager/pkg/types"
)

// ResourceNotificationHandler can handle notifications by processing the Payload
type ResourceNotificationHandler interface {
	// OnCreate is called when a notification for creating a resource arrives
	OnCreate(ctx context.Context, payload json.RawMessage)

	// OnUpdate is called when a notification for modifying a resource arrives
	OnUpdate(ctx context.Context, payload json.RawMessage)

	// OnDelete is called when a notification for deleting a resource arrives
	OnDelete(ctx context.Context, payload json.RawMessage)
}

// Consumer allows consuming notifications by picking the correct handler to process it
type Consumer struct {
	Handlers map[types.ObjectType]ResourceNotificationHandler
}

// Consume consumes a notification and passes it to the correct handler for further processing
func (c *Consumer) Consume(ctx context.Context, n *types.Notification) {
	notificationHandler, found := c.Handlers[n.Resource]

	if !found {
		log.C(ctx).Warnf("No notification handler found for notification for resource %s. Ignoring notification...", n.Resource)
		return
	}

	correlationID := n.CorrelationID
	if correlationID == "" {
		correlationID = n.ID
	}
	entry := log.C(ctx).WithField(log.FieldCorrelationID, correlationID)
	ctx = log.ContextWithLogger(ctx, entry)

	switch n.Type {
	case types.CREATED:
		notificationHandler.OnCreate(ctx, n.Payload)
	case types.MODIFIED:
		notificationHandler.OnUpdate(ctx, n.Payload)
	case types.DELETED:
		notificationHandler.OnDelete(ctx, n.Payload)
	}
}
