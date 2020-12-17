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

	"github.com/Peripli/service-manager/pkg/types"
)

//go:generate smgen storage platform github.com/Peripli/service-manager/pkg/types
// Platform entity
type Platform struct {
	BaseEntity
	Type              string         `db:"type"`
	Name              string         `db:"name"`
	Description       sql.NullString `db:"description"`
	Username          string         `db:"username"`
	OldUsername       string         `db:"old_username"`
	Password          string         `db:"password"`
	OldPassword       string         `db:"old_password"`
	Integrity         []byte         `db:"integrity"`
	Active            bool           `db:"active"`
	CredentialsActive bool           `db:"credentials_active"`
	LastActive        time.Time      `db:"last_active"`
	Technical         bool           `db:"technical"`
}

func (p *Platform) FromObject(object types.Object) (storage.Entity, error) {
	platform, ok := object.(*types.Platform)
	if !ok {
		return nil, fmt.Errorf("object is not of type Platform")
	}
	result := &Platform{
		BaseEntity: BaseEntity{
			ID:             platform.ID,
			CreatedAt:      platform.CreatedAt,
			UpdatedAt:      platform.UpdatedAt,
			PagingSequence: platform.PagingSequence,
			Ready:          platform.Ready,
		},
		Type:              platform.Type,
		Name:              platform.Name,
		Description:       toNullString(platform.Description),
		Active:            platform.Active,
		CredentialsActive: platform.CredentialsActive,
		Technical:         platform.Technical,
		LastActive:        platform.LastActive,
	}

	if platform.Description != "" {
		result.Description.Valid = true
	}

	if platform.GetIntegrity() != nil {
		result.Integrity = platform.Integrity
	}

	if platform.Credentials != nil && platform.Credentials.Basic != nil {
		result.Username = platform.Credentials.Basic.Username
		result.Password = platform.Credentials.Basic.Password
	}

	if platform.OldCredentials != nil && platform.OldCredentials.Basic != nil {
		result.OldUsername = platform.OldCredentials.Basic.Username
		result.OldPassword = platform.OldCredentials.Basic.Password
	}
	return result, nil
}

func (p *Platform) ToObject() (types.Object, error) {
	platform := &types.Platform{
		Base: types.Base{
			ID:             p.ID,
			CreatedAt:      p.CreatedAt,
			UpdatedAt:      p.UpdatedAt,
			PagingSequence: p.PagingSequence,
			Ready:          p.Ready,
		},
		Type:              p.Type,
		Name:              p.Name,
		Description:       p.Description.String,
		Active:            p.Active,
		CredentialsActive: p.CredentialsActive,
		LastActive:        p.LastActive,
		Technical:         p.Technical,
		Integrity:         p.Integrity,
	}
	if len(p.Username) > 0 || len(p.Password) > 0 {
		platform.Credentials = &types.Credentials{
			Basic: &types.Basic{
				Username: p.Username,
				Password: p.Password,
			},
		}
	}
	if len(p.OldUsername) > 0 || len(p.OldPassword) > 0 {
		platform.OldCredentials = &types.Credentials{
			Basic: &types.Basic{
				Username: p.OldUsername,
				Password: p.OldPassword,
			},
		}
	}
	return platform, nil
}
