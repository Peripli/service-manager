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

package sm

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/Peripli/service-manager/api"
	"github.com/Peripli/service-manager/server"

	"github.com/Peripli/service-manager/config"
	"github.com/Peripli/service-manager/pkg/env"

	"github.com/Peripli/service-manager/api/filters"
	"github.com/Peripli/service-manager/cf"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
	"github.com/Peripli/service-manager/storage/postgres"
	"github.com/sirupsen/logrus"
)

// ServiceManager  struct
type ServiceManager struct {
	ctx    context.Context
	server *server.Server
}

// New returns service-manager server with default setup. The function panics on bad configuration
func New(ctx context.Context, cancel context.CancelFunc, env env.Environment) *ServiceManager {
	// graceful shutdown and handle interrupts
	handleInterrupts(ctx, cancel)

	// setup config from env
	cfg, err := config.New(env)
	if err != nil {
		panic(fmt.Errorf("error loading configuration: %s", err))
	}
	if err := cfg.Validate(); err != nil {
		panic(fmt.Sprintf("error validating config: %s", err))
	}

	// setup logging
	log.SetupLogging(cfg.Log)

	// setup storage
	storage, err := storage.Use(ctx, postgres.Storage, cfg.Storage.URI)
	if err != nil {
		panic(fmt.Sprintf("error using storage: %s", err))
	}

	// setup core API
	coreAPI, err := api.New(ctx, storage, cfg.API)
	if err != nil {
		panic(fmt.Sprintf("error creating core API: %s", err))
	}

	// setup server and add relevant global middleware
	srv := server.New(cfg.Server, coreAPI)
	srv.Router.Use(filters.NewRecoveryMiddleware())

	return &ServiceManager{
		server: srv,
	}
}

// DefaultEnv creates a default environment that can be used to boot up a Service Manager
func DefaultEnv() env.Environment {
	set := env.EmptyFlagSet()
	config.AddPFlags(set)

	environment, err := env.New(set)
	if err != nil {
		panic(fmt.Errorf("error loading environment: %s", err))
	}
	if err := cf.SetCFOverrides(environment); err != nil {
		panic(fmt.Errorf("error setting CF environment values: %s", err))
	}
	return environment
}

// RegisterPlugins adds plugins to the Service Manager
func (sm *ServiceManager) RegisterPlugins(plugins ...web.Plugin) {
	sm.server.API.RegisterPlugins(plugins...)
}

// RegisterFilters adds filters to the Service Manager
func (sm *ServiceManager) RegisterFilters(filters ...web.Filter) {
	sm.server.API.RegisterFilters(filters...)
}

// RegisterControllers adds controllers to the Service Manager
func (sm *ServiceManager) RegisterControllers(controllers ...web.Controller) {
	sm.server.API.RegisterControllers(controllers...)
}

// ChainMatching starts the Service Manager
func (sm *ServiceManager) Run() {
	sm.server.Run(sm.ctx)
}

func handleInterrupts(ctx context.Context, cancelFunc context.CancelFunc) {
	term := make(chan os.Signal)
	signal.Notify(term, os.Interrupt)
	go func() {
		select {
		case <-term:
			logrus.Error("Received OS interrupt, exiting gracefully...")
			cancelFunc()
		case <-ctx.Done():
			return
		}
	}()
}
