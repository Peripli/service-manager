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

var _ PostgresEntity = &Notification{}

const NotificationTable = "notifications"

func (*Notification) LabelEntity() PostgresLabel {
	return &NotificationLabel{}
}

func (*Notification) TableName() string {
	return NotificationTable
}

func (e *Notification) NewLabel(id, entityID, key, value string) storage.Label {
	now := pq.NullTime{
		Time:  time.Now(),
		Valid: true,
	}
	return &NotificationLabel{
		BaseLabelEntity: BaseLabelEntity{
			ID:        sql.NullString{String: id, Valid: id != ""},
			Key:       sql.NullString{String: key, Valid: key != ""},
			Val:       sql.NullString{String: value, Valid: value != ""},
			CreatedAt: now,
			UpdatedAt: now,
		},
		NotificationID: sql.NullString{String: entityID, Valid: entityID != ""},
	}
}

func (e *Notification) RowsToList(rows *sqlx.Rows) (types.ObjectList, error) {
	rowCreator := func() EntityLabelRow {
		return &struct {
			*Notification
			NotificationLabel `db:"notification_labels"`
		}{}
	}
	result := &types.Notifications{
		Notifications: make([]*types.Notification, 0),
	}
	err := rowsToList(rows, rowCreator, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

type NotificationLabel struct {
	BaseLabelEntity
	NotificationID sql.NullString `db:"notification_id"`
}

func (el NotificationLabel) LabelsTableName() string {
	return "notification_labels"
}

func (el NotificationLabel) ReferenceColumn() string {
	return "notification_id"
}
