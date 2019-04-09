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
	"time"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
	"github.com/gofrs/uuid"
	"github.com/lib/pq"
)

type BaseEntity struct {
	ID        string    `db:"id"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func (e *BaseEntity) GetID() string {
	return e.ID
}

func (e *BaseEntity) BuildLabels(labels types.Labels, newLabel func(id, key, value string) storage.Label) ([]storage.Label, error) {
	var result []storage.Label
	for key, values := range labels {
		for _, labelValue := range values {
			UUID, err := uuid.NewV4()
			if err != nil {
				return nil, fmt.Errorf("could not generate GUID for broker label: %s", err)
			}
			result = append(result, newLabel(UUID.String(), key, labelValue))
		}
	}

	return result, nil
}

type BaseLabelEntity struct {
	ID        sql.NullString `db:"id"`
	Key       sql.NullString `db:"key"`
	Val       sql.NullString `db:"val"`
	CreatedAt pq.NullTime    `db:"created_at"`
	UpdatedAt pq.NullTime    `db:"updated_at"`
}

func (el BaseLabelEntity) GetKey() string {
	return el.Key.String
}

func (el BaseLabelEntity) GetValue() string {
	return el.Val.String
}

func (el BaseLabelEntity) LabelsPrimaryColumn() string {
	return "id"
}
