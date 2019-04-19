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

package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"

	"github.com/Peripli/service-manager/storage"
	"github.com/Peripli/service-manager/storage/postgres"
)

// NotificationStorage storage for getting and listening for notifications
//go:generate counterfeiter . NotificationStorage
type NotificationStorage interface {
	storage.Storage

	// GetLastRevision returns the last received notification revision
	GetLastRevision(ctx context.Context) (int64, error)

	// NewConnection returns new connection with callback for events
	NewConnection(eventCallback func(isRunning bool, err error)) NotificationConnection
}

// NewNotificationStorage returns storage for notifications
func NewNotificationStorage(st storage.Storage, settings Settings) (NotificationStorage, error) {
	pgStorage, ok := st.(*postgres.PostgresStorage)
	if !ok {
		return nil, errors.New("expected notification storage to be Postgres")
	}
	return &notificationStorage{
		PostgresStorage: pgStorage,
		settings:        settings,
	}, nil
}

type notificationStorage struct {
	*postgres.PostgresStorage
	settings Settings
}

func (ns *notificationStorage) GetLastRevision(ctx context.Context) (int64, error) {
	notification := &postgres.Notification{}
	sqlString := fmt.Sprintf("SELECT revision FROM %s ORDER BY DESC LIMIT 1", postgres.NotificationTable)
	err := ns.SelectContext(ctx, notification, sqlString)
	if err != nil && err != sql.ErrNoRows {
		return 0, fmt.Errorf("could not get last notification revision from db %v", err)
	}
	return notification.Revision, nil
}

func (ns *notificationStorage) NewConnection(eventCallback func(isRunning bool, err error)) NotificationConnection {
	return pq.NewListener(ns.settings.URI, ns.settings.MinReconnectInterval, ns.settings.MaxReconnectInterval, func(event pq.ListenerEventType, err error) {
		switch event {
		case pq.ListenerEventConnected, pq.ListenerEventReconnected:
			eventCallback(true, err)
		case pq.ListenerEventDisconnected, pq.ListenerEventConnectionAttemptFailed:
			eventCallback(false, err)
		}
	})
}
