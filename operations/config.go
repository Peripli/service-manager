package operations

import (
	"fmt"
	"time"
)

const (
	minJobTimeout      = 5 * time.Minute
	minCleanupInterval = 10 * time.Minute
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
		JobTimeout:      10 * time.Minute,
		CleanupInterval: 60 * time.Minute,
		Pools:           []PoolSettings{},
	}
}

// Validate validates the Operations settings
func (s *Settings) Validate() error {
	if s.JobTimeout < minJobTimeout {
		return fmt.Errorf("validate Settings: JobTimeout must be larger than %s", minJobTimeout)
	}
	if s.CleanupInterval < minCleanupInterval {
		return fmt.Errorf("validate Settings: CleanupInterval must be larger than %s", minCleanupInterval)
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
	Resource string `mapstructure:"resourcec" description:"name of the resource for which a worker pool is created"`
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
