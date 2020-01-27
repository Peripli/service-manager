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
	"time"

	"github.com/Peripli/service-manager/operations"

	"github.com/Peripli/service-manager/pkg/env"

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
	osbc "github.com/kubernetes-sigs/go-open-service-broker-client/v2"
)

// ServiceManagerBuilder type is an extension point that allows adding additional filters, plugins and
// controllers before running ServiceManager.
type ServiceManagerBuilder struct {
	*web.API

	Storage             *storage.InterceptableTransactionalRepository
	Notificator         storage.Notificator
	NotificationCleaner *storage.NotificationCleaner
	OperationMaintainer *operations.Maintainer
	OSBClientProvider   osbc.CreateFunc
	ctx                 context.Context
	wg                  *sync.WaitGroup
	cfg                 *config.Settings
	securityBuilder     *SecurityBuilder
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
func New(ctx context.Context, cancel context.CancelFunc, e env.Environment, cfg *config.Settings) (*ServiceManagerBuilder, error) {
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

	pgNotificator, err := postgres.NewNotificator(smStorage, cfg.Storage)
	if err != nil {
		return nil, fmt.Errorf("could not create notificator: %v", err)
	}

	apiOptions := &api.Options{
		Repository:        interceptableRepository,
		APISettings:       cfg.API,
		OperationSettings: cfg.Operations,
		WSSettings:        cfg.WebSocket,
		Notificator:       pgNotificator,
		WaitGroup:         waitGroup,
	}
	API, err := api.New(ctx, e, apiOptions)
	if err != nil {
		return nil, fmt.Errorf("error creating core api: %s", err)
	}

	securityBuilder, securityFilters := NewSecurityBuilder()
	API.RegisterFiltersAfter(filters.LoggingFilterName, securityFilters...)

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

	operationMaintainer := operations.NewMaintainer(ctx, interceptableRepository, cfg.Operations)
	osbClientProvider := osb.NewBrokerClientProvider(cfg.HTTPClient.SkipSSLValidation, int(cfg.HTTPClient.ResponseHeaderTimeout.Seconds()))

	smb := &ServiceManagerBuilder{
		API:                 API,
		Storage:             interceptableRepository,
		Notificator:         pgNotificator,
		NotificationCleaner: notificationCleaner,
		OperationMaintainer: operationMaintainer,
		ctx:                 ctx,
		wg:                  waitGroup,
		cfg:                 cfg,
		securityBuilder:     securityBuilder,
		OSBClientProvider:   osbClientProvider,
	}

	smb.RegisterPlugins(osb.NewCatalogFilterByVisibilityPlugin(interceptableRepository))
	smb.RegisterPluginsBefore(osb.CheckInstanceOwnerhipPluginName, osb.NewStoreServiceInstancesPlugin(interceptableRepository))
	smb.RegisterPluginsBefore(osb.StoreServiceInstancePluginName, osb.NewCheckVisibilityPlugin(interceptableRepository))
	smb.RegisterPlugins(osb.NewCheckPlatformIDPlugin(interceptableRepository))

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
		WithCreateAroundTxInterceptorProvider(types.PlatformType, &interceptors.GeneratePlatformCredentialsInterceptorProvider{}).Register().
		WithCreateOnTxInterceptorProvider(types.VisibilityType, &interceptors.VisibilityCreateNotificationsInterceptorProvider{}).Register().
		WithUpdateOnTxInterceptorProvider(types.VisibilityType, &interceptors.VisibilityUpdateNotificationsInterceptorProvider{}).Register().
		WithDeleteOnTxInterceptorProvider(types.VisibilityType, &interceptors.VisibilityDeleteNotificationsInterceptorProvider{}).Register().
		WithCreateOnTxInterceptorProvider(types.ServiceBrokerType, &interceptors.BrokerNotificationsCreateInterceptorProvider{}).Before(interceptors.BrokerCreateCatalogInterceptorName).Register().
		WithUpdateOnTxInterceptorProvider(types.ServiceBrokerType, &interceptors.BrokerNotificationsUpdateInterceptorProvider{}).Before(interceptors.BrokerUpdateCatalogInterceptorName).Register().
		WithDeleteOnTxInterceptorProvider(types.ServiceBrokerType, &interceptors.BrokerNotificationsDeleteInterceptorProvider{}).After(interceptors.BrokerDeleteCatalogInterceptorName).Register().
		WithCreateAroundTxInterceptorProvider(types.ServiceInstanceType, &interceptors.ServiceInstanceCreateInterceptorProvider{
			OSBClientCreateFunc: osbClientProvider,
			Repository:          interceptableRepository,
			TenantKey:           cfg.Multitenancy.LabelKey,
			PollingInterval:     cfg.Operations.PollingInterval,
		}).Register().
		WithUpdateAroundTxInterceptorProvider(types.ServiceInstanceType, &interceptors.ServiceInstanceUpdateInterceptorProvider{
			OSBClientCreateFunc: osbClientProvider,
			Repository:          interceptableRepository,
			TenantKey:           cfg.Multitenancy.LabelKey,
			PollingInterval:     cfg.Operations.PollingInterval,
		}).Register().
		WithDeleteAroundTxInterceptorProvider(types.ServiceInstanceType, &interceptors.ServiceInstanceDeleteInterceptorProvider{
			OSBClientCreateFunc: osbClientProvider,
			Repository:          interceptableRepository,
			TenantKey:           cfg.Multitenancy.LabelKey,
			PollingInterval:     cfg.Operations.PollingInterval,
		}).Register().
		WithCreateAroundTxInterceptorProvider(types.ServiceBindingType, &interceptors.ServiceBindingCreateInterceptorProvider{
			OSBClientCreateFunc: osbClientProvider,
			Repository:          interceptableRepository,
			TenantKey:           cfg.Multitenancy.LabelKey,
			PollingInterval:     cfg.Operations.PollingInterval,
		}).Register().
		WithDeleteAroundTxInterceptorProvider(types.ServiceBindingType, &interceptors.ServiceBindingDeleteInterceptorProvider{
			OSBClientCreateFunc: osbClientProvider,
			Repository:          interceptableRepository,
			TenantKey:           cfg.Multitenancy.LabelKey,
			PollingInterval:     cfg.Operations.PollingInterval,
		}).Register()

	return smb, nil
}

// Build builds the Service Manager
func (smb *ServiceManagerBuilder) Build() *ServiceManager {
	if smb.securityBuilder != nil {
		smb.securityBuilder.Build()
	}

	if err := smb.installHealth(); err != nil {
		log.C(smb.ctx).Panic(err)
	}

	// setup server and add relevant global middleware
	srv := server.New(smb.cfg.Server, smb.API)
	srv.Use(filters.NewRecoveryMiddleware())

	// start the operation maintainer
	smb.OperationMaintainer.Run()

	if err := smb.registerSMPlatform(); err != nil {
		log.C(smb.ctx).Panic(err)
	}

	return &ServiceManager{
		ctx:                 smb.ctx,
		wg:                  smb.wg,
		Server:              srv,
		Notificator:         smb.Notificator,
		NotificationCleaner: smb.NotificationCleaner,
	}
}

func (smb *ServiceManagerBuilder) registerSMPlatform() error {
	if _, err := smb.Storage.Create(smb.ctx, &types.Platform{
		Base: types.Base{
			ID:        types.SMPlatform,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Labels:    make(map[string][]string),
		},
		Type: types.SMPlatform,
		Name: types.SMPlatform,
	}); err != nil {
		if err == util.ErrAlreadyExistsInStorage {
			log.C(smb.ctx).Infof("platform %s already exists in SMDB...", types.SMPlatform)
			return nil
		}
		return fmt.Errorf("could not register %s platform during bootstrap: %s", types.SMPlatform, err)
	}

	return nil
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

func (smb *ServiceManagerBuilder) RegisterExtension(registry Extendable) *ServiceManagerBuilder {
	if err := registry.Extend(smb.ctx, smb); err != nil {
		log.D().Panicf("Could not register extension: %s", err)
	}
	return smb
}

func (smb *ServiceManagerBuilder) WithCreateAroundTxInterceptorProvider(objectType types.ObjectType, provider storage.CreateAroundTxInterceptorProvider) *interceptorRegistrationBuilder {
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
			smb.Storage.AddCreateAroundTxInterceptorProvider(objectType, provider, order)
			return smb
		},
	}
}

