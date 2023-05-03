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
	"github.com/Peripli/service-manager/pkg/instance_sharing"
	"github.com/tidwall/sjson"
	"net/http"
	"strconv"

	"github.com/gofrs/uuid"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/storage"
)

const BrokerUpdateCatalogInterceptorName = "BrokerUpdateCatalogInterceptor"

// BrokerUpdateCatalogInterceptorProvider provides a broker interceptor for update operations
type BrokerUpdateCatalogInterceptorProvider struct {
	CatalogFetcher func(ctx context.Context, broker *types.ServiceBroker) ([]byte, error)
	CatalogLoader  func(ctx context.Context, brokerID string, repository storage.Repository) (*types.ServiceOfferings, error)
}

func (c *BrokerUpdateCatalogInterceptorProvider) Provide() storage.UpdateInterceptor {
	return &brokerUpdateCatalogInterceptor{
		CatalogFetcher: c.CatalogFetcher,
		CatalogLoader:  c.CatalogLoader,
	}
}

func (c *BrokerUpdateCatalogInterceptorProvider) Name() string {
	return BrokerUpdateCatalogInterceptorName
}

type brokerUpdateCatalogInterceptor struct {
	CatalogFetcher func(ctx context.Context, broker *types.ServiceBroker) ([]byte, error)
	CatalogLoader  func(ctx context.Context, brokerID string, repository storage.Repository) (*types.ServiceOfferings, error)
}

// AroundTxUpdate fetches the broker catalog before the transaction, so it can be stored later on in the transaction
func (c *brokerUpdateCatalogInterceptor) AroundTxUpdate(h storage.InterceptUpdateAroundTxFunc) storage.InterceptUpdateAroundTxFunc {
	return func(ctx context.Context, obj types.Object, labelChanges ...*types.LabelChange) (types.Object, error) {
		broker := obj.(*types.ServiceBroker)
		if err := brokerCatalogAroundTx(ctx, broker, c.CatalogFetcher); err != nil {
			return nil, err
		}

		return h(ctx, broker, labelChanges...)
	}
}

