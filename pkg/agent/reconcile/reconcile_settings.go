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

package reconcile

import (
	"fmt"
)

// DefaultProxyBrokerPrefix prefix for brokers registered by the proxy
const DefaultProxyBrokerPrefix = "sm-"

// Settings type represents the sbproxy settings
type Settings struct {
	LegacyURL           string   `mapstructure:"legacy_url"`
	MaxParallelRequests int      `mapstructure:"max_parallel_requests"`
	URL                 string   `mapstructure:"url"`
	BrokerPrefix        string   `mapstructure:"broker_prefix"`
	BrokerBlacklist     []string `mapstructure:"broker_blacklist"`
	TakeoverEnabled     bool     `mapstructure:"takeover_enabled"`
}

// DefaultSettings creates default proxy settings
func DefaultSettings() *Settings {
	return &Settings{
		LegacyURL:           "",
		MaxParallelRequests: 5,
		URL:                 "",
		BrokerPrefix:        DefaultProxyBrokerPrefix,
		BrokerBlacklist:     []string{},
		TakeoverEnabled:     true,
	}
}

// Validate validates that the configuration contains all mandatory properties
func (c *Settings) Validate() error {
	if c.LegacyURL == "" {
		return fmt.Errorf("validate settings: missing legacy url")
	}
	if c.URL == "" {
		return fmt.Errorf("validate settings: missing url")
	}
	if c.MaxParallelRequests <= 0 {
		return fmt.Errorf("validate settings: max parallel requests must be positive number")
	}

	return nil
}