func (smb *ServiceManagerBuilder) WithCreateOnTxInterceptorProvider(objectType types.ObjectType, provider storage.CreateOnTxInterceptorProvider) *interceptorRegistrationBuilder {
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
			smb.Storage.AddCreateOnTxInterceptorProvider(objectType, provider, order)
			return smb
		},
	}
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
			smb.Storage.AddCreateInterceptorProvider(objectType, provider, order)
			return smb
		},
	}
}

func (smb *ServiceManagerBuilder) WithUpdateAroundTxInterceptorProvider(objectType types.ObjectType, provider storage.UpdateAroundTxInterceptorProvider) *interceptorRegistrationBuilder {
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
			smb.Storage.AddUpdateAroundTxInterceptorProvider(objectType, provider, order)
			return smb
		},
	}
}

func (smb *ServiceManagerBuilder) WithUpdateOnTxInterceptorProvider(objectType types.ObjectType, provider storage.UpdateOnTxInterceptorProvider) *interceptorRegistrationBuilder {
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
			smb.Storage.AddUpdateOnTxInterceptorProvider(objectType, provider, order)
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
			smb.Storage.AddUpdateInterceptorProvider(objectType, provider, order)
			return smb
		},
	}
}

func (smb *ServiceManagerBuilder) WithDeleteAroundTxInterceptorProvider(objectType types.ObjectType, provider storage.DeleteAroundTxInterceptorProvider) *interceptorRegistrationBuilder {
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
			smb.Storage.AddDeleteAroundTxInterceptorProvider(objectType, provider, order)
			return smb
		},
	}
}

func (smb *ServiceManagerBuilder) WithDeleteOnTxInterceptorProvider(objectType types.ObjectType, provider storage.DeleteOnTxInterceptorProvider) *interceptorRegistrationBuilder {
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
			smb.Storage.AddDeleteOnTxInterceptorProvider(objectType, provider, order)
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
			smb.Storage.AddDeleteInterceptorProvider(objectType, provider, order)
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
	smb.RegisterFilters(
		filters.NewServiceInstanceVisibilityFilter(smb.Storage, labelKey),
		filters.NewServiceBindingVisibilityFilter(smb.Storage, labelKey),
	)

	smb.RegisterPlugins(osb.NewCheckInstanceOwnershipPlugin(smb.Storage, labelKey))

	smb.WithCreateOnTxInterceptorProvider(types.ServiceInstanceType, &interceptors.ServiceInstanceCreateInsterceptorProvider{
		TenantIdentifier: labelKey,
	}).AroundTxAfter(interceptors.ServiceInstanceCreateInterceptorProviderName).Register()
	smb.WithCreateOnTxInterceptorProvider(types.OperationType, &interceptors.OperationsCreateInsterceptorProvider{
		TenantIdentifier: labelKey,
	}).Register()

	return smb
}

// Security provides mechanism to apply authentication and authorization with a builder pattern
func (smb *ServiceManagerBuilder) Security() *SecurityBuilder {
	return smb.securityBuilder.Reset()
}
