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
	"errors"
	"net/http"
	"time"

	"github.com/Peripli/service-manager/pkg/filter"
	osbc "github.com/pmorie/go-open-service-broker-client/v2"

	"github.com/Peripli/service-manager/api/common"

	"encoding/json"

	"bytes"

	"strings"

	"github.com/Peripli/service-manager/api/osb"
	"github.com/Peripli/service-manager/rest"
	"github.com/Peripli/service-manager/storage"
	"github.com/Peripli/service-manager/types"
	"github.com/satori/go.uuid"
	uuid "github.com/satori/go.uuid"
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

func validateBrokerCredentials(brokerCredentials *types.Credentials) error {
	if brokerCredentials == nil || brokerCredentials.Basic == nil {
		return errors.New("missing broker credentials")
	}
	if brokerCredentials.Basic.Username == "" {
		return errors.New("missing broker username")
	}
	if brokerCredentials.Basic.Password == "" {
		return errors.New("missing broker password")
	}
	return nil
}

func validateBroker(broker *types.Broker) error {
	if broker.Name == "" {
		return errors.New("missing broker name")
	}
	if broker.BrokerURL == "" {
		return errors.New("missing broker url")
	}
	return validateBrokerCredentials(broker.Credentials)
}

func (c *Controller) createBroker(request *filter.Request) (*filter.Response, error) {
	logrus.Debug("Creating new broker")

	broker := &types.Broker{}
	if err := rest.ReadJSONBody(request, broker); err != nil {
		return nil, err
	}

	if err := validateBroker(broker); err != nil {
		return nil, types.NewErrorResponse(err, http.StatusBadRequest, "BadRequest")
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
	err = common.HandleUniqueError(err, "broker")
	if err != nil {
		return nil, err
	}

	broker.Credentials = nil
	broker.Catalog = nil
	return rest.NewJSONResponse(http.StatusCreated, broker)
}

func (c *Controller) getBroker(request *filter.Request) (*filter.Response, error) {
	brokerID := request.PathParams[reqBrokerID]
	logrus.Debugf("Getting broker with id %s", brokerID)

	broker, err := c.BrokerStorage.Get(brokerID)
	err = common.HandleNotFoundError(err, "broker", brokerID)
	if err != nil {
		return nil, err
	}

	broker.Credentials = nil
	broker.Catalog = nil
	return rest.NewJSONResponse(http.StatusOK, broker)
}

func (c *Controller) getAllBrokers(request *filter.Request) (*filter.Response, error) {
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

	return rest.NewJSONResponse(http.StatusOK, struct {
		Brokers []types.Broker `json:"brokers"`
	}{
		Brokers: brokers,
	})
}

func (c *Controller) deleteBroker(request *filter.Request) (*filter.Response, error) {
	brokerID := request.PathParams[reqBrokerID]
	logrus.Debugf("Deleting broker with id %s", brokerID)

	err := c.BrokerStorage.Delete(brokerID)
	err = common.HandleNotFoundError(err, "broker", brokerID)
	if err != nil {
		return nil, err
	}
	return rest.NewJSONResponse(http.StatusOK, map[string]int{})
}

func (c *Controller) patchBroker(request *filter.Request) (*filter.Response, error) {
	brokerID := request.PathParams[reqBrokerID]
	logrus.Debugf("Updating updateBroker with id %s", brokerID)

	updateBroker := &types.Broker{}
	if err := rest.ReadJSONBody(request, updateBroker); err != nil {
		return nil, err
	}

	updateBroker.UpdatedAt = time.Now().UTC()
	updateBroker.ID = brokerID

	if updateBroker.Credentials != nil {
		err := validateBrokerCredentials(updateBroker.Credentials)
		if err != nil {
			return nil, types.NewErrorResponse(err, http.StatusBadRequest, "BadRequest")
		}
	}

	broker, err := c.BrokerStorage.Get(brokerID)
	err = common.HandleNotFoundError(err, "broker", brokerID)
	if err != nil {
		return nil, err
	}

	updateData, err := json.Marshal(updateBroker)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(updateData, broker); err != nil {
		return nil, err
	}

	catalog, err := c.getBrokerCatalog(broker)
	if err != nil {
		return nil, err
	}

	isCatalogModified := !bytes.Equal(broker.Catalog, catalog)
	if isCatalogModified {
		updateBroker.Catalog = catalog
	}

	err = c.BrokerStorage.Update(updateBroker)
	err = common.CheckErrors(
		common.HandleNotFoundError(err, "broker", brokerID),
		common.HandleUniqueError(err, "broker"),
	)
	if err != nil {
		return nil, err
	}
	broker.Credentials = nil
	broker.Catalog = nil

	return rest.NewJSONResponse(http.StatusOK, broker)
}

func (c *Controller) getBrokerCatalog(broker *types.Broker) (json.RawMessage, error) {
	osbClient, err := osb.Client(c.OSBClientCreateFunc, broker)
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
