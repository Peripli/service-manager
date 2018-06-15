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

// Package sm contains logic for setting up the service manager server
package sm

import (
	"context"
	"fmt"

	"github.com/Peripli/service-manager/api"
	"github.com/Peripli/service-manager/server"
	"github.com/Peripli/service-manager/storage"
	"github.com/Peripli/service-manager/storage/postgres"
	"github.com/sirupsen/logrus"
)

// New creates service manager server
func New(ctx context.Context, serverEnv server.Environment) (*server.Server, error) {
	config, err := server.NewConfig(serverEnv)
	if err != nil {
		return nil, fmt.Errorf("error loading configuration: %v", err)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %v", err)
	}

	setUpLogging(config.Log)

	storage, err := storage.Use(ctx, postgres.Storage, config.DB.URI)
	if err != nil {
		return nil, fmt.Errorf("error using storage: %v", err)
	}

	defaultAPI := api.Default(storage, serverEnv)
	srv, err := server.New(defaultAPI, config)
	if err != nil {
		return nil, fmt.Errorf("error creating server: %v", err)
	}
	return srv, nil
}

func setUpLogging(settings server.LogSettings) {
	level, err := logrus.ParseLevel(settings.Level)
	if err != nil {
		logrus.Fatal("Could not parse log level configuration")
	}
	logrus.SetLevel(level)
	if settings.Format == "json" {
		logrus.SetFormatter(&logrus.JSONFormatter{})
	} else {
		logrus.SetFormatter(&logrus.TextFormatter{})
	}
}
