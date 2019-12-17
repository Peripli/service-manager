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
	PoolSize        int           `mapstructure:"pool_size" description:"pool size denoting the maximum number of concurrent API operations capable of being processed per API resource"`
	JobTimeout      time.Duration `mapstructure:"job_timeout" description:"timeout for async operations"`
	CleanupInterval time.Duration `mapstructure:"cleanup_interval" description:"cleanup interval of old operations"`
}

// DefaultSettings returns default values for API settings
func DefaultSettings() *Settings {
	return &Settings{
		PoolSize:        1000,
		JobTimeout:      10 * time.Minute,
		CleanupInterval: 60 * time.Minute,
	}
}

// Validate validates the Operations settings
func (s *Settings) Validate() error {
	if s.PoolSize <= 0 {
		return fmt.Errorf("validate Settings: PoolSize must be larger than 0")
	}
	if s.JobTimeout < minJobTimeout {
		return fmt.Errorf("validate Settings: JobTimeout must be larger than %s", minJobTimeout)
	}
	if s.CleanupInterval < minCleanupInterval {
		return fmt.Errorf("validate Settings: CleanupInterval must be larger than %s", minCleanupInterval)
	}

	return nil
}

// OperationError holds an error message returned from an execution of an async job
type OperationError struct {
	Message string `json:"message"`
}
