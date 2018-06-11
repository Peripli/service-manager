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

	"github.com/Peripli/service-manager/api/common"

	"github.com/Peripli/service-manager/rest"
	"github.com/Peripli/service-manager/storage"
	"github.com/Peripli/service-manager/types"
	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
)

const reqBrokerID = "broker_id"

// Controller broker controller
type Controller struct {
	BrokerStorage storage.Broker
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

func (ctrl *Controller) addBroker(response http.ResponseWriter, request *http.Request) error {
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

	err = ctrl.BrokerStorage.Create(broker)
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

func (ctrl *Controller) getBroker(response http.ResponseWriter, request *http.Request) error {
	brokerID := getBrokerID(request)
	logrus.Debugf("Getting broker with id %s", brokerID)

	broker, err := ctrl.BrokerStorage.Get(brokerID)
	err = common.HandleNotFoundError(err, "broker", brokerID)
	if err != nil {
		return err
	}
	broker.Credentials = nil

	return rest.SendJSON(response, http.StatusOK, broker)
}

func (ctrl *Controller) getAllBrokers(response http.ResponseWriter, request *http.Request) error {
	logrus.Debug("Getting all brokers")

	var brokers []types.Broker
	var err error

	names := request.URL.Query()["name"]
	if len(names) == 0 {
		brokers, err = ctrl.BrokerStorage.GetAll()
	} else {
		brokers, err = ctrl.BrokerStorage.GetByName(names)
	}

	if err != nil {
		return err
	}

	type brokerResponse struct {
		Brokers []types.Broker `json:"brokers"`
	}
	return rest.SendJSON(response, http.StatusOK, brokerResponse{brokers})
}

func (ctrl *Controller) deleteBroker(response http.ResponseWriter, request *http.Request) error {
	brokerID := getBrokerID(request)
	logrus.Debugf("Deleting broker with id %s", brokerID)

	err := ctrl.BrokerStorage.Delete(brokerID)
	err = common.HandleNotFoundError(err, "broker", brokerID)
	if err != nil {
		return err
	}
	return rest.SendJSON(response, http.StatusOK, map[string]int{})
}

func (ctrl *Controller) updateBroker(response http.ResponseWriter, request *http.Request) error {
	brokerID := getBrokerID(request)
	logrus.Debugf("Updating broker with id %s", brokerID)

	broker := &types.Broker{}
	if err := rest.ReadJSONBody(request, broker); err != nil {
		logrus.Error("Invalid request body")
		return err
	}

	broker.ID = brokerID
	broker.UpdatedAt = time.Now().UTC()

	brokerStorage := ctrl.BrokerStorage
	if broker.Credentials != nil {
		err := validateBrokerCredentials(broker.Credentials)
		if err != nil {
			return types.NewErrorResponse(err, http.StatusBadRequest, "BadRequest")
		}
	}
	err := brokerStorage.Update(broker)
	err = common.CheckErrors(
		common.HandleNotFoundError(err, "broker", brokerID),
		common.HandleUniqueError(err, "broker"),
	)
	if err != nil {
		return err
	}

	updatedBroker, err := brokerStorage.Get(broker.ID)
	if err != nil {
		logrus.Error("Failed to retrieve updated broker")
		return err
	}

	return rest.SendJSON(response, http.StatusOK, updatedBroker)
}
