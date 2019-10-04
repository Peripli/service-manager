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
	"database/sql"
	"fmt"
	"net/http"
	"sync"

	"github.com/Peripli/service-manager/api/plugins"

	"github.com/Peripli/service-manager/pkg/health"

	"github.com/Peripli/service-manager/pkg/httpclient"

	"github.com/Peripli/service-manager/api/osb"

	"github.com/Peripli/service-manager/storage/catalog"

	"github.com/Peripli/service-manager/pkg/security"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage/interceptors"

	"github.com/Peripli/service-manager/api"
	"github.com/Peripli/service-manager/api/healthcheck"
	"github.com/Peripli/service-manager/config"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/server"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/storage"
	"github.com/Peripli/service-manager/storage/postgres"

	"github.com/Peripli/service-manager/api/filters"
	"github.com/Peripli/service-manager/pkg/web"
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
	cfg                 *config.Settings
}

// ServiceManager  struct
type ServiceManager struct {
	ctx                 context.Context
	wg                  *sync.WaitGroup
	Server              *server.Server
	Notificator         storage.Notificator
	NotificationCleaner *storage.NotificationCleaner
}

// New returns service-manager Server with default setup
func New(ctx context.Context, cancel context.CancelFunc, cfg *config.Settings) (*ServiceManagerBuilder, error) {
	var err error
	if err = cfg.Validate(); err != nil {
		return nil, fmt.Errorf("error validating configuration: %s", err)
	}

	httpclient.Configure(cfg.HTTPClient)

	// Setup logging
	ctx, err = log.Configure(ctx, cfg.Log)
	if err != nil {
		return nil, fmt.Errorf("error configuring logging,: %s", err)
	}

	util.HandleInterrupts(ctx, cancel)

	// Setup storage
	log.C(ctx).Info("Setting up Service Manager storage...")
	smStorage := &postgres.Storage{
		ConnectFunc: func(driver string, url string) (*sql.DB, error) {
			return sql.Open(driver, url)
		},
	}

	// Decorate the storage with credentials encryption/decryption
	encryptingDecorator := storage.EncryptingDecorator(ctx, &security.AESEncrypter{}, smStorage)

	// Initialize the storage with graceful termination
	var transactionalRepository storage.TransactionalRepository
	waitGroup := &sync.WaitGroup{}
	if transactionalRepository, err = storage.InitializeWithSafeTermination(ctx, smStorage, cfg.Storage, waitGroup, encryptingDecorator); err != nil {
		return nil, fmt.Errorf("error opening storage: %s", err)
	}

	// Wrap the repository with logic that runs interceptors
	interceptableRepository := storage.NewInterceptableTransactionalRepository(transactionalRepository)

	// Setup core API
	log.C(ctx).Info("Setting up Service Manager core API...")

	pgNotificator, err := postgres.NewNotificator(smStorage, interceptableRepository, cfg.Storage)
	if err != nil {
		return nil, fmt.Errorf("could not create notificator: %v", err)
	}

	apiOptions := &api.Options{
		Repository:  interceptableRepository,
		APISettings: cfg.API,
		WSSettings:  cfg.WebSocket,
		Notificator: pgNotificator,
	}
	API, err := api.New(ctx, apiOptions)
	if err != nil {
		return nil, fmt.Errorf("error creating core api: %s", err)
	}

	storageHealthIndicator, err := storage.NewSQLHealthIndicator(storage.PingFunc(smStorage.PingContext))
	if err != nil {
		return nil, fmt.Errorf("error creating storage health indicator: %s", err)
	}

	API.SetIndicator(storageHealthIndicator)
	API.SetIndicator(healthcheck.NewPlatformIndicator(ctx, interceptableRepository, nil))

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
		cfg:                 cfg,
	}

	smb.RegisterPlugins(plugins.NewCatalogFilterByVisibilityPlugin(interceptableRepository))

	// Register default interceptors that represent the core SM business logic
	smb.
		WithCreateInterceptorProvider(types.ServiceBrokerType, &interceptors.BrokerCreateCatalogInterceptorProvider{
			CatalogFetcher: osb.CatalogFetcher(http.DefaultClient.Do, cfg.API.OSBVersion),
		}).Register().
		WithUpdateInterceptorProvider(types.ServiceBrokerType, &interceptors.BrokerUpdateCatalogInterceptorProvider{
			CatalogFetcher: osb.CatalogFetcher(http.DefaultClient.Do, cfg.API.OSBVersion),
			CatalogLoader:  catalog.Load,
		}).Register().
		WithDeleteInterceptorProvider(types.ServiceBrokerType, &interceptors.BrokerDeleteCatalogInterceptorProvider{
			CatalogLoader: catalog.Load,
		}).Register().
		WithCreateInterceptorProvider(types.PlatformType, &interceptors.GenerateCredentialsInterceptorProvider{}).Register().
		WithCreateInterceptorProvider(types.VisibilityType, &interceptors.VisibilityCreateNotificationsInterceptorProvider{}).Register().
		WithUpdateInterceptorProvider(types.VisibilityType, &interceptors.VisibilityUpdateNotificationsInterceptorProvider{}).Register().
		WithDeleteInterceptorProvider(types.VisibilityType, &interceptors.VisibilityDeleteNotificationsInterceptorProvider{}).Register().
		WithCreateInterceptorProvider(types.ServiceBrokerType, &interceptors.BrokerNotificationsCreateInterceptorProvider{}).Before(interceptors.BrokerCreateCatalogInterceptorName).Register().
		WithUpdateInterceptorProvider(types.ServiceBrokerType, &interceptors.BrokerNotificationsUpdateInterceptorProvider{}).Before(interceptors.BrokerUpdateCatalogInterceptorName).Register().
		WithDeleteInterceptorProvider(types.ServiceBrokerType, &interceptors.BrokerNotificationsDeleteInterceptorProvider{}).After(interceptors.BrokerDeleteCatalogInterceptorName).Register()

	return smb, nil
}

