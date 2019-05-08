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

package interceptors

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Peripli/service-manager/storage/catalog"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/storage"
	osbc "github.com/pmorie/go-open-service-broker-client/v2"
)

const BrokerUpdateCatalogInterceptorName = "BrokerUpdateCatalogInterceptor"

// BrokerUpdateCatalogInterceptorProvider provides a broker interceptor for update operations
type BrokerUpdateCatalogInterceptorProvider struct {
	OsbClientCreateFunc osbc.CreateFunc
}

func (c *BrokerUpdateCatalogInterceptorProvider) Provide() storage.UpdateInterceptor {
	return &brokerUpdateCatalogInterceptor{
		OSBClientCreateFunc: c.OsbClientCreateFunc,
	}
}

func (c *BrokerUpdateCatalogInterceptorProvider) Name() string {
	return BrokerUpdateCatalogInterceptorName
}

type brokerUpdateCatalogInterceptor struct {
	OSBClientCreateFunc osbc.CreateFunc
}

// AroundTxUpdate fetches the broker catalog before the transaction, so it can be stored later on in the transaction
func (c *brokerUpdateCatalogInterceptor) AroundTxUpdate(h storage.InterceptUpdateAroundTxFunc) storage.InterceptUpdateAroundTxFunc {
	return func(ctx context.Context, obj types.Object, labelChanges ...*query.LabelChange) (types.Object, error) {
		broker := obj.(*types.ServiceBroker)
		catalog, err := getBrokerCatalog(ctx, c.OSBClientCreateFunc, broker) // keep catalog to be stored later
		if err != nil {
			return nil, err
		}
		if broker.Services, err = osbCatalogToOfferings(catalog, broker.ID); err != nil {
			return nil, err
		}

		return h(ctx, broker, labelChanges...)
	}
}

