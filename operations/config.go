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

package operations

import (
	"fmt"
	"time"
)

const (
	MinPoolSize   = 10
	minTimePeriod = time.Second
)

// Settings type to be loaded from the environment
type Settings struct {
	JobTimeout      time.Duration  `mapstructure:"job_timeout" description:"timeout for async operations"`
	CleanupInterval time.Duration  `mapstructure:"cleanup_interval" description:"cleanup interval of old operations"`
	Pools           []PoolSettings `mapstructure:"pools" description:"defines the different available worker pools"`
}

// DefaultSettings returns default values for API settings
func DefaultSettings() *Settings {
	return &Settings{
		JobTimeout:      5 * time.Minute,
		CleanupInterval: 10 * time.Minute,
		Pools:           []PoolSettings{},
	}
}

// Validate validates the Operations settings
func (s *Settings) Validate() error {
	if s.JobTimeout <= minTimePeriod {
		return fmt.Errorf("validate Settings: JobTimeout must be larger than %s", minTimePeriod)
	}
	if s.CleanupInterval <= minTimePeriod {
		return fmt.Errorf("validate Settings: CleanupInterval must be larger than %s", minTimePeriod)
	}
	for _, pool := range s.Pools {
		if err := pool.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// PoolSettings defines the settings for a worker pool
type PoolSettings struct {
	Resource string `mapstructure:"resource" description:"name of the resource for which a worker pool is created"`
	Size     int    `mapstructure:"size" description:"size of the worker pool"`
}

// Validate validates the Pool settings
func (ps *PoolSettings) Validate() error {
	if ps.Size <= 0 {
		return fmt.Errorf("validate Settings: Pool size for resource '%s' must be larger than 0", ps.Resource)
	}

	return nil
}

// OperationError holds an error message returned from an execution of an async job
type OperationError struct {
	Message string `json:"message"`
}
