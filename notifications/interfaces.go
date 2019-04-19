/*
 * Copyright 2018 The Service Manager Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package notifications

import (
	"context"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/web"
)

// NotificationQueue is used for processing
type NotificationQueue interface {
	// New enqueues new notification for processing. If queue is full - error is returned.
	// It should not block or execute heavy operations.
	Enqueue(notification *types.Notification) error

	// Next returns the next notification which has to be processed.
	// If there are no new notifications the call will block.
	// If error is returned this means that the NotificationQueue is no longer valid.
	Next() (*types.Notification, error)

	// Close closes the queue
	Close()

	// ID returns unique queue identifier
	ID() string
}

// Notificator is used for receiving notifications for SM events
type Notificator interface {
	// Start starts the Notificator
	Start(ctx context.Context) error

	// RegisterConsumer returns queue with received notifications from Postgres.
	// Returns notification queue, last_known_revision and error if any
	RegisterConsumer(userContext web.UserContext) (NotificationQueue, int64, error)

	// UnregisterConsumer must be called to stop receiving notifications in the queue
	UnregisterConsumer(queue NotificationQueue) error
}
