package postgres

import (
	"database/sql"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"time"
)

var _ PostgresEntity = &BlockedClient{}

const BlockedClientTable = "blocked_clients"

func (*BlockedClient) LabelEntity() PostgresLabel {
	return &BlockClientLabel{}
}

func (*BlockedClient) TableName() string {
	return BlockedClientTable
}

func (*BlockedClient) NewLabel(id, entityID, key, value string) storage.Label {
	now := pq.NullTime{
		Time:  time.Now(),
		Valid: true,
	}
	return &BlockClientLabel{
		BaseLabelEntity: BaseLabelEntity{
			ID:        sql.NullString{String: id, Valid: id != ""},
			Key:       sql.NullString{String: key, Valid: key != ""},
			Val:       sql.NullString{String: value, Valid: value != ""},
			CreatedAt: now,
			UpdatedAt: now,
		},
		BlockedClientID: sql.NullString{String: entityID, Valid: entityID != ""},
	}
}

func (e *BlockedClient) RowsToList(rows *sqlx.Rows) (types.ObjectList, error) {
	rowCreator := func() EntityLabelRow {
		return &struct {
			*BlockedClient
			BlockClientLabel `db:"blocked_clients_labels"`
		}{}
	}
	result := &types.BlockedClients{
		BlockedClients: make([]*types.BlockedClient, 0),
	}
	err := rowsToList(rows, rowCreator, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

type BlockClientLabel struct {
	BaseLabelEntity
	BlockedClientID sql.NullString `db:"blocked_client_id"`
}

func (el BlockClientLabel) LabelsTableName() string {
	return "blocked_clients_labels"
}

func (el BlockClientLabel) ReferenceColumn() string {
	return "blocked_client_id"
}
