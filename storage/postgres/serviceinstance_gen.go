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

var _ PostgresEntity = &ServiceInstance{}

const ServiceInstanceTable = "service_instances"

func (*ServiceInstance) LabelEntity() PostgresLabel {
	return &ServiceInstanceLabel{}
}

func (*ServiceInstance) TableName() string {
	return ServiceInstanceTable
}

func (e *ServiceInstance) NewLabel(id, entityID, key, value string) storage.Label {
	now := pq.NullTime{
		Time:  time.Now(),
		Valid: true,
	}
	return &ServiceInstanceLabel{
		BaseLabelEntity: BaseLabelEntity{
			ID:        sql.NullString{String: id, Valid: id != ""},
			Key:       sql.NullString{String: key, Valid: key != ""},
			Val:       sql.NullString{String: value, Valid: value != ""},
			CreatedAt: now,
			UpdatedAt: now,
		},
		ServiceInstanceID: sql.NullString{String: entityID, Valid: entityID != ""},
	}
}

func (e *ServiceInstance) RowsToList(rows *sqlx.Rows) (types.ObjectList, error) {
	rowCreator := func() EntityLabelRow {
		return &struct {
			*ServiceInstance
			ServiceInstanceLabel `db:"service_instance_labels"`
		}{}
	}
	result := &types.ServiceInstances{
		ServiceInstances: make([]*types.ServiceInstance, 0),
	}
	err := rowsToList(rows, rowCreator, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

type ServiceInstanceLabel struct {
	BaseLabelEntity
	ServiceInstanceID sql.NullString `db:"service_instance_id"`
}

func (el ServiceInstanceLabel) LabelsTableName() string {
	return "service_instance_labels"
}

func (el ServiceInstanceLabel) ReferenceColumn() string {
	return "service_instance_id"
}
