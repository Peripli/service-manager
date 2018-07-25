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
	"github.com/Peripli/service-manager/authentication"
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
	OAuth   authentication.OAuthSettings
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
			Security: api.Security{
				EncryptionKey: "",
				URI:           "",
			},
		},
		OAuth: authentication.OAuthSettings{
			ClientID: "",
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
	if c.Security.EncryptionKey == "" {
		return fmt.Errorf("validate Settings: SecurityEncryptionkey missing")
	}
	if len(c.Security.URI) == 0 {
		return fmt.Errorf("validate Settings: SecurityURI missing")
	}
	if (len(c.OAuth.ClientID)) == 0 {
		return fmt.Errorf("validate Settings: CLIClientID missing")
	}
	return nil
}
