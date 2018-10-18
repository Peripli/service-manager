/*
 * Copyright 2018 The Service Manager Authors
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

package reconcile

import (
	"context"
	"sync"

	"github.com/Peripli/service-manager/pkg/log"

	"strings"

	"encoding/json"

	"fmt"

	"github.com/Peripli/service-manager/pkg/sbproxy/platform"
	"github.com/Peripli/service-manager/pkg/sbproxy/sm"
	"github.com/pkg/errors"
	osbc "github.com/pmorie/go-open-service-broker-client/v2"
	"github.com/sirupsen/logrus"
)

// ProxyBrokerPrefix prefixes names of brokers registered at the platform
const ProxyBrokerPrefix = "sm-proxy-"

// ReconcilationTask type represents a registration task that takes care of propagating broker creations
// and deletions to the platform. It reconciles the state of the proxy brokers in the platform to match
// the desired state provided by the Service Manager.
// TODO if the reg credentials are changed (the ones under cf.reg) we need to update the already registered brokers
type ReconcilationTask struct {
	group          *sync.WaitGroup
	platformClient platform.Client
	smClient       sm.Client
	proxyPath      string
	ctx            context.Context
}

// Settings type represents the sbproxy settings
type Settings struct {
	URL      string
	Username string
	Password string
}

// DefaultSettings creates default proxy settings
func DefaultSettings() *Settings {
	return &Settings{
		URL:      "",
		Username: "",
		Password: "",
	}
}

// NewTask builds a new ReconcilationTask
func NewTask(ctx context.Context, group *sync.WaitGroup, platformClient platform.Client, smClient sm.Client, proxyPath string) *ReconcilationTask {
	return &ReconcilationTask{
		group:          group,
		platformClient: platformClient,
		smClient:       smClient,
		proxyPath:      proxyPath,
		ctx:            ctx,
	}
}

// Validate validates that the configuration contains all mandatory properties
func (c *Settings) Validate() error {
	if c.URL == "" {
		return fmt.Errorf("validate settings: missing host")
	}
	return nil
}

// Run executes the registration task that is responsible for reconciling the state of the proxy brokers at the
// platform with the brokers provided by the Service Manager
func (r ReconcilationTask) Run() {
	logger := log.C(r.ctx)
	logger.Debug("STARTING scheduled reconciliation task...")

	r.group.Add(1)
	defer r.group.Done()
	r.run()

	logger.Debug("FINISHED scheduled reconciliation task...")
}

func (r ReconcilationTask) run() {
	// get all the registered proxy brokers from the platform
	brokersFromPlatform, err := r.getBrokersFromPlatform()
	if err != nil {
		log.C(r.ctx).WithError(err).Error("An error occurred while obtaining already registered brokers")
		return
	}

	// get all the brokers that are in SM and for which a proxy broker should be present in the platform
	brokersFromSM, err := r.getBrokersFromSM()
	if err != nil {
		log.C(r.ctx).WithError(err).Error("An error occurred while obtaining brokers from Service Manager")
		return
	}

	// control logic - make sure current state matches desired state
	r.reconcileBrokers(brokersFromPlatform, brokersFromSM)
}

// reconcileBrokers attempts to reconcile the current brokers state in the platform (existingBrokers)
// to match the desired broker state coming from the Service Manager (payloadBrokers).
func (r ReconcilationTask) reconcileBrokers(existingBrokers []platform.ServiceBroker, payloadBrokers []platform.ServiceBroker) {
	existingMap := convertBrokersRegListToMap(existingBrokers)
	for _, payloadBroker := range payloadBrokers {
		var err error
		existingBroker := existingMap[payloadBroker.GUID]
		delete(existingMap, payloadBroker.GUID)
		if existingBroker == nil {
			err = r.createBrokerRegistration(&payloadBroker)
		} else {
			err = r.fetchBrokerCatalog(existingBroker)
		}
		if err == nil {
			r.enableServiceAccessVisibilities(&payloadBroker)
		}
	}

	for _, existingBroker := range existingMap {
		r.deleteBrokerRegistration(existingBroker)
	}
}

func (r ReconcilationTask) getBrokersFromPlatform() ([]platform.ServiceBroker, error) {
	logger := log.C(r.ctx)
	logger.Debug("ReconcilationTask task getting proxy brokers from platform...")
	registeredBrokers, err := r.platformClient.GetBrokers(r.ctx)
	if err != nil {
		return nil, errors.Wrap(err, "error getting brokers from platform")
	}

	brokersFromPlatform := make([]platform.ServiceBroker, 0, len(registeredBrokers))
	for _, broker := range registeredBrokers {
		if !r.isProxyBroker(broker) {
			continue
		}

		logger.WithFields(logBroker(&broker)).Debug("ReconcilationTask task FOUND registered proxy broker... ")
		brokersFromPlatform = append(brokersFromPlatform, broker)
	}
	logger.Debugf("ReconcilationTask task SUCCESSFULLY retrieved %d proxy brokers from platform", len(brokersFromPlatform))
	return brokersFromPlatform, nil
}

func (r ReconcilationTask) getBrokersFromSM() ([]platform.ServiceBroker, error) {
	logger := log.C(r.ctx)
	logger.Debug("ReconcilationTask task getting brokers from Service Manager")

	proxyBrokers, err := r.smClient.GetBrokers(r.ctx)
	if err != nil {
		return nil, errors.Wrap(err, "error getting brokers from SM")
	}

	brokersFromSM := make([]platform.ServiceBroker, 0, len(proxyBrokers))
	for _, broker := range proxyBrokers {
		brokerReg := platform.ServiceBroker{
			GUID:      broker.ID,
			BrokerURL: broker.BrokerURL,
			Catalog:   broker.Catalog,
			Metadata:  broker.Metadata,
		}
		brokersFromSM = append(brokersFromSM, brokerReg)
	}
	logger.Debugf("ReconcilationTask task SUCCESSFULLY retrieved %d brokers from Service Manager", len(brokersFromSM))

	return brokersFromSM, nil
}

func (r ReconcilationTask) fetchBrokerCatalog(broker *platform.ServiceBroker) (err error) {
	if f, isFetcher := r.platformClient.(platform.CatalogFetcher); isFetcher {
		logger := log.C(r.ctx)
		logger.WithFields(logBroker(broker)).Debugf("ReconcilationTask task refetching catalog for broker")
		if err = f.Fetch(r.ctx, broker); err != nil {
			logger.WithFields(logBroker(broker)).WithError(err).Error("Error during fetching catalog...")
		} else {
			logger.WithFields(logBroker(broker)).Debug("ReconcilationTask task SUCCESSFULLY refetched catalog for broker")
		}
	}
	return
}

func (r ReconcilationTask) createBrokerRegistration(broker *platform.ServiceBroker) (err error) {
	logger := log.C(r.ctx)
	logger.WithFields(logBroker(broker)).Info("ReconcilationTask task attempting to create proxy for broker in platform...")

	createRequest := &platform.CreateServiceBrokerRequest{
		Name:      ProxyBrokerPrefix + broker.GUID,
		BrokerURL: r.proxyPath + "/" + broker.GUID,
	}

	var b *platform.ServiceBroker
	if b, err = r.platformClient.CreateBroker(r.ctx, createRequest); err != nil {
		logger.WithFields(logBroker(broker)).WithError(err).Error("Error during broker creation")
	} else {
		logger.WithFields(logBroker(b)).Infof("ReconcilationTask task SUCCESSFULLY created proxy for broker at platform under name [%s] accessible at [%s]", createRequest.Name, createRequest.BrokerURL)
	}
	return
}

func (r ReconcilationTask) deleteBrokerRegistration(broker *platform.ServiceBroker) {
	logger := log.C(r.ctx)
	logger.WithFields(logBroker(broker)).Info("ReconcilationTask task attempting to delete broker from platform...")

	deleteRequest := &platform.DeleteServiceBrokerRequest{
		GUID: broker.GUID,
		Name: broker.Name,
	}

	if err := r.platformClient.DeleteBroker(r.ctx, deleteRequest); err != nil {
		logger.WithFields(logBroker(broker)).WithError(err).Error("Error during broker deletion")
	} else {
		logger.WithFields(logBroker(broker)).Infof("ReconcilationTask task SUCCESSFULLY deleted proxy broker from platform with name [%s]", deleteRequest.Name)
	}
}

func (r ReconcilationTask) enableServiceAccessVisibilities(broker *platform.ServiceBroker) {
	if f, isEnabler := r.platformClient.(platform.ServiceAccess); isEnabler {
		emptyContext := emptyContext()
		logger := log.C(r.ctx)
		logger.WithFields(logBroker(broker)).Info("ReconcilationTask task attempting to enable service access for broker...")

		catalog := broker.Catalog
		if catalog == nil {
			logger.WithFields(logBroker(broker)).Error("Error enabling service access due to missing catalog details")
			return
		}

		for _, service := range catalog.Services {
			logger.WithFields(logService(service)).Debug("ReconcilationTask task attempting to enable service access for service...")
			if err := f.EnableAccessForService(r.ctx, emptyContext, service.ID); err != nil {
				logger.WithFields(logService(service)).WithError(err).Errorf("Error enabling service access for service with ID=%s...", service.ID)
			}
			logger.WithFields(logService(service)).Debug("ReconcilationTask task finished enabling service access for service...")
		}
		logger.WithFields(logBroker(broker)).Infof("ReconcilationTask task finished enabling service access for broker")
	}
}

func (r ReconcilationTask) isProxyBroker(broker platform.ServiceBroker) bool {
	return strings.HasPrefix(broker.BrokerURL, r.proxyPath)
}

func logBroker(broker *platform.ServiceBroker) logrus.Fields {
	return logrus.Fields{
		"broker_guid": broker.GUID,
		"broker_name": broker.Name,
		"broker_url":  broker.BrokerURL,
	}
}

func logService(service osbc.Service) logrus.Fields {
	return logrus.Fields{
		"service_guid": service.ID,
		"service_name": service.Name,
	}
}

func emptyContext() json.RawMessage {
	return json.RawMessage(`{}`)
}

func convertBrokersRegListToMap(brokerList []platform.ServiceBroker) map[string]*platform.ServiceBroker {
	brokerRegMap := make(map[string]*platform.ServiceBroker, len(brokerList))

	for i, broker := range brokerList {
		smID := broker.BrokerURL[strings.LastIndex(broker.BrokerURL, "/")+1:]
		brokerRegMap[smID] = &brokerList[i]
	}
	return brokerRegMap
}
