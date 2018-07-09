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

package servicemanager

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/Peripli/service-manager/rest"
	"github.com/Peripli/service-manager/server"

	"github.com/Peripli/service-manager/app"
	"github.com/Peripli/service-manager/cf"
	"github.com/Peripli/service-manager/config"
	"github.com/Peripli/service-manager/pkg/env"
	"github.com/Peripli/service-manager/pkg/plugin"
	"github.com/Peripli/service-manager/pkg/web"

	"github.com/sirupsen/logrus"
)

// ServiceManager service manager struct
type ServiceManager struct {
	context context.Context
	config  *config.Settings
	api     *rest.API
}

// RegisterPlugins adds plugins to the Service Manager
func (sm *ServiceManager) RegisterPlugins(plugins ...plugin.Plugin) {
	sm.api.RegisterPlugins(plugins...)
}

// RegisterFilters adds filters to the Service Manager
func (sm *ServiceManager) RegisterFilters(filters ...web.Filter) {
	sm.api.RegisterFilters(filters...)
}

// Run starts the Service Manager
func (sm *ServiceManager) Run() {
	sm.getServer().Run(sm.context)
}

func (sm *ServiceManager) getServer() *server.Server {
	srv, err := app.New(sm.context, &app.Parameters{
		Settings: sm.config,
		API:      sm.api,
	})
	if err != nil {
		panic(fmt.Sprintf("error creating SM server: %s", err))
	}
	return srv
}

// New returns service-manager server with default setup. The function panics on bad configuration
func New(ctx context.Context, cancel context.CancelFunc) *ServiceManager {
	handleInterrupts(ctx, cancel)

	return &ServiceManager{
		context: ctx,
		config:  getConfig(ctx),
		api:     &rest.API{},
	}
}

func getConfig(ctx context.Context) *config.Settings {
	set := env.EmptyFlagSet()
	config.AddPFlags(set)

	env, err := env.New(set)
	if err != nil {
		panic(fmt.Sprintf("error loading environment: %s", err))
	}
	if err := cf.SetCFOverrides(env); err != nil {
		panic(fmt.Sprintf("error setting CF environment values: %s", err))
	}
	cfg, err := config.New(env)
	if err != nil {
		panic(fmt.Sprintf("error loading configuration: %s", err))
	}
	return cfg
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
