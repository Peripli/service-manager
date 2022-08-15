// GENERATED. DO NOT MODIFY!

package postgres

import (
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/storage"

	"database/sql"
	"time"
)

var _ PostgresEntity = &Broker{}

const BrokerTable = "brokers"

func (*Broker) LabelEntity() PostgresLabel {
	return &BrokerLabel{}
}

func (*Broker) TableName() string {
	return BrokerTable
}

func (e *Broker) NewLabel(id, entityID, key, value string) storage.Label {
	now := pq.NullTime{
		Time:  time.Now(),
		Valid: true,
	}
	return &BrokerLabel{
		BaseLabelEntity: BaseLabelEntity{
			ID:        sql.NullString{String: id, Valid: id != ""},
			Key:       sql.NullString{String: key, Valid: key != ""},
			Val:       sql.NullString{String: value, Valid: value != ""},
			CreatedAt: now,
			UpdatedAt: now,
		},
		BrokerID: sql.NullString{String: entityID, Valid: entityID != ""},
	}
}

func (e *Broker) RowsToList(rows *sqlx.Rows) (types.ObjectList, error) {
	rowCreator := func() EntityLabelRow {
		return &struct {
			*Broker
			BrokerLabel `db:"broker_labels"`
		}{}
	}
	result := &types.ServiceBrokers{
		ServiceBrokers: make([]*types.ServiceBroker, 0),
	}
	err := rowsToList(rows, rowCreator, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

type BrokerLabel struct {
	BaseLabelEntity
	BrokerID sql.NullString `db:"broker_id"`
}

func (el BrokerLabel) LabelsTableName() string {
	return "broker_labels"
}

func (el BrokerLabel) ReferenceColumn() string {
	return "broker_id"
}
