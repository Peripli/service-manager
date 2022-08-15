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

var _ PostgresEntity = &ServicePlan{}

const ServicePlanTable = "service_plans"

func (*ServicePlan) LabelEntity() PostgresLabel {
	return &ServicePlanLabel{}
}

func (*ServicePlan) TableName() string {
	return ServicePlanTable
}

func (e *ServicePlan) NewLabel(id, entityID, key, value string) storage.Label {
	now := pq.NullTime{
		Time:  time.Now(),
		Valid: true,
	}
	return &ServicePlanLabel{
		BaseLabelEntity: BaseLabelEntity{
			ID:        sql.NullString{String: id, Valid: id != ""},
			Key:       sql.NullString{String: key, Valid: key != ""},
			Val:       sql.NullString{String: value, Valid: value != ""},
			CreatedAt: now,
			UpdatedAt: now,
		},
		ServicePlanID: sql.NullString{String: entityID, Valid: entityID != ""},
	}
}

func (e *ServicePlan) RowsToList(rows *sqlx.Rows) (types.ObjectList, error) {
	rowCreator := func() EntityLabelRow {
		return &struct {
			*ServicePlan
			ServicePlanLabel `db:"service_plan_labels"`
		}{}
	}
	result := &types.ServicePlans{
		ServicePlans: make([]*types.ServicePlan, 0),
	}
	err := rowsToList(rows, rowCreator, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

type ServicePlanLabel struct {
	BaseLabelEntity
	ServicePlanID sql.NullString `db:"service_plan_id"`
}

func (el ServicePlanLabel) LabelsTableName() string {
	return "service_plan_labels"
}

func (el ServicePlanLabel) ReferenceColumn() string {
	return "service_plan_id"
}
