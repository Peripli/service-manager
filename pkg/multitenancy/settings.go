package multitenancy

import "fmt"

type Settings struct {
	LabelKey string `mapstructure:"label_key"`
}

func DefaultSettings() *Settings {
	return &Settings{
		LabelKey: "",
	}
}

// Validate validates the httpclient settings
func (s *Settings) Validate() error {
	if len(s.LabelKey) == 0 {
		return fmt.Errorf("validate multitenancy settings: label_key should not be empty")
	}

	return nil
}
