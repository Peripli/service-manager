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
	"database/sql"
	"fmt"

	sqlxtypes "github.com/jmoiron/sqlx/types"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/storage"
)

//go:generate smgen storage ServicePlan github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types
type ServicePlan struct {
	BaseEntity
	Name        string `db:"name"`
	Description string `db:"description"`

	Free          bool         `db:"free"`
	Bindable      sql.NullBool `db:"bindable"`
	PlanUpdatable sql.NullBool `db:"plan_updateable"`
	CatalogID     string       `db:"catalog_id"`
	CatalogName   string       `db:"catalog_name"`

	Metadata               sqlxtypes.JSONText `db:"metadata"`
	Schemas                sqlxtypes.JSONText `db:"schemas"`
	MaximumPollingDuration int                `db:"maximum_polling_duration"`
	MaintenanceInfo        sqlxtypes.JSONText `db:"maintenance_info"`

	ServiceOfferingID string `db:"service_offering_id"`
}

func (sp *ServicePlan) ToObject() (types.Object, error) {
	return &types.ServicePlan{
		Base: types.Base{
			ID:             sp.ID,
			CreatedAt:      sp.CreatedAt,
			UpdatedAt:      sp.UpdatedAt,
			PagingSequence: sp.PagingSequence,
			Ready:          sp.Ready,
		},
		Name:                   sp.Name,
		Description:            sp.Description,
		CatalogID:              sp.CatalogID,
		CatalogName:            sp.CatalogName,
		Free:                   &sp.Free,
		Bindable:               toBoolPointer(sp.Bindable),
		PlanUpdatable:          toBoolPointer(sp.PlanUpdatable),
		Metadata:               getJSONRawMessage(sp.Metadata),
		Schemas:                getJSONRawMessage(sp.Schemas),
		MaximumPollingDuration: sp.MaximumPollingDuration,
		MaintenanceInfo:        getJSONRawMessage(sp.MaintenanceInfo),
		ServiceOfferingID:      sp.ServiceOfferingID,
	}, nil
}

func (sp *ServicePlan) FromObject(object types.Object) (storage.Entity, error) {
	plan, ok := object.(*types.ServicePlan)
	if !ok {
		return nil, fmt.Errorf("object is not of type ServicePlan")
	}

	isFree := func() bool {
		if plan.Free == nil {
			//If not specified, plan should be free as default
			return true
		} else {
			return *plan.Free
		}
	}

	return &ServicePlan{
		BaseEntity: BaseEntity{
			ID:             plan.ID,
			CreatedAt:      plan.CreatedAt,
			UpdatedAt:      plan.UpdatedAt,
			PagingSequence: plan.PagingSequence,
			Ready:          plan.Ready,
		},
		Name:                   plan.Name,
		Description:            plan.Description,
		Free:                   isFree(),
		Bindable:               toNullBool(plan.Bindable),
		PlanUpdatable:          toNullBool(plan.PlanUpdatable),
		CatalogID:              plan.CatalogID,
		CatalogName:            plan.CatalogName,
		Metadata:               getJSONText(plan.Metadata),
		Schemas:                getJSONText(plan.Schemas),
		MaximumPollingDuration: plan.MaximumPollingDuration,
		MaintenanceInfo:        getJSONText(plan.MaintenanceInfo),
		ServiceOfferingID:      plan.ServiceOfferingID,
	}, nil
}
