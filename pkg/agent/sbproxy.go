/*
 * Copyright 2019 The Service Manager Authors
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

package agent

import (
	"sync"

	filters2 "github.com/Peripli/service-manager/pkg/security/filters"

	"github.com/Peripli/service-manager/pkg/agent/authn"

	"github.com/Peripli/service-manager/api/configuration"
	secfilters "github.com/Peripli/service-manager/api/filters"
	"github.com/Peripli/service-manager/pkg/agent/notifications/handlers"
	"github.com/Peripli/service-manager/pkg/types"

	"fmt"

	"github.com/Peripli/service-manager/api/healthcheck"
	"github.com/Peripli/service-manager/pkg/agent/logging"
	"github.com/Peripli/service-manager/pkg/health"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/util"

	"context"
	"time"

	"github.com/Peripli/service-manager/api/filters"
	"github.com/Peripli/service-manager/pkg/agent/notifications"
	"github.com/Peripli/service-manager/pkg/agent/platform"
	"github.com/Peripli/service-manager/pkg/agent/reconcile"
	"github.com/Peripli/service-manager/pkg/agent/sm"
	"github.com/Peripli/service-manager/pkg/env"
	"github.com/Peripli/service-manager/pkg/server"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/spf13/pflag"
)

const (
	// BrokerPathParam for the broker id
	BrokerPathParam = "brokerID"

	// APIPrefix for the Proxy OSB API
	APIPrefix = "/v1/osb"

	// Path for the Proxy OSB API
	Path = APIPrefix + "/{" + BrokerPathParam + "}"
)

// SMProxyBuilder type is an extension point that allows adding additional filters, plugins and
// controllers before running SMProxy.
type SMProxyBuilder struct {
	*web.API

	ctx                   context.Context
	cfg                   *Settings
	group                 *sync.WaitGroup
	reconciler            *reconcile.Reconciler
	notificationsProducer *notifications.Producer
}

// SMProxy  struct
type SMProxy struct {
	*server.Server

	ctx                   context.Context
	group                 *sync.WaitGroup
	reconciler            *reconcile.Reconciler
	notificationsProducer *notifications.Producer
}

// DefaultEnv creates a default environment that can be used to boot up a Service Broker proxy
func DefaultEnv(ctx context.Context, additionalPFlags ...func(set *pflag.FlagSet)) (env.Environment, error) {
	set := pflag.NewFlagSet("Configuration Flags", pflag.ExitOnError)

	AddPFlags(set)
	for _, addFlags := range additionalPFlags {
		addFlags(set)
	}

	return env.New(ctx, set)
}

// New creates service broker proxy that is configured from the provided environment and platform client.
func New(ctx context.Context, cancel context.CancelFunc, environment env.Environment, settings *Settings, platformClient platform.Client) (*SMProxyBuilder, error) {
	if err := settings.Validate(); err != nil {
		return nil, fmt.Errorf("error validating settings: %s", err)
	}

	var err error
	ctx, err = log.Configure(ctx, settings.Log)
	if err != nil {
		return nil, fmt.Errorf("error configuring logging: %s", err)
	}

	log.AddHook(&logging.ErrorLocationHook{})

	util.HandleInterrupts(ctx, cancel)
	filters := []web.Filter{
		&filters.Logging{},
	}
	authnSettings := settings.Authentication
	if len(authnSettings.User) != 0 && len(authnSettings.Password) != 0 {
		filters = append(filters, authn.NewBasicAuthnFilter(authnSettings.User, authnSettings.Password))
	}
	if len(authnSettings.TokenIssuerURL) != 0 {
		bearerAuthnFilter, err := secfilters.NewOIDCAuthnFilter(ctx, settings.Authentication.TokenIssuerURL, settings.Authentication.ClientID)
		if err != nil {
			return nil, err
		}
		filters = append(filters, bearerAuthnFilter)
	}
	filters = append(filters, filters2.NewRequiredAuthnFilter())

	api := &web.API{
		Controllers: []web.Controller{
			&configuration.Controller{
				Environment: environment,
			},
		},
		Filters:  filters,
		Registry: health.NewDefaultRegistry(),
	}

	smClient, err := sm.NewClient(settings.Sm)
	if err != nil {
		return nil, fmt.Errorf("error create service manager client: %s", err)
	}

	notificationsProducer, err := notifications.NewProducer(settings.Producer, settings.Sm)
	if err != nil {
		return nil, fmt.Errorf("error creating notifications producer: %s", err)
	}

	smPath := settings.Reconcile.URL + APIPrefix
	proxyPathPattern := settings.Reconcile.LegacyURL + APIPrefix + "/%s"

	resyncer := reconcile.NewResyncer(settings.Reconcile, platformClient, smClient, smPath, proxyPathPattern)
	consumer := &notifications.Consumer{
		Handlers: map[types.ObjectType]notifications.ResourceNotificationHandler{
			types.ServiceBrokerType: &handlers.BrokerResourceNotificationsHandler{
				BrokerClient:    platformClient.Broker(),
				CatalogFetcher:  platformClient.CatalogFetcher(),
				ProxyPrefix:     settings.Reconcile.BrokerPrefix,
				SMPath:          smPath,
				BrokerBlacklist: settings.Reconcile.BrokerBlacklist,
				TakeoverEnabled: settings.Reconcile.TakeoverEnabled,
			},
			types.VisibilityType: &handlers.VisibilityResourceNotificationsHandler{
				VisibilityClient: platformClient.Visibility(),
				ProxyPrefix:      settings.Reconcile.BrokerPrefix,
				BrokerBlacklist:  settings.Reconcile.BrokerBlacklist,
			},
		},
	}
	reconciler := &reconcile.Reconciler{
		Resyncer: resyncer,
		Consumer: consumer,
	}
	var group sync.WaitGroup
	return &SMProxyBuilder{
		API:                   api,
		ctx:                   ctx,
		cfg:                   settings,
		group:                 &group,
		reconciler:            reconciler,
		notificationsProducer: notificationsProducer,
	}, nil
}

// Build builds the Service Manager
func (smb *SMProxyBuilder) Build() *SMProxy {
	if err := smb.installHealth(); err != nil {
		log.C(smb.ctx).Panic(err)
	}

	srv := server.New(smb.cfg.Server, smb.API)
	srv.Use(filters.NewRecoveryMiddleware())

	return &SMProxy{
		Server:                srv,
		ctx:                   smb.ctx,
		group:                 smb.group,
		reconciler:            smb.reconciler,
		notificationsProducer: smb.notificationsProducer,
	}
}

func (smb *SMProxyBuilder) installHealth() error {
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
	}, smb.group)

	return nil
}

// Run starts the proxy
func (p *SMProxy) Run() {
	defer waitWithTimeout(p.ctx, p.group, p.Server.Config.ShutdownTimeout)

	messages := p.notificationsProducer.Start(p.ctx, p.group)
	p.reconciler.Reconcile(p.ctx, messages, p.group)

	log.C(p.ctx).Info("Running SBProxy...")
	p.Server.Run(p.ctx, p.group)

	p.group.Wait()
}

// waitWithTimeout waits for a WaitGroup to finish for a certain duration and times out afterwards
// WaitGroup parameter should be pointer or else the copy won't get notified about .Done() calls
func waitWithTimeout(ctx context.Context, group *sync.WaitGroup, timeout time.Duration) {
	c := make(chan struct{})
	go func() {
		defer close(c)
		group.Wait()
	}()
	select {
	case <-c:
		log.C(ctx).Debugf("Timeout WaitGroup %+v finished successfully", group)
	case <-time.After(timeout):
		log.C(ctx).Fatal("Shutdown took more than ", timeout)
		close(c)
	}
}
