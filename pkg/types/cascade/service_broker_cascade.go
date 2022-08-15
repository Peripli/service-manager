/*
 * Copyright 2018 The Service Manager Authors
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

// Package types contains the Service Manager web entities
package cascade

import (
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/query"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types"
)

type ServiceBrokerCascade struct {
	*types.ServiceBroker
}

func (sb *ServiceBrokerCascade) GetChildrenCriterion() ChildrenCriterion {
	var planIDs []string
	for _, serviceOffering := range sb.Services {
		for _, servicePlan := range serviceOffering.Plans {
			planIDs = append(planIDs, servicePlan.ID)
		}
	}
	return ChildrenCriterion{
		types.ServiceInstanceType: {{query.ByField(query.InOperator, "service_plan_id", planIDs...)}},
	}
}
