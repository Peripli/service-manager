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
	"github.com/Peripli/service-manager/storage"
	sqlxtypes "github.com/jmoiron/sqlx/types"

	"github.com/Peripli/service-manager/pkg/types"
)

// ServiceInstance entity
//go:generate smgen storage ServiceInstance github.com/Peripli/service-manager/pkg/types
type ServiceInstance struct {
	BaseEntity
	Name            string             `db:"name"`
	ServicePlanID   string             `db:"service_plan_id"`
	PlatformID      string             `db:"platform_id"`
	MaintenanceInfo sqlxtypes.JSONText `db:"maintenance_info"`
	Context         sqlxtypes.JSONText `db:"context"`
	PreviousValues  sqlxtypes.JSONText `db:"previous_values"`
	Usable          bool               `db:"usable"`
	Ready           bool               `db:"ready"`
}

func (si *ServiceInstance) ToObject() types.Object {
	return &types.ServiceInstance{
		Base: types.Base{
			ID:             si.ID,
			CreatedAt:      si.CreatedAt,
			UpdatedAt:      si.UpdatedAt,
			Labels:         map[string][]string{},
			PagingSequence: si.PagingSequence,
		},
		Name:            si.Name,
		ServicePlanID:   si.ServicePlanID,
		PlatformID:      si.PlatformID,
		MaintenanceInfo: getJSONRawMessage(si.MaintenanceInfo),
		Context:         getJSONRawMessage(si.Context),
		PreviousValues:  getJSONRawMessage(si.PreviousValues),
		Usable:          si.Usable,
		Ready:           si.Ready,
	}
}

func (*ServiceInstance) FromObject(object types.Object) (storage.Entity, bool) {
	serviceInstance, ok := object.(*types.ServiceInstance)
	if !ok {
		return nil, false
	}

	si := &ServiceInstance{
		BaseEntity: BaseEntity{
			ID:             serviceInstance.ID,
			CreatedAt:      serviceInstance.CreatedAt,
			UpdatedAt:      serviceInstance.UpdatedAt,
			PagingSequence: serviceInstance.PagingSequence,
		},
		Name:            serviceInstance.Name,
		ServicePlanID:   serviceInstance.ServicePlanID,
		PlatformID:      serviceInstance.PlatformID,
		MaintenanceInfo: getJSONText(serviceInstance.MaintenanceInfo),
		Context:         getJSONText(serviceInstance.Context),
		PreviousValues:  getJSONText(serviceInstance.PreviousValues),
		Usable:          serviceInstance.Usable,
		Ready:           serviceInstance.Ready,
	}

	return si, true
}
