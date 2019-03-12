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
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/Peripli/service-manager/pkg/types"
	sqlxtypes "github.com/jmoiron/sqlx/types"
)

//
//func init() {
//	RegisterEntity(types.ServicePlanType, &ServicePlan{})
//}

type ServicePlan struct {
	ID          string    `db:"id"`
	Name        string    `db:"name"`
	Description string    `db:"description"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`

	Free          bool   `db:"free"`
	Bindable      bool   `db:"bindable"`
	PlanUpdatable bool   `db:"plan_updateable"`
	CatalogID     string `db:"catalog_id"`
	CatalogName   string `db:"catalog_name"`

	Metadata sqlxtypes.JSONText `db:"metadata"`
	Schemas  sqlxtypes.JSONText `db:"schemas"`

	ServiceOfferingID string `db:"service_offering_id"`
}

func (sp *ServicePlan) SetID(id string) {
	sp.ID = id
}

func (sp *ServicePlan) LabelEntity() LabelEntity {
	return nil
}

func (sp *ServicePlan) GetID() string {
	return sp.ID
}

func (sp *ServicePlan) TableName() string {
	return servicePlanTable
}

func (sp *ServicePlan) PrimaryColumn() string {
	return "id"
}

func (sp *ServicePlan) RowsToList(rows *sqlx.Rows) (types.ObjectList, error) {
	result := &types.ServicePlans{}
	for rows.Next() {
		var item ServicePlan
		if err := rows.StructScan(&item); err != nil {
			return nil, err
		}
		result.Add(item.ToObject())
	}
	return result, nil
}

func (sp *ServicePlan) ToObject() types.Object {
	return &types.ServicePlan{
		Base: &types.Base{
			ID:        sp.ID,
			CreatedAt: sp.CreatedAt,
			UpdatedAt: sp.UpdatedAt,
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

func (sp *ServicePlan) FromObject(object types.Object) Entity {
	plan := object.(*types.ServicePlan)
	return &ServicePlan{
		ID:                plan.ID,
		Name:              plan.Name,
		Description:       plan.Description,
		CreatedAt:         plan.CreatedAt,
		UpdatedAt:         plan.UpdatedAt,
		Free:              plan.Free,
		Bindable:          plan.Bindable,
		PlanUpdatable:     plan.PlanUpdatable,
		CatalogID:         plan.CatalogID,
		CatalogName:       plan.CatalogName,
		Metadata:          getJSONText(plan.Metadata),
		Schemas:           getJSONText(plan.Schemas),
		ServiceOfferingID: plan.ServiceOfferingID,
	}
}
