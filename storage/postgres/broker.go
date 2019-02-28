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
	"time"

	"github.com/Peripli/service-manager/pkg/types"
)

func init() {
	RegisterEntity(types.BrokerType, Broker{})
}

//go:generate ./generate_entity.sh Broker Labels
// Broker entity
type Broker struct {
	ID          string         `db:"id"`
	Name        string         `db:"name"`
	Description sql.NullString `db:"description"`
	CreatedAt   time.Time      `db:"created_at"`
	UpdatedAt   time.Time      `db:"updated_at"`
	BrokerURL   string         `db:"broker_url"`
	Username    string         `db:"username"`
	Password    string         `db:"password"`
}

func (br Broker) ToObject() types.Object {
	broker := &types.Broker{
		ID:          br.ID,
		Name:        br.Name,
		Description: br.Description.String,
		CreatedAt:   br.CreatedAt,
		UpdatedAt:   br.UpdatedAt,
		BrokerURL:   br.BrokerURL,
		Credentials: &types.Credentials{
			Basic: &types.Basic{
				Username: br.Username,
				Password: br.Password,
			},
		},
		Labels: make(map[string][]string),
	}
	return broker
}

func (b Broker) FromObject(obj types.Object) Entity {
	if obj == nil {
		return Broker{}
	}
	broker := obj.(*types.Broker)
	res := Broker{
		ID:          broker.ID,
		Description: toNullString(broker.Description),
		Name:        broker.Name,
		BrokerURL:   broker.BrokerURL,
		CreatedAt:   broker.CreatedAt,
		UpdatedAt:   broker.UpdatedAt,
	}

	if broker.Description != "" {
		b.Description.Valid = true
	}
	if broker.Credentials != nil && broker.Credentials.Basic != nil {
		b.Username = broker.Credentials.Basic.Username
		b.Password = broker.Credentials.Basic.Password
	}
	return res
}
