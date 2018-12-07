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
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/Peripli/service-manager/api"
	"github.com/Peripli/service-manager/api/healthcheck"
	"github.com/Peripli/service-manager/config"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/security"
	"github.com/Peripli/service-manager/pkg/server"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/storage"
	"github.com/Peripli/service-manager/storage/postgres"

	"github.com/Peripli/service-manager/api/filters"
	"github.com/Peripli/service-manager/cf"
	"github.com/Peripli/service-manager/pkg/env"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/spf13/pflag"
)

// ServiceManagerBuilder type is an extension point that allows adding additional filters, plugins and
// controllers before running ServiceManager.
type ServiceManagerBuilder struct {
	*web.API

	ctx context.Context
	cfg *server.Settings
}

// ServiceManager  struct
type ServiceManager struct {
	ctx    context.Context
	Server *server.Server
}

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
	// setup config from env
	cfg, err := config.New(env)
	if err != nil {
		panic(fmt.Errorf("error loading configuration: %s", err))
	}
	if err := cfg.Validate(); err != nil {
		panic(fmt.Sprintf("error validating configuration: %s", err))
	}

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: cfg.API.SkipSSLValidation}

	// setup logging
	ctx = log.Configure(ctx, cfg.Log)

	// goroutines that log must be started after the log has been configured
	util.HandleInterrupts(ctx, cancel)

	// setup smStorage
	smStorage, err := storage.Use(ctx, postgres.Storage, cfg.Storage)
	if err != nil {
		panic(fmt.Sprintf("error using smStorage: %s", err))
	}

	securityStorage := smStorage.Security()
	if err := initializeSecureStorage(ctx, securityStorage); err != nil {
		panic(fmt.Sprintf("error initialzing secure storage: %v", err))
	}

	encrypter := &security.TwoLayerEncrypter{
		Fetcher: securityStorage.Fetcher(),
	}

	// setup core api
	API, err := api.New(ctx, smStorage, cfg.API, encrypter)
	if err != nil {
		panic(fmt.Sprintf("error creating core api: %s", err))
	}

	API.AddHealthIndicator(&storage.HealthIndicator{Pinger: storage.PingFunc(smStorage.Ping)})

	return &ServiceManagerBuilder{
		ctx: ctx,
		cfg: cfg.Server,
		API: API,
	}
}

// Build builds the Service Manager
func (smb *ServiceManagerBuilder) Build() *ServiceManager {
	// setup server and add relevant global middleware
	smb.installHealth()

	srv := server.New(smb.cfg, smb.API)
	srv.Use(filters.NewRecoveryMiddleware())

	return &ServiceManager{
		ctx:    smb.ctx,
		Server: srv,
	}
}

func (smb *ServiceManagerBuilder) installHealth() {
	if len(smb.HealthIndicators()) > 0 {
		smb.RegisterControllers(healthcheck.NewController(smb.HealthIndicators(), smb.HealthAggregationPolicy()))
	}
}

// Run starts the Service Manager
func (sm *ServiceManager) Run() {
	sm.Server.Run(sm.ctx)
}

func initializeSecureStorage(ctx context.Context, secureStorage storage.Security) error {
	ctx, cancelFunc := context.WithTimeout(ctx, 2*time.Second)
	defer cancelFunc()
	if err := secureStorage.Lock(ctx); err != nil {
		return err
	}
	keyFetcher := secureStorage.Fetcher()
	encryptionKey, err := keyFetcher.GetEncryptionKey(ctx)
	if err != nil {
		return err
	}
	if len(encryptionKey) == 0 {
		logger := log.C(ctx)
		logger.Debug("No encryption key is present. Generating new one...")
		newEncryptionKey := make([]byte, 32)
		if _, err := rand.Read(newEncryptionKey); err != nil {
			return fmt.Errorf("could not generate encryption key: %v", err)
		}
		keySetter := secureStorage.Setter()
		if err := keySetter.SetEncryptionKey(ctx, newEncryptionKey); err != nil {
			return err
		}
		logger.Debug("Successfully generated new encryption key")
	}
	return secureStorage.Unlock(ctx)
}
