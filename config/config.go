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
	"fmt"
	"time"

	"github.com/Peripli/service-manager/api"
	"github.com/Peripli/service-manager/log"
	"github.com/Peripli/service-manager/server"
	"github.com/Peripli/service-manager/storage"
)

// Settings type to be loaded from the env
type Config struct {
	Server  server.Settings
	Storage storage.Settings
	Log     log.Settings
	API     api.Settings
}

func DefaultConfig() *Config {
	config := &Config{
		Server: server.Settings{
			Host:            "127.0.0.1",
			Port:            8080,
			RequestTimeout:  time.Millisecond * time.Duration(3000),
			ShutdownTimeout: time.Millisecond * time.Duration(3000),
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
		},
	}
	return config
}

// New creates a configuration from the provided env
func New(env Environment) (*Config, error) {
	config := DefaultConfig()

	if err := env.CreatePFlags(config); err != nil {
		return nil, err
	}

	if err := env.Load(); err != nil {
		return nil, err
	}
	if err := env.Unmarshal(config); err != nil {
		return nil, err
	}

	return config, nil
}

// Validate validates that the configuration contains all mandatory properties
func (c *Config) Validate() error {
	if c.Server.Port == 0 {
		return fmt.Errorf("validate Settings: Port missing")
	}
	if c.Server.RequestTimeout == 0 {
		return fmt.Errorf("validate Settings: RequestTimeout missing")
	}
	if c.Server.ShutdownTimeout == 0 {
		return fmt.Errorf("validate Settings: ShutdownTimeout missing")
	}
	if len(c.Log.Level) == 0 {
		return fmt.Errorf("validate Settings: LogLevel missing")
	}
	if len(c.Log.Format) == 0 {
		return fmt.Errorf("validate Settings: LogFormat missing")
	}
	if len(c.Storage.URI) == 0 {
		return fmt.Errorf("validate Settings: StorageURI missing")
	}
	if (len(c.API.TokenIssuerURL)) == 0 {
		return fmt.Errorf("validate Settings: APITokenIssuerURL missing")
	}
	return nil
}
