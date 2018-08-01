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
	"crypto/rand"
	"fmt"
	"os"
	"os/signal"

	"github.com/Peripli/service-manager/api"
	"github.com/Peripli/service-manager/config"
	"github.com/Peripli/service-manager/security"
	"github.com/Peripli/service-manager/server"
	"github.com/Peripli/service-manager/storage"
	"github.com/Peripli/service-manager/storage/postgres"

	"github.com/Peripli/service-manager/api/filters"
	"github.com/Peripli/service-manager/cf"
	"github.com/Peripli/service-manager/pkg/env"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
)

// DefaultEnv creates a default environment that can be used to boot up a Service Manager
func DefaultEnv(additionalPFlags ...func(set *pflag.FlagSet)) env.Environment {
	set := env.EmptyFlagSet()
	config.AddPFlags(set)
	for _, addFlags := range additionalPFlags {
		addFlags(set)
	}

	environment, err := env.New(set)
	if err != nil {
		panic(fmt.Errorf("error loading environment: %s", err))
	}
	if err := cf.SetCFOverrides(environment); err != nil {
		panic(fmt.Errorf("error setting CF environment values: %s", err))
	}
	return environment
}

// New returns service-manager Server with default setup. The function panics on bad configuration
func New(ctx context.Context, cancel context.CancelFunc, env env.Environment) *ServiceManagerBuilder {
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

	// setup smStorage
	smStorage, err := storage.Use(ctx, postgres.Storage, cfg.Storage.URI, []byte(cfg.API.Security.EncryptionKey))
	if err != nil {
		panic(fmt.Sprintf("error using smStorage: %s", err))
	}

	securityStorage := smStorage.Security()
	if err := initializeSecureStorage(securityStorage); err != nil {
		panic(fmt.Sprintf("error initialzing secure storage: %v", err))
	}

	encrypter := &security.TwoLayerEncrypter{
		Fetcher: securityStorage.Fetcher(),
	}

	// setup core API
	API, err := api.New(ctx, smStorage, cfg.API, encrypter)
	if err != nil {
		panic(fmt.Sprintf("error creating core API: %s", err))
	}

	return &ServiceManagerBuilder{
		ctx:    ctx,
		cancel: cancel,
		cfg:    cfg.Server,
		API:    API,
	}
}

// ServiceManager  struct
type ServiceManager struct {
	ctx    context.Context
	Server *server.Server
}

// Run starts the Service Manager
func (sm *ServiceManager) Run() {
	sm.Server.Run(sm.ctx)
}

// ServiceManagerBuilder type is an extension point that allows adding additional filters, plugins and
// controllers before running ServiceManager.
type ServiceManagerBuilder struct {
	ctx    context.Context
	cancel context.CancelFunc
	cfg    server.Settings
	API    *web.API
}

// RegisterPlugins adds plugins to the Service Manager
func (smb *ServiceManagerBuilder) RegisterPlugins(plugins ...web.Plugin) {
	smb.API.RegisterPlugins(plugins...)
}

// RegisterFilters adds filters to the Service Manager
func (smb *ServiceManagerBuilder) RegisterFilters(filters ...web.Filter) {
	smb.API.RegisterFilters(filters...)
}

// RegisterControllers adds controllers to the Service Manager
func (smb *ServiceManagerBuilder) RegisterControllers(controllers ...web.Controller) {
	smb.API.RegisterControllers(controllers...)
}

// Build builds the Service Manager
func (smb *ServiceManagerBuilder) Build() *ServiceManager {
	// setup server and add relevant global middleware
	srv := server.New(smb.cfg, smb.API)
	srv.Router.Use(filters.NewRecoveryMiddleware())

	return &ServiceManager{
		ctx:    smb.ctx,
		Server: srv,
	}
}

func initializeSecureStorage(secureStorage storage.Security) error {
	keyFetcher := secureStorage.Fetcher()
	encryptionKey, err := keyFetcher.GetEncryptionKey()
	if err != nil {
		return err
	}
	if len(encryptionKey) == 0 {
		logrus.Debug("No encryption key is present. Generating new one...")
		newEncryptionKey := make([]byte, 32)
		if _, err := rand.Read(newEncryptionKey); err != nil {
			return fmt.Errorf("Could not generate encryption key: %v", err)
		}
		keySetter := secureStorage.Setter()
		if err := keySetter.SetEncryptionKey(newEncryptionKey); err != nil {
			return err
		}
		logrus.Debug("Successfully generated new encryption key")
	}
	return nil
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
