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

package catalog

import (
	"encoding/json"
	"net/http"

	"github.com/Peripli/service-manager/rest"
	"github.com/Peripli/service-manager/storage"
	"github.com/sirupsen/logrus"
)

type Controller struct {
	BrokerStorage storage.Broker
}

type brokerServices struct {
	ID      string          `json:"id"`
	Name    string          `json:"name"`
	Catalog json.RawMessage `json:"catalog"`
}

type aggregatedCatalog struct {
	Brokers []brokerServices `json:"brokers"`
}

func (ctrl *Controller) getCatalog(writer http.ResponseWriter, request *http.Request) error {
	logrus.Debugf("Aggregating all broker catalogs")
	brokers, err := ctrl.BrokerStorage.GetAll()
	if err != nil {
		return err
	}

	brokerIds := request.URL.Query()["broker_id"]

	logrus.Debugf("Filtering based on the provided query parameters: %s", brokerIds)

	services := make([]brokerServices, 0, len(brokers)+1)
	for i, _ := range brokers {
		if shouldBrokerBeAdded(brokers[i].ID, brokerIds) {
			services = append(services, brokerServices{
				ID:      brokers[i].ID,
				Name:    brokers[i].Name,
				Catalog: brokers[i].Catalog,
			})
		}
	}

	return rest.SendJSON(writer, http.StatusOK, aggregatedCatalog{services})
}

func shouldBrokerBeAdded(brokerId string, brokerIds []string) bool {
	if len(brokerIds) != 0 {
		for _, bID := range brokerIds {
			if brokerId == bID {
				return true
			}
		}
		return false
	}
	return true
}
