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
	"errors"

	"github.com/Peripli/service-manager/pkg/types"
)

var _ NotificationQueue = &notificationQueue{}

// ErrQueueClosed is returned when queue is closed
var ErrQueueClosed = errors.New("queue closed")

// NewNotificationQueue returns new NotificationQueue with specific size
func NewNotificationQueue(size int) NotificationQueue {
	return &notificationQueue{
		isClosed:             false,
		size:                 size,
		notificationsChannel: make(chan *types.Notification, size),
	}
}

type notificationQueue struct {
	isClosed             bool
	size                 int
	notificationsChannel chan *types.Notification
}

// Enqueue enqueues new notification for processing. If queue is full - error is returned.
// It should not block or execute heavy operations.
func (nq *notificationQueue) Enqueue(notification *types.Notification) error {
	if nq.isClosed {
		return ErrQueueClosed
	}
	if len(nq.notificationsChannel) >= nq.size {
		return errors.New("notification queue is full")
	}
	nq.notificationsChannel <- notification
	return nil
}

// Next returns the next notification which has to be processed.
// If there are no new notifications the call will block.
// If error is returned this means that the NotificationQueue is no longer valid.
func (nq *notificationQueue) Next() (*types.Notification, error) {
	notification, ok := <-nq.notificationsChannel
	if !ok {
		return nil, ErrQueueClosed
	}
	return notification, nil
}

// Close closes the queue.
// Any calls to Next or Enqueue will return ErrQueueClosed
// Any subsequent calls to close does nothing
func (nq *notificationQueue) Close() {
	nq.isClosed = true
	close(nq.notificationsChannel)
}
