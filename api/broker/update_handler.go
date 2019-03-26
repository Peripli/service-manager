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

	"github.com/Peripli/service-manager/storage/catalog"

	"github.com/Peripli/service-manager/pkg/extension"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/security"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/storage"
	"github.com/gofrs/uuid"
	osbc "github.com/pmorie/go-open-service-broker-client/v2"
)

const UpdateBrokerInterceptor = "update-broker"

type updateInterceptorProvider struct {
	osbClientCreateFunc osbc.CreateFunc
	encrypter           security.Encrypter
	repository          storage.Repository
}

func (c *updateInterceptorProvider) Provide() extension.UpdateInterceptor {
	return &UpdateBrokerHook{
		OSBClientCreateFunc: c.osbClientCreateFunc,
		Encrypter:           c.encrypter,
		Repository:          c.repository,
	}
}
func (c *updateInterceptorProvider) Name() string {
	return UpdateBrokerInterceptor
}

type UpdateBrokerHook struct {
	OSBClientCreateFunc osbc.CreateFunc
	Encrypter           security.Encrypter
	Repository          storage.Repository
	catalog             *osbc.CatalogResponse
}

func (c *UpdateBrokerHook) OnAPIUpdate(h extension.InterceptUpdateOnAPI) extension.InterceptUpdateOnAPI {
	return func(ctx context.Context, changes *extension.UpdateContext) (types.Object, error) {
		obj, err := c.Repository.Get(ctx, types.ServiceBrokerType, changes.Object.GetID())
		if err != nil {
			return nil, err
		}
		broker := obj.(*types.ServiceBroker)
		if err = transformBrokerCredentials(ctx, broker, c.Encrypter.Decrypt); err != nil {
			return nil, err
		}
		if err = util.BytesToObject(changes.ObjectChanges, broker); err != nil {
			return nil, err
		}
		if c.catalog, err = getBrokerCatalog(ctx, c.OSBClientCreateFunc, broker); err != nil {
			return nil, err
		}
		serviceOfferings, err := osbCatalogToOfferings(c.catalog, broker)
		broker.Services = serviceOfferings
		if err = transformBrokerCredentials(ctx, broker, c.Encrypter.Encrypt); err != nil {
			return nil, err
		}
		changes.Object = broker
		result, err := h(ctx, changes)
		if err != nil {
			return nil, err
		}
		if secured, ok := result.(types.Secured); ok {
			secured.SetCredentials(nil)
		}
		return result, nil
	}
}

