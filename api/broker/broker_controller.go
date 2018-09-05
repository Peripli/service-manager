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
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/security"
	osbc "github.com/pmorie/go-open-service-broker-client/v2"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/Peripli/service-manager/storage"
	"github.com/gofrs/uuid"
)

const (
	reqBrokerID  = "broker_id"
	catalogParam = "catalog"
)

// Controller broker controller
type Controller struct {
	BrokerStorage       storage.Broker
	OSBClientCreateFunc osbc.CreateFunc
	Encrypter           security.Encrypter
}

var _ web.Controller = &Controller{}

func (c *Controller) createBroker(r *web.Request) (*web.Response, error) {
	ctx := r.Context()
	log.C(ctx).Debug("Creating new broker")

	broker := &types.Broker{}
	if err := util.BytesToObject(r.Body, broker); err != nil {
		return nil, err
	}

	UUID, err := uuid.NewV4()
	if err != nil {
		return nil, fmt.Errorf("could not generate GUID for broker: %s", err)
	}

	broker.ID = UUID.String()

	currentTime := time.Now().UTC()
	broker.CreatedAt = currentTime
	broker.UpdatedAt = currentTime

	catalog, err := c.getBrokerCatalog(ctx, broker)
	if err != nil {
		return nil, err
	}
	broker.Catalog = catalog

	if err := transformBrokerCredentials(ctx, broker, c.Encrypter.Encrypt); err != nil {
		return nil, err
	}
	if err := c.BrokerStorage.Create(ctx, broker); err != nil {
		return nil, util.HandleStorageError(err, "broker", broker.ID)
	}

	broker.Credentials = nil
	broker.Catalog = nil
	return util.NewJSONResponse(http.StatusCreated, broker)
}

func (c *Controller) getBroker(r *web.Request) (*web.Response, error) {
	brokerID := r.PathParams[reqBrokerID]
	ctx := r.Context()
	log.C(ctx).Debugf("Getting broker with id %s", brokerID)

	broker, err := c.BrokerStorage.Get(ctx, brokerID)
	if err != nil {
		return nil, util.HandleStorageError(err, "broker", brokerID)
	}

	broker.Credentials = nil
	broker.Catalog = nil
	return util.NewJSONResponse(http.StatusOK, broker)
}

func (c *Controller) getAllBrokers(r *web.Request) (*web.Response, error) {
	ctx := r.Context()
	log.C(ctx).Debug("Getting all brokers")
	brokers, err := c.BrokerStorage.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	removeCatalog := strings.ToLower(r.FormValue(catalogParam)) != "true"
	for _, broker := range brokers {
		broker.Credentials = nil
		if removeCatalog {
			broker.Catalog = nil
		}
	}

	return util.NewJSONResponse(http.StatusOK, &types.Brokers{
		Brokers: brokers,
	})
}

func (c *Controller) deleteBroker(r *web.Request) (*web.Response, error) {
	brokerID := r.PathParams[reqBrokerID]
	ctx := r.Context()
	log.C(ctx).Debugf("Deleting broker with id %s", brokerID)

	if err := c.BrokerStorage.Delete(ctx, brokerID); err != nil {
		return nil, util.HandleStorageError(err, "broker", brokerID)
	}
	return util.NewJSONResponse(http.StatusOK, map[string]int{})
}

func (c *Controller) patchBroker(r *web.Request) (*web.Response, error) {
	brokerID := r.PathParams[reqBrokerID]
	ctx := r.Context()
	log.C(ctx).Debugf("Updating updateBroker with id %s", brokerID)

	broker, err := c.BrokerStorage.Get(ctx, brokerID)
	if err != nil {
		return nil, util.HandleStorageError(err, "broker", brokerID)
	}

	createdAt := broker.CreatedAt

	if err := util.BytesToObject(r.Body, broker); err != nil {
		return nil, err
	}

	if err := transformBrokerCredentials(ctx, broker, c.Encrypter.Encrypt); err != nil {
		return nil, err
	}

	catalog, err := c.getBrokerCatalog(ctx, broker)
	if err != nil {
		return nil, err
	}

	broker.ID = brokerID
	broker.Catalog = catalog
	broker.CreatedAt = createdAt
	broker.UpdatedAt = time.Now().UTC()

	if err := c.BrokerStorage.Update(ctx, broker); err != nil {
		return nil, util.HandleStorageError(err, "broker", brokerID)
	}

	broker.Credentials = nil
	broker.Catalog = nil

	return util.NewJSONResponse(http.StatusOK, broker)
}

func (c *Controller) getBrokerCatalog(ctx context.Context, broker *types.Broker) (json.RawMessage, error) {
	osbClient, err := osbClient(ctx, c.OSBClientCreateFunc, broker)
	if err != nil {
		return nil, err
	}
	catalog, err := osbClient.GetCatalog()
	if err != nil {
		return nil, fmt.Errorf("Error fetching catalog from broker %s: %v", broker.Name, err)
	}

	bytes, err := json.Marshal(catalog)
	if err != nil {
		return nil, err
	}

	return json.RawMessage(bytes), nil
}

func osbClient(ctx context.Context, createFunc osbc.CreateFunc, broker *types.Broker) (osbc.Client, error) {
	config := clientConfigForBroker(broker)
	log.C(ctx).Debug("Building OSB client for serviceBroker with name: ", config.Name, " accessible at: ", config.URL)
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

func transformBrokerCredentials(ctx context.Context, broker *types.Broker, transformationFunc func(context.Context, []byte) ([]byte, error)) error {
	if broker.Credentials != nil {
		transformedPassword, err := transformationFunc(ctx, []byte(broker.Credentials.Basic.Password))
		if err != nil {
			return err
		}
		broker.Credentials.Basic.Password = string(transformedPassword)
	}
	return nil
}
