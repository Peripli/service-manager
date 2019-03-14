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
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	osbc "github.com/pmorie/go-open-service-broker-client/v2"
)

type catalogPlanWithServiceOfferingID struct {
	*osbc.Plan
	ServiceOffering *types.ServiceOffering
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

func getBrokerCatalog(ctx context.Context, osbClientCreateFunc osbc.CreateFunc, broker *types.ServiceBroker) (*osbc.CatalogResponse, error) {
	osbClient, err := osbcClient(ctx, osbClientCreateFunc, broker)
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

func transformBrokerCredentials(ctx context.Context, broker *types.ServiceBroker, transformationFunc func(context.Context, []byte) ([]byte, error)) error {
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
