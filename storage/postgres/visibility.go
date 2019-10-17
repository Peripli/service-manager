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

	"github.com/Peripli/service-manager/storage"

	"github.com/Peripli/service-manager/pkg/types"
)

//go:generate smgen storage Visibility github.com/Peripli/service-manager/pkg/types
type Visibility struct {
	BaseEntity
	PlatformID    sql.NullString `db:"platform_id"`
	ServicePlanID string         `db:"service_plan_id"`
}

func (v *Visibility) ToObject() types.Object {
	return &types.Visibility{
		Base: types.Base{
			ID:             v.ID,
			CreatedAt:      v.CreatedAt,
			UpdatedAt:      v.UpdatedAt,
			Labels:         make(map[string][]string),
			PagingSequence: v.PagingSequence,
		},
		PlatformID:    v.PlatformID.String,
		ServicePlanID: v.ServicePlanID,
	}
}

func (v *Visibility) FromObject(visibility types.Object) (storage.Entity, bool) {
	vis, ok := visibility.(*types.Visibility)
	if !ok {
		return nil, false
	}
	return &Visibility{
		BaseEntity: BaseEntity{
			ID:             vis.ID,
			CreatedAt:      vis.CreatedAt,
			UpdatedAt:      vis.UpdatedAt,
			PagingSequence: vis.PagingSequence,
		},
		PlatformID:    toNullString(vis.PlatformID),
		ServicePlanID: vis.ServicePlanID,
	}, true
}
