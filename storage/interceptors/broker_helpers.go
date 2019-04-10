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
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gofrs/uuid"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	osbc "github.com/pmorie/go-open-service-broker-client/v2"
)

func convertExistingServiceOfferringsToMaps(serviceOfferings []*types.ServiceOffering) (map[string]*types.ServiceOffering, map[string][]*types.ServicePlan) {
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

func getBrokerCatalog(ctx context.Context, osbClientCreateFunc osbc.CreateFunc, broker *types.ServiceBroker) (*osbc.CatalogResponse, error) {
	osbClient, err := osbcClient(ctx, osbClientCreateFunc, broker)
	if err != nil {
		return nil, err
	}
	catalog, err := osbClient.GetCatalog()
	if err != nil {
		return nil, &util.HTTPError{
			ErrorType:   "BrokerError",
			Description: fmt.Sprintf("error fetching actualBrokerCatalog from broker %s: %v", broker.Name, err),
			StatusCode:  http.StatusBadRequest,
		}
	}

	return catalog, nil
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

func osbcClient(ctx context.Context, createFunc osbc.CreateFunc, broker *types.ServiceBroker) (osbc.Client, error) {
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

// osbCatalogToOfferings converts a broker catalog to SM entities. The service offerings ids and service plans ids are auto generated.
func osbCatalogToOfferings(catalog *osbc.CatalogResponse, brokerID string) ([]*types.ServiceOffering, error) {
	var result []*types.ServiceOffering
	for serviceIndex := range catalog.Services {
		service := catalog.Services[serviceIndex]
		serviceOffering := &types.ServiceOffering{}
		err := osbcCatalogServiceToServiceOffering(serviceOffering, &service, brokerID)
		if err != nil {
			return nil, err
		}

		if err := serviceOffering.Validate(); err != nil {
			return nil, &util.HTTPError{
				ErrorType:   "BadRequest",
				Description: fmt.Sprintf("service offering constructed during catalog insertion for broker %s is invalid: %s", brokerID, err),
				StatusCode:  http.StatusBadRequest,
			}
		}

		for planIndex := range service.Plans {
			servicePlan := &types.ServicePlan{}
			err := osbcCatalogPlanToServicePlan(&service.Plans[planIndex], serviceOffering, servicePlan)
			if err != nil {
				return nil, err
			}

			if err := servicePlan.Validate(); err != nil {
				return nil, &util.HTTPError{
					ErrorType:   "BadRequest",
					Description: fmt.Sprintf("service plan constructed during catalog insertion for broker %s is invalid: %s", brokerID, err),
					StatusCode:  http.StatusBadRequest,
				}
			}
			serviceOffering.Plans = append(serviceOffering.Plans, servicePlan)
		}
		result = append(result, serviceOffering)
	}
	return result, nil
}

func osbcCatalogServiceToServiceOffering(serviceOffering *types.ServiceOffering, service *osbc.Service, brokerID string) error {
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

	serviceUUID, err := uuid.NewV4()
	if err != nil {
		return fmt.Errorf("could not generate GUID for service: %s", err)
	}
	now := time.Now().UTC()

	serviceOffering.ID = serviceUUID.String()
	serviceOffering.BrokerID = brokerID
	serviceOffering.CreatedAt = now
	serviceOffering.UpdatedAt = now
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

func osbcCatalogPlanToServicePlan(plan *osbc.Plan, serviceOffering *types.ServiceOffering, servicePlan *types.ServicePlan) error {
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

	planUUID, err := uuid.NewV4()
	if err != nil {
		return fmt.Errorf("could not generate GUID for service_plan: %s", err)
	}

	now := time.Now().UTC()

	servicePlan.ID = planUUID.String()
	servicePlan.CreatedAt = now
	servicePlan.UpdatedAt = now

	servicePlan.Name = plan.Name
	servicePlan.Description = plan.Description
	servicePlan.CatalogID = plan.ID
	servicePlan.CatalogName = plan.Name
	servicePlan.Free = boolPointerToBool(plan.Free, servicePlan.Free)
	servicePlan.Bindable = boolPointerToBool(plan.Bindable, serviceOffering.Bindable)
	servicePlan.PlanUpdatable = boolPointerToBool(&serviceOffering.PlanUpdatable, servicePlan.PlanUpdatable)
	servicePlan.Metadata = json.RawMessage(planMetadataBytes)
	servicePlan.Schemas = schemasBytes
	servicePlan.ServiceOfferingID = serviceOffering.ID

	return nil
}

func boolPointerToBool(value *bool, defaultValue bool) bool {
	if value == nil {
		return defaultValue
	}
	return *value
}
