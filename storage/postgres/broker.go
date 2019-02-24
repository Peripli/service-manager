/*
 * Copyright 2018 The Service Manager Authors
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

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/gofrs/uuid"
	"github.com/jmoiron/sqlx"
)

func init() {
	RegisterEntity(types.BrokerType, Broker{})
}

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

func (b Broker) Labels() EntityLabels {
	return brokerLabels{}
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

func (Broker) Empty() Entity {
	return Broker{}
}

func (b Broker) RowsToList(rows *sqlx.Rows) (types.ObjectList, error) {
	brokers := make(map[string]*types.Broker)
	labels := make(map[string]map[string][]string)
	result := &types.Brokers{}
	for rows.Next() {
		row := struct {
			*Broker
			*BrokerLabel `db:"broker_labels"`
		}{}
		if err := rows.StructScan(&row); err != nil {
			return nil, err
		}
		broker, ok := brokers[row.Broker.ID]
		if !ok {
			broker = row.Broker.ToObject().(*types.Broker)
			brokers[row.Broker.ID] = broker
			result.Add(broker)
		}
		if labels[broker.ID] == nil {
			labels[broker.ID] = make(map[string][]string)
		}
		labels[broker.ID][row.BrokerLabel.Key.String] = append(labels[broker.ID][row.BrokerLabel.Key.String], row.BrokerLabel.Val.String)
	}

	for _, obj := range result.Brokers {
		obj.Labels = labels[obj.ID]
	}
	return result, nil
}

func (Broker) PrimaryColumn() string {
	return "id"
}

func (Broker) TableName() string {
	return brokerTable
}

func (b Broker) GetID() string {
	return b.ID
}

type BrokerLabel struct {
	ID        sql.NullString `db:"id"`
	Key       sql.NullString `db:"key"`
	Val       sql.NullString `db:"val"`
	CreatedAt *time.Time     `db:"created_at"`
	UpdatedAt *time.Time     `db:"updated_at"`
	BrokerID  sql.NullString `db:"broker_id"`
}

func (bl BrokerLabel) TableName() string {
	return brokerLabelsTable
}

func (bl BrokerLabel) PrimaryColumn() string {
	return "id"
}

func (bl BrokerLabel) ReferenceColumn() string {
	return "broker_id"
}

func (bl BrokerLabel) Empty() Label {
	return BrokerLabel{}
}

func (bl BrokerLabel) New(entityID, id, key, value string) Label {
	now := time.Now()
	return BrokerLabel{
		ID:        toNullString(id),
		Key:       toNullString(key),
		Val:       toNullString(value),
		BrokerID:  toNullString(entityID),
		CreatedAt: &now,
		UpdatedAt: &now,
	}
}

func (bl BrokerLabel) GetKey() string {
	return bl.Key.String
}

func (bl BrokerLabel) GetValue() string {
	return bl.Val.String
}

type brokerLabels []*BrokerLabel

func (bl brokerLabels) Single() Label {
	return &BrokerLabel{}
}

func (bl brokerLabels) PrimaryColumn() string {
	return "id"
}

func (bl brokerLabels) TableName() string {
	return brokerLabelsTable
}

func (brokerLabels) ReferenceColumn() string {
	return "broker_id"
}

func (bls brokerLabels) FromDTO(entityID string, labels types.Labels) ([]Label, error) {
	var result []Label
	now := time.Now()
	for key, values := range labels {
		for _, labelValue := range values {
			UUID, err := uuid.NewV4()
			if err != nil {
				return nil, fmt.Errorf("could not generate GUID for broker label: %s", err)
			}
			id := UUID.String()
			bLabel := &BrokerLabel{
				ID:        toNullString(id),
				Key:       toNullString(key),
				Val:       toNullString(labelValue),
				CreatedAt: &now,
				UpdatedAt: &now,
				BrokerID:  toNullString(entityID),
			}
			result = append(result, bLabel)
		}
	}
	return result, nil
}

func (bls brokerLabels) ToDTO() types.Labels {
	labelValues := make(map[string][]string)
	for _, label := range bls {
		values, exists := labelValues[label.Key.String]
		if exists {
			labelValues[label.Key.String] = append(values, label.Val.String)
		} else {
			labelValues[label.Key.String] = []string{label.Val.String}
		}
	}
	return labelValues
}
