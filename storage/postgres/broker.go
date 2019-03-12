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

	"github.com/gofrs/uuid"

	"github.com/Peripli/service-manager/storage"

	"github.com/Peripli/service-manager/pkg/types"
)

func InstallBroker(scheme *storage.Scheme) {
	scheme.Introduce(&types.Broker{}, &Broker{}, &BrokerTransformer{})
}

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

type BrokerTransformer struct {
}

func (b *BrokerTransformer) EntityFromStorage(entity storage.Entity) (types.Object, bool) {
	br, ok := entity.(*Broker)
	if !ok {
		return nil, false
	}
	broker := &types.Broker{
		ID:          br.ID,
		CreatedAt:   br.CreatedAt,
		UpdatedAt:   br.UpdatedAt,
		Labels:      map[string][]string{},
		Name:        br.Name,
		Description: br.Description.String,
		BrokerURL:   br.BrokerURL,
		Credentials: &types.Credentials{
			Basic: &types.Basic{
				Username: br.Username,
				Password: br.Password,
			},
		},
	}
	return broker, true
}

func (*BrokerTransformer) EntityToStorage(object types.Object) (storage.Entity, bool) {
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

func (*BrokerTransformer) LabelsToStorage(entityID string, objectType types.ObjectType, labels types.Labels) ([]storage.Label, bool, error) {
	if objectType != types.BrokerType {
		return nil, false, nil
	}
	var result []storage.Label
	now := time.Now()
	for key, values := range labels {
		for _, labelValue := range values {
			UUID, err := uuid.NewV4()
			if err != nil {
				return nil, false, fmt.Errorf("could not generate GUID for broker label: %s", err)
			}
			id := UUID.String()
			bLabel := &BrokerLabel{
				ID:        sql.NullString{String: id, Valid: id != ""},
				Key:       sql.NullString{String: key, Valid: key != ""},
				Val:       sql.NullString{String: labelValue, Valid: labelValue != ""},
				CreatedAt: &now,
				UpdatedAt: &now,
				BrokerID:  sql.NullString{String: entityID, Valid: entityID != ""},
			}
			result = append(result, bLabel)
		}
	}
	return result, true, nil
}
