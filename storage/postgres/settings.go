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
	"fmt"
	"time"
)

// Settings type to be loaded from the environment
type Settings struct {
	NotificationQueuesSize int           `mapstructure:"notification_queues_size"`
	MinReconnectInterval   time.Duration `mapstructure:"min_reconnect_interval"`
	MaxReconnectInterval   time.Duration `mapstructure:"max_reconnect_interval"`
}

// DefaultSettings returns default values for Notificator settings
func DefaultSettings() *Settings {
	return &Settings{
		NotificationQueuesSize: 100,
		MinReconnectInterval:   time.Millisecond * 200,
		MaxReconnectInterval:   time.Second * 20,
	}
}

// Validate validates the Notificator settings
func (s *Settings) Validate() error {
	if s.NotificationQueuesSize < 1 {
		return fmt.Errorf("notification queues size (%d) should be at lest 1", s.NotificationQueuesSize)
	}
	if s.MinReconnectInterval > s.MaxReconnectInterval {
		return fmt.Errorf("min reconnect interval (%s) should not be greater than max reconnect interval (%s)",
			s.MinReconnectInterval, s.MaxReconnectInterval)
	}
	return nil
}
