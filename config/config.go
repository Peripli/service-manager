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
	"github.com/Peripli/service-manager/security"
	"github.com/Peripli/service-manager/server"
	"github.com/Peripli/service-manager/storage"
	"github.com/spf13/pflag"
)

// Environment represents an abstraction over the env from which Service Manager configuration will be loaded
//go:generate counterfeiter . Environment
type Environment interface {
	Get(key string) interface{}
	Set(key string, value interface{})
	Unmarshal(value interface{}) error
	BindPFlag(key string, flag *pflag.Flag) error
}

// Settings is used to setup the Service Manager
type Settings struct {
	Server   server.Settings
	Storage  storage.Settings
	Security security.Settings
	Log      log.Settings
	API      api.Settings
}

// File describes the name, path and the format of the file to be used to load the configuration in the env
type File struct {
	Name     string
	Location string
	Format   string
}

// DefaultSettings returns the default values for configuring the Service Manager
func DefaultSettings() *Settings {
	config := &Settings{
		Server: server.Settings{
			Port:            8080,
			RequestTimeout:  time.Millisecond * time.Duration(3000),
			ShutdownTimeout: time.Millisecond * time.Duration(3000),
		},
		Storage: storage.Settings{
			URI: "",
		},
		Security: security.Settings{
			EncryptionKey: "",
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

// DefaultFile holds the default SM config file properties
func DefaultFile() File {
	return File{
		Name:     "application",
		Location: ".",
		Format:   "yml",
	}
}

// AddPFlags adds the SM config flags to the provided flag set
func AddPFlags(set *pflag.FlagSet) {
	CreatePFlags(set, DefaultSettings())
	CreatePFlags(set, struct{ File File }{File: DefaultFile()})
}

// SMFlagSet creates an empty flag set and adds the default se of flags to it
func SMFlagSet() *pflag.FlagSet {
	set := pflag.NewFlagSet("Service Manager Configuration Flags", pflag.ExitOnError)
	set.AddFlagSet(pflag.CommandLine)
	return set
}

// New creates a configuration from the provided env
func New(env Environment) (*Settings, error) {
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
	return nil
}
