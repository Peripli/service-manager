
// GENERATED. DO NOT MODIFY!

package postgres

import (
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
	"github.com/jmoiron/sqlx"
	
	
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

func (e *ServiceOffering) NewLabel(id, key, value string) storage.Label {
	now := time.Now()
	return &ServiceOfferingLabel{
		BaseLabelEntity: BaseLabelEntity{
			ID:        sql.NullString{String: id, Valid: id != ""},
			Key:       sql.NullString{String: key, Valid: key != ""},
			Val:       sql.NullString{String: value, Valid: value != ""},
			CreatedAt: &now,
			UpdatedAt: &now,
		},
		ServiceOfferingID:  sql.NullString{String: e.ID, Valid: e.ID != ""},
	}
}

func (e *ServiceOffering) RowsToList(rows *sqlx.Rows) (types.ObjectList, error) {
	row := struct {
		*ServiceOffering
		ServiceOfferingLabel `db:"service_offering_labels"`
	}{}
	result := &types.ServiceOfferings{
		ServiceOfferings: make([]*types.ServiceOffering, 0),
	}		
	err := rowsToList(rows, &row, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

type ServiceOfferingLabel struct {
	BaseLabelEntity
	ServiceOfferingID  sql.NullString `db:"service_offering_id"`
}

func (el ServiceOfferingLabel) LabelsTableName() string {
	return "service_offering_labels"
}

func (el ServiceOfferingLabel) ReferenceColumn() string {
	return "service_offering_id"
}
