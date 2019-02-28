// GENERATED. DO NOT MODIFY!

package postgres

import (
	"database/sql"
	"fmt"
	"github.com/jmoiron/sqlx"
	"time"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/gofrs/uuid"
)

func (Broker) Empty() Entity {
	return Broker{}
}

func (Broker) PrimaryColumn() string {
	return "id"
}

func (Broker) TableName() string {
	return brokerTable
}

func (e Broker) GetID() string {
	return e.ID
}

func (e Broker) Labels() EntityLabels {
    return brokerLabels{}
}

func (e Broker) RowsToList(rows *sqlx.Rows) (types.ObjectList, error) {
    entities := make(map[string]*types.Broker)
	labels := make(map[string]map[string][]string)
	result := &types.Brokers{
		Brokers: make([]*types.Broker, 0),
	}
	for rows.Next() {
		row := struct {
			*Broker
			*BrokerLabel `db:broker_labels`
		}{}
		if err := rows.StructScan(&row); err != nil {
			return nil, err
		}
		entity, ok := entities[row.Broker.ID]
		if !ok {
			entity = row.Broker.ToObject().(*types.Broker)
			entities[row.Broker.ID] = entity
			result.Brokers = append(result.Brokers, entity)
		}
		if labels[entity.ID] == nil {
			labels[entity.ID] = make(map[string][]string)
		}
		labels[entity.ID][row.BrokerLabel.Key.String] = append(labels[entity.ID][row.BrokerLabel.Key.String], row.BrokerLabel.Val.String)
	}

	for _, b := range result.Brokers {
		b.Labels = labels[b.ID]
	}
	return result, nil
}


type BrokerLabel struct {
	ID        sql.NullString `db:"id"`
	Key       sql.NullString `db:"key"`
	Val       sql.NullString `db:"val"`
	CreatedAt *time.Time     `db:"created_at"`
	UpdatedAt *time.Time     `db:"updated_at"`
	BrokerID  sql.NullString `db:"broker_id"`
}

func (el BrokerLabel) TableName() string {
	return "broker_labels"
}

func (el BrokerLabel) PrimaryColumn() string {
	return "id"
}

func (el BrokerLabel) ReferenceColumn() string {
	return "broker_id"
}

func (el BrokerLabel) Empty() Label {
	return BrokerLabel{}
}

func (el BrokerLabel) New(entityID, id, key, value string) Label {
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

func (el BrokerLabel) GetKey() string {
	return el.Key.String
}

func (el BrokerLabel) GetValue() string {
	return el.Val.String
}

type brokerLabels []*BrokerLabel

func (el brokerLabels) Single() Label {
	return &BrokerLabel{}
}

func (el brokerLabels) FromDTO(entityID string, labels types.Labels) ([]Label, error) {
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

func (els brokerLabels) ToDTO() types.Labels {
	labelValues := make(map[string][]string)
	for _, label := range els {
		values, exists := labelValues[label.Key.String]
		if exists {
			labelValues[label.Key.String] = append(values, label.Val.String)
		} else {
			labelValues[label.Key.String] = []string{label.Val.String}
		}
	}
	return labelValues
}

