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
	"github.com/Peripli/service-manager/pkg/instance_sharing"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/util"
	schemas2 "github.com/Peripli/service-manager/schemas"
	"github.com/Peripli/service-manager/storage"
	"github.com/gofrs/uuid"
	"github.com/tidwall/sjson"
	"net/http"
	"time"
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
			return &util.HTTPError{
				ErrorType:   "BadRequest",
				Description: fmt.Sprintf("service offering constructed during catalog insertion for broker with name %s is invalid: %s", broker.Name, err),
				StatusCode:  http.StatusBadRequest,
			}
		}
		var sharedPlanFound = false
		for _, servicePlan := range service.Plans {
			if servicePlanUsesReservedNameForReferencePlan(servicePlan) {
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
				return &util.HTTPError{
					ErrorType:   "BadRequest",
					Description: fmt.Sprintf("service plan constructed during catalog insertion for broker with name %s is invalid: %s", broker.Name, err),
					StatusCode:  http.StatusBadRequest,
				}
			}
			if servicePlan.IsShareablePlan() {
				if !isBindablePlan(service, servicePlan) {
					return util.HandleInstanceSharingError(util.ErrPlanMustBeBindable, servicePlan.ID)
				}
				sharedPlanFound = true
			}
		}
		if sharedPlanFound {
			referencePlan := generateReferencePlanObject(service.ID)
			service.Plans = append(service.Plans, referencePlan)
			// Adds OSB spec properties only
			referencePlanOSBObj := convertReferencePlanObjectToOSBPlan(referencePlan)
			// The path should append reference plan into service plans json
			catalogJsonPath := fmt.Sprintf("services.%d.plans.-1", serviceIndex)
			catalogJson, err := sjson.SetBytes(broker.Catalog, catalogJsonPath, referencePlanOSBObj)
			if err != nil {
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

func generateReferencePlanObject(serviceOfferingId string) *types.ServicePlan {
	referencePlan := new(types.ServicePlan)

	UUID, err := uuid.NewV4()
	if err != nil {
		panic(fmt.Errorf("could not generate GUID for ServicePlan: %s", err))
	}

	referencePlan.ID = UUID.String()
	referencePlan.CatalogID = UUID.String()
	referencePlan.CatalogName = instance_sharing.ReferencePlanName
	referencePlan.Name = instance_sharing.ReferencePlanName
	referencePlan.Description = instance_sharing.ReferencePlanDescription
	referencePlan.ServiceOfferingID = serviceOfferingId
	referencePlan.Bindable = newTrue()
	referencePlan.Ready = true
	referencePlan.CreatedAt = time.Now()
	referencePlan.UpdatedAt = time.Now()
	schemas, err := schemas2.SchemasLoader("reference_plan.json")
	if err == nil {
		var planSchema map[string]json.RawMessage
		err = json.Unmarshal(schemas, &planSchema)
		if err == nil {
			referencePlan.Schemas = planSchema["schemas"]
			referencePlan.Metadata = planSchema["metadata"]
		}
	}

	return referencePlan
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

func newTrue() *bool {
	b := true
	return &b
}
