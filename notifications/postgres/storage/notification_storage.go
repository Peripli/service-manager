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
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
	"github.com/Peripli/service-manager/storage/postgres"
)

// NotificationStorage storage for getting and listening for notifications
//go:generate counterfeiter . NotificationStorage
type NotificationStorage interface {
	GetNotification(ctx context.Context, id string) (*types.Notification, error)

	// GetLastRevision returns the last received notification revision
	GetLastRevision(ctx context.Context) (int64, error)

	// NewConnection returns new connection with callback for events
	NewConnection(eventCallback func(isRunning bool, err error)) NotificationConnection
}

func NewNotificationStorage(st storage.Storage, storageURI string, minReconnectInterval time.Duration, maxReconnectInterval time.Duration) (NotificationStorage, error) {
	pgStorage, ok := st.(*postgres.PostgresStorage)
	if !ok {
		return nil, errors.New("expected notification storage to be Postgres")
	}
	return &notificationStorage{
		storage:              pgStorage,
		storageURI:           storageURI,
		minReconnectInterval: minReconnectInterval,
		maxReconnectInterval: maxReconnectInterval,
	}, nil
}

type notificationStorage struct {
	storage              *postgres.PostgresStorage
	storageURI           string
	minReconnectInterval time.Duration
	maxReconnectInterval time.Duration
}

func (ns *notificationStorage) GetLastRevision(ctx context.Context) (int64, error) {
	result := make([]*postgres.Notification, 0, 1)
	sqlString := fmt.Sprintf("SELECT revision FROM %s ORDER BY revision DESC LIMIT 1", postgres.NotificationTable)
	err := ns.storage.SelectContext(ctx, &result, sqlString)
	if err != nil {
		return 0, fmt.Errorf("could not get last notification revision from db %v", err)
	}
	if len(result) == 0 {
		return 0, nil
	}
	return result[0].Revision, nil
}

func (ns *notificationStorage) GetNotification(ctx context.Context, id string) (*types.Notification, error) {
	notificationObj, err := ns.storage.Get(ctx, types.NotificationType, id)
	if err != nil {
		return nil, err
	}
	return notificationObj.(*types.Notification), nil
}

func (ns *notificationStorage) NewConnection(eventCallback func(isRunning bool, err error)) NotificationConnection {
	return pq.NewListener(ns.storageURI, ns.minReconnectInterval, ns.maxReconnectInterval, func(event pq.ListenerEventType, err error) {
		switch event {
		case pq.ListenerEventConnected, pq.ListenerEventReconnected:
			eventCallback(true, err)
		case pq.ListenerEventDisconnected, pq.ListenerEventConnectionAttemptFailed:
			eventCallback(false, err)
		}
	})
}
