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
	if err = toHTTPError(err); err != nil {
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
	if err = toHTTPError(err); err != nil {
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
	if err = toHTTPError(err); err != nil {
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
	if err = toHTTPError(err); err != nil {
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
	if err = toHTTPError(err); err != nil {
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
	if err = toHTTPError(err); err != nil {
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
	if err = toHTTPError(err); err != nil {
		return nil, err
	}

	return &broker.UpdateInstanceResponse{
		UpdateInstanceResponse: *response,
	}, nil
}

// ValidateBrokerAPIVersion implements pmorie/osb-broker-lib/pkg/broker/Interface.ValidateBrokerAPIVersion by
// checking that the version provided as parameter matches the latest supported version
func (b *BusinessLogic) ValidateBrokerAPIVersion(version string) error {
	return nil
}

func (b *BusinessLogic) osbClient(request *http.Request) (osbc.Client, error) {
	vars := mux.Vars(request)
	brokerID, ok := vars[BrokerIDPathParam]
	if !ok {
		logrus.Debugf("error creating OSB client: brokerID path parameter not found")
		return nil, createHTTPErrorResponse("Invalid broker id path parameter", "BadRequest", http.StatusBadRequest)
	}
	logrus.Debugf("Obtained path parameter [brokerID = %s] from mux vars", brokerID)

	serviceBroker, err := b.brokerStorage.Get(brokerID)
	if err == storage.ErrNotFound {
		logrus.Debugf("service broker with id %s not found", brokerID)
		return nil, createHTTPErrorResponse(fmt.Sprintf("Could not find broker with id: %s", brokerID), "NotFound", http.StatusNotFound)
	} else if err != nil {
		logrus.Errorf("error obtaining serviceBroker with id %s from storage: %s", brokerID, err)
		return nil, fmt.Errorf("Internal Server Error")
	}
	return OSBClient(b.createFunc, serviceBroker)
}

func clientConfigForBroker(broker *types.Broker) *osbc.ClientConfiguration {
	config := osbc.DefaultClientConfiguration()
	config.Name = broker.Name
	config.URL = broker.BrokerURL
	config.AuthConfig = &osbc.AuthConfig{
		BasicAuthConfig: &osbc.BasicAuthConfig{
			Username: broker.Credentials.Basic.Username,
			Password: broker.Credentials.Basic.Password,
		},
	}
	return config
}

func OSBClient(createFunc osbc.CreateFunc, broker *types.Broker) (osbc.Client, error) {
	config := clientConfigForBroker(broker)
	logrus.Debug("Building OSB client for serviceBroker with name: ", config.Name, " accessible at: ", config.URL)
	return createFunc(config)
}

func toHTTPError(err error) error {
	if err == nil {
		return nil
	}
	if _, isHTTPError := osbc.IsHTTPError(err); isHTTPError {
		return err
	}
	return createHTTPErrorResponse(err.Error(), "BadRequest", http.StatusBadRequest)
}

func createHTTPErrorResponse(description, errorMessage string, statusCode int) *osbc.HTTPStatusCodeError {
	return &osbc.HTTPStatusCodeError{
		Description:  &description,
		ErrorMessage: &errorMessage,
		StatusCode:   statusCode,
	}
}
