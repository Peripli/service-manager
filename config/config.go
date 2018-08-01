/*
 * Copyright 2018 The Service Manager Authors
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */

package config

import (
	"time"

	"github.com/Peripli/service-manager/api"
	"github.com/Peripli/service-manager/pkg/env"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/server"
	"github.com/Peripli/service-manager/storage"
	"github.com/spf13/pflag"
)

// Settings is used to setup the Service Manager
type Settings struct {
	Server  server.Settings
	Storage storage.Settings
	Log     log.Settings
	API     api.Settings
}

// AddPFlags adds the SM config flags to the provided flag set
func AddPFlags(set *pflag.FlagSet) {
	env.CreatePFlags(set, DefaultSettings())
	env.CreatePFlagsForConfigFile(set)
}

// DefaultSettings returns the default values for configuring the Service Manager
func DefaultSettings() *Settings {
	config := &Settings{
		Server: server.Settings{
			Port:            8080,
			RequestTimeout:  time.Second * 3,
			ShutdownTimeout: time.Second * 3,
		},
		Storage: storage.Settings{
			URI: "",
		},
		Log: log.Settings{
			Level:  "debug",
			Format: "text",
		},
		API: api.Settings{
			TokenIssuerURL: "",
			ClientID:       "sm",
			Security: api.Security{
				EncryptionKey: "",
			},
		},
	}
	return config
}

// New creates a configuration from the provided env
func New(env env.Environment) (*Settings, error) {
	config := &Settings{}
	if err := env.Unmarshal(config); err != nil {
		return nil, err
	}

	return config, nil
}

// Validate validates that the configuration contains all mandatory properties
func (c *Settings) Validate() error {
	if err := c.Server.Validate(); err != nil {
		return err
	}
	if err := c.Log.Validate(); err != nil {
		return err
	}
	if err := c.API.Validate(); err != nil {
		return err
	}
	if err := c.Storage.Validate(); err != nil {
		return err
	}
	return nil
}
