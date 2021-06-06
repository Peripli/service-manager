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
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/schemas"
	"github.com/Peripli/service-manager/storage"
	"github.com/gofrs/uuid"
	"github.com/tidwall/sjson"
	"net/http"
)

const BrokerCreateCatalogInterceptorName = "BrokerCreateCatalogInterceptor"

type BrokerCreateCatalogInterceptorProvider struct {
	CatalogFetcher func(ctx context.Context, broker *types.ServiceBroker) ([]byte, error)
}

func (c *BrokerCreateCatalogInterceptorProvider) Name() string {
	return BrokerCreateCatalogInterceptorName
}

func (c *BrokerCreateCatalogInterceptorProvider) Provide() storage.CreateInterceptor {
	return &brokerCreateCatalogInterceptor{
		CatalogFetcher: c.CatalogFetcher,
	}

}

type brokerCreateCatalogInterceptor struct {
	CatalogFetcher func(ctx context.Context, broker *types.ServiceBroker) ([]byte, error)
}

func (c *brokerCreateCatalogInterceptor) AroundTxCreate(h storage.InterceptCreateAroundTxFunc) storage.InterceptCreateAroundTxFunc {
	return func(ctx context.Context, obj types.Object) (types.Object, error) {
		broker := obj.(*types.ServiceBroker)
		if err := brokerCatalogAroundTx(ctx, broker, c.CatalogFetcher); err != nil {
			return nil, err
		}

		return h(ctx, broker)
	}
}

func (c *brokerCreateCatalogInterceptor) OnTxCreate(f storage.InterceptCreateOnTxFunc) storage.InterceptCreateOnTxFunc {
	return func(ctx context.Context, storage storage.Repository, obj types.Object) (types.Object, error) {
		var createdObj types.Object
		var err error
		if createdObj, err = f(ctx, storage, obj); err != nil {
			return nil, err
		}
		broker := obj.(*types.ServiceBroker)

		for _, service := range broker.Services {
			if _, err := storage.Create(ctx, service); err != nil {
				return nil, err
			}
			for _, servicePlan := range service.Plans {
				if _, err := storage.Create(ctx, servicePlan); err != nil {
					return nil, err
				}
			}
		}

		return createdObj, nil
	}
}

func brokerCatalogAroundTx(ctx context.Context, broker *types.ServiceBroker, fetcher func(ctx context.Context, broker *types.ServiceBroker) ([]byte, error)) error {
	catalogBytes, err := fetcher(ctx, broker)
	if err != nil {
		return err
	}
	broker.Catalog = catalogBytes
	catalogResponse := struct {
		Services []*types.ServiceOffering `json:"services"`
	}{}
	if err := util.BytesToObject(catalogBytes, &catalogResponse); err != nil {
		log.C(ctx).Errorf("Failed to create the catalog: %s", err)
		return err
	}

	for serviceIndex, service := range catalogResponse.Services {
		service.CatalogID = service.ID
		service.CatalogName = service.Name
		service.BrokerID = broker.ID
		service.CreatedAt = broker.UpdatedAt
		service.UpdatedAt = broker.UpdatedAt
		service.Ready = broker.GetReady()
		UUID, err := uuid.NewV4()
		if err != nil {
			return err
		}
		service.ID = UUID.String()
		if err := service.Validate(); err != nil {
			errorDescription := fmt.Sprintf("service offering constructed during catalog insertion for broker with name %s is invalid: %s", broker.Name, err)
			log.C(ctx).Errorf(errorDescription)
			return &util.HTTPError{
				ErrorType:   "BadRequest",
				Description: errorDescription,
				StatusCode:  http.StatusBadRequest,
			}
		}
		var sharedPlanFound = false
		for _, servicePlan := range service.Plans {
			if servicePlanUsesReservedNameForReferencePlan(servicePlan) {
				log.C(ctx).Errorf("%s: %s", util.ErrCatalogUsesReservedPlanName, instance_sharing.ReferencePlanName)
				return util.HandleInstanceSharingError(util.ErrCatalogUsesReservedPlanName, instance_sharing.ReferencePlanName)
			}
			servicePlan.CatalogID = servicePlan.ID
			servicePlan.CatalogName = servicePlan.Name
			servicePlan.ServiceOfferingID = service.ID
			servicePlan.CreatedAt = broker.UpdatedAt
			servicePlan.UpdatedAt = broker.UpdatedAt
			servicePlan.Ready = broker.GetReady()
			UUID, err := uuid.NewV4()
			if err != nil {
				return err
			}
			servicePlan.ID = UUID.String()
			if err := servicePlan.Validate(); err != nil {
				errorDescription := fmt.Sprintf("service plan constructed during catalog insertion for broker with name %s is invalid: %s", broker.Name, err)
				log.C(ctx).Errorf(errorDescription)
				return &util.HTTPError{
					ErrorType:   "BadRequest",
					Description: errorDescription,
					StatusCode:  http.StatusBadRequest,
				}
			}
			if servicePlan.SupportsInstanceSharing() {
				if !isBindablePlan(service, servicePlan) {
					log.C(ctx).Errorf("%s: %s", util.ErrPlanMustBeBindable, servicePlan.ID)
					return util.HandleInstanceSharingError(util.ErrPlanMustBeBindable, servicePlan.ID)
				}
				sharedPlanFound = true
			}
		}
		if sharedPlanFound {
			referencePlan, err := schemas.CreatePlanOutOfSchema(schemas.BuildReferencePlanSchema(),service.ID)
			if err != nil {
				err := fmt.Errorf("error setting reference schema for the reference plan: %s", err)
				log.C(ctx).WithError(err)
				return err
			}
			service.Plans = append(service.Plans, referencePlan)
			// Adds OSB spec properties only
			referencePlanOSBObj := convertReferencePlanObjectToOSBPlan(referencePlan)
			// The path should append reference plan into service plans json
			catalogJsonPath := fmt.Sprintf("services.%d.plans.-1", serviceIndex)
			catalogJson, err := sjson.SetBytes(broker.Catalog, catalogJsonPath, referencePlanOSBObj)
			if err != nil {
				log.C(ctx).Errorf("Failed to create the reference plan: %s", err)
				return err
			}
			broker.Catalog = catalogJson
		}
	}
	broker.Services = catalogResponse.Services

	return nil
}

func convertReferencePlanObjectToOSBPlan(plan *types.ServicePlan) interface{} {
	return map[string]interface{}{
		"id":          plan.ID,
		"name":        plan.Name,
		"description": plan.Description,
		"bindable":    plan.Bindable,
		"metadata":    plan.Metadata,
		"schemas":     plan.Schemas,
	}
}

func isBindablePlan(service *types.ServiceOffering, plan *types.ServicePlan) bool {
	if plan.Bindable != nil {
		return *plan.Bindable
	}
	return service.Bindable
}

func servicePlanUsesReservedNameForReferencePlan(servicePlan *types.ServicePlan) bool {
	return servicePlan.Name == instance_sharing.ReferencePlanName || servicePlan.CatalogName == instance_sharing.ReferencePlanName
}
