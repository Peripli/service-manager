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

package notification_connection

import "github.com/lib/pq"

// NotificationConnection is a connection for listening for notifications
//go:generate counterfeiter . NotificationConnection
type NotificationConnection interface {
	// Listen starts listening a channel
	Listen(channel string) error

	// Unlisten stops listening a channel
	Unlisten(channel string) error

	// Close closes the connection
	Close() error

	// Ping the remote server to make sure it's alive.  Non-nil return value means
	// that there is no active connection.
	Ping() error

	// NotificationChannel returns channel for receiving notifications
	NotificationChannel() <-chan *pq.Notification
}
