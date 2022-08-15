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

var _ PostgresEntity = &Platform{}

const PlatformTable = "platforms"

func (*Platform) LabelEntity() PostgresLabel {
	return &PlatformLabel{}
}

func (*Platform) TableName() string {
	return PlatformTable
}

func (e *Platform) NewLabel(id, entityID, key, value string) storage.Label {
	now := pq.NullTime{
		Time:  time.Now(),
		Valid: true,
	}
	return &PlatformLabel{
		BaseLabelEntity: BaseLabelEntity{
			ID:        sql.NullString{String: id, Valid: id != ""},
			Key:       sql.NullString{String: key, Valid: key != ""},
			Val:       sql.NullString{String: value, Valid: value != ""},
			CreatedAt: now,
			UpdatedAt: now,
		},
		PlatformID: sql.NullString{String: entityID, Valid: entityID != ""},
	}
}

func (e *Platform) RowsToList(rows *sqlx.Rows) (types.ObjectList, error) {
	rowCreator := func() EntityLabelRow {
		return &struct {
			*Platform
			PlatformLabel `db:"platform_labels"`
		}{}
	}
	result := &types.Platforms{
		Platforms: make([]*types.Platform, 0),
	}
	err := rowsToList(rows, rowCreator, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

type PlatformLabel struct {
	BaseLabelEntity
	PlatformID sql.NullString `db:"platform_id"`
}

func (el PlatformLabel) LabelsTableName() string {
	return "platform_labels"
}

func (el PlatformLabel) ReferenceColumn() string {
	return "platform_id"
}
