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

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/storage"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types"
)

//go:generate smgen storage Visibility github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types
type Visibility struct {
	BaseEntity
	PlatformID    sql.NullString `db:"platform_id"`
	ServicePlanID string         `db:"service_plan_id"`
}

func (v *Visibility) ToObject() (types.Object, error) {
	return &types.Visibility{
		Base: types.Base{
			ID:             v.ID,
			CreatedAt:      v.CreatedAt,
			UpdatedAt:      v.UpdatedAt,
			Labels:         make(map[string][]string),
			PagingSequence: v.PagingSequence,
			Ready:          v.Ready,
		},
		PlatformID:    v.PlatformID.String,
		ServicePlanID: v.ServicePlanID,
	}, nil
}

func (v *Visibility) FromObject(visibility types.Object) (storage.Entity, error) {
	vis, ok := visibility.(*types.Visibility)
	if !ok {
		return nil, fmt.Errorf("object is not of type Visibility")
	}
	return &Visibility{
		BaseEntity: BaseEntity{
			ID:             vis.ID,
			CreatedAt:      vis.CreatedAt,
			UpdatedAt:      vis.UpdatedAt,
			PagingSequence: vis.PagingSequence,
			Ready:          vis.Ready,
		},
		PlatformID:    toNullString(vis.PlatformID),
		ServicePlanID: vis.ServicePlanID,
	}, nil
}
