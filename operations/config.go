package operations

import (
	"fmt"
	"time"
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
	return nil
}

// JobError represents an error during execution of a scheduled job
// It contains the OperationID so that the error handler can retry to
// either set the operation state Success if operation was successful
// or to Failed if operation was not successful
type JobError struct {
	error
	OperationID         string
	OperationSuccessful bool
}

func (je JobError) Error() string {
	return fmt.Sprintf("job for operation wih ID %s failed to execute: %s", je.OperationID, je.error)
}
