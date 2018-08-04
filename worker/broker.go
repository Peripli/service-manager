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

package worker

import (
	"encoding/json"
	"fmt"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/security"
	"github.com/Peripli/service-manager/storage"
	"github.com/Peripli/service-manager/work"
	osbc "github.com/pmorie/go-open-service-broker-client/v2"
	"github.com/sirupsen/logrus"
)

type Broker struct {
	OSBClientCreateFunc osbc.CreateFunc
	Encrypter           security.Encrypter
	BrokerStorage       storage.Broker
}

func (c *Broker) Supports(job work.Job) bool {
	return job.EntityType == work.EntityBroker
}

func (c *Broker) Work(job work.Job) error {
	if job.EntityType != work.EntityBroker {
		return fmt.Errorf("Broker work can work only on broker entities")
	}
	switch job.Action {
	case work.ActionCreate:
		broker := &types.Broker{}
		if err := util.BytesToObject(job.Data, broker); err != nil {
			return err
		}
		broker.ID = job.EntityId
		catalog, err := c.getBrokerCatalog(broker)
		if err != nil {
			return err
		}
		broker.Catalog = catalog

		if err := transformBrokerCredentials(broker, c.Encrypter.Encrypt); err != nil {
			return err
		}
		if err := c.BrokerStorage.Create(broker); err != nil {
			return util.HandleStorageError(err, "broker", broker.ID)
		}
		return nil
	case work.ActionUpdate:
	case work.ActionDelete:
		if err := c.BrokerStorage.Delete(job.EntityId); err != nil {
			// errors should be saved so they can be returned when an entity is described
			return util.HandleStorageError(err, "broker", job.EntityId)
		}
	}
	return nil
}

func (c *Broker) getBrokerCatalog(broker *types.Broker) (json.RawMessage, error) {
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

func transformBrokerCredentials(broker *types.Broker, transformationFunc func([]byte) ([]byte, error)) error {
	if broker.Credentials != nil {
		transformedPassword, err := transformationFunc([]byte(broker.Credentials.Basic.Password))
		if err != nil {
			return err
		}
		broker.Credentials.Basic.Password = string(transformedPassword)
	}
	return nil
}
