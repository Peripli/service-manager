/*
 *    Copyright 2018 The Service Manager Authors
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

// Package app contains logic for setting up the service manager server
package app

import (
	"context"
	"fmt"

	"github.com/Peripli/service-manager/api"
	"github.com/Peripli/service-manager/config"
	"github.com/Peripli/service-manager/log"
	"github.com/Peripli/service-manager/rest"
	"github.com/Peripli/service-manager/server"
	"github.com/Peripli/service-manager/storage"
	"github.com/Peripli/service-manager/storage/postgres"
	"github.com/Peripli/service-manager/filters/auth"
)

// Parameters contains settings for configuring a Service Manager server and optional extensions API
type Parameters struct {
	Settings *config.Settings

	// API can define REST API extensions
	API *rest.API
}

// New creates a SM server
func New(ctx context.Context, params *Parameters) (*server.Server, error) {
	if err := params.Settings.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %v", err)
	}

	log.SetupLogging(params.Settings.Log)

	storage, err := storage.Use(ctx, postgres.Storage, params.Settings.Storage.URI)
	if err != nil {
		return nil, fmt.Errorf("error using storage: %v", err)
	}

	coreAPI := api.New(storage, params.Settings.API)
	registerDefaultFilters(coreAPI, storage, params.Settings)
	if params.API != nil {
		coreAPI.RegisterControllers(params.API.Controllers...)
		coreAPI.RegisterFilters(params.API.Filters...)
	}

	return server.New(coreAPI, params.Settings.Server)
}

func registerDefaultFilters(api *rest.API, storage storage.Storage, cfg *config.Settings) {
	authFilter := auth.NewAuthenticationFilter(storage.Credentials(), cfg.API.TokenIssuerURL, cfg.CLI.ClientID)
	api.RegisterFilters(authFilter.Filters()...)
}
