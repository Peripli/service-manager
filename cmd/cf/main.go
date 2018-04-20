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

package main

import (
	"context"
	"github.com/Peripli/service-manager/env"
	"github.com/Peripli/service-manager/server"
	"fmt"
	"github.com/Peripli/service-manager/storage"
	"github.com/Peripli/service-manager/storage/postgres"
	"github.com/Peripli/service-manager/api"
	cfenv "github.com/Peripli/service-manager/cmd/cf/env"
)

func main() {
	ctx := context.Background()
	server, err := CreateServer(ctx, cfenv.New(env.Default()))
	if err != nil {
		panic(err)
	}
	server.Run(ctx)
}

// CreateServer creates service manager server
func CreateServer(ctx context.Context, serverEnv server.Environment) (*server.Server, error) {
	config, err := server.NewConfiguration(serverEnv)
	if err != nil {
		return nil, fmt.Errorf("Error loading configuration: %v", err)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("Configuration validation failed: %v", err)
	}

	storage, err := storage.Use(ctx, postgres.Storage, config.DbURI)
	if err != nil {
		return nil, fmt.Errorf("Error using storage: %v", err)
	}
	defaultAPI := api.Default(storage)

	srv, err := server.New(defaultAPI, config)
	if err != nil {
		return nil, fmt.Errorf("Error creating server: %v", err)
	}
	return srv, nil
}