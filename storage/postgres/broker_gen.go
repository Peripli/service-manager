// GENERATED. DO NOT MODIFY!

package postgres

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
	"github.com/gofrs/uuid"
	"github.com/jmoiron/sqlx"
)

func InstallBroker(scheme *storage.Scheme) {
	scheme.Introduce(&Broker{})
}

func (e *Broker) BuildLabels(labels types.Labels) ([]storage.Label, error) {
	var result []storage.Label
	now := time.Now()
	for key, values := range labels {
		for _, labelValue := range values {
			UUID, err := uuid.NewV4()
			if err != nil {
				return nil, fmt.Errorf("could not generate GUID for broker label: %s", err)
			}
			id := UUID.String()
			bLabel := &BrokerLabel{
				ID:        sql.NullString{String: id, Valid: id != ""},
				Key:       sql.NullString{String: key, Valid: key != ""},
				Val:       sql.NullString{String: labelValue, Valid: labelValue != ""},
				CreatedAt: &now,
				UpdatedAt: &now,
				BrokerID:  sql.NullString{String: e.ID, Valid: e.ID != ""},
			}
			result = append(result, bLabel)
		}
	}
	return result, nil
}

func (*Broker) PrimaryColumn() string {
	return "id"
}

func (*Broker) TableName() string {
	return "brokers"
}

func (e *Broker) LabelEntity() PostgresLabel {
	return &BrokerLabel{}
}

func (e *Broker) NewLabel(id, key, value string) storage.Label {
	now := time.Now()
	return &BrokerLabel{
		ID:        sql.NullString{String: id, Valid: id != ""},
		Key:       sql.NullString{String: key, Valid: key != ""},
		Val:       sql.NullString{String: value, Valid: value != ""},
		CreatedAt: &now,
		UpdatedAt: &now,
		BrokerID:  sql.NullString{String: e.ID, Valid: e.ID != ""},
	}
}

func (e *Broker) RowsToList(rows *sqlx.Rows) (types.ObjectList, error) {
	row := struct {
		*Broker
		*BrokerLabel `db:"broker_labels"`
	}{}
	result := &types.Brokers{}
	err := rowsToList(rows, row, result)
	if err != nil {
		return nil, err
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

func (el *BrokerLabel) LabelsPrimaryColumn() string {
	return "id"
}

func (el *BrokerLabel) LabelsTableName() string {
	return "broker_labels"
}

func (el *BrokerLabel) ReferenceColumn() string {
	return "broker_id"
}

func (el *BrokerLabel) GetKey() string {
	return el.Key.String
}

func (el *BrokerLabel) GetValue() string {
	return el.Val.String
}
