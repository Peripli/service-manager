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
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/gofrs/uuid"
)

func init() {
	RegisterEntity(types.VisibilityType, &Visibility{})
}

type Visibility struct {
	ID            string         `db:"id"`
	PlatformID    sql.NullString `db:"platform_id"`
	ServicePlanID string         `db:"service_plan_id"`
	CreatedAt     time.Time      `db:"created_at"`
	UpdatedAt     time.Time      `db:"updated_at"`
}

func (v *Visibility) GetID() string {
	return v.ID
}

func (v *Visibility) TableName() string {
	return visibilityTable
}

func (v *Visibility) PrimaryColumn() string {
	return "id"
}

func (v *Visibility) Empty() Entity {
	return &Visibility{}
}

func (v *Visibility) RowsToList(rows *sqlx.Rows) (types.ObjectList, error) {
	entities := make(map[string]*types.Visibility)
	labels := make(map[string]map[string][]string)
	result := &types.Visibilities{
		Visibilities: make([]*types.Visibility, 0),
	}
	for rows.Next() {
		row := struct {
			*Visibility
			*VisibilityLabel `db:"visibility_labels"`
		}{}
		if err := rows.StructScan(&row); err != nil {
			return nil, err
		}
		entity, ok := entities[row.Visibility.ID]
		if !ok {
			entity = row.Visibility.ToObject().(*types.Visibility)
			entities[row.Visibility.ID] = entity
			result.Visibilities = append(result.Visibilities, entity)
		}
		if labels[entity.ID] == nil {
			labels[entity.ID] = make(map[string][]string)
		}
		labels[entity.ID][row.VisibilityLabel.Key.String] = append(labels[entity.ID][row.VisibilityLabel.Key.String], row.VisibilityLabel.Val.String)
	}

	for _, b := range result.Visibilities {
		b.Labels = labels[b.ID]
	}
	return result, nil
}

func (v *Visibility) Labels() EntityLabels {
	return &visibilityLabels{}
}

func (v *Visibility) ToObject() types.Object {
	return &types.Visibility{
		ID:            v.ID,
		PlatformID:    v.PlatformID.String,
		ServicePlanID: v.ServicePlanID,
		CreatedAt:     v.CreatedAt,
		UpdatedAt:     v.UpdatedAt,
		Labels:        make(map[string][]string),
	}
}

func (v *Visibility) FromObject(visibility types.Object) Entity {
	vis := visibility.(*types.Visibility)
	return &Visibility{
		ID: vis.ID,
		// API cannot send nulls right now and storage cannot store empty string for this column as it is FK
		PlatformID:    toNullString(vis.PlatformID),
		ServicePlanID: vis.ServicePlanID,
		CreatedAt:     vis.CreatedAt,
		UpdatedAt:     vis.UpdatedAt,
	}
}

type visibilityLabels []*VisibilityLabel

func (vls *visibilityLabels) Single() Label {
	return &VisibilityLabel{}
}

func (vls *visibilityLabels) FromDTO(entityID string, labels types.Labels) ([]Label, error) {
	var result []Label
	now := time.Now()
	for key, values := range labels {
		for _, labelValue := range values {
			UUID, err := uuid.NewV4()
			if err != nil {
				return nil, fmt.Errorf("could not generate GUID for visibility label: %s", err)
			}
			id := UUID.String()
			visLabel := &VisibilityLabel{
				ID:                  toNullString(id),
				Key:                 toNullString(key),
				Val:                 toNullString(labelValue),
				CreatedAt:           &now,
				UpdatedAt:           &now,
				ServiceVisibilityID: toNullString(entityID),
			}
			result = append(result, visLabel)
		}
	}
	return result, nil
}

func (vls *visibilityLabels) ToDTO() types.Labels {
	labelValues := make(map[string][]string)
	for _, label := range *vls {
		values, exists := labelValues[label.Key.String]
		if exists {
			labelValues[label.Key.String] = append(values, label.Val.String)
		} else {
			labelValues[label.Key.String] = []string{label.Val.String}
		}
	}
	return labelValues
}

type VisibilityLabel struct {
	ID                  sql.NullString `db:"id"`
	Key                 sql.NullString `db:"key"`
	Val                 sql.NullString `db:"val"`
	CreatedAt           *time.Time     `db:"created_at"`
	UpdatedAt           *time.Time     `db:"updated_at"`
	ServiceVisibilityID sql.NullString `db:"visibility_id"`
}

func (*VisibilityLabel) TableName() string {
	return visibilityLabelsTable
}

func (*VisibilityLabel) PrimaryColumn() string {
	return "id"
}

func (*VisibilityLabel) ReferenceColumn() string {
	return "visibility_id"
}

func (*VisibilityLabel) Empty() Label {
	return &VisibilityLabel{}
}

func (*VisibilityLabel) New(entityID, id, key, value string) Label {
	now := time.Now()
	return &VisibilityLabel{
		ID:                  toNullString(id),
		Key:                 toNullString(key),
		Val:                 toNullString(value),
		ServiceVisibilityID: toNullString(entityID),
		CreatedAt:           &now,
		UpdatedAt:           &now,
	}
}

func (vl *VisibilityLabel) GetKey() string {
	return vl.Key.String
}

func (vl *VisibilityLabel) GetValue() string {
	return vl.Val.String
}
