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
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/Peripli/service-manager/pkg/types"
)

func init() {
	RegisterEntity(types.PlatformType, &Platform{})
}

// Platform entity
type Platform struct {
	ID          string         `db:"id"`
	Type        string         `db:"type"`
	Name        string         `db:"name"`
	Description sql.NullString `db:"description"`
	CreatedAt   time.Time      `db:"created_at"`
	UpdatedAt   time.Time      `db:"updated_at"`
	Username    string         `db:"username"`
	Password    string         `db:"password"`
}

func (p *Platform) GetID() string {
	return p.ID
}

func (p *Platform) TableName() string {
	return platformTable
}

func (p *Platform) PrimaryColumn() string {
	return "id"
}

func (p *Platform) Empty() Entity {
	return &Platform{}
}

func (p *Platform) RowsToList(rows *sqlx.Rows) (types.ObjectList, error) {
	return nil, nil
}

func (p *Platform) Labels() EntityLabels {
	return nil
}

func (p *Platform) FromObject(object types.Object) Entity {
	platform := object.(*types.Platform)
	result := &Platform{
		ID:          platform.ID,
		Type:        platform.Type,
		Name:        platform.Name,
		CreatedAt:   platform.CreatedAt,
		Description: toNullString(platform.Description),
		UpdatedAt:   platform.UpdatedAt,
	}

	if platform.Description != "" {
		result.Description.Valid = true
	}
	if platform.Credentials != nil && platform.Credentials.Basic != nil {
		result.Username = platform.Credentials.Basic.Username
		result.Password = platform.Credentials.Basic.Password
	}
	return result
}

func (p *Platform) ToObject() types.Object {
	return &types.Platform{
		ID:          p.ID,
		Type:        p.Type,
		Name:        p.Name,
		Description: p.Description.String,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
		Credentials: &types.Credentials{
			Basic: &types.Basic{
				Username: p.Username,
				Password: p.Password,
			},
		},
	}
}
