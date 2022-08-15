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
	"fmt"

	sqlxtypes "github.com/jmoiron/sqlx/types"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/storage"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types"
)

// ServiceBinding entity
//go:generate smgen storage ServiceBinding github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types
type ServiceBinding struct {
	BaseEntity
	Name              string                 `db:"name"`
	ServiceInstanceID string                 `db:"service_instance_id"`
	SyslogDrainURL    sql.NullString         `db:"syslog_drain_url"`
	RouteServiceURL   sql.NullString         `db:"route_service_url"`
	VolumeMounts      sqlxtypes.NullJSONText `db:"volume_mounts"`
	Endpoints         sqlxtypes.NullJSONText `db:"endpoints"`
	Context           sqlxtypes.JSONText     `db:"context"`
	BindResource      sqlxtypes.JSONText     `db:"bind_resource"`
	Credentials       string                 `db:"credentials"`
	Integrity         []byte                 `db:"integrity"`
}

func (sb *ServiceBinding) ToObject() (types.Object, error) {
	return &types.ServiceBinding{
		Base: types.Base{
			ID:             sb.ID,
			CreatedAt:      sb.CreatedAt,
			UpdatedAt:      sb.UpdatedAt,
			Labels:         map[string][]string{},
			PagingSequence: sb.PagingSequence,
			Ready:          sb.Ready,
		},
		Name:              sb.Name,
		ServiceInstanceID: sb.ServiceInstanceID,
		SyslogDrainURL:    sb.SyslogDrainURL.String,
		RouteServiceURL:   sb.RouteServiceURL.String,
		VolumeMounts:      getJSONRawMessage(sb.VolumeMounts.JSONText),
		Endpoints:         getJSONRawMessage(sb.Endpoints.JSONText),
		Context:           getJSONRawMessage(sb.Context),
		BindResource:      getJSONRawMessage(sb.BindResource),
		Credentials:       getJSONRawMessageFromString(sb.Credentials),
		Integrity:         sb.Integrity,
	}, nil
}

func (*ServiceBinding) FromObject(object types.Object) (storage.Entity, error) {
	serviceBinding, ok := object.(*types.ServiceBinding)
	if !ok {
		return nil, fmt.Errorf("object is not of type ServiceBinding")
	}

	sb := &ServiceBinding{
		BaseEntity: BaseEntity{
			ID:             serviceBinding.ID,
			CreatedAt:      serviceBinding.CreatedAt,
			UpdatedAt:      serviceBinding.UpdatedAt,
			PagingSequence: serviceBinding.PagingSequence,
			Ready:          serviceBinding.Ready,
		},
		Name:              serviceBinding.Name,
		ServiceInstanceID: serviceBinding.ServiceInstanceID,
		SyslogDrainURL:    toNullString(serviceBinding.SyslogDrainURL),
		RouteServiceURL:   toNullString(serviceBinding.RouteServiceURL),
		VolumeMounts:      getNullJSONText(serviceBinding.VolumeMounts),
		Endpoints:         getNullJSONText(serviceBinding.Endpoints),
		Context:           getJSONText(serviceBinding.Context),
		BindResource:      getJSONText(serviceBinding.BindResource),
		Credentials:       getStringFromJSONRawMessage(serviceBinding.Credentials),
		Integrity:         serviceBinding.Integrity,
	}

	return sb, nil
}
