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
		if err := storage.Broker().Create(ctx, broker); err != nil {
			return util.HandleStorageError(err, "broker", broker.ID)
		}
		for _, service := range catalog.Services {
			serviceOffering, err := osbcCatalogServiceToServiceOffering(&service)
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
				return fmt.Errorf("service offering constructed during catalog insertion for broker %s is invalid: %s", broker.ID, err)
			}
			if err := storage.ServiceOffering().Create(ctx, serviceOffering); err != nil {
				return util.HandleStorageError(err, "service_offering", service.ID)
			}
			for _, plan := range service.Plans {
				servicePlan, err := osbcCatalogPlanToServicePlan(&catalogPlanWithServiceOfferingID{
					Plan:            &plan,
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
					return fmt.Errorf("service plan constructed during catalog insertion for broker %s is invalid: %s", broker.ID, err)
				}

				if err := storage.ServicePlan().Create(ctx, servicePlan); err != nil {
					return util.HandleStorageError(err, "service_plan", plan.ID)
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

func (c *Controller) getAllBrokers(r *web.Request) (*web.Response, error) {
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

	createdAt := broker.CreatedAt
	if err := transformBrokerCredentials(ctx, broker, c.Encrypter.Decrypt); err != nil {
		return nil, err
	}

	if err := util.BytesToObject(r.Body, broker); err != nil {
		return nil, err
	}

	catalog, err := c.getBrokerCatalog(ctx, broker)
	if err != nil {
		return nil, err
	}

	if err := transformBrokerCredentials(ctx, broker, c.Encrypter.Encrypt); err != nil {
		return nil, err
	}

	broker.ID = brokerID
	broker.CreatedAt = createdAt
	broker.UpdatedAt = time.Now().UTC()
	broker.Credentials = nil

	log.C(ctx).Debugf("Updating catalog storage for broker with id %s", brokerID)
	if err := c.Repository.InTransaction(ctx, func(ctx context.Context, storage storage.Warehouse) error {
		if err := storage.Broker().Update(ctx, broker); err != nil {
			return util.HandleStorageError(err, "broker", broker.ID)
		}

		existingServiceOfferings, err := storage.ServiceOffering().ListWithServicePlansByBrokerID(ctx, broker.ID)
		if err != nil {
			return err
		}
		existingServicesOfferingsMap, existingServicePlansMap := convertExistingCatalogToMaps(existingServiceOfferings)

		catalogServices, catalogPlansMap, err := getBrokerCatalogServicesAndPlans(catalog)
		if err != nil {
			return err
		}

		catalogPlans := make([]*catalogPlanWithServiceOfferingID, len(catalogServices))

		for _, catalogService := range catalogServices {
			existingServiceOffering, ok := existingServicesOfferingsMap[catalogService.ID]
			delete(existingServicesOfferingsMap, catalogService.ID)
			if ok {
				existingServiceOffering.UpdatedAt = time.Now().UTC()
				if err := updateServiceOfferingWithCatalogService(existingServiceOffering, catalogService); err != nil {
					return err
				}

				if err := existingServiceOffering.Validate(); err != nil {
					return fmt.Errorf("service offering constructed during catalog update for broker %s is invalid: %s", broker.ID, err)
				}
				if err := c.Repository.ServiceOffering().Update(ctx, existingServiceOffering); err != nil {
					return util.HandleStorageError(err, "service_offering", existingServiceOffering.ID)
				}
			} else {
				serviceUUID, err := uuid.NewV4()
				if err != nil {
					return fmt.Errorf("could not generate GUID for service_plan: %s", err)
				}
				serviceOffering := &types.ServiceOffering{}
				serviceOffering.ID = serviceUUID.String()
				serviceOffering.CreatedAt = time.Now().UTC()
				serviceOffering.UpdatedAt = time.Now().UTC()
				serviceOffering.BrokerID = broker.ID
				if err := updateServiceOfferingWithCatalogService(serviceOffering, catalogService); err != nil {
					return err
				}

				if err := serviceOffering.Validate(); err != nil {
					return fmt.Errorf("service offering constructed during catalog update for broker %s is invalid: %s", broker.ID, err)
				}
				if err := c.Repository.ServiceOffering().Create(ctx, serviceOffering); err != nil {
					return util.HandleStorageError(err, "service_offering", existingServiceOffering.ID)
				}
			}

			catalogPlansForService := catalogPlansMap[catalogService.ID]
			for _, catalogPlanOfCatalogService := range catalogPlansForService {
				catalogPlan := &catalogPlanWithServiceOfferingID{
					Plan:            catalogPlanOfCatalogService,
					ServiceOffering: existingServiceOffering,
				}
				catalogPlans = append(catalogPlans, catalogPlan)
			}
		}

		for _, existingServiceOffering := range existingServicesOfferingsMap {
			if err := c.Repository.ServiceOffering().Delete(ctx, existingServiceOffering.ID); err != nil {
				return util.HandleStorageError(err, "service_offering", existingServiceOffering.ID)
			}
		}

		for _, catalogPlan := range catalogPlans {
			existingServicePlan, ok := existingServicePlansMap[catalogPlan.ID]
			delete(existingServicePlansMap, catalogPlan.ID)
			if ok {
				existingServicePlan.UpdatedAt = time.Now().UTC()
				if err := updateServicePlanWithCatalogPlan(existingServicePlan, catalogPlan); err != nil {
					return err
				}

				if err := existingServicePlan.Validate(); err != nil {
					return fmt.Errorf("service plan constructed during catalog update for broker %s is invalid: %s", broker.ID, err)
				}
				if err := c.Repository.ServicePlan().Update(ctx, existingServicePlan); err != nil {
					return util.HandleStorageError(err, "service_plan", existingServicePlan.ID)
				}
			} else {
				planUUID, err := uuid.NewV4()
				if err != nil {
					return fmt.Errorf("could not generate GUID for service_plan: %s", err)
				}
				servicePlan := &types.ServicePlan{}
				servicePlan.ID = planUUID.String()
				servicePlan.CreatedAt = time.Now().UTC()
				servicePlan.UpdatedAt = time.Now().UTC()

				if err := updateServicePlanWithCatalogPlan(servicePlan, catalogPlan); err != nil {
					return err
				}
				if err := servicePlan.Validate(); err != nil {
					return fmt.Errorf("service plan constructed during catalog update for broker %s is invalid: %s", broker.ID, err)
				}
				if err := c.Repository.ServicePlan().Create(ctx, servicePlan); err != nil {
					return util.HandleStorageError(err, "service_plan", existingServicePlan.ID)
				}
			}
		}

		for _, existingServicePlan := range existingServicePlansMap {
			if err := c.Repository.ServiceOffering().Delete(ctx, existingServicePlan.ID); err != nil {
				return util.HandleStorageError(err, "service_plan", existingServicePlan.ID)
			}
		}

		return nil
	}); err != nil {
		return nil, err
	}
	log.C(ctx).Debugf("Successfully updated catalog storage for broker with id %s", brokerID)

	return util.NewJSONResponse(http.StatusOK, broker)
}

func convertExistingCatalogToMaps(serviceOfferings []*types.ServiceOffering) (map[string]*types.ServiceOffering, map[string]*types.ServicePlan) {
	serviceOfferingsMap := make(map[string]*types.ServiceOffering, len(serviceOfferings))
	servicePlansMap := make(map[string]*types.ServicePlan, 0)

	for _, serviceOffering := range serviceOfferings {
		serviceOfferingsMap[serviceOffering.CatalogID] = serviceOffering
		for _, servicePlan := range serviceOffering.Plans {
			servicePlansMap[servicePlan.CatalogID] = servicePlan
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
		return nil, fmt.Errorf("error fetching catalog from broker %s: %s", broker.Name, err)
	}

	return catalog, nil
}

func getBrokerCatalogServicesAndPlans(catalog *osbc.CatalogResponse) ([]*osbc.Service, map[string][]*osbc.Plan, error) {
	services := make([]*osbc.Service, len(catalog.Services))
	plans := make(map[string][]*osbc.Plan, len(catalog.Services))

	for _, service := range catalog.Services {
		services = append(services, &service)
		plansForService := make([]*osbc.Plan, len(service.Plans))
		for _, plan := range service.Plans {
			plansForService = append(plansForService, &plan)
		}
		plans[service.ID] = plansForService
	}
	return services, plans, nil
}

func osbcCatalogServiceToServiceOffering(service *osbc.Service) (*types.ServiceOffering, error) {
	serviceTagsBytes, err := json.Marshal(service.Tags)
	if err != nil {
		return nil, fmt.Errorf("could not marshal service tags: %s", err)
	}
	serviceRequiresBytes, err := json.Marshal(service.Requires)
	if err != nil {
		return nil, fmt.Errorf("could not marshal service requires: %s", err)
	}
	serviceMetadataBytes, err := json.Marshal(service.Metadata)
	if err != nil {
		return nil, fmt.Errorf("could not marshal service metadata: %s", err)
	}

	return &types.ServiceOffering{
		Name:                 service.Name,
		Description:          service.Description,
		Bindable:             service.Bindable,
		InstancesRetrievable: true,
		BindingsRetrievable:  service.BindingsRetrievable,
		PlanUpdatable:        boolPointerToBool(service.PlanUpdatable, false),
		CatalogID:            service.ID,
		CatalogName:          service.Name,
		Tags:                 json.RawMessage(serviceTagsBytes),
		Requires:             json.RawMessage(serviceRequiresBytes),
		Metadata:             json.RawMessage(serviceMetadataBytes),
	}, nil
}

func osbcCatalogPlanToServicePlan(plan *catalogPlanWithServiceOfferingID) (*types.ServicePlan, error) {
	planMetadataBytes, err := json.Marshal(plan.Metadata)
	if err != nil {
		return nil, fmt.Errorf("could not marshal service_plan metadata: %s", err)
	}
	return &types.ServicePlan{
		Name:              plan.Name,
		Description:       plan.Description,
		CatalogID:         plan.ID,
		CatalogName:       plan.Name,
		Free:              boolPointerToBool(plan.Free, false),
		Bindable:          boolPointerToBool(plan.Bindable, plan.ServiceOffering.Bindable),
		PlanUpdatable:     boolPointerToBool(&plan.ServiceOffering.PlanUpdatable, false),
		Metadata:          json.RawMessage(planMetadataBytes),
		ServiceOfferingID: plan.ServiceOffering.ID,
	}, nil
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

func updateServiceOfferingWithCatalogService(serviceOffering *types.ServiceOffering, catalogService *osbc.Service) error {
	var err error
	if catalogService.PlanUpdatable == nil {
		catalogService.PlanUpdatable = &serviceOffering.PlanUpdatable
	}
	serviceOffering, err = osbcCatalogServiceToServiceOffering(catalogService)
	if err != nil {
		return err
	}
	return nil
}

func updateServicePlanWithCatalogPlan(servicePlan *types.ServicePlan, catalogPlan *catalogPlanWithServiceOfferingID) error {
	var err error
	if catalogPlan.Bindable == nil {
		catalogPlan.Bindable = &servicePlan.Bindable
	}
	if catalogPlan.Free == nil {
		catalogPlan.Free = &servicePlan.Free
	}

	servicePlan, err = osbcCatalogPlanToServicePlan(catalogPlan)
	if err != nil {
		return err
	}
	return nil
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

func resyncBrokerCatalog() {

}

func resyncServiceOfferings() {

}

func resyncServicePlans() {

}

type catalogPlanWithServiceOfferingID struct {
	*osbc.Plan
	ServiceOffering *types.ServiceOffering
}
