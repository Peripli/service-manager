// GENERATED. DO NOT MODIFY!

package postgres

import (
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"database/sql"
	"time"
)

var _ PostgresEntity = &BrokerVisibility{}

const BrokerVisibilityTable = "broker_visibilities"

func (*BrokerVisibility) LabelEntity() PostgresLabel {
	return &BrokerVisibilityLabel{}
}

func (*BrokerVisibility) TableName() string {
	return BrokerVisibilityTable
}

func (e *BrokerVisibility) NewLabel(id, key, value string) storage.Label {
	now := pq.NullTime{
		Time:  time.Now(),
		Valid: true,
	}
	return &BrokerVisibilityLabel{
		BaseLabelEntity: BaseLabelEntity{
			ID:        sql.NullString{String: id, Valid: id != ""},
			Key:       sql.NullString{String: key, Valid: key != ""},
			Val:       sql.NullString{String: value, Valid: value != ""},
			CreatedAt: now,
			UpdatedAt: now,
		},
		BrokerVisibilityID: sql.NullString{String: e.ID, Valid: e.ID != ""},
	}
}

func (e *BrokerVisibility) RowsToList(rows *sqlx.Rows) (types.ObjectList, error) {
	rowCreator := func() EntityLabelRow {
		return &struct {
			*BrokerVisibility
			BrokerVisibilityLabel `db:"broker_visibility_labels"`
		}{}
	}
	result := &types.BrokerVisibilities{
		BrokerVisibilities: make([]*types.BrokerVisibility, 0),
	}
	err := rowsToList(rows, rowCreator, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

type BrokerVisibilityLabel struct {
	BaseLabelEntity
	BrokerVisibilityID sql.NullString `db:"broker_visibility_id"`
}

func (el BrokerVisibilityLabel) LabelsTableName() string {
	return "broker_visibility_labels"
}

func (el BrokerVisibilityLabel) ReferenceColumn() string {
	return "broker_visibility_id"
}
