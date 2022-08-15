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

var _ PostgresEntity = &ServiceOffering{}

const ServiceOfferingTable = "service_offerings"

func (*ServiceOffering) LabelEntity() PostgresLabel {
	return &ServiceOfferingLabel{}
}

func (*ServiceOffering) TableName() string {
	return ServiceOfferingTable
}

func (e *ServiceOffering) NewLabel(id, entityID, key, value string) storage.Label {
	now := pq.NullTime{
		Time:  time.Now(),
		Valid: true,
	}
	return &ServiceOfferingLabel{
		BaseLabelEntity: BaseLabelEntity{
			ID:        sql.NullString{String: id, Valid: id != ""},
			Key:       sql.NullString{String: key, Valid: key != ""},
			Val:       sql.NullString{String: value, Valid: value != ""},
			CreatedAt: now,
			UpdatedAt: now,
		},
		ServiceOfferingID: sql.NullString{String: entityID, Valid: entityID != ""},
	}
}

func (e *ServiceOffering) RowsToList(rows *sqlx.Rows) (types.ObjectList, error) {
	rowCreator := func() EntityLabelRow {
		return &struct {
			*ServiceOffering
			ServiceOfferingLabel `db:"service_offering_labels"`
		}{}
	}
	result := &types.ServiceOfferings{
		ServiceOfferings: make([]*types.ServiceOffering, 0),
	}
	err := rowsToList(rows, rowCreator, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

type ServiceOfferingLabel struct {
	BaseLabelEntity
	ServiceOfferingID sql.NullString `db:"service_offering_id"`
}

func (el ServiceOfferingLabel) LabelsTableName() string {
	return "service_offering_labels"
}

func (el ServiceOfferingLabel) ReferenceColumn() string {
	return "service_offering_id"
}
