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
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
	sqlxtypes "github.com/jmoiron/sqlx/types"
)

//go:generate smgen storage ServicePlan github.com/Peripli/service-manager/pkg/types
type ServicePlan struct {
	BaseEntity
	Name        string `db:"name"`
	Description string `db:"description"`

	Free          bool   `db:"free"`
	Bindable      bool   `db:"bindable"`
	PlanUpdatable bool   `db:"plan_updateable"`
	CatalogID     string `db:"catalog_id"`
	CatalogName   string `db:"catalog_name"`

	Metadata sqlxtypes.JSONText `db:"metadata"`
	Schemas  sqlxtypes.JSONText `db:"schemas"`

	ServiceOfferingID string `db:"service_offering_id"`
}

func (sp *ServicePlan) ToObject() types.Object {
	return &types.ServicePlan{
		Base: types.Base{
			ID:             sp.ID,
			CreatedAt:      sp.CreatedAt,
			UpdatedAt:      sp.UpdatedAt,
			PagingSequence: sp.PagingSequence,
		},
		Name:              sp.Name,
		Description:       sp.Description,
		CatalogID:         sp.CatalogID,
		CatalogName:       sp.CatalogName,
		Free:              sp.Free,
		Bindable:          sp.Bindable,
		PlanUpdatable:     sp.PlanUpdatable,
		Metadata:          getJSONRawMessage(sp.Metadata),
		Schemas:           getJSONRawMessage(sp.Schemas),
		ServiceOfferingID: sp.ServiceOfferingID,
	}
}

func (sp *ServicePlan) FromObject(object types.Object) (storage.Entity, bool) {
	plan, ok := object.(*types.ServicePlan)
	if !ok {
		return nil, false
	}
	return &ServicePlan{
		BaseEntity: BaseEntity{
			ID:             plan.ID,
			CreatedAt:      plan.CreatedAt,
			UpdatedAt:      plan.UpdatedAt,
			PagingSequence: plan.PagingSequence,
		},
		Name:              plan.Name,
		Description:       plan.Description,
		Free:              plan.Free,
		Bindable:          plan.Bindable,
		PlanUpdatable:     plan.PlanUpdatable,
		CatalogID:         plan.CatalogID,
		CatalogName:       plan.CatalogName,
		Metadata:          getJSONText(plan.Metadata),
		Schemas:           getJSONText(plan.Schemas),
		ServiceOfferingID: plan.ServiceOfferingID,
	}, true
}
