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
	"github.com/Peripli/service-manager/pkg/security"
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
	Repository storage.Repository

	OSBClientCreateFunc osbc.CreateFunc
	Encrypter           security.Encrypter
}

var _ web.Controller = &Controller{}

type catalogPlanWithServiceOfferingID struct {
	*osbc.Plan
	ServiceOffering *types.ServiceOffering
}

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

	if err := transformBrokerCredentials(ctx, broker, c.Encrypter.Encrypt); err != nil {
		return nil, err
	}

	if err := c.Repository.InTransaction(ctx, func(ctx context.Context, storage storage.Warehouse) error {
		var brokerID string
		if brokerID, err = storage.Broker().Create(ctx, broker); err != nil {
			return util.HandleStorageError(err, "broker", broker.ID)
		}
		for _, service := range catalog.Services {
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
			serviceOffering.BrokerID = brokerID

			if err := serviceOffering.Validate(); err != nil {
				return &util.HTTPError{
					ErrorType:   "BadRequest",
					Description: fmt.Sprintf("service offering constructed during catalog insertion for broker %s is invalid: %s", broker.ID, err),
					StatusCode:  http.StatusBadRequest,
				}
			}

			var serviceID string
			if serviceID, err = storage.ServiceOffering().Create(ctx, serviceOffering); err != nil {
				return util.HandleStorageError(err, "service_offering", service.ID)
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

				if _, err := storage.ServicePlan().Create(ctx, servicePlan); err != nil {
					return util.HandleStorageError(err, "service_plan", service.Plans[planIndex].ID)
				}
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	broker.Credentials = nil
	return util.NewJSONResponse(http.StatusCreated, broker)
}

func (c *Controller) getBroker(r *web.Request) (*web.Response, error) {
	brokerID := r.PathParams[reqBrokerID]
	ctx := r.Context()
	log.C(ctx).Debugf("Getting broker with id %s", brokerID)

	broker, err := c.Repository.Broker().Get(ctx, brokerID)
	if err != nil {
		return nil, util.HandleStorageError(err, "broker", brokerID)
	}

	broker.Credentials = nil
	return util.NewJSONResponse(http.StatusOK, broker)
}

func (c *Controller) listBrokers(r *web.Request) (*web.Response, error) {
	var brokers []*types.Broker
	var err error
	ctx := r.Context()
	log.C(ctx).Debug("Getting all brokers")
	includeCatalog := strings.ToLower(r.FormValue(catalogParam)) == "true"

	brokers, err = c.Repository.Broker().List(ctx)
	if err != nil {
		return nil, err
	}
	if includeCatalog {
		for _, broker := range brokers {
			offerings, err := c.Repository.ServiceOffering().ListWithServicePlansByBrokerID(ctx, broker.ID)
			if err != nil {
				return nil, err
			}
			broker.Services = offerings

		}
	}

	for _, broker := range brokers {
		broker.Credentials = nil
	}

	return util.NewJSONResponse(http.StatusOK, &types.Brokers{
		Brokers: brokers,
	})
}

func (c *Controller) deleteBroker(r *web.Request) (*web.Response, error) {
	brokerID := r.PathParams[reqBrokerID]
	ctx := r.Context()
	log.C(ctx).Debugf("Deleting broker with id %s", brokerID)

	if err := c.Repository.Broker().Delete(ctx, brokerID); err != nil {
		return nil, util.HandleStorageError(err, "broker", brokerID)
	}
	return util.NewJSONResponse(http.StatusOK, map[string]int{})
}

func (c *Controller) patchBroker(r *web.Request) (*web.Response, error) {
	brokerID := r.PathParams[reqBrokerID]
	ctx := r.Context()
	log.C(ctx).Debugf("Updating updateBroker with id %s", brokerID)

	broker, err := c.Repository.Broker().Get(ctx, brokerID)
	if err != nil {
		return nil, util.HandleStorageError(err, "broker", brokerID)
	}

	if err := transformBrokerCredentials(ctx, broker, c.Encrypter.Decrypt); err != nil {
		return nil, err
	}

	createdAt := broker.CreatedAt
	if err := util.BytesToObject(r.Body, broker); err != nil {
		return nil, err
	}

	broker.ID = brokerID
	broker.CreatedAt = createdAt
	broker.UpdatedAt = time.Now().UTC()

	catalog, err := c.getBrokerCatalog(ctx, broker)
	if err != nil {
		return nil, err
	}

	if err := transformBrokerCredentials(ctx, broker, c.Encrypter.Encrypt); err != nil {
		return nil, err
	}

	if err := c.resyncBrokerAndCatalog(ctx, broker, catalog); err != nil {
		return nil, err
	}

	broker.Credentials = nil
	return util.NewJSONResponse(http.StatusOK, broker)
}

func convertExistingCatalogToMaps(serviceOfferings []*types.ServiceOffering) (map[string]*types.ServiceOffering, map[string]*types.ServicePlan) {
	serviceOfferingsMap := make(map[string]*types.ServiceOffering)
	servicePlansMap := make(map[string]*types.ServicePlan)

	for serviceOfferingIndex := range serviceOfferings {
		serviceOfferingsMap[serviceOfferings[serviceOfferingIndex].CatalogID] = serviceOfferings[serviceOfferingIndex]
		for servicePlanIndex := range serviceOfferings[serviceOfferingIndex].Plans {
			servicePlansMap[serviceOfferings[serviceOfferingIndex].Plans[servicePlanIndex].CatalogID] = serviceOfferings[serviceOfferingIndex].Plans[servicePlanIndex]
		}
	}

	return serviceOfferingsMap, servicePlansMap
}

func (c *Controller) getBrokerCatalog(ctx context.Context, broker *types.Broker) (*osbc.CatalogResponse, error) {
	osbClient, err := osbcClient(ctx, c.OSBClientCreateFunc, broker)
	if err != nil {
		return nil, err
	}
	catalog, err := osbClient.GetCatalog()
	if err != nil {
		return nil, &util.HTTPError{
			ErrorType:   "BrokerError",
			Description: fmt.Sprintf("error fetching catalog from broker %s: %v", broker.Name, err),
			StatusCode:  http.StatusBadRequest,
		}
	}

	return catalog, nil
}

func getBrokerCatalogServicesAndPlans(catalog *osbc.CatalogResponse) ([]*osbc.Service, map[string][]*osbc.Plan, error) {
	services := make([]*osbc.Service, 0, len(catalog.Services))
	plans := make(map[string][]*osbc.Plan)

	for serviceIndex := range catalog.Services {
		services = append(services, &catalog.Services[serviceIndex])
		plansForService := make([]*osbc.Plan, 0)
		for planIndex := range catalog.Services[serviceIndex].Plans {
			plansForService = append(plansForService, &catalog.Services[serviceIndex].Plans[planIndex])
		}
		plans[catalog.Services[serviceIndex].ID] = plansForService
	}
	return services, plans, nil
}

func osbcCatalogServiceToServiceOffering(serviceOffering *types.ServiceOffering, service *osbc.Service) error {
	serviceTagsBytes, err := json.Marshal(service.Tags)
	if err != nil {
		return fmt.Errorf("could not marshal service tags: %s", err)
	}
	serviceRequiresBytes, err := json.Marshal(service.Requires)
	if err != nil {
		return fmt.Errorf("could not marshal service requires: %s", err)
	}
	serviceMetadataBytes, err := json.Marshal(service.Metadata)
	if err != nil {
		return fmt.Errorf("could not marshal service metadata: %s", err)
	}

	serviceOffering.Name = service.Name
	serviceOffering.Description = service.Description
	serviceOffering.Bindable = service.Bindable
	serviceOffering.InstancesRetrievable = service.BindingsRetrievable
	serviceOffering.BindingsRetrievable = service.BindingsRetrievable
	serviceOffering.PlanUpdatable = boolPointerToBool(service.PlanUpdatable, serviceOffering.PlanUpdatable)
	serviceOffering.CatalogID = service.ID
	serviceOffering.CatalogName = service.Name
	serviceOffering.Tags = json.RawMessage(serviceTagsBytes)
	serviceOffering.Requires = json.RawMessage(serviceRequiresBytes)
	serviceOffering.Metadata = json.RawMessage(serviceMetadataBytes)

	return nil
}

func osbcCatalogPlanToServicePlan(servicePlan *types.ServicePlan, plan *catalogPlanWithServiceOfferingID) error {
	planMetadataBytes, err := json.Marshal(plan.Metadata)
	if err != nil {
		return fmt.Errorf("could not marshal plan metadata: %s", err)
	}
	schemasBytes := make([]byte, 0)
	if plan.Schemas != nil {
		schemasBytes, err = json.Marshal(plan.Schemas)
		if err != nil {
			return fmt.Errorf("could not marshal plan service instance create schema: %s", err)
		}
	}

	servicePlan.Name = plan.Plan.Name
	servicePlan.Description = plan.Plan.Description
	servicePlan.CatalogID = plan.Plan.ID
	servicePlan.CatalogName = plan.Plan.Name
	servicePlan.Free = boolPointerToBool(plan.Plan.Free, servicePlan.Free)
	servicePlan.Bindable = boolPointerToBool(plan.Plan.Bindable, plan.ServiceOffering.Bindable)
	servicePlan.PlanUpdatable = boolPointerToBool(&plan.ServiceOffering.PlanUpdatable, servicePlan.PlanUpdatable)
	servicePlan.Metadata = json.RawMessage(planMetadataBytes)
	servicePlan.Schemas = schemasBytes
	servicePlan.ServiceOfferingID = plan.ServiceOffering.ID

	return nil
}

func osbcClient(ctx context.Context, createFunc osbc.CreateFunc, broker *types.Broker) (osbc.Client, error) {
	config := osbc.DefaultClientConfiguration()
	config.Name = broker.Name
	config.URL = broker.BrokerURL
	config.AuthConfig = &osbc.AuthConfig{
		BasicAuthConfig: &osbc.BasicAuthConfig{
			Username: broker.Credentials.Basic.Username,
			Password: broker.Credentials.Basic.Password,
		},
	}
	log.C(ctx).Debug("Building OSB client for service broker with name: ", config.Name, " accessible at: ", config.URL)
	return createFunc(config)
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

func boolPointerToBool(value *bool, defaultValue bool) bool {
	if value == nil {
		return defaultValue
	}
	return *value
}

func (c *Controller) resyncBrokerAndCatalog(ctx context.Context, broker *types.Broker, catalog *osbc.CatalogResponse) error {
	log.C(ctx).Debugf("Updating catalog storage for broker with id %s", broker.ID)
	if err := c.Repository.InTransaction(ctx, func(ctx context.Context, txStorage storage.Warehouse) error {
		if err := txStorage.Broker().Update(ctx, broker); err != nil {
			return util.HandleStorageError(err, "broker", broker.ID)
		}

		existingServiceOfferingsWithServicePlans, err := txStorage.ServiceOffering().ListWithServicePlansByBrokerID(ctx, broker.ID)
		if err != nil {
			return err
		}

		existingServicesOfferingsMap, existingServicePlansMap := convertExistingCatalogToMaps(existingServiceOfferingsWithServicePlans)
		log.C(ctx).Debugf("Found %d services and %d plans currently known for broker", len(existingServicesOfferingsMap), len(existingServicePlansMap))

		catalogServices, catalogPlansMap, err := getBrokerCatalogServicesAndPlans(catalog)
		if err != nil {
			return err
		}
		log.C(ctx).Debugf("Found %d services and %d plans in catalog for broker with id %s", len(catalogServices), len(catalogPlansMap), broker.ID)

		catalogPlans := make([]*catalogPlanWithServiceOfferingID, 0)

		log.C(ctx).Debugf("Resyncing service offerings for broker with id %s...", broker.ID)
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
						Description: fmt.Sprintf("service offering constructed during catalog update for broker %s is invalid: %s", broker.ID, err),
						StatusCode:  http.StatusBadRequest,
					}
				}
				if err := txStorage.ServiceOffering().Update(ctx, existingServiceOffering); err != nil {
					return util.HandleStorageError(err, "service_offering", existingServiceOffering.ID)
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
				existingServiceOffering.BrokerID = broker.ID

				if err := existingServiceOffering.Validate(); err != nil {
					return &util.HTTPError{
						ErrorType:   "BadRequest",
						Description: fmt.Sprintf("service offering constructed during catalog update for broker %s is invalid: %s", broker.ID, err),
						StatusCode:  http.StatusBadRequest,
					}
				}

				var dbServiceID string
				if dbServiceID, err = txStorage.ServiceOffering().Create(ctx, existingServiceOffering); err != nil {
					return util.HandleStorageError(err, "service_offering", existingServiceOffering.ID)
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
			if err := txStorage.ServiceOffering().Delete(ctx, existingServiceOffering.ID); err != nil {
				return util.HandleStorageError(err, "service_offering", existingServiceOffering.ID)
			}
		}
		log.C(ctx).Debugf("Successfully resynced service offerings for broker with id %s", broker.ID)

		log.C(ctx).Debugf("Resyncing service plans for broker with id %s", broker.ID)
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
						Description: fmt.Sprintf("service plan constructed during catalog update for broker %s is invalid: %s", broker.ID, err),
						StatusCode:  http.StatusBadRequest,
					}
				}

				if err := txStorage.ServicePlan().Update(ctx, existingServicePlan); err != nil {
					return util.HandleStorageError(err, "service_plan", existingServicePlan.ID)
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
						Description: fmt.Sprintf("service plan constructed during catalog update for broker %s is invalid: %s", broker.ID, err),
						StatusCode:  http.StatusBadRequest,
					}
				}

				if _, err := txStorage.ServicePlan().Create(ctx, servicePlan); err != nil {
					return util.HandleStorageError(err, "service_plan", existingServicePlan.ID)
				}
			}
		}

		for _, existingServicePlan := range existingServicePlansMap {
			if err := txStorage.ServicePlan().Delete(ctx, existingServicePlan.ID); err != nil {
				if err == util.ErrNotFoundInStorage {
					// If the service for the plan was deleted, plan would already be gone
					continue
				}
				return util.HandleStorageError(err, "service_plan", existingServicePlan.ID)
			}
		}
		log.C(ctx).Debugf("Successfully resynced service plans for broker with id %s", broker.ID)

		return nil
	}); err != nil {
		return err
	}
	log.C(ctx).Debugf("Successfully updated catalog storage for broker with id %s", broker.ID)

	return nil
}
