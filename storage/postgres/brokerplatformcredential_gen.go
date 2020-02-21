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

var _ PostgresEntity = &BrokerPlatformCredential{}

const BrokerPlatformCredentialTable = "broker_platform_credentials"

func (*BrokerPlatformCredential) LabelEntity() PostgresLabel {
	return &BrokerPlatformCredentialLabel{}
}

func (*BrokerPlatformCredential) TableName() string {
	return BrokerPlatformCredentialTable
}

func (bpc *BrokerPlatformCredential) NewLabel(id, key, value string) storage.Label {
	now := pq.NullTime{
		Time:  time.Now(),
		Valid: true,
	}
	return &BrokerPlatformCredentialLabel{
		BaseLabelEntity: BaseLabelEntity{
			ID:        sql.NullString{String: id, Valid: id != ""},
			Key:       sql.NullString{String: key, Valid: key != ""},
			Val:       sql.NullString{String: value, Valid: value != ""},
			CreatedAt: now,
			UpdatedAt: now,
		},
		BrokerPlatformCredentialID: sql.NullString{String: bpc.ID, Valid: bpc.ID != ""},
	}
}

func (bpc *BrokerPlatformCredential) RowsToList(rows *sqlx.Rows) (types.ObjectList, error) {
	rowCreator := func() EntityLabelRow {
		return &struct {
			*BrokerPlatformCredential
			BrokerPlatformCredentialLabel `db:"broker_platform_credential_labels"`
		}{}
	}
	result := &types.BrokerPlatformCredentials{
		BrokerPlatformCredentials: make([]*types.BrokerPlatformCredential, 0),
	}
	err := rowsToList(rows, rowCreator, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

type BrokerPlatformCredentialLabel struct {
	BaseLabelEntity
	BrokerPlatformCredentialID sql.NullString `db:"broker_platform_credential_id"`
}

func (el BrokerPlatformCredentialLabel) LabelsTableName() string {
	return "broker_platform_credential_labels"
}

func (el BrokerPlatformCredentialLabel) ReferenceColumn() string {
	return "broker_platform_credential_id"
}
