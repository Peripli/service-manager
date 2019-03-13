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

	"github.com/Peripli/service-manager/storage"

	"github.com/Peripli/service-manager/pkg/types"
)

//go:generate smgen storage broker labels github.com/Peripli/service-manager/pkg/types
// Broker entity
type Broker struct {
	BaseEntity
	Name        string         `db:"name"`
	Description sql.NullString `db:"description"`
	BrokerURL   string         `db:"broker_url"`
	Username    string         `db:"username"`
	Password    string         `db:"password"`
}

func (e *Broker) ToObject() types.Object {
	broker := &types.Broker{
		BaseLabelled: types.BaseLabelled{
			Base: types.Base{
				ID:        e.ID,
				CreatedAt: e.CreatedAt,
				UpdatedAt: e.UpdatedAt,
			},
			Labels: map[string][]string{},
		},
		Name:        e.Name,
		Description: e.Description.String,
		BrokerURL:   e.BrokerURL,
		Credentials: &types.Credentials{
			Basic: &types.Basic{
				Username: e.Username,
				Password: e.Password,
			},
		},
	}
	return broker
}

func (*Broker) FromObject(object types.Object) (storage.Entity, bool) {
	broker, ok := object.(*types.Broker)
	if !ok {
		return nil, false
	}
	b := &Broker{
		BaseEntity: BaseEntity{
			ID:        broker.ID,
			CreatedAt: broker.CreatedAt,
			UpdatedAt: broker.UpdatedAt,
		},
		Description: toNullString(broker.Description),
		Name:        broker.Name,
		BrokerURL:   broker.BrokerURL,
	}
	if broker.Credentials != nil && broker.Credentials.Basic != nil {
		b.Username = broker.Credentials.Basic.Username
		b.Password = broker.Credentials.Basic.Password
	}
	return b, true
}
