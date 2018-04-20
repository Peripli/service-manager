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

package osb

import (
	"net/http"

	"fmt"

	"github.com/Peripli/service-manager/storage"
	"github.com/Peripli/service-manager/types"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	osbc "github.com/pmorie/go-open-service-broker-client/v2"
	"github.com/pmorie/osb-broker-lib/pkg/broker"
)

// BrokerIDPathParam is used as a key for broker id path parameter

// BusinessLogic provides an implementation of the pmorie/osb-broker-lib/pkg/broker/Interface interface.
type BusinessLogic struct {
	createFunc    osbc.CreateFunc
	brokerStorage storage.Broker
}

var _ broker.Interface = &BusinessLogic{}

// NewBusinessLogic creates an OSB business logic containing logic to proxy OSB calls
func NewBusinessLogic(createFunc osbc.CreateFunc, brokerStorage storage.Broker) *BusinessLogic {
	return &BusinessLogic{
		createFunc:    createFunc,
		brokerStorage: brokerStorage,
	}
}

// GetCatalog implements pmorie/osb-broker-lib/pkg/broker/Interface.GetCatalog by
// proxying the method to the underlying implementation to an underlying service broker
// the id of which should be provided as path parameter
func (b *BusinessLogic) GetCatalog(c *broker.RequestContext) (*broker.CatalogResponse, error) {
	client, err := b.osbClient(c.Request)
	if err != nil {
		return nil, err
	}
	response, err := client.GetCatalog()
	if err != nil {
		return nil, err
	}

	return &broker.CatalogResponse{
		CatalogResponse: *response,
	}, nil
}

// Provision implements pmorie/osb-broker-lib/pkg/broker/Interface.Provision by
// proxying the method to the underlying implementation to an underlying service broker
// the id of which should be provided as path parameter
func (b *BusinessLogic) Provision(request *osbc.ProvisionRequest, c *broker.RequestContext) (*broker.ProvisionResponse, error) {
	client, err := b.osbClient(c.Request)
	if err != nil {
		return nil, err
	}

	response, err := client.ProvisionInstance(request)
	if err != nil {
		return nil, err
	}

	return &broker.ProvisionResponse{
		ProvisionResponse: *response,
	}, nil
}

// Deprovision implements pmorie/osb-broker-lib/pkg/broker/Interface.Deprovision by
// proxying the method to the underlying implementation to an underlying service broker
// the id of which should be provided as path parameter
func (b *BusinessLogic) Deprovision(request *osbc.DeprovisionRequest, c *broker.RequestContext) (*broker.DeprovisionResponse, error) {
	client, err := b.osbClient(c.Request)
	if err != nil {
		return nil, err
	}
	response, err := client.DeprovisionInstance(request)
	if err != nil {
		return nil, err
	}

	return &broker.DeprovisionResponse{
		DeprovisionResponse: *response,
	}, nil
}

// LastOperation implements pmorie/osb-broker-lib/pkg/broker/Interface.LastOperation by
// proxying the method to the underlying implementation to an underlying service broker
// the id of which should be provided as path parameter
func (b *BusinessLogic) LastOperation(request *osbc.LastOperationRequest, c *broker.RequestContext) (*broker.LastOperationResponse, error) {
	client, err := b.osbClient(c.Request)
	if err != nil {
		return nil, err
	}
	response, err := client.PollLastOperation(request)
	if err != nil {
		return nil, err
	}

	return &broker.LastOperationResponse{
		LastOperationResponse: *response,
	}, nil
}

// Bind implements pmorie/osb-broker-lib/pkg/broker/Interface.Bind by
// proxying the method to the underlying implementation to an underlying service broker
// the id of which should be provided as path parameter
func (b *BusinessLogic) Bind(request *osbc.BindRequest, c *broker.RequestContext) (*broker.BindResponse, error) {
	client, err := b.osbClient(c.Request)
	if err != nil {
		return nil, err
	}

	response, err := client.Bind(request)
	if err != nil {
		return nil, err
	}

	return &broker.BindResponse{
		BindResponse: *response,
	}, nil

}

// Unbind implements pmorie/osb-broker-lib/pkg/broker/Interface.Unbind by
// proxying the method to the underlying implementation to an underlying service broker
// the id of which should be provided as path parameter
func (b *BusinessLogic) Unbind(request *osbc.UnbindRequest, c *broker.RequestContext) (*broker.UnbindResponse, error) {
	client, err := b.osbClient(c.Request)
	if err != nil {
		return nil, err
	}

	response, err := client.Unbind(request)
	if err != nil {
		return nil, err
	}

	return &broker.UnbindResponse{
		UnbindResponse: *response,
	}, nil
}

// Update implements pmorie/osb-broker-lib/pkg/broker/Interface.Update by
// proxying the method to the underlying implementation to an underlying service broker
// the id of which should be provided as path parameter
func (b *BusinessLogic) Update(request *osbc.UpdateInstanceRequest, c *broker.RequestContext) (*broker.UpdateInstanceResponse, error) {
	client, err := b.osbClient(c.Request)
	if err != nil {
		return nil, err
	}

	response, err := client.UpdateInstance(request)
	if err != nil {
		return nil, err
	}

	return &broker.UpdateInstanceResponse{
		UpdateInstanceResponse: *response,
	}, nil
}

// ValidateBrokerAPIVersion implements pmorie/osb-broker-lib/pkg/broker/Interface.ValidateBrokerAPIVersion by
// checking that the version provided as parameter matches the latest supported version
func (b *BusinessLogic) ValidateBrokerAPIVersion(version string) error {
	expectedVersion := osbc.LatestAPIVersion().HeaderValue()
	if version != expectedVersion {
		return fmt.Errorf("error validating OSB Version: expected %s but was %s", expectedVersion, version)
	}
	return nil
}

func clientConfigForBroker(broker *types.Broker) *osbc.ClientConfiguration {
	config := osbc.DefaultClientConfiguration()
	config.Name = broker.Name
	config.URL = broker.URL
	config.AuthConfig = &osbc.AuthConfig{
		BasicAuthConfig: &osbc.BasicAuthConfig{
			Username: broker.User,
			Password: broker.Password,
		},
	}
	return config
}

func (b *BusinessLogic) osbClient(request *http.Request) (osbc.Client, error) {
	vars := mux.Vars(request)
	brokerID, ok := vars[BrokerIDPathParam]
	logrus.Debugf("Obtained path parameter [brokerID = %s] from mux vars", brokerID)
	if !ok {
		return nil, fmt.Errorf("error creating OSB client: brokerID path parameter not found")
	}
	serviceBroker, err := b.brokerStorage.Find(request.Context(), brokerID)
	if err != nil {
		return nil, fmt.Errorf("error obtaining serviceBroker with id %s from storage: %s", brokerID, err)
	}
	config := clientConfigForBroker(serviceBroker)
	logrus.Debug("Building OSB client for serviceBroker with name: ", config.Name, " accessible at: ", config.URL)
	return b.createFunc(config)
}
