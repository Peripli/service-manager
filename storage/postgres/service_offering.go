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

package postgres

import (
	"fmt"

	sqlxtypes "github.com/jmoiron/sqlx/types"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/storage"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types"
)

//go:generate smgen storage ServiceOffering github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types
type ServiceOffering struct {
	BaseEntity
	Name        string `db:"name"`
	Description string `db:"description"`

	Bindable             bool   `db:"bindable"`
	InstancesRetrievable bool   `db:"instances_retrievable"`
	BindingsRetrievable  bool   `db:"bindings_retrievable"`
	PlanUpdatable        bool   `db:"plan_updateable"`
	AllowContextUpdates  bool   `db:"allow_context_updates"`
	CatalogID            string `db:"catalog_id"`
	CatalogName          string `db:"catalog_name"`

	Tags     sqlxtypes.JSONText `db:"tags"`
	Requires sqlxtypes.JSONText `db:"requires"`
	Metadata sqlxtypes.JSONText `db:"metadata"`

	BrokerID string `db:"broker_id"`

	Plans []*ServicePlan `db:"-"`
}

func (e *ServiceOffering) ToObject() (types.Object, error) {
	var plans []*types.ServicePlan
	for _, plan := range e.Plans {
		planObject, err := plan.ToObject()
		if err != nil {
			return nil, fmt.Errorf("converting service offering to object failed while converting plans: %s", err)
		}
		plans = append(plans, planObject.(*types.ServicePlan))
	}

	return &types.ServiceOffering{
		Base: types.Base{
			ID:             e.ID,
			CreatedAt:      e.CreatedAt,
			UpdatedAt:      e.UpdatedAt,
			PagingSequence: e.PagingSequence,
			Ready:          e.Ready,
		},
		Name:                 e.Name,
		Description:          e.Description,
		Bindable:             e.Bindable,
		InstancesRetrievable: e.InstancesRetrievable,
		BindingsRetrievable:  e.BindingsRetrievable,
		PlanUpdatable:        e.PlanUpdatable,
		AllowContextUpdates:  e.AllowContextUpdates,
		CatalogID:            e.CatalogID,
		CatalogName:          e.CatalogName,
		Tags:                 getJSONRawMessage(e.Tags),
		Requires:             getJSONRawMessage(e.Requires),
		Metadata:             getJSONRawMessage(e.Metadata),
		BrokerID:             e.BrokerID,
		Plans:                plans,
	}, nil
}

func (*ServiceOffering) FromObject(object types.Object) (storage.Entity, error) {
	offering, ok := object.(*types.ServiceOffering)
	if !ok {
		return nil, fmt.Errorf("object is not of type ServiceOffering")
	}
	servicePlanDTO := &ServicePlan{}
	var plans []*ServicePlan
	for _, plan := range offering.Plans {
		entity, err := servicePlanDTO.FromObject(plan)
		if err != nil {
			return nil, fmt.Errorf("converting service offering from object failed while converting plans: %s", err)
		}
		plans = append(plans, entity.(*ServicePlan))
	}

	result := &ServiceOffering{
		BaseEntity: BaseEntity{
			ID:             offering.ID,
			CreatedAt:      offering.CreatedAt,
			UpdatedAt:      offering.UpdatedAt,
			PagingSequence: offering.PagingSequence,
			Ready:          offering.Ready,
		},
		Name:                 offering.Name,
		Description:          offering.Description,
		Bindable:             offering.Bindable,
		InstancesRetrievable: offering.InstancesRetrievable,
		BindingsRetrievable:  offering.BindingsRetrievable,
		PlanUpdatable:        offering.PlanUpdatable,
		AllowContextUpdates:  offering.AllowContextUpdates,
		CatalogID:            offering.CatalogID,
		CatalogName:          offering.CatalogName,
		Tags:                 getJSONText(offering.Tags),
		Requires:             getJSONText(offering.Requires),
		Metadata:             getJSONText(offering.Metadata),
		BrokerID:             offering.BrokerID,
		Plans:                plans,
	}

	return result, nil
}
