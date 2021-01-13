package agents

import (
	"encoding/json"
	"fmt"
)

type Settings struct {
	Versions string `mapstructure:"versions"`
}

func DefaultSettings() *Settings {
	return &Settings{
		Versions: "{}",
	}
}

func (s *Settings) Validate() error {
	var supportedVersions = make(map[string][]string)
	err := json.Unmarshal([]byte(s.Versions), &supportedVersions)
	if err != nil {
		return fmt.Errorf("validate agents settings: failed to unmarshal versions, invalid json: %s", err.Error())
	}
	return nil

}