// OnTxUpdate stores the previously fetched broker catalog, in the transaction in which the broker is being updated
func (c *brokerUpdateCatalogInterceptor) OnTxUpdate(f storage.InterceptUpdateOnTxFunc) storage.InterceptUpdateOnTxFunc {
	return func(ctx context.Context, txStorage storage.Repository, oldObj, newObj types.Object, labelChanges ...*query.LabelChange) (types.Object, error) {
		oldBroker := oldObj.(*types.ServiceBroker)

		existingServiceOfferingsWithServicePlans, err := catalog.Load(ctx, oldBroker.GetID(), txStorage)
		if err != nil {
			return nil, fmt.Errorf("error getting catalog for broker with id %s from SM DB: %s", brokerID, err)
		}

		oldBroker.Services = existingServiceOfferingsWithServicePlans.ServiceOfferings

		updatedObject, err := f(ctx, txStorage, oldObj, newObj, labelChanges...)
		if err != nil {
			return nil, err
		}

		updatedBroker := updatedObject.(*types.ServiceBroker)
		brokerID := updatedBroker.GetID()

		existingServicesOfferingsMap, existingServicePlansPerOfferingMap := convertExistingServiceOfferringsToMaps(existingServiceOfferingsWithServicePlans.ServiceOfferings)
		log.C(ctx).Debugf("Found %d services currently known for broker", len(existingServicesOfferingsMap))

		catalogServices, catalogPlansMap, err := getBrokerCatalogServicesAndPlans(updatedBroker.Services)
		if err != nil {
			return nil, err
		}
		log.C(ctx).Debugf("Found %d services and %d plans in catalog for broker with id %s", len(catalogServices), len(catalogPlansMap), brokerID)

		log.C(ctx).Debugf("Resyncing service offerings for broker with id %s...", brokerID)
		for _, catalogService := range catalogServices {
			existingServiceOffering, ok := existingServicesOfferingsMap[catalogService.CatalogID]
			if ok {
				delete(existingServicesOfferingsMap, catalogService.CatalogID)
				catalogService.ID = existingServiceOffering.ID
				catalogService.CreatedAt = existingServiceOffering.CreatedAt
				catalogService.UpdatedAt = existingServiceOffering.UpdatedAt

				if err := catalogService.Validate(); err != nil {
					return nil, &util.HTTPError{
						ErrorType:   "BadRequest",
						Description: fmt.Sprintf("service offering constructed during catalog update for broker %s is invalid: %s", brokerID, err),
						StatusCode:  http.StatusBadRequest,
					}
				}
				if _, err := txStorage.Update(ctx, catalogService); err != nil {
					return nil, err
				}
			} else {
				if err := catalogService.Validate(); err != nil {
					return nil, &util.HTTPError{
						ErrorType:   "BadRequest",
						Description: fmt.Sprintf("service offering constructed during catalog update for broker %s is invalid: %s", brokerID, err),
						StatusCode:  http.StatusBadRequest,
					}
				}

				var dbServiceID string
				if dbServiceID, err = txStorage.Create(ctx, catalogService); err != nil {
					return nil, err
				}
				catalogService.ID = dbServiceID
			}

			catalogPlansForService := catalogPlansMap[catalogService.CatalogID]
			for catalogPlanOfCatalogServiceIndex := range catalogPlansForService {
				catalogPlansForService[catalogPlanOfCatalogServiceIndex].ServiceOfferingID = catalogService.ID
			}
		}

		for _, existingServiceOffering := range existingServicesOfferingsMap {
			byID := query.ByField(query.EqualsOperator, "id", existingServiceOffering.ID)
			if _, err := txStorage.Delete(ctx, types.ServiceOfferingType, byID); err != nil {
				return nil, err
			}
		}
		log.C(ctx).Debugf("Successfully resynced service offerings for broker with id %s", brokerID)

		log.C(ctx).Debugf("Resyncing service plans for broker with id %s", brokerID)
		for serviceOfferingCatalogID, catalogPlans := range catalogPlansMap {
			// for each catalog plan of this service
			for _, catalogPlan := range catalogPlans {
				var newPlansMapping []*types.ServicePlan
				// after each iteration take the existing plans for the service again as if a previous match was found,
				// the existing plans will be reduced by one
				existingServicePlans, ok := existingServicePlansPerOfferingMap[serviceOfferingCatalogID]
				if ok {
					var existingPlanUpdated *types.ServicePlan
					// for each plan in SMDB for this service
					for _, existingServicePlan := range existingServicePlans {
						if existingServicePlan.CatalogID == catalogPlan.CatalogID {
							// found a match means an update should happen
							existingPlanUpdated = catalogPlan
							existingPlanUpdated.ID = existingServicePlan.ID
							existingPlanUpdated.CreatedAt = existingServicePlan.CreatedAt
							existingPlanUpdated.UpdatedAt = existingServicePlan.UpdatedAt
						} else {
							newPlansMapping = append(newPlansMapping, existingServicePlan)
						}
					}
					if existingPlanUpdated != nil {
						if err := existingPlanUpdated.Validate(); err != nil {
							return nil, &util.HTTPError{
								ErrorType:   "BadRequest",
								Description: fmt.Sprintf("service plan constructed during catalog update for broker %s is invalid: %s", brokerID, err),
								StatusCode:  http.StatusBadRequest,
							}
						}

						if _, err := txStorage.Update(ctx, existingPlanUpdated); err != nil {
							return nil, err
						}

						// we found a match for an existing plan so we remove it from the ones that will be deleted at the end
						existingServicePlansPerOfferingMap[serviceOfferingCatalogID] = newPlansMapping
					} else {
						if err := createPlan(ctx, txStorage, catalogPlan, brokerID); err != nil {
							return nil, err
						}
					}
				} else {
					// for this one we didnt even find an existing service in the initially loaded list, so create it
					if err := createPlan(ctx, txStorage, catalogPlan, brokerID); err != nil {
						return nil, err
					}
				}
			}
		}

		for _, existingServicePlansForOffering := range existingServicePlansPerOfferingMap {
			for _, existingServicePlan := range existingServicePlansForOffering {
				byID := query.ByField(query.EqualsOperator, "id", existingServicePlan.ID)
				if _, err := txStorage.Delete(ctx, types.ServicePlanType, byID); err != nil {
					if err == util.ErrNotFoundInStorage {
						// If the service for the plan was deleted, plan would already be gone
						continue
					}
					return nil, err
				}
			}
		}

		updatedBroker.Services = catalogServices

		log.C(ctx).Debugf("Successfully resynced service plans for broker with id %s", brokerID)
		return updatedBroker, nil
	}
}

func createPlan(ctx context.Context, txStorage storage.Repository, servicePlan *types.ServicePlan, brokerID string) error {
	if err := servicePlan.Validate(); err != nil {
		return &util.HTTPError{
			ErrorType:   "BadRequest",
			Description: fmt.Sprintf("service plan constructed during catalog update for broker %s is invalid: %s", brokerID, err),
			StatusCode:  http.StatusBadRequest,
		}
	}

	if _, err := txStorage.Create(ctx, servicePlan); err != nil {
		return err
	}
	return nil
}
