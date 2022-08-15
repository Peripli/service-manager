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

package postgres

import (
	"database/sql"
	"encoding/json"
	"fmt"
	sqlxtypes "github.com/jmoiron/sqlx/types"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/storage"
)

// ServiceInstance entity
//go:generate smgen storage ServiceInstance github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types
type ServiceInstance struct {
	BaseEntity
	Name                 string             `db:"name"`
	ServicePlanID        string             `db:"service_plan_id"`
	PlatformID           string             `db:"platform_id"`
	ReferencedInstanceID sql.NullString     `db:"referenced_instance_id"`
	DashboardURL         sql.NullString     `db:"dashboard_url"`
	MaintenanceInfo      sqlxtypes.JSONText `db:"maintenance_info"`
	Context              sqlxtypes.JSONText `db:"context"`
	PreviousValues       sqlxtypes.JSONText `db:"previous_values"`
	UpdateValues         sqlxtypes.JSONText `db:"update_values"`
	Usable               bool               `db:"usable"`
	Shared               sql.NullBool       `db:"shared"`
}

func (si *ServiceInstance) ToObject() (types.Object, error) {
	var updateValues types.InstanceUpdateValues
	if si.UpdateValues != nil {
		if err := json.Unmarshal(si.UpdateValues, &updateValues); err != nil {
			return nil, err
		}
	}
	return &types.ServiceInstance{
		Base: types.Base{
			ID:             si.ID,
			CreatedAt:      si.CreatedAt,
			UpdatedAt:      si.UpdatedAt,
			Labels:         map[string][]string{},
			PagingSequence: si.PagingSequence,
			Ready:          si.Ready,
		},
		Name:                 si.Name,
		ServicePlanID:        si.ServicePlanID,
		PlatformID:           si.PlatformID,
		ReferencedInstanceID: si.ReferencedInstanceID.String,
		DashboardURL:         si.DashboardURL.String,
		MaintenanceInfo:      getJSONRawMessage(si.MaintenanceInfo),
		Context:              getJSONRawMessage(si.Context),
		PreviousValues:       getJSONRawMessage(si.PreviousValues),
		UpdateValues:         updateValues,
		Usable:               si.Usable,
		Shared:               toBoolPointer(si.Shared),
	}, nil
}

func (*ServiceInstance) FromObject(object types.Object) (storage.Entity, error) {
	serviceInstance, ok := object.(*types.ServiceInstance)
	if !ok {
		return nil, fmt.Errorf("object is not of type ServiceInstance")
	}

	newStateBytes, err := json.Marshal(serviceInstance.UpdateValues)
	if err != nil {
		return nil, err
	}

	si := &ServiceInstance{
		BaseEntity: BaseEntity{
			ID:             serviceInstance.ID,
			CreatedAt:      serviceInstance.CreatedAt,
			UpdatedAt:      serviceInstance.UpdatedAt,
			PagingSequence: serviceInstance.PagingSequence,
			Ready:          serviceInstance.Ready,
		},
		Name:                 serviceInstance.Name,
		ServicePlanID:        serviceInstance.ServicePlanID,
		PlatformID:           serviceInstance.PlatformID,
		ReferencedInstanceID: toNullString(serviceInstance.ReferencedInstanceID),
		DashboardURL:         toNullString(serviceInstance.DashboardURL),
		MaintenanceInfo:      getJSONText(serviceInstance.MaintenanceInfo),
		Context:              getJSONText(serviceInstance.Context),
		PreviousValues:       getJSONText(serviceInstance.PreviousValues),
		UpdateValues:         newStateBytes,
		Usable:               serviceInstance.Usable,
		Shared:               toNullBool(serviceInstance.Shared),
	}

	return si, nil
}
