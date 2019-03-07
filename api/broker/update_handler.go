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
	"time"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/security"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/storage"
	"github.com/gofrs/uuid"
	osbc "github.com/pmorie/go-open-service-broker-client/v2"
)

type UpdateBrokerHook struct {
	OSBClientCreateFunc osbc.CreateFunc
	Encrypter           security.Encrypter
	catalog             *osbc.CatalogResponse
}

func (c *UpdateBrokerHook) OnAPI(ctx context.Context, oldObject types.Object, apiFunc func(object types.Object) (types.Object, error), changes ...*query.LabelChange) (types.Object, error) {
	broker := oldObject.(*types.Broker)
	var err error
	if err = transformBrokerCredentials(ctx, broker, c.Encrypter.Decrypt); err != nil {
		return nil, err
	}
	var obj types.Object
	obj, err = apiFunc(broker)
	if err != nil {
		return nil, err
	}
	result := obj.(*types.Broker)
	c.catalog, err = getBrokerCatalog(ctx, c.OSBClientCreateFunc, result)
	if err != nil {
		return nil, err
	}

	if err := transformBrokerCredentials(ctx, result, c.Encrypter.Encrypt); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *UpdateBrokerHook) OnTransaction(ctx context.Context, storage storage.Warehouse, transactionFunc func() (oldObject, newObject types.Object, err error)) error {
	_, newObject, err := transactionFunc()
	if err != nil {
		return util.HandleStorageError(err, "broker")
	}
	brokerID := newObject.GetID()
	existingServiceOfferingsWithServicePlans, err := storage.ServiceOffering().ListWithServicePlansByBrokerID(ctx, brokerID)
	if err != nil {
		return fmt.Errorf("error getting catalog for broker with id %s from SM DB: %s", brokerID, err)

	}

	existingServicesOfferingsMap, existingServicePlansMap := convertExistingCatalogToMaps(existingServiceOfferingsWithServicePlans)
	log.C(ctx).Debugf("Found %d services and %d plans currently known for broker", len(existingServicesOfferingsMap), len(existingServicePlansMap))

	catalogServices, catalogPlansMap, err := getBrokerCatalogServicesAndPlans(c.catalog)
	if err != nil {
		return err
	}
	log.C(ctx).Debugf("Found %d services and %d plans in catalog for broker with id %s", len(catalogServices), len(catalogPlansMap), brokerID)

	catalogPlans := make([]*catalogPlanWithServiceOfferingID, 0)

	log.C(ctx).Debugf("Resyncing service offerings for broker with id %s...", brokerID)
	for _, catalogService := range catalogServices {
		existingServiceOffering, ok := existingServicesOfferingsMap[catalogService.ID]
		delete(existingServicesOfferingsMap, catalogService.ID)
		if ok {
			if err := osbcCatalogServiceToServiceOffering(existingServiceOffering, catalogService); err != nil {
				return err
			}
			existingServiceOffering.UpdatedAt = time.Now().UTC()

			if err := existingServiceOffering.Validate(); err != nil {
				return &util.HTTPError{
					ErrorType:   "BadRequest",
					Description: fmt.Sprintf("service offering constructed during catalog update for broker %s is invalid: %s", brokerID, err),
					StatusCode:  http.StatusBadRequest,
				}
			}
			if _, err := storage.Update(ctx, existingServiceOffering); err != nil {
				return util.HandleStorageError(err, "service_offering")
			}
		} else {
			serviceUUID, err := uuid.NewV4()
			if err != nil {
				return fmt.Errorf("could not generate GUID for service_plan: %s", err)
			}
			existingServiceOffering = &types.ServiceOffering{}
			if err := osbcCatalogServiceToServiceOffering(existingServiceOffering, catalogService); err != nil {
				return err
			}
			existingServiceOffering.ID = serviceUUID.String()
			existingServiceOffering.CreatedAt = time.Now().UTC()
			existingServiceOffering.UpdatedAt = time.Now().UTC()
			existingServiceOffering.BrokerID = brokerID

			if err := existingServiceOffering.Validate(); err != nil {
				return &util.HTTPError{
					ErrorType:   "BadRequest",
					Description: fmt.Sprintf("service offering constructed during catalog update for broker %s is invalid: %s", brokerID, err),
					StatusCode:  http.StatusBadRequest,
				}
			}

			var dbServiceID string
			if dbServiceID, err = storage.Create(ctx, existingServiceOffering); err != nil {
				return util.HandleStorageError(err, "service_offering")
			}
			existingServiceOffering.ID = dbServiceID
		}

		catalogPlansForService := catalogPlansMap[catalogService.ID]
		for catalogPlanOfCatalogServiceIndex := range catalogPlansForService {
			catalogPlan := &catalogPlanWithServiceOfferingID{
				Plan:            catalogPlansForService[catalogPlanOfCatalogServiceIndex],
				ServiceOffering: existingServiceOffering,
			}
			catalogPlans = append(catalogPlans, catalogPlan)
		}
	}

	for _, existingServiceOffering := range existingServicesOfferingsMap {
		byID := query.ByField(query.EqualsOperator, "id", existingServiceOffering.ID)
		if _, err := storage.Delete(ctx, types.ServiceOfferingType, byID); err != nil {
			return util.HandleStorageError(err, "service_offering")
		}
	}
	log.C(ctx).Debugf("Successfully resynced service offerings for broker with id %s", brokerID)

	log.C(ctx).Debugf("Resyncing service plans for broker with id %s", brokerID)
	for _, catalogPlan := range catalogPlans {
		existingServicePlan, ok := existingServicePlansMap[catalogPlan.ID]
		delete(existingServicePlansMap, catalogPlan.ID)
		if ok {
			if err := osbcCatalogPlanToServicePlan(existingServicePlan, catalogPlan); err != nil {
				return err
			}
			existingServicePlan.UpdatedAt = time.Now().UTC()

			if err := existingServicePlan.Validate(); err != nil {
				return &util.HTTPError{
					ErrorType:   "BadRequest",
					Description: fmt.Sprintf("service plan constructed during catalog update for broker %s is invalid: %s", brokerID, err),
					StatusCode:  http.StatusBadRequest,
				}
			}

			if _, err := storage.Update(ctx, existingServicePlan); err != nil {
				return util.HandleStorageError(err, "service_plan")
			}
		} else {
			planUUID, err := uuid.NewV4()
			if err != nil {
				return fmt.Errorf("could not generate GUID for service_plan: %s", err)
			}
			servicePlan := &types.ServicePlan{}
			if err := osbcCatalogPlanToServicePlan(servicePlan, catalogPlan); err != nil {
				return err
			}
			servicePlan.ID = planUUID.String()
			servicePlan.CreatedAt = time.Now().UTC()
			servicePlan.UpdatedAt = time.Now().UTC()
			if err := servicePlan.Validate(); err != nil {
				return &util.HTTPError{
					ErrorType:   "BadRequest",
					Description: fmt.Sprintf("service plan constructed during catalog update for broker %s is invalid: %s", brokerID, err),
					StatusCode:  http.StatusBadRequest,
				}
			}

			if _, err := storage.Create(ctx, servicePlan); err != nil {
				return util.HandleStorageError(err, "service_plan")
			}
		}
	}

	for _, existingServicePlan := range existingServicePlansMap {
		byID := query.ByField(query.EqualsOperator, "id", existingServicePlan.ID)
		if _, err := storage.Delete(ctx, types.ServicePlanType, byID); err != nil {
			if err == util.ErrNotFoundInStorage {
				// If the service for the plan was deleted, plan would already be gone
				continue
			}
			return util.HandleStorageError(err, "service_plan")
		}
	}
	log.C(ctx).Debugf("Successfully resynced service plans for broker with id %s", brokerID)
	return nil
}
