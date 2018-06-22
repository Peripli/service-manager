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
	"github.com/Peripli/service-manager/types"
)

// Controller catalog controller
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

func (c *Controller) getCatalog(writer http.ResponseWriter, request *http.Request) error {
	logrus.Debugf("Aggregating all broker catalogs")
	brokers, err := c.BrokerStorage.GetAll()
	if err != nil {
		return err
	}

	resultServices := make([]brokerServices, 0, len(brokers)+1)
	queryBrokerIDs := request.URL.Query()["broker_id"]
	if len(queryBrokerIDs) != 0 {
		logrus.Debugf("Filtering based on the provided query parameters: %s", queryBrokerIDs)
		filterBrokersByID(brokers, queryBrokerIDs, &resultServices)
	} else {
		retrieveAllBrokers(brokers, &resultServices)
	}

	return rest.SendJSON(writer, http.StatusOK, aggregatedCatalog{resultServices})
}

func filterBrokersByID(dbBrokers []types.Broker, queryBrokerIDs []string, filteredBrokers *[]brokerServices) {
	for _, queryBrokerID := range queryBrokerIDs{
		for _, dbBroker := range dbBrokers {
			if queryBrokerID == dbBroker.ID {
				*filteredBrokers = append(*filteredBrokers, brokerServices{
					ID: 	dbBroker.ID,
					Name: 	dbBroker.Name,
					Catalog:dbBroker.Catalog,
				})
				break
			}
		}
	}
}

func retrieveAllBrokers(dbBrokers []types.Broker, brokers *[]brokerServices) {
	for _, dbBroker := range dbBrokers {
		*brokers = append(*brokers, brokerServices{
			ID: 	dbBroker.ID,
			Name: 	dbBroker.Name,
			Catalog:dbBroker.Catalog,
		})
	}
}
