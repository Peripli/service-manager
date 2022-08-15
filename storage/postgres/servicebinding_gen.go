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

var _ PostgresEntity = &ServiceBinding{}

const ServiceBindingTable = "service_bindings"

func (*ServiceBinding) LabelEntity() PostgresLabel {
	return &ServiceBindingLabel{}
}

func (*ServiceBinding) TableName() string {
	return ServiceBindingTable
}

func (e *ServiceBinding) NewLabel(id, entityID, key, value string) storage.Label {
	now := pq.NullTime{
		Time:  time.Now(),
		Valid: true,
	}
	return &ServiceBindingLabel{
		BaseLabelEntity: BaseLabelEntity{
			ID:        sql.NullString{String: id, Valid: id != ""},
			Key:       sql.NullString{String: key, Valid: key != ""},
			Val:       sql.NullString{String: value, Valid: value != ""},
			CreatedAt: now,
			UpdatedAt: now,
		},
		ServiceBindingID: sql.NullString{String: entityID, Valid: entityID != ""},
	}
}

func (e *ServiceBinding) RowsToList(rows *sqlx.Rows) (types.ObjectList, error) {
	rowCreator := func() EntityLabelRow {
		return &struct {
			*ServiceBinding
			ServiceBindingLabel `db:"service_binding_labels"`
		}{}
	}
	result := &types.ServiceBindings{
		ServiceBindings: make([]*types.ServiceBinding, 0),
	}
	err := rowsToList(rows, rowCreator, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

type ServiceBindingLabel struct {
	BaseLabelEntity
	ServiceBindingID sql.NullString `db:"service_binding_id"`
}

func (el ServiceBindingLabel) LabelsTableName() string {
	return "service_binding_labels"
}

func (el ServiceBindingLabel) ReferenceColumn() string {
	return "service_binding_id"
}