// Build builds the Service Manager
func (smb *ServiceManagerBuilder) Build() *ServiceManager {
	if err := smb.installHealth(); err != nil {
		log.C(smb.ctx).Panic(err)
	}

	// setup server and add relevant global middleware
	srv := server.New(smb.cfg.Server, smb.API)
	srv.Use(filters.NewRecoveryMiddleware())

	return &ServiceManager{
		ctx:                 smb.ctx,
		wg:                  smb.wg,
		Server:              srv,
		Notificator:         smb.Notificator,
		NotificationCleaner: smb.NotificationCleaner,
	}
}

func (smb *ServiceManagerBuilder) installHealth() error {
	healthz, thresholds, err := health.Configure(smb.ctx, smb.HealthIndicators, smb.cfg.Health)
	if err != nil {
		return err
	}

	smb.RegisterControllers(healthcheck.NewController(healthz, thresholds))

	if err := healthz.Start(); err != nil {
		return err
	}

	util.StartInWaitGroupWithContext(smb.ctx, func(c context.Context) {
		<-c.Done()
		log.C(c).Debug("Context cancelled. Stopping health checks...")
		if err := healthz.Stop(); err != nil {
			log.C(c).Error(err)
		}
	}, smb.wg)

	return nil
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

// EnableMultitenancy enables multitenancy resources for Service Manager by labeling them with appropriate tenant value
func (smb *ServiceManagerBuilder) EnableMultitenancy(labelKey string, extractTenantFunc func(*web.Request) (string, error)) *ServiceManagerBuilder {
	if len(labelKey) == 0 {
		log.D().Panic("labelKey should be provided")
	}
	if extractTenantFunc == nil {
		log.D().Panic("extractTenantFunc should be provided")
	}

	multitenancyFilters := filters.NewMultitenancyFilters(labelKey, extractTenantFunc)
	smb.RegisterFiltersAfter(filters.ProtectedLabelsFilterName, multitenancyFilters...)
	return smb
}
