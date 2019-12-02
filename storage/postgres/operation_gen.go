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

var _ PostgresEntity = &Operation{}

const OperationTable = "operations"

func (*Operation) LabelEntity() PostgresLabel {
	return &OperationLabel{}
}

func (*Operation) TableName() string {
	return OperationTable
}

func (e *Operation) NewLabel(id, key, value string) storage.Label {
	now := pq.NullTime{
		Time:  time.Now(),
		Valid: true,
	}
	return &OperationLabel{
		BaseLabelEntity: BaseLabelEntity{
			ID:        sql.NullString{String: id, Valid: id != ""},
			Key:       sql.NullString{String: key, Valid: key != ""},
			Val:       sql.NullString{String: value, Valid: value != ""},
			CreatedAt: now,
			UpdatedAt: now,
		},
		OperationID: sql.NullString{String: e.ID, Valid: e.ID != ""},
	}
}

func (e *Operation) RowsToList(rows *sqlx.Rows) (types.ObjectList, error) {
	rowCreator := func() EntityLabelRow {
		return &struct {
			*Operation
			OperationLabel `db:"operation_labels"`
		}{}
	}
	result := &types.Operations{
		Operations: make([]*types.Operation, 0),
	}
	err := rowsToList(rows, rowCreator, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

type OperationLabel struct {
	BaseLabelEntity
	OperationID sql.NullString `db:"operation_id"`
}

func (el OperationLabel) LabelsTableName() string {
	return "operation_labels"
}

func (el OperationLabel) ReferenceColumn() string {
	return "operation_id"
}
