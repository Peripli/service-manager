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

var _ PostgresEntity = &Visibility{}

const VisibilityTable = "visibilities"

func (*Visibility) LabelEntity() PostgresLabel {
	return &VisibilityLabel{}
}

func (*Visibility) TableName() string {
	return VisibilityTable
}

func (e *Visibility) NewLabel(id, key, value string) storage.Label {
	now := pq.NullTime{
		Time:  time.Now(),
		Valid: true,
	}
	return &VisibilityLabel{
		BaseLabelEntity: BaseLabelEntity{
			ID:        sql.NullString{String: id, Valid: id != ""},
			Key:       sql.NullString{String: key, Valid: key != ""},
			Val:       sql.NullString{String: value, Valid: value != ""},
			CreatedAt: now,
			UpdatedAt: now,
		},
		VisibilityID: sql.NullString{String: e.ID, Valid: e.ID != ""},
	}
}

func (e *Visibility) RowsToList(rows *sqlx.Rows) (types.ObjectList, error) {
	rowCreator := func() EntityLabelRow {
		return &struct {
			*Visibility
			VisibilityLabel `db:"visibility_labels"`
		}{}
	}
	result := &types.Visibilities{
		Visibilities: make([]*types.Visibility, 0),
	}
	err := rowsToList(rows, rowCreator, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

type VisibilityLabel struct {
	BaseLabelEntity
	VisibilityID sql.NullString `db:"visibility_id"`
}

func (el VisibilityLabel) LabelsTableName() string {
	return "visibility_labels"
}

func (el VisibilityLabel) ReferenceColumn() string {
	return "visibility_id"
}
