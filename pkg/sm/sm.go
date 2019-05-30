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
	"crypto/tls"
	"database/sql"
	"fmt"
	"net/http"
	"sync"

	"github.com/Peripli/service-manager/pkg/security"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage/interceptors"
	osbc "github.com/pmorie/go-open-service-broker-client/v2"

	"github.com/Peripli/service-manager/api"
	"github.com/Peripli/service-manager/api/healthcheck"
	"github.com/Peripli/service-manager/config"
	"github.com/Peripli/service-manager/pkg/log"
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

	Storage             *storage.InterceptableTransactionalRepository
	Notificator         storage.Notificator
	NotificationCleaner *storage.NotificationCleaner
	ctx                 context.Context
	wg                  *sync.WaitGroup
	cfg                 *server.Settings
}

// ServiceManager  struct
type ServiceManager struct {
	ctx                 context.Context
	wg                  *sync.WaitGroup
	Server              *server.Server
	Notificator         storage.Notificator
	NotificationCleaner *storage.NotificationCleaner
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
	if err = cfg.Validate(); err != nil {
		panic(fmt.Sprintf("error validating configuration: %s", err))
	}

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: cfg.API.SkipSSLValidation}

	// setup logging
	ctx = log.Configure(ctx, cfg.Log)

	// goroutines that log must be started after the log has been configured
	util.HandleInterrupts(ctx, cancel)

	// setup smStorage
	log.C(ctx).Info("Setting up Service Manager storage...")

	waitGroup := &sync.WaitGroup{}

	smStorage := &postgres.Storage{
		ConnectFunc: func(driver string, url string) (*sql.DB, error) {
			return sql.Open(driver, url)
		},
	}

	// Initialize the storage with graceful termination
	if err := storage.InitializeWithSafeTermination(ctx, smStorage, cfg.Storage, waitGroup); err != nil {
		panic(fmt.Sprintf("error opening storage: %s", err))
	}

	// Decorate the storage with credentials ecryption/decryption
	encryptingRepository, err := storage.NewEncryptingRepository(ctx, smStorage, &security.AESEncrypter{})
	if err != nil {
		panic(fmt.Sprintf("error setting up encrypting repository: %s", err))
	}

	// Decorate the storage with transactional interceptors
	interceptableRepository := storage.NewInterceptableTransactionalRepository(encryptingRepository)

	// setup core api
	log.C(ctx).Info("Setting up Service Manager core API...")

	pgNotificator, err := postgres.NewNotificator(smStorage, cfg.Storage)
	if err != nil {
		panic(fmt.Sprintf("could not create notificator: %v", err))
	}

	apiOptions := &api.Options{
		Repository:  interceptableRepository,
		APISettings: cfg.API,
		WSSettings:  cfg.WebSocket,
		Notificator: pgNotificator,
	}
	API, err := api.New(ctx, apiOptions)
	if err != nil {
		panic(fmt.Sprintf("error creating core api: %s", err))
	}

	API.HealthIndicators = append(API.HealthIndicators, &storage.HealthIndicator{Pinger: storage.PingFunc(smStorage.Ping)})

	notificationCleaner := &storage.NotificationCleaner{
		Storage:  interceptableRepository,
		Settings: *cfg.Storage,
	}

	smb := &ServiceManagerBuilder{
		API:                 API,
		Storage:             interceptableRepository,
		Notificator:         pgNotificator,
		NotificationCleaner: notificationCleaner,
		ctx:                 ctx,
		wg:                  waitGroup,
		cfg:                 cfg.Server,
	}

	smb.
		WithCreateInterceptorProvider(types.ServiceBrokerType, &interceptors.BrokerCreateCatalogInterceptorProvider{
			OsbClientCreateFunc: newOSBClient(cfg.API.SkipSSLValidation),
		}).Register().
		WithUpdateInterceptorProvider(types.ServiceBrokerType, &interceptors.BrokerUpdateCatalogInterceptorProvider{
			OsbClientCreateFunc: newOSBClient(cfg.API.SkipSSLValidation),
		}).Register().
		WithDeleteInterceptorProvider(types.ServiceBrokerType, &interceptors.BrokerDeleteCatalogInterceptorProvider{
			OsbClientCreateFunc: newOSBClient(cfg.API.SkipSSLValidation),
		}).Register().
		WithCreateInterceptorProvider(types.PlatformType, &interceptors.GenerateCredentialsInterceptorProvider{}).Register().
		WithCreateInterceptorProvider(types.VisibilityType, &interceptors.VisibilityCreateNotificationsInterceptorProvider{}).Register().
		WithUpdateInterceptorProvider(types.VisibilityType, &interceptors.VisibilityUpdateNotificationsInterceptorProvider{}).Register().
		WithDeleteInterceptorProvider(types.VisibilityType, &interceptors.VisibilityDeleteNotificationsInterceptorProvider{}).Register().
		WithCreateInterceptorProvider(types.ServiceBrokerType, &interceptors.BrokerNotificationsCreateInterceptorProvider{}).Before(interceptors.BrokerCreateCatalogInterceptorName).Register().
		WithUpdateInterceptorProvider(types.ServiceBrokerType, &interceptors.BrokerNotificationsUpdateInterceptorProvider{}).Before(interceptors.BrokerUpdateCatalogInterceptorName).Register().
		WithDeleteInterceptorProvider(types.ServiceBrokerType, &interceptors.BrokerNotificationsDeleteInterceptorProvider{}).After(interceptors.BrokerDeleteCatalogInterceptorName).Register()

	return smb
}