// OnTxUpdate stores the previously fetched broker catalog, in the transaction in which the broker is being updated
func (c *brokerUpdateCatalogInterceptor) OnTxUpdate(f storage.InterceptUpdateOnTxFunc) storage.InterceptUpdateOnTxFunc {
	return func(ctx context.Context, txStorage storage.Repository, oldObj, newObj types.Object, labelChanges ...*types.LabelChange) (types.Object, error) {
		oldBroker := oldObj.(*types.ServiceBroker)

		existingServiceOfferingsWithServicePlans, err := c.CatalogLoader(ctx, oldBroker.GetID(), txStorage)
		if err != nil {
			return nil, fmt.Errorf("error getting catalog for broker with id %s from SM DB: %s", oldBroker.GetID(), err)
		}

		oldBroker.Services = existingServiceOfferingsWithServicePlans.ServiceOfferings

		err = reuseReferenceInstancePlans(oldObj.(*types.ServiceBroker), newObj.(*types.ServiceBroker))
		if err != nil {
			return nil, err
		}

		_, err = f(ctx, txStorage, oldObj, newObj, labelChanges...)
		if err != nil {
			return nil, err
		}

		newBrokerObj := newObj.(*types.ServiceBroker)
		brokerID := newObj.GetID()

		existingServicesOfferingsMap, existingServicePlansPerOfferingMap := convertExistingServiceOfferingsToMaps(existingServiceOfferingsWithServicePlans.ServiceOfferings)
		log.C(ctx).Debugf("Found %d services currently known for broker", len(existingServicesOfferingsMap))

		catalogServices, catalogPlansMap, err := getBrokerCatalogServicesAndPlans(newBrokerObj.Services)
		if err != nil {
			return nil, err
		}
		log.C(ctx).Debugf("Found %d services and %d plans in catalog for broker with id %s", len(catalogServices), len(catalogPlansMap), brokerID)
		log.C(ctx).Debugf("Resyncing service offerings for broker with id %s...", brokerID)
		offeringsToBeCreated := make([]*types.ServiceOffering, 0)
		for _, catalogService := range catalogServices {
			existingServiceOffering, offeringExists := existingServicesOfferingsMap[catalogService.CatalogID]
			if offeringExists {
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
				if _, err := txStorage.Update(ctx, catalogService, types.LabelChanges{}); err != nil {
					return nil, err
				}
			} else {
				UUID, err := uuid.NewV4()
				if err != nil {
					return nil, err
				}
				catalogService.ID = UUID.String()
				if err := catalogService.Validate(); err != nil {
					return nil, &util.HTTPError{
						ErrorType:   "BadRequest",
						Description: fmt.Sprintf("service offering constructed during catalog update for broker %s is invalid: %s", brokerID, err),
						StatusCode:  http.StatusBadRequest,
					}
				}

				catalogService := catalogService
				offeringsToBeCreated = append(offeringsToBeCreated, catalogService)
			}

			catalogPlansForService := catalogPlansMap[catalogService.CatalogID]
			for catalogPlanOfCatalogServiceIndex := range catalogPlansForService {
				catalogPlansForService[catalogPlanOfCatalogServiceIndex].ServiceOfferingID = catalogService.ID
			}
		}

		for _, existingServiceOffering := range existingServicesOfferingsMap {
			byID := query.ByField(query.EqualsOperator, "id", existingServiceOffering.ID)
			if err := txStorage.Delete(ctx, types.ServiceOfferingType, byID); err != nil {
				return nil, err
			}
		}

		for _, offering := range offeringsToBeCreated {
			if _, err = txStorage.Create(ctx, offering); err != nil {
				return nil, err
			}
		}
		log.C(ctx).Debugf("Successfully resynced service offerings for broker with id %s", brokerID)

		log.C(ctx).Debugf("Resyncing service plans for broker with id %s", brokerID)
		plansToBeCreated := make([]*types.ServicePlan, 0)
		for serviceOfferingCatalogID, catalogPlans := range catalogPlansMap {
			// for each catalog plan of this service
			for _, catalogPlan := range catalogPlans {
				var newPlansMapping []*types.ServicePlan
				// after each iteration take the existing plans for the service again as if a previous match was found,
				// the existing plans will be reduced by one
				existingServicePlans, plansExist := existingServicePlansPerOfferingMap[serviceOfferingCatalogID]
				if plansExist {
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
						hasSharedInstances, err := planHasSharedInstances(txStorage, ctx, existingPlanUpdated.ID)
						if err != nil {
							return nil, err
						}
						if hasSharedInstances && !existingPlanUpdated.SupportsInstanceSharing() {
							return nil, util.HandleInstanceSharingError(util.ErrSharedPlanHasReferences, existingPlanUpdated.ID)
						}
						if err := existingPlanUpdated.Validate(); err != nil {
							return nil, &util.HTTPError{
								ErrorType:   "BadRequest",
								Description: fmt.Sprintf("service plan constructed during catalog update for broker %s is invalid: %s", brokerID, err),
								StatusCode:  http.StatusBadRequest,
							}
						}

						if _, err := txStorage.Update(ctx, existingPlanUpdated, types.LabelChanges{}); err != nil {
							return nil, err
						}

						// we found a match for an existing plan so we remove it from the ones that will be deleted at the end
						existingServicePlansPerOfferingMap[serviceOfferingCatalogID] = newPlansMapping
					} else {
						catalogPlan := catalogPlan
						plansToBeCreated = append(plansToBeCreated, catalogPlan)
					}
				} else {
					// for this one we didnt even find an existing service in the initially loaded list, so create it
					catalogPlan := catalogPlan
					plansToBeCreated = append(plansToBeCreated, catalogPlan)
				}
			}
		}

		for _, existingServicePlansForOffering := range existingServicePlansPerOfferingMap {
			for _, existingServicePlan := range existingServicePlansForOffering {
				byID := query.ByField(query.EqualsOperator, "id", existingServicePlan.ID)
				if err := txStorage.Delete(ctx, types.ServicePlanType, byID); err != nil {
					if err == util.ErrNotFoundInStorage {
						// If the service for the plan was deleted, plan would already be gone
						continue
					}
					return nil, err
				}
			}
		}

		for _, plan := range plansToBeCreated {
			if err := createPlan(ctx, txStorage, plan, brokerID); err != nil {
				return nil, err
			}
		}

		newBrokerObj.Services = catalogServices

		log.C(ctx).Debugf("Successfully resynced service plans for broker with id %s", brokerID)
		return newBrokerObj, nil
	}
}

