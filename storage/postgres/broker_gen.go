// GENERATED. DO NOT MODIFY!

package postgres

import (
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
	"github.com/jmoiron/sqlx"

	"database/sql"
	"time"
)

//func (e *Broker) SetID(id string) {
//	e.ID = id
//}
//
//func (e *Broker) GetID() string {
//	return e.ID
//}

func (Broker) PrimaryColumn() string {
	return "id"
}

func (Broker) TableName() string {
	return "brokers"
}

func (e Broker) LabelEntity() LabelEntity {
	return &BrokerLabel{}
}

func (e Broker) RowsToList(rows *sqlx.Rows) (types.ObjectList, error) {
	row := struct {
		*Broker
		*BrokerLabel `db:"broker_labels"`
	}{}
	result := &types.Brokers{}
	return result, rowsToList(rows, row, result)
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

func (el *BrokerLabel) NewLabelInstance() storage.Label {
	return &BrokerLabel{}
}

func (el *BrokerLabel) New(entityID, id, key, value string) storage.Label {
	now := time.Now()
	return &BrokerLabel{
		ID:        sql.NullString{String: id, Valid: id != ""},
		Key:       sql.NullString{String: key, Valid: key != ""},
		Val:       sql.NullString{String: value, Valid: value != ""},
		CreatedAt: &now,
		UpdatedAt: &now,
		BrokerID:  sql.NullString{String: entityID, Valid: entityID != ""},
	}
}

func (el *BrokerLabel) GetKey() string {
	return el.Key.String
}

func (el *BrokerLabel) GetValue() string {
	return el.Val.String
}