// Build builds the Service Manager
func (smb *ServiceManagerBuilder) Build() *ServiceManager {
	// setup server and add relevant global middleware
	smb.installHealth()

	srv := server.New(smb.cfg, smb.API)
	srv.Use(filters.NewRecoveryMiddleware())

	return &ServiceManager{
		ctx:                 smb.ctx,
		wg:                  smb.wg,
		Server:              srv,
		Notificator:         smb.Notificator,
		NotificationCleaner: smb.NotificationCleaner,
	}
}

func (smb *ServiceManagerBuilder) installHealth() {
	if len(smb.HealthIndicators) > 0 {
		smb.RegisterControllers(healthcheck.NewController(smb.HealthIndicators, smb.HealthAggregationPolicy))
	}
}

// Run starts the Service Manager
func (sm *ServiceManager) Run() {
	log.C(sm.ctx).Info("Running Service Manager...")

	if err := sm.Notificator.Start(sm.ctx, sm.wg); err != nil {
		log.C(sm.ctx).WithError(err).Panicf("could not start Service Manager notificator")
	}
	if err := sm.NotificationCleaner.Start(sm.ctx, sm.wg); err != nil {
		log.C(sm.ctx).WithError(err).Panicf("could not start Service Manager notification cleaner")
	}

	sm.Server.Run(sm.ctx, sm.wg)

	sm.wg.Wait()
}

func newOSBClient(skipSsl bool) osbc.CreateFunc {
	return func(configuration *osbc.ClientConfiguration) (osbc.Client, error) {
		configuration.Insecure = skipSsl
		return osbc.NewClient(configuration)
	}
}

func (smb *ServiceManagerBuilder) RegisterNotificationReceiversFilter(filterFunc storage.ReceiversFilterFunc) {
	smb.Notificator.RegisterFilter(filterFunc)
}

func (smb *ServiceManagerBuilder) WithCreateInterceptorProvider(objectType types.ObjectType, provider storage.CreateInterceptorProvider) *interceptorRegistrationBuilder {
	return &interceptorRegistrationBuilder{
		order: storage.InterceptorOrder{
			OnTxPosition: storage.InterceptorPosition{
				PositionType: storage.PositionNone,
			},
			AroundTxPosition: storage.InterceptorPosition{
				PositionType: storage.PositionNone,
			},
		},
		registrationFunc: func(order storage.InterceptorOrder) *ServiceManagerBuilder {
			smb.Storage.AddCreateInterceptorProvider(objectType, storage.OrderedCreateInterceptorProvider{
				CreateInterceptorProvider: provider,
				InterceptorOrder:          order,
			})
			return smb
		},
	}
}

func (smb *ServiceManagerBuilder) WithUpdateInterceptorProvider(objectType types.ObjectType, provider storage.UpdateInterceptorProvider) *interceptorRegistrationBuilder {
	return &interceptorRegistrationBuilder{
		order: storage.InterceptorOrder{
			OnTxPosition: storage.InterceptorPosition{
				PositionType: storage.PositionNone,
			},
			AroundTxPosition: storage.InterceptorPosition{
				PositionType: storage.PositionNone,
			},
		},
		registrationFunc: func(order storage.InterceptorOrder) *ServiceManagerBuilder {
			smb.Storage.AddUpdateInterceptorProvider(objectType, storage.OrderedUpdateInterceptorProvider{
				UpdateInterceptorProvider: provider,
				InterceptorOrder:          order,
			})
			return smb
		},
	}
}

func (smb *ServiceManagerBuilder) WithDeleteInterceptorProvider(objectType types.ObjectType, provider storage.DeleteInterceptorProvider) *interceptorRegistrationBuilder {
	return &interceptorRegistrationBuilder{
		order: storage.InterceptorOrder{
			OnTxPosition: storage.InterceptorPosition{
				PositionType: storage.PositionNone,
			},
			AroundTxPosition: storage.InterceptorPosition{
				PositionType: storage.PositionNone,
			},
		},
		registrationFunc: func(order storage.InterceptorOrder) *ServiceManagerBuilder {
			smb.Storage.AddDeleteInterceptorProvider(objectType, storage.OrderedDeleteInterceptorProvider{
				DeleteInterceptorProvider: provider,
				InterceptorOrder:          order,
			})
			return smb
		},
	}
}
