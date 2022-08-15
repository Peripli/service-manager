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
	"time"

	notificationConnection "github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/storage/postgres/notification_connection"

	"github.com/lib/pq"
)

//go:generate counterfeiter . notificationConnectionCreator
type notificationConnectionCreator interface {
	// NewConnection returns new connection with callback for events
	NewConnection(eventCallback func(isRunning bool, err error)) notificationConnection.NotificationConnection
}

type notificationConnectionCreatorImpl struct {
	storageURI           string
	skipSSLValidation    bool
	minReconnectInterval time.Duration
	maxReconnectInterval time.Duration
}

func (ncci *notificationConnectionCreatorImpl) NewConnection(eventCallback func(isRunning bool, err error)) notificationConnection.NotificationConnection {
	sslModeParam := ""
	if ncci.skipSSLValidation {
		sslModeParam = "?sslmode=disable"
	}
	return pq.NewListener(ncci.storageURI+sslModeParam, ncci.minReconnectInterval, ncci.maxReconnectInterval, func(event pq.ListenerEventType, err error) {
		switch event {
		case pq.ListenerEventConnected, pq.ListenerEventReconnected:
			eventCallback(true, err)
		case pq.ListenerEventDisconnected, pq.ListenerEventConnectionAttemptFailed:
			eventCallback(false, err)
		}
	})
}
