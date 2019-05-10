package ws

import (
	"fmt"
	"time"
)

type Settings struct {
	PingTimeout  time.Duration `mapstructure:"ping_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

// DefaultSettings return the default values for ws server
func DefaultSettings() *Settings {
	return &Settings{
		PingTimeout:  time.Second * 5,
		WriteTimeout: time.Second * 5,
	}
}

// Validate validates the ws server settings
func (s *Settings) Validate() error {
	if s.PingTimeout <= 0 {
		return fmt.Errorf("validate ws settings: PingTimeut should be > 0")
	}

	if s.WriteTimeout <= 0 {
		return fmt.Errorf("validate ws settings: WriteTimeout should be > 0")
	}

	return nil
}
