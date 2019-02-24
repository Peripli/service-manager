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
			*ServicePlan `pdDB:"service_plans"`
		}{}

		if err := rows.StructScan(&row); err != nil {
			return nil, err
		}

		if serviceOffering, ok := services[row.ServiceOffering.ID]; !ok {
			serviceOffering = row.ServiceOffering.ToObject().(*types.ServiceOffering)
			serviceOffering.Plans = append(serviceOffering.Plans, row.ServicePlan.ToObject().(*types.ServicePlan))

			services[row.ServiceOffering.ID] = serviceOffering
			result = append(result, serviceOffering)
		} else {
			serviceOffering.Plans = append(serviceOffering.Plans, row.ServicePlan.ToObject().(*types.ServicePlan))
		}
	}

	return result, nil
}
