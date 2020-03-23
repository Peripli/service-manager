package osb

import (
	"fmt"
	"time"
)

type Settings struct {
	Timeout time.Duration `mapstructure:"timeout"`
}

// DefaultSettings return the default values for OSB requests
func DefaultSettings() *Settings {
	return &Settings{
		Timeout: time.Second * 30,
	}
}

// Validate validates the OSB requests settings
func (s *Settings) Validate() error {
	if s.Timeout <= 0 {
		return fmt.Errorf("validate osb settings: Timeout should be > 0")
	}

	return nil
}
