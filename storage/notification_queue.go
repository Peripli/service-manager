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

package storage

import (
	"fmt"
	"sync"

	"github.com/gofrs/uuid"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types"
)

var _ NotificationQueue = &notificationQueue{}

// NewNotificationQueue returns new NotificationQueue with specific size
func NewNotificationQueue(size int) (*notificationQueue, error) {
	idBytes, err := uuid.NewV4()
	if err != nil {
		return nil, fmt.Errorf("could not generate uuid %v", err)
	}
	return &notificationQueue{
		isClosed:             false,
		size:                 size,
		notificationsChannel: make(chan *types.Notification, size),
		mutex:                &sync.Mutex{},
		id:                   idBytes.String(),
	}, nil
}

type notificationQueue struct {
	isClosed             bool
	size                 int
	notificationsChannel chan *types.Notification
	mutex                *sync.Mutex
	id                   string
}

// Enqueue adds a new notification for processing. If queue is full ErrQueueFull should be returned.
// It should not block or execute heavy operations.
func (nq *notificationQueue) Enqueue(notification *types.Notification) error {
	nq.mutex.Lock()
	defer nq.mutex.Unlock()
	if nq.isClosed {
		return ErrQueueClosed
	}
	if len(nq.notificationsChannel) >= nq.size {
		return ErrQueueFull
	}
	nq.notificationsChannel <- notification
	return nil
}

// Channel returns the go channel with received notifications which has to be processed.
// If error is returned this means that the NotificationQueue is no longer valid.
func (nq *notificationQueue) Channel() <-chan *types.Notification {
	return nq.notificationsChannel
}

// Close closes the queue.
// Any subsequent calls to Next or Enqueue will return ErrQueueClosed.
// Any subsequent calls to Close does nothing.
func (nq *notificationQueue) Close() {
	nq.mutex.Lock()
	defer nq.mutex.Unlock()
	if nq.isClosed {
		return
	}
	nq.isClosed = true
	close(nq.notificationsChannel)
}

func (nq *notificationQueue) ID() string {
	return nq.id
}
