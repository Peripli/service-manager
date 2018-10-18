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

package sbproxy

import (
	"github.com/Peripli/service-manager/pkg/env"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/sbproxy/reconcile"
	"github.com/Peripli/service-manager/pkg/sbproxy/sm"
	"github.com/Peripli/service-manager/pkg/server"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/spf13/pflag"
)

// Settings type holds all config properties for the sbproxy
type Settings struct {
	Server    *server.Settings    `mapstructure:"server"`
	Log       *log.Settings       `mapstructure:"log"`
	Sm        *sm.Settings        `mapstructure:"sm"`
	Reconcile *reconcile.Settings `mapstructure:"app"`
}

// DefaultSettings returns default value for the proxy settings
func DefaultSettings() *Settings {
	return &Settings{
		Server:    server.DefaultSettings(),
		Log:       log.DefaultSettings(),
		Sm:        sm.DefaultSettings(),
		Reconcile: reconcile.DefaultSettings(),
	}
}

// NewSettings creates new proxy settings from the specified environment
func NewSettings(env env.Environment) (*Settings, error) {
	config := DefaultSettings()
	if err := env.Unmarshal(config); err != nil {
		return nil, err
	}

	return config, nil
}

// AddPFlags adds the SM config flags to the provided flag set
func AddPFlags(set *pflag.FlagSet) {
	env.CreatePFlags(set, DefaultSettings())

	env.CreatePFlagsForConfigFile(set)
}

// Validate validates that the configuration contains all mandatory properties
func (c *Settings) Validate() error {
	validatable := []util.InputValidator{c.Server, c.Log, c.Sm, c.Reconcile}

	for _, item := range validatable {
		if err := item.Validate(); err != nil {
			return err
		}
	}
	return nil
}
