/*
 * Copyright 2018 The Service Manager Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package catalog

import (
	"context"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/query"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/storage"
)

// Load fetches the catalog of the broker with the given ID from the storage
func Load(ctx context.Context, brokerID string, repository storage.Repository) (*types.ServiceOfferings, error) {
	serviceOfferings, err := repository.List(ctx, types.ServiceOfferingType, query.ByField(query.EqualsOperator, "broker_id", brokerID))
	if err != nil {
		return nil, err
	}
	result := serviceOfferings.(*types.ServiceOfferings)
	if serviceOfferings.Len() == 0 {
		return result, nil
	}
	var serviceOfferingIDs []string
	for _, so := range result.ServiceOfferings {
		serviceOfferingIDs = append(serviceOfferingIDs, so.ID)
	}
	servicePlansForOffering := make(map[string][]*types.ServicePlan)
	if len(serviceOfferingIDs) > 0 {
		servicePlans, err := repository.List(ctx, types.ServicePlanType, query.ByField(query.InOperator, "service_offering_id", serviceOfferingIDs...))
		if err != nil {
			return nil, err
		}
		for i := 0; i < servicePlans.Len(); i++ {
			plan := servicePlans.ItemAt(i).(*types.ServicePlan)
			servicePlansForOffering[plan.ServiceOfferingID] = append(servicePlansForOffering[plan.ServiceOfferingID], plan)
		}
	}
	for _, service := range result.ServiceOfferings {
		service.Plans = servicePlansForOffering[service.ID]
	}
	return result, nil
}
