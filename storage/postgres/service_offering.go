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
	"github.com/Peripli/service-manager/storage"
	sqlxtypes "github.com/jmoiron/sqlx/types"

	"github.com/Peripli/service-manager/pkg/types"
)

//go:generate smgen storage ServiceOffering github.com/Peripli/service-manager/pkg/types
type ServiceOffering struct {
	BaseEntity
	Name        string `db:"name"`
	Description string `db:"description"`

	Bindable             bool   `db:"bindable"`
	InstancesRetrievable bool   `db:"instances_retrievable"`
	BindingsRetrievable  bool   `db:"bindings_retrievable"`
	PlanUpdatable        bool   `db:"plan_updateable"`
	CatalogID            string `db:"catalog_id"`
	CatalogName          string `db:"catalog_name"`

	Tags     sqlxtypes.JSONText `db:"tags"`
	Requires sqlxtypes.JSONText `db:"requires"`
	Metadata sqlxtypes.JSONText `db:"metadata"`

	BrokerID       string `db:"broker_id"`
	PagingSequence int64  `db:"paging_sequence,auto_increment"`

	Plans []*ServicePlan `db:"-"`
}

func (e *ServiceOffering) ToObject() types.Object {
	var plans []*types.ServicePlan
	for _, plan := range e.Plans {
		plans = append(plans, plan.ToObject().(*types.ServicePlan))
	}
	return &types.ServiceOffering{
		Base: types.Base{
			ID:             e.ID,
			CreatedAt:      e.CreatedAt,
			UpdatedAt:      e.UpdatedAt,
			PagingSequence: e.PagingSequence,
		},
		Name:                 e.Name,
		Description:          e.Description,
		Bindable:             e.Bindable,
		InstancesRetrievable: e.InstancesRetrievable,
		BindingsRetrievable:  e.BindingsRetrievable,
		PlanUpdatable:        e.PlanUpdatable,
		CatalogID:            e.CatalogID,
		CatalogName:          e.CatalogName,
		Tags:                 getJSONRawMessage(e.Tags),
		Requires:             getJSONRawMessage(e.Requires),
		Metadata:             getJSONRawMessage(e.Metadata),
		BrokerID:             e.BrokerID,
		Plans:                plans,
	}
}

func (*ServiceOffering) FromObject(object types.Object) (storage.Entity, bool) {
	offering, ok := object.(*types.ServiceOffering)
	if !ok {
		return nil, false
	}
	servicePlanDTO := &ServicePlan{}
	var plans []*ServicePlan
	for _, plan := range offering.Plans {
		if entity, isServicePlan := servicePlanDTO.FromObject(plan); isServicePlan {
			plans = append(plans, entity.(*ServicePlan))
		}
	}
	result := &ServiceOffering{
		BaseEntity: BaseEntity{
			ID:        offering.ID,
			CreatedAt: offering.CreatedAt,
			UpdatedAt: offering.UpdatedAt,
		},
		Name:                 offering.Name,
		Description:          offering.Description,
		Bindable:             offering.Bindable,
		InstancesRetrievable: offering.InstancesRetrievable,
		BindingsRetrievable:  offering.BindingsRetrievable,
		PlanUpdatable:        offering.PlanUpdatable,
		CatalogID:            offering.CatalogID,
		CatalogName:          offering.CatalogName,
		Tags:                 getJSONText(offering.Tags),
		Requires:             getJSONText(offering.Requires),
		Metadata:             getJSONText(offering.Metadata),
		BrokerID:             offering.BrokerID,
		PagingSequence:       offering.PagingSequence,
		Plans:                plans,
	}
	return result, true
}