func (c *UpdateBrokerHook) OnTransactionUpdate(f extension.InterceptUpdateOnTransaction) extension.InterceptUpdateOnTransaction {
	return func(ctx context.Context, txStorage storage.Warehouse, oldObject types.Object, changes *extension.UpdateContext) (types.Object, error) {
		newObject, err := f(ctx, txStorage, oldObject, changes)
		if err != nil {
			return nil, err
		}
		broker := newObject.(*types.ServiceBroker)
		brokerID := broker.GetID()
		existingServiceOfferingsWithServicePlans, err := catalog.Load(ctx, brokerID, txStorage)
		if err != nil {
			return nil, fmt.Errorf("error getting catalog for broker with id %s from SM DB: %s", brokerID, err)
		}

		existingServicesOfferingsMap, existingServicePlansPerOfferringMap := convertExistingServiceOfferringsToMaps(existingServiceOfferingsWithServicePlans.ServiceOfferings)
		log.C(ctx).Debugf("Found %d services currently known for broker", len(existingServicesOfferingsMap))

		catalogServices, catalogPlansMap, err := getBrokerCatalogServicesAndPlans(c.catalog)
		if err != nil {
			return nil, err
		}
		log.C(ctx).Debugf("Found %d services and %d plans in catalog for broker with id %s", len(catalogServices), len(catalogPlansMap), brokerID)

		catalogPlans := make([]*catalogPlanWithServiceOfferingID, 0)

		log.C(ctx).Debugf("Resyncing service offerings for broker with id %s...", brokerID)
		for _, catalogService := range catalogServices {
			existingServiceOffering, ok := existingServicesOfferingsMap[catalogService.ID]
			delete(existingServicesOfferingsMap, catalogService.ID)
			if ok {
				if err := osbcCatalogServiceToServiceOffering(existingServiceOffering, catalogService); err != nil {
					return nil, err
				}
				existingServiceOffering.UpdatedAt = time.Now().UTC()

				if err := existingServiceOffering.Validate(); err != nil {
					return nil, &util.HTTPError{
						ErrorType:   "BadRequest",
						Description: fmt.Sprintf("service offering constructed during catalog update for broker %s is invalid: %s", brokerID, err),
						StatusCode:  http.StatusBadRequest,
					}
				}
				if _, err := txStorage.Update(ctx, existingServiceOffering); err != nil {
					return nil, util.HandleStorageError(err, "service_offering")
				}
			} else {
				serviceUUID, err := uuid.NewV4()
				if err != nil {
					return nil, fmt.Errorf("could not generate GUID for service_plan: %s", err)
				}
				existingServiceOffering = &types.ServiceOffering{}
				if err := osbcCatalogServiceToServiceOffering(existingServiceOffering, catalogService); err != nil {
					return nil, err
				}
				existingServiceOffering.ID = serviceUUID.String()
				existingServiceOffering.CreatedAt = time.Now().UTC()
				existingServiceOffering.UpdatedAt = time.Now().UTC()
				existingServiceOffering.BrokerID = brokerID

				if err := existingServiceOffering.Validate(); err != nil {
					return nil, &util.HTTPError{
						ErrorType:   "BadRequest",
						Description: fmt.Sprintf("service offering constructed during catalog update for broker %s is invalid: %s", brokerID, err),
						StatusCode:  http.StatusBadRequest,
					}
				}

				var dbServiceID string
				if dbServiceID, err = txStorage.Create(ctx, existingServiceOffering); err != nil {
					return nil, util.HandleStorageError(err, "service_offering")
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
			if _, err := txStorage.Delete(ctx, types.ServiceOfferingType, byID); err != nil {
				return nil, util.HandleStorageError(err, "service_offering")
			}
		}
		log.C(ctx).Debugf("Successfully resynced service offerings for broker with id %s", brokerID)

		log.C(ctx).Debugf("Resyncing service plans for broker with id %s", brokerID)
		for _, catalogPlan := range catalogPlans {
			existingServicePlans, ok := existingServicePlansPerOfferringMap[catalogPlan.ServiceOffering.CatalogID]
			if ok {
				var existingPlan *types.ServicePlan
				var newPlansMapping []*types.ServicePlan
				for _, existingServicePlan := range existingServicePlans {
					if existingServicePlan.CatalogID == catalogPlan.ID {
						existingPlan = existingServicePlan
					} else {
						newPlansMapping = append(newPlansMapping, existingServicePlan)
					}
				}
				if existingPlan != nil {
					if err := osbcCatalogPlanToServicePlan(existingPlan, catalogPlan); err != nil {
						return nil, err
					}
					existingPlan.UpdatedAt = time.Now().UTC()

					if err := existingPlan.Validate(); err != nil {
						return nil, &util.HTTPError{
							ErrorType:   "BadRequest",
							Description: fmt.Sprintf("service plan constructed during catalog update for broker %s is invalid: %s", brokerID, err),
							StatusCode:  http.StatusBadRequest,
						}
					}

					if _, err := txStorage.Update(ctx, existingPlan); err != nil {
						return nil, util.HandleStorageError(err, "service_plan")
					}
					existingServicePlansPerOfferringMap[catalogPlan.ServiceOffering.CatalogID] = newPlansMapping
				} else {
					if err := createPlan(ctx, txStorage, catalogPlan, brokerID); err != nil {
						return nil, err
					}
				}
			} else {
				if err := createPlan(ctx, txStorage, catalogPlan, brokerID); err != nil {
					return nil, err
				}
			}
		}

		for _, existingServicePlansForOffering := range existingServicePlansPerOfferringMap {
			for _, existingServicePlan := range existingServicePlansForOffering {
				byID := query.ByField(query.EqualsOperator, "id", existingServicePlan.ID)
				if _, err := txStorage.Delete(ctx, types.ServicePlanType, byID); err != nil {
					if err == util.ErrNotFoundInStorage {
						// If the service for the plan was deleted, plan would already be gone
						continue
					}
					return nil, util.HandleStorageError(err, "service_plan")
				}
			}
		}

		brokerServices, err := catalog.Load(ctx, brokerID, txStorage)
		if err != nil {
			return nil, fmt.Errorf("error getting catalog for broker with id %s from SM DB: %s", brokerID, err)
		}
		broker.Services = brokerServices.ServiceOfferings

		log.C(ctx).Debugf("Successfully resynced service plans for broker with id %s", brokerID)
		return broker, nil
	}
}

func createPlan(ctx context.Context, txStorage storage.Warehouse, catalogPlan *catalogPlanWithServiceOfferingID, brokerID string) error {
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

	if _, err := txStorage.Create(ctx, servicePlan); err != nil {
		return util.HandleStorageError(err, "service_plan")
	}
	return nil
}
