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

	"github.com/Peripli/service-manager/pkg/types"
)

const (
	minTimePeriod = time.Nanosecond
)

// Settings type to be loaded from the environment
type Settings struct {
	ActionTimeout                  time.Duration `mapstructure:"action_timeout" description:"timeout for async operations"`
	ReconciliationOperationTimeout time.Duration `mapstructure:"reconciliation_operation_timeout" description:"the maximum allowed timeout for auto rescheduling of operation actions"`
	CascadeOrphanMitigationTimeout time.Duration `mapstructure:"cascade_orphan_mitigation_timeout" description:"maximum allowed timeout for orphan mitigation of cascade operations"`

	PollCascadeInterval     time.Duration `mapstructure:"poll_cascade_interval" description:"poll interval for cascade operations"`
	CleanupInterval         time.Duration `mapstructure:"cleanup_interval" description:"cleanup interval of old operations"`
	MaintainerRetryInterval time.Duration `mapstructure:"maintainer_retry_interval" description:"maintenance retry interval"`
	Lifespan                time.Duration `mapstructure:"lifespan" description:"after that time is passed since its creation, the operation can be cleaned up by the maintainer"`

	ReschedulingInterval time.Duration `mapstructure:"rescheduling_interval" description:"the interval between auto rescheduling of operation actions"`
	PollingInterval      time.Duration `mapstructure:"polling_interval" description:"the interval between polls for async requests"`

	DefaultPoolSize               int            `mapstructure:"default_pool_size" description:"default worker pool size"`
	DefaultCascadePollingPoolSize int            `mapstructure:"default_cascade_polling_pool_size" description:"default worker pool size"`
	Pools                         []PoolSettings `mapstructure:"pools" description:"defines the different available worker pools"`

	SMSupportedPlatformType []string `mapstructure:"sm_supported_platform_type" description:"defines the value of the supported platform aliases for the SM platform"`
}

// DefaultSettings returns default values for API settings
func DefaultSettings() *Settings {
	return &Settings{
		ActionTimeout:                  15 * time.Minute,
		ReconciliationOperationTimeout: 7 * 24 * time.Hour,
		CascadeOrphanMitigationTimeout: 6 * time.Hour,
		CleanupInterval:                1 * time.Hour,
		MaintainerRetryInterval:        10 * time.Minute,
		Lifespan:                       7 * 24 * time.Hour,
		ReschedulingInterval:           1 * time.Hour,
		PollingInterval:                4 * time.Second,
		PollCascadeInterval:            4 * time.Second,
		DefaultPoolSize:                20,
		DefaultCascadePollingPoolSize:  20,
		Pools:                          []PoolSettings{},
		SMSupportedPlatformType:        []string{types.SMPlatform},
	}
}

// Validate validates the Operations settings
func (s *Settings) Validate() error {
	if s.ActionTimeout <= minTimePeriod {
		return fmt.Errorf("validate Settings: ActionTimeout must be larger than %s", minTimePeriod)
	}
	if s.CleanupInterval <= minTimePeriod {
		return fmt.Errorf("validate Settings: CleanupInterval must be larger than %s", minTimePeriod)
	}
	if s.ReconciliationOperationTimeout <= minTimePeriod {
		return fmt.Errorf("validate Settings: ReconciliationOperationTimeout must be larger than %s", minTimePeriod)
	}
	if s.ReschedulingInterval <= minTimePeriod {
		return fmt.Errorf("validate Settings: ReschedulingInterval must be larger than %s", minTimePeriod)
	}
	if s.PollingInterval <= minTimePeriod {
		return fmt.Errorf("validate Settings: PollingInterval must be larger than %s", minTimePeriod)
	}
	if s.MaintainerRetryInterval <= minTimePeriod {
		return fmt.Errorf("validate Settings: MaintainerRetryInterval must be larger than %s", minTimePeriod)
	}
	if s.DefaultPoolSize <= 0 {
		return fmt.Errorf("validate Settings: DefaultPoolSize must be larger than 0")
	}
	if s.DefaultCascadePollingPoolSize <= 0 {
		return fmt.Errorf("validate Settings: DefaultCascadePollingPoolSize must be larger than 0")
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
