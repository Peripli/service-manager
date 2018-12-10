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

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/types"
)

type serviceOfferingStorage struct {
	db pgDB
}

func (sos *serviceOfferingStorage) Create(ctx context.Context, serviceOffering *types.ServiceOffering) (string, error) {
	so := &ServiceOffering{}
	so.FromDTO(serviceOffering)
	return create(ctx, sos.db, serviceOfferingTable, so)
}

func (sos *serviceOfferingStorage) Get(ctx context.Context, id string) (*types.ServiceOffering, error) {
	serviceOffering := &ServiceOffering{}
	if err := get(ctx, sos.db, id, serviceOfferingTable, serviceOffering); err != nil {
		return nil, err
	}
	return serviceOffering.ToDTO(), nil
}

func (sos *serviceOfferingStorage) List(ctx context.Context) ([]*types.ServiceOffering, error) {
	var serviceOfferings []ServiceOffering
	err := list(ctx, sos.db, serviceOfferingTable, map[string][]string{}, &serviceOfferings)
	if err != nil || len(serviceOfferings) == 0 {
		return []*types.ServiceOffering{}, err
	}
	serviceOfferingDTOs := make([]*types.ServiceOffering, 0, len(serviceOfferings))
	for _, so := range serviceOfferings {
		serviceOfferingDTOs = append(serviceOfferingDTOs, so.ToDTO())
	}
	return serviceOfferingDTOs, nil
}

func (sos *serviceOfferingStorage) ListByCatalogName(ctx context.Context, name string) ([]*types.ServiceOffering, error) {
	var serviceOfferings []ServiceOffering
	err := list(ctx, sos.db, serviceOfferingTable, map[string][]string{"catalog_name": {name}}, &serviceOfferings)
	if err != nil || len(serviceOfferings) == 0 {
		return []*types.ServiceOffering{}, err
	}
	serviceOfferingDTOs := make([]*types.ServiceOffering, 0, len(serviceOfferings))
	for _, so := range serviceOfferings {
		serviceOfferingDTOs = append(serviceOfferingDTOs, so.ToDTO())
	}
	return serviceOfferingDTOs, nil
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
	WHERE %[1]s.broker_id=$1;`, serviceOfferingTable, servicePlanTable)

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
			serviceOffering = row.ServiceOffering.ToDTO()
			serviceOffering.Plans = append(serviceOffering.Plans, row.ServicePlan.ToDTO())

			services[row.ServiceOffering.ID] = serviceOffering
			result = append(result, serviceOffering)
		} else {
			serviceOffering.Plans = append(serviceOffering.Plans, row.ServicePlan.ToDTO())
		}
	}

	return result, nil
}

func (sos *serviceOfferingStorage) Delete(ctx context.Context, id string) error {
	return remove(ctx, sos.db, id, serviceOfferingTable)
}

func (sos *serviceOfferingStorage) Update(ctx context.Context, serviceOffering *types.ServiceOffering) error {
	so := &ServiceOffering{}
	so.FromDTO(serviceOffering)
	return update(ctx, sos.db, serviceOfferingTable, so)

}
