/*
 *    Copyright 2018 The Service Manager Authors
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

package broker

import (
	"net/http"
	"time"

	osbc "github.com/pmorie/go-open-service-broker-client/v2"

	"encoding/json"

	"strings"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
	"github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
)

const (
	reqBrokerID  = "broker_id"
	catalogParam = "catalog"
)

// Controller broker controller
type Controller struct {
	BrokerStorage       storage.Broker
	OSBClientCreateFunc osbc.CreateFunc
}

var _ web.Controller = &Controller{}

func (c *Controller) createBroker(request *web.Request) (*web.Response, error) {
	logrus.Debug("Creating new broker")

	broker := &types.Broker{}
	if err := util.UnmarshalAndValidate(request.Body, broker); err != nil {
		return nil, err
	}

	uuid, err := uuid.NewV4()
	if err != nil {
		logrus.Error("Could not generate GUID")
		return nil, err
	}

	broker.ID = uuid.String()

	currentTime := time.Now().UTC()
	broker.CreatedAt = currentTime
	broker.UpdatedAt = currentTime

	catalog, err := c.getBrokerCatalog(broker)
	if err != nil {
		return nil, err
	}
	broker.Catalog = catalog

	err = c.BrokerStorage.Create(broker)
	err = storage.HandleUniqueError(err, "broker")
	if err != nil {
		return nil, err
	}

	broker.Credentials = nil
	broker.Catalog = nil
	return util.NewJSONResponse(http.StatusCreated, broker)
}

func (c *Controller) getBroker(request *web.Request) (*web.Response, error) {
	brokerID := request.PathParams[reqBrokerID]
	logrus.Debugf("Getting broker with id %s", brokerID)

	broker, err := c.BrokerStorage.Get(brokerID)
	err = storage.HandleNotFoundError(err, "broker", brokerID)
	if err != nil {
		return nil, err
	}

	broker.Credentials = nil
	broker.Catalog = nil
	return util.NewJSONResponse(http.StatusOK, broker)
}

func (c *Controller) getAllBrokers(request *web.Request) (*web.Response, error) {
	logrus.Debug("Getting all brokers")
	brokers, err := c.BrokerStorage.GetAll()
	if err != nil {
		return nil, err
	}
	withCatalog := request.FormValue(catalogParam)
	if strings.ToLower(withCatalog) != "true" {
		for i := 0; i < len(brokers); i++ {
			brokers[i].Catalog = nil
		}
	}

	return util.NewJSONResponse(http.StatusOK, &types.Brokers{
		Brokers: brokers,
	})
}

func (c *Controller) deleteBroker(request *web.Request) (*web.Response, error) {
	brokerID := request.PathParams[reqBrokerID]
	logrus.Debugf("Deleting broker with id %s", brokerID)

	err := c.BrokerStorage.Delete(brokerID)
	err = storage.HandleNotFoundError(err, "broker", brokerID)
	if err != nil {
		return nil, err
	}
	return util.NewJSONResponse(http.StatusOK, map[string]int{})
}

func (c *Controller) patchBroker(request *web.Request) (*web.Response, error) {
	brokerID := request.PathParams[reqBrokerID]
	logrus.Debugf("Updating updateBroker with id %s", brokerID)

	broker, err := c.BrokerStorage.Get(brokerID)
	err = storage.HandleNotFoundError(err, "broker", brokerID)
	if err != nil {
		return nil, err
	}

	if err := util.UnmarshalAndValidate(request.Body, broker); err != nil {
		return nil, err
	}

	catalog, err := c.getBrokerCatalog(broker)
	if err != nil {
		return nil, err
	}

	broker.Catalog = catalog
	broker.UpdatedAt = time.Now().UTC()

	err = c.BrokerStorage.Update(broker)
	err = storage.CheckErrors(
		storage.HandleNotFoundError(err, "broker", brokerID),
		storage.HandleUniqueError(err, "broker"),
	)
	if err != nil {
		return nil, err
	}

	broker.Credentials = nil
	broker.Catalog = nil

	return util.NewJSONResponse(http.StatusOK, broker)
}

func (c *Controller) getBrokerCatalog(broker *types.Broker) (json.RawMessage, error) {
	osbClient, err := osbClient(c.OSBClientCreateFunc, broker)
	if err != nil {
		return nil, err
	}
	catalog, err := osbClient.GetCatalog()
	if err != nil {
		return nil, err
	}

	bytes, err := json.Marshal(catalog)
	if err != nil {
		return nil, err
	}

	return json.RawMessage(bytes), nil
}

func osbClient(createFunc osbc.CreateFunc, broker *types.Broker) (osbc.Client, error) {
	config := clientConfigForBroker(broker)
	logrus.Debug("Building OSB client for serviceBroker with name: ", config.Name, " accessible at: ", config.URL)
	return createFunc(config)
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
