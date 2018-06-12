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

	osbc "github.com/pmorie/go-open-service-broker-client/v2"

	"github.com/Peripli/service-manager/api/common"

	"encoding/json"

	"bytes"

	"strings"

	"github.com/Peripli/service-manager/api/osb"
	"github.com/Peripli/service-manager/rest"
	"github.com/Peripli/service-manager/storage"
	"github.com/Peripli/service-manager/types"
	"github.com/gorilla/mux"
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

func validateBrokerCredentials(brokerCredentials *types.Credentials) error {
	if brokerCredentials == nil || brokerCredentials.Basic == nil {
		return errors.New("Missing broker credentials")
	}
	if brokerCredentials.Basic.Username == "" {
		return errors.New("Missing broker username")
	}
	if brokerCredentials.Basic.Password == "" {
		return errors.New("Missing broker password")
	}
	return nil
}

func validateBroker(broker *types.Broker) error {
	if broker.Name == "" {
		return errors.New("Missing broker name")
	}
	if broker.BrokerURL == "" {
		return errors.New("Missing broker url")
	}
	return validateBrokerCredentials(broker.Credentials)
}

func (c *Controller) createBroker(response http.ResponseWriter, request *http.Request) error {
	logrus.Debug("Creating new broker")

	broker := &types.Broker{}
	if err := rest.ReadJSONBody(request, broker); err != nil {
		return err
	}

	if err := validateBroker(broker); err != nil {
		return types.NewErrorResponse(err, http.StatusBadRequest, "BadRequest")
	}

	uuid, err := uuid.NewV4()
	if err != nil {
		logrus.Error("Could not generate GUID")
		return err
	}

	broker.ID = uuid.String()

	currentTime := time.Now().UTC()
	broker.CreatedAt = currentTime
	broker.UpdatedAt = currentTime

	catalog, err := c.getBrokerCatalog(broker)
	if err != nil {
		return err
	}
	broker.Catalog = catalog

	err = c.BrokerStorage.Create(broker)
	err = common.HandleUniqueError(err, "broker")
	if err != nil {
		return err
	}

	broker.Credentials = nil
	return rest.SendJSON(response, http.StatusCreated, broker)
}

func getBrokerID(request *http.Request) string {
	return mux.Vars(request)[reqBrokerID]
}

func (c *Controller) getBrokerCatalog(broker *types.Broker) (json.RawMessage, error) {
	osbClient, err := osb.OSBClient(c.OSBClientCreateFunc, broker)
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

func (c *Controller) getBroker(response http.ResponseWriter, request *http.Request) error {
	brokerID := getBrokerID(request)
	logrus.Debugf("Getting broker with id %s", brokerID)

	broker, err := c.BrokerStorage.Get(brokerID)
	err = common.HandleNotFoundError(err, "broker", brokerID)
	if err != nil {
		return err
	}
	broker.Credentials = nil
	broker.Catalog = nil
	return rest.SendJSON(response, http.StatusOK, broker)
}

func (c *Controller) getAllBrokers(response http.ResponseWriter, request *http.Request) error {
	logrus.Debug("Getting all brokers")
	brokers, err := c.BrokerStorage.GetAll()
	if err != nil {
		return err
	}
	withCatalog := request.FormValue(catalogParam)
	if strings.ToLower(withCatalog) != "true" {
		for i := 0; i < len(brokers); i++ {
			brokers[i].Catalog = nil
		}
	}

	type brokerResponse struct {
		Brokers []types.Broker `json:"brokers"`
	}
	return rest.SendJSON(response, http.StatusOK, brokerResponse{brokers})
}

func (c *Controller) deleteBroker(response http.ResponseWriter, request *http.Request) error {
	brokerID := getBrokerID(request)
	logrus.Debugf("Deleting broker with id %s", brokerID)

	err := c.BrokerStorage.Delete(brokerID)
	err = common.HandleNotFoundError(err, "broker", brokerID)
	if err != nil {
		return err
	}
	return rest.SendJSON(response, http.StatusOK, map[string]int{})
}

func (c *Controller) updateBroker(response http.ResponseWriter, request *http.Request) error {
	brokerID := getBrokerID(request)
	logrus.Debugf("Updating broker with id %s", brokerID)

	broker := &types.Broker{}
	if err := rest.ReadJSONBody(request, broker); err != nil {
		logrus.Error("Invalid request body")
		return err
	}

	broker.ID = brokerID
	broker.UpdatedAt = time.Now().UTC()

	if broker.Credentials != nil {
		err := validateBrokerCredentials(broker.Credentials)
		if err != nil {
			return types.NewErrorResponse(err, http.StatusBadRequest, "BadRequest")
		}
	}

	brokerFromDb, err := c.BrokerStorage.Get(broker.ID)
	if err != nil {
		logrus.Error("Failed to retrieve updated broker")
		return err
	}

	catalog, err := c.getBrokerCatalog(brokerFromDb)
	if err != nil {
		return types.NewErrorResponse(err, http.StatusBadRequest, "BadRequest")
	}

	isCatalogModified := !bytes.Equal(brokerFromDb.Catalog, catalog)
	if isCatalogModified {
		broker.Catalog = catalog
	}

	err = c.BrokerStorage.Update(broker)
	err = common.CheckErrors(
		common.HandleNotFoundError(err, "broker", brokerID),
		common.HandleUniqueError(err, "broker"),
	)
	if err != nil {
		return err
	}

	return rest.SendJSON(response, http.StatusOK, map[string]string{})
}
