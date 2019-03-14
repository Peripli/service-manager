/*
 *    Copyright 2018 The Service Manager Authors
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */

package postgres

import (
	"context"
	"fmt"

	"github.com/Peripli/service-manager/storage"
	sqlxtypes "github.com/jmoiron/sqlx/types"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/types"
)

//go:generate smgen storage ServiceOffering github.com/Peripli/service-manager/pkg/types
type ServiceOffering struct {
	BaseEntity
	Name        string `db:"name"`
	Description string `db:"description"`

	Bindable             bool   `db:"bindable"`
	InstancesRetrievable bool   `db:"instances_retrievable"`
	BindingsRetrievable  bool   `db:"bindings_retrievable"`
	PlanUpdatable        bool   `db:"plan_updateable"`
	CatalogID            string `db:"catalog_id"`
	CatalogName          string `db:"catalog_name"`

	Tags     sqlxtypes.JSONText `db:"tags"`
	Requires sqlxtypes.JSONText `db:"requires"`
	Metadata sqlxtypes.JSONText `db:"metadata"`

	BrokerID string `db:"broker_id"`
}

func (so *ServiceOffering) ToObject() types.Object {
	return &types.ServiceOffering{
		Base: types.Base{
			ID:        so.ID,
			CreatedAt: so.CreatedAt,
			UpdatedAt: so.UpdatedAt,
		},
		Name:                 so.Name,
		Description:          so.Description,
		Bindable:             so.Bindable,
		InstancesRetrievable: so.InstancesRetrievable,
		BindingsRetrievable:  so.BindingsRetrievable,
		PlanUpdatable:        so.PlanUpdatable,
		CatalogID:            so.CatalogID,
		CatalogName:          so.CatalogName,
		Tags:                 getJSONRawMessage(so.Tags),
		Requires:             getJSONRawMessage(so.Requires),
		Metadata:             getJSONRawMessage(so.Metadata),
		BrokerID:             so.BrokerID,
	}
}

func (*ServiceOffering) FromObject(object types.Object) (storage.Entity, bool) {
	offering, ok := object.(*types.ServiceOffering)
	if !ok {
		return nil, false
	}
	result := &ServiceOffering{
		BaseEntity: BaseEntity{
			ID:        offering.ID,
			CreatedAt: offering.CreatedAt,
			UpdatedAt: offering.UpdatedAt,
		},
		Name:                 offering.Name,
		Description:          offering.Description,
		Bindable:             offering.Bindable,
		InstancesRetrievable: offering.InstancesRetrievable,
		BindingsRetrievable:  offering.BindingsRetrievable,
		PlanUpdatable:        offering.PlanUpdatable,
		CatalogID:            offering.CatalogID,
		CatalogName:          offering.CatalogName,
		Tags:                 getJSONText(offering.Tags),
		Requires:             getJSONText(offering.Requires),
		Metadata:             getJSONText(offering.Metadata),
		BrokerID:             offering.BrokerID,
	}
	return result, true
}

type serviceOfferingStorage struct {
	db     pgDB
	scheme *storage.Scheme
}

func (sos *serviceOfferingStorage) ListWithServicePlansByBrokerID(ctx context.Context, brokerID string) ([]*types.ServiceOffering, error) {
	query := fmt.Sprintf(`SELECT 
		%[1]s.*,
		%[2]s.id "%[2]s.id",
		%[2]s.name "%[2]s.name",
		%[2]s.description "%[2]s.description",
		%[2]s.created_at "%[2]s.created_at",
		%[2]s.updated_at "%[2]s.updated_at",
		%[2]s.free "%[2]s.free",
		%[2]s.bindable "%[2]s.bindable",
		%[2]s.plan_updateable "%[2]s.plan_updateable",
		%[2]s.catalog_id "%[2]s.catalog_id",
		%[2]s.catalog_name "%[2]s.catalog_name",
		%[2]s.metadata "%[2]s.metadata",
		%[2]s.schemas "%[2]s.schemas",
		%[2]s.service_offering_id "%[2]s.service_offering_id"
	FROM %[1]s 
	JOIN %[2]s ON %[1]s.id = %[2]s.service_offering_id
	WHERE %[1]s.broker_id=$1;`, ServiceOfferingTable, ServicePlanTable)

	log.C(ctx).Debugf("Executing query %s", query)
	rows, err := sos.db.QueryxContext(ctx, query, brokerID)

	defer func() {
		if err := rows.Close(); err != nil {
			log.C(ctx).Errorf("Could not release connection when checking database s. Error: %s", err)
		}
	}()
	if err != nil {
		return nil, checkSQLNoRows(err)
	}

	services := make(map[string]*types.ServiceOffering)
	result := make([]*types.ServiceOffering, 0)

	for rows.Next() {
		row := struct {
			*ServiceOffering
			*ServicePlan `db:"service_plans"`
		}{}

		if err := rows.StructScan(&row); err != nil {
			return nil, err
		}

		if serviceOffering, ok := services[row.ServiceOffering.ID]; !ok {
			object := row.ServiceOffering.ToObject()
			serviceOffering = object.(*types.ServiceOffering)
			serviceOffering.Plans = append(serviceOffering.Plans, row.ServicePlan.ToObject().(*types.ServicePlan))

			services[row.ServiceOffering.ID] = serviceOffering
			result = append(result, serviceOffering)
		} else {
			serviceOffering.Plans = append(serviceOffering.Plans, row.ServicePlan.ToObject().(*types.ServicePlan))
		}
	}

	return result, nil
}