// reuseReferenceInstancePlans checks whether you have a reference plan already or not.
// if the catalog has a reference plan already, it does not re-generate it, but uses the existing one.
// so we don't have multiple reference plans or changes of IDs on the existing reference plans that might be in use by other instances.
func reuseReferenceInstancePlans(oldBroker *types.ServiceBroker, newBroker *types.ServiceBroker) error {
	var newServicesByCatalogID = make(map[string]*types.ServiceOffering)
	var newServicesIdxByCatalogID = make(map[string]int)
	for index, newService := range newBroker.Services {
		newServicesByCatalogID[newService.CatalogID] = newService
		newServicesIdxByCatalogID[newService.CatalogID] = index
	}
	for _, oldService := range oldBroker.Services {
		for _, oldPlan := range oldService.Plans {
			if oldPlan.Name == instance_sharing.ReferencePlanName {
				if newServicesByCatalogID[oldService.CatalogID] != nil {
					newService := newServicesByCatalogID[oldService.CatalogID]
					for newPlanIndex, newPlan := range newService.Plans {
						if newPlan.Name == instance_sharing.ReferencePlanName {
							newPlan.CatalogID = oldPlan.CatalogID
							newPlan.ID = oldPlan.ID
							jsonPath := fmt.Sprintf("services.%d.plans.%d.id", newServicesIdxByCatalogID[newService.CatalogID], newPlanIndex)
							catalog, err := sjson.SetBytes(newBroker.Catalog, jsonPath, oldPlan.ID)
							if err != nil {
								return err
							}
							newBroker.Catalog = catalog
							break
						}
					}
				}
				break
			}
		}
	}
	return nil
}

func createPlan(ctx context.Context, txStorage storage.Repository, servicePlan *types.ServicePlan, brokerID string) error {
	if servicePlan.Name == instance_sharing.ReferencePlanName && servicePlan.ID != "" {
		UUID, err := uuid.NewV4()
		if err != nil {
			return err
		}
		servicePlan.ID = UUID.String()
	}
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

func convertExistingServiceOfferingsToMaps(serviceOfferings []*types.ServiceOffering) (map[string]*types.ServiceOffering, map[string][]*types.ServicePlan) {
	serviceOfferingsMap := make(map[string]*types.ServiceOffering)
	servicePlansMap := make(map[string][]*types.ServicePlan)

	for serviceOfferingIndex := range serviceOfferings {
		serviceOfferingsMap[serviceOfferings[serviceOfferingIndex].CatalogID] = serviceOfferings[serviceOfferingIndex]
		for servicePlanIndex := range serviceOfferings[serviceOfferingIndex].Plans {
			servicePlansMap[serviceOfferings[serviceOfferingIndex].CatalogID] = append(servicePlansMap[serviceOfferings[serviceOfferingIndex].CatalogID], serviceOfferings[serviceOfferingIndex].Plans[servicePlanIndex])
		}
	}

	return serviceOfferingsMap, servicePlansMap
}

func getBrokerCatalogServicesAndPlans(serviceOfferings []*types.ServiceOffering) ([]*types.ServiceOffering, map[string][]*types.ServicePlan, error) {
	services := make([]*types.ServiceOffering, 0, len(serviceOfferings))
	plans := make(map[string][]*types.ServicePlan)

	for serviceIndex := range serviceOfferings {
		services = append(services, serviceOfferings[serviceIndex])
		plansForService := make([]*types.ServicePlan, 0)
		for planIndex := range serviceOfferings[serviceIndex].Plans {
			plansForService = append(plansForService, serviceOfferings[serviceIndex].Plans[planIndex])
		}
		plans[serviceOfferings[serviceIndex].CatalogID] = plansForService
	}
	return services, plans, nil
}

func planHasSharedInstances(storage storage.Repository, ctx context.Context, planID string) (bool, error) {
	byServicePlanID := query.ByField(query.EqualsOperator, "service_plan_id", planID)
	bySharedValue := query.ByField(query.EqualsOperator, "shared", strconv.FormatBool(true))
	sharedInstancesCount, err := storage.Count(ctx, types.ServiceInstanceType, byServicePlanID, bySharedValue)
	if err != nil {
		return false, err
	}
	if sharedInstancesCount > 0 {
		return true, nil
	}
	return false, nil
}
