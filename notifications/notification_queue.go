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
	"fmt"

	"github.com/gofrs/uuid"

	"github.com/Peripli/service-manager/pkg/types"
)

var _ NotificationQueue = &notificationQueue{}

// NewNotificationQueue returns new NotificationQueue with specific size
func NewNotificationQueue(size int) (NotificationQueue, error) {
	idBytes, err := uuid.NewV4()
	if err != nil {
		return nil, fmt.Errorf("could not generate uuid %v", err)
	}
	return &notificationQueue{
		isClosed:             false,
		size:                 size,
		notificationsChannel: make(chan *types.Notification, size),
		id:                   idBytes.String(),
	}, nil
}

type notificationQueue struct {
	isClosed             bool
	size                 int
	notificationsChannel chan *types.Notification
	id                   string
}

func (nq *notificationQueue) Enqueue(notification *types.Notification) error {
	if nq.isClosed {
		return ErrQueueClosed
	}
	if len(nq.notificationsChannel) >= nq.size {
		return ErrQueueFull
	}
	nq.notificationsChannel <- notification
	return nil
}

func (nq *notificationQueue) Next() (*types.Notification, error) {
	notification, ok := <-nq.notificationsChannel
	if !ok {
		return nil, ErrQueueClosed
	}
	return notification, nil
}

func (nq *notificationQueue) Close() {
	nq.isClosed = true
	close(nq.notificationsChannel)
}

func (nq *notificationQueue) ID() string {
	return nq.id
}
