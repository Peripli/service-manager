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
	"os"

	"github.com/Peripli/service-manager/api"
	cfenv "github.com/Peripli/service-manager/cmd/cf/env"
	k8senv "github.com/Peripli/service-manager/cmd/k8s/env"
	"github.com/Peripli/service-manager/env"
	"github.com/Peripli/service-manager/server"
	"github.com/Peripli/service-manager/storage"
	"github.com/Peripli/service-manager/storage/postgres"
	"github.com/sirupsen/logrus"
)

// NewServer creates service manager server
func NewServer(ctx context.Context, serverEnv server.Environment) (*server.Server, error) {
	config, err := server.NewConfiguration(serverEnv)
	if err != nil {
		return nil, fmt.Errorf("Error loading configuration: %v", err)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("Configuration validation failed: %v", err)
	}

	setUpLogging(config.LogLevel, config.LogFormat)

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

func setUpLogging(logLevel string, logFormat string) {
	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		logrus.Fatal("Could not parse log level configuration")
	}
	logrus.SetLevel(level)
	if logFormat == "json" {
		logrus.SetFormatter(&logrus.JSONFormatter{})
	} else {
		logrus.SetFormatter(&logrus.TextFormatter{})
	}
}

func getK8SConfigFile() *env.ConfigFile {
	configFileLocation := os.Getenv(k8senv.K8SConfigLocationEnvVarName)
	if configFileLocation == "" {
		logrus.Fatalf("Expected %s environment variable to be set", k8senv.K8SConfigLocationEnvVarName)
	}
	return &env.ConfigFile{
		Path:   configFileLocation,
		Name:   "application",
		Format: "yaml",
	}
}

// GetEnvironment returns the specific server.Environment required to run the Service Manager
func GetEnvironment() server.Environment {
	envFlag := os.Getenv("SM_RUN_ENV")

	switch envFlag {
	case "cf":
		return cfenv.New(env.Default())
	case "k8s":
		return k8senv.New(env.New(getK8SConfigFile(), "SM"))
	default:
		return env.Default()
	}
}
