/*
 *    Copyright 2018 The Service Manager Authors
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */

package postgres

import (
	"github.com/Peripli/service-manager/notifications/postgres/storage"
)

// Settings type to be loaded from the environment
type Settings struct {
	NotificationQueuesSize int               `mapstructure:"notification_queues_size"`
	StorageSettings        *storage.Settings `mapstructure:"storage"`
}

// DefaultSettings returns default values for notificator settings
func DefaultSettings() *Settings {
	return &Settings{
		NotificationQueuesSize: 100,
		StorageSettings:        storage.DefaultSettings(),
	}
}

// Validate validates the notificator settings
func (s *Settings) Validate() error {
	return s.StorageSettings.Validate()
}
