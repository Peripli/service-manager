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

	"github.com/Peripli/service-manager/storage"

	"github.com/jmoiron/sqlx"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/gofrs/uuid"
)

type Visibility struct {
	BaseEntity
	PlatformID    sql.NullString `db:"platform_id"`
	ServicePlanID string         `db:"service_plan_id"`
}

func (v *Visibility) LabelEntity() PostgresLabel {
	return &VisibilityLabel{}
}

func (v *Visibility) TableName() string {
	return visibilityTable
}

func (v *Visibility) PrimaryColumn() string {
	return "id"
}

func (v *Visibility) RowsToList(rows *sqlx.Rows) (types.ObjectList, error) {
	row := struct {
		*Visibility
		*VisibilityLabel `db:"visibility_labels"`
	}{}
	result := &types.Visibilities{}
	err := rowsToList(rows, row, result)
	return result, err
}

//TODO make these generated as well (everything is easy execept user,pass -> credentials
func (v *Visibility) ToObject() types.Object {
	return &types.Visibility{
		ID:            v.ID,
		CreatedAt:     v.CreatedAt,
		UpdatedAt:     v.UpdatedAt,
		Labels:        make(map[string][]string),
		PlatformID:    v.PlatformID.String,
		ServicePlanID: v.ServicePlanID,
	}
}

func (v *Visibility) FromObject(visibility types.Object) PostgresEntity {
	vis := visibility.(*types.Visibility)
	return &Visibility{
		BaseEntity: BaseEntity{
			ID:        vis.ID,
			CreatedAt: vis.CreatedAt,
			UpdatedAt: vis.UpdatedAt,
		},
		// API cannot send nulls right now and storage cannot store empty string for this column as it is FK
		PlatformID:    toNullString(vis.PlatformID),
		ServicePlanID: vis.ServicePlanID,
	}
}

type visibilityLabels []*VisibilityLabel

func (vls *visibilityLabels) Single() PostgresLabel {
	return &VisibilityLabel{}
}

func (vls *visibilityLabels) FromDTO(entityID string, labels types.Labels) ([]PostgresLabel, error) {
	var result []PostgresLabel
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

func (vl *VisibilityLabel) NewLabelInstance() storage.Label {
	return &VisibilityLabel{}
}

func (vl *VisibilityLabel) LabelsTableName() string {
	return visibilityLabelsTable
}

func (vl *VisibilityLabel) LabelsPrimaryColumn() string {
	return "id"
}

func (*VisibilityLabel) ReferenceColumn() string {
	return "visibility_id"
}

func (*VisibilityLabel) Empty() PostgresLabel {
	return &VisibilityLabel{}
}

func (*VisibilityLabel) New(entityID, id, key, value string) storage.Label {
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
