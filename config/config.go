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
	"github.com/Peripli/service-manager/pkg/agents"
	"github.com/Peripli/service-manager/storage/cache"

	"github.com/Peripli/service-manager/pkg/multitenancy"

	"github.com/Peripli/service-manager/operations"

	"github.com/Peripli/service-manager/pkg/httpclient"

	"github.com/Peripli/service-manager/api"
	"github.com/Peripli/service-manager/pkg/env"
	"github.com/Peripli/service-manager/pkg/health"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/server"
	"github.com/Peripli/service-manager/pkg/ws"
	"github.com/Peripli/service-manager/storage"
	"github.com/spf13/pflag"
)

// Settings is used to setup the Service Manager
type Settings struct {
	Server       *server.Settings
	Cache        *cache.Settings
	Storage      *storage.Settings
	Log          *log.Settings
	API          *api.Settings
	Operations   *operations.Settings
	WebSocket    *ws.Settings
	HTTPClient   *httpclient.Settings
	Health       *health.Settings
	Multitenancy *multitenancy.Settings
	Agents       *agents.Settings
}

// AddPFlags adds the SM config flags to the provided flag set
func AddPFlags(set *pflag.FlagSet) {
	env.CreatePFlags(set, DefaultSettings())
	env.CreatePFlagsForConfigFile(set)
}

// DefaultSettings returns the default values for configuring the Service Manager
func DefaultSettings() *Settings {
	return &Settings{
		Cache:        cache.DefaultSettings(),
		Server:       server.DefaultSettings(),
		Storage:      storage.DefaultSettings(),
		Log:          log.DefaultSettings(),
		API:          api.DefaultSettings(),
		Operations:   operations.DefaultSettings(),
		WebSocket:    ws.DefaultSettings(),
		HTTPClient:   httpclient.DefaultSettings(),
		Health:       health.DefaultSettings(),
		Multitenancy: multitenancy.DefaultSettings(),
		Agents:       agents.DefaultSettings(),
	}
}

// New creates a configuration from the provided env
func New(env env.Environment) (*Settings, error) {
	config := DefaultSettings()
	if err := env.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("error loading configuration: %s", err)
	}
	return config, nil
}

// Validate validates that the configuration contains all mandatory properties
func (c *Settings) Validate() error {
	validatable := []interface {
		Validate() error
	}{c.Server, c.Storage, c.Log, c.Health, c.API, c.Operations, c.WebSocket, c.Multitenancy, c.Agents, c.HTTPClient}

	for _, item := range validatable {
		if err := item.Validate(); err != nil {
			return err
		}
	}
	return nil
}
