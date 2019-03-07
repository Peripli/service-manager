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

package broker

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Peripli/service-manager/pkg/security"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/storage"
	"github.com/gofrs/uuid"
	osbc "github.com/pmorie/go-open-service-broker-client/v2"
)

type CreateBrokerHook struct {
	OSBClientCreateFunc osbc.CreateFunc
	Encrypter           security.Encrypter
	catalog             *osbc.CatalogResponse
}

func (c *CreateBrokerHook) OnAPI(ctx context.Context, apiFunc func() (types.Object, error)) (types.Object, error) {
	result, err := apiFunc()
	if err != nil {
		return nil, err
	}
	broker := result.(*types.Broker)
	c.catalog, err = getBrokerCatalog(ctx, c.OSBClientCreateFunc, broker) // keep catalog to be stored later
	if err != nil {
		return nil, err
	}
	if err = transformBrokerCredentials(ctx, broker, c.Encrypter.Encrypt); err != nil {
		return nil, err
	}
	return broker, nil
}

func (c *CreateBrokerHook) OnTransaction(ctx context.Context, storage storage.Warehouse, transactionFunc func() (types.Object, error)) error {
	object, err := transactionFunc() // create broker first
	if err != nil {
		return err
	}
	broker := object.(*types.Broker)
	for _, service := range c.catalog.Services {
		serviceOffering := &types.ServiceOffering{}
		err := osbcCatalogServiceToServiceOffering(serviceOffering, &service)
		if err != nil {
			return err
		}
		serviceUUID, err := uuid.NewV4()
		if err != nil {
			return fmt.Errorf("could not generate GUID for service: %s", err)
		}
		serviceOffering.ID = serviceUUID.String()
		serviceOffering.CreatedAt = broker.CreatedAt
		serviceOffering.UpdatedAt = broker.UpdatedAt
		serviceOffering.BrokerID = broker.ID

		if err := serviceOffering.Validate(); err != nil {
			return &util.HTTPError{
				ErrorType:   "BadRequest",
				Description: fmt.Sprintf("service offering constructed during catalog insertion for broker %s is invalid: %s", broker.ID, err),
				StatusCode:  http.StatusBadRequest,
			}
		}

		var serviceID string
		if serviceID, err = storage.Create(ctx, serviceOffering); err != nil {
			return util.HandleStorageError(err, "service_offering")
		}
		serviceOffering.ID = serviceID
		for planIndex := range service.Plans {
			servicePlan := &types.ServicePlan{}
			err := osbcCatalogPlanToServicePlan(servicePlan, &catalogPlanWithServiceOfferingID{
				Plan:            &service.Plans[planIndex],
				ServiceOffering: serviceOffering,
			})
			if err != nil {
				return err
			}
			planUUID, err := uuid.NewV4()
			if err != nil {
				return fmt.Errorf("could not generate GUID for service_plan: %s", err)
			}
			servicePlan.ID = planUUID.String()
			servicePlan.CreatedAt = broker.CreatedAt
			servicePlan.UpdatedAt = broker.UpdatedAt

			if err := servicePlan.Validate(); err != nil {
				return &util.HTTPError{
					ErrorType:   "BadRequest",
					Description: fmt.Sprintf("service plan constructed during catalog insertion for broker %s is invalid: %s", broker.ID, err),
					StatusCode:  http.StatusBadRequest,
				}
			}

			if _, err := storage.Create(ctx, servicePlan); err != nil {
				return util.HandleStorageError(err, "service_plan")
			}
		}
	}
	return nil
}

func (*CreateBrokerHook) Supports(objectType types.ObjectType) bool {
	return objectType == types.BrokerType
}
