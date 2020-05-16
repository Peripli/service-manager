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
package cascadetypes

import (
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/tidwall/gjson"
)

type ServiceBrokerCascade struct {}

func (sb *ServiceBrokerCascade) GetChildrenCriteria() map[types.ObjectType][]query.Criterion {
	plansIDs := gjson.GetBytes(sb.Catalog, `services.#.plans.#.id`)
	serviceOfferingIDs := gjson.GetBytes(sb.Catalog, `services.#.id`)
	return map[types.ObjectType][]query.Criterion{
		types.ServiceInstanceType: {query.ByField(query.InOperator, "service_plan_id", plansIDs.Value().([]string)...)},
		types.ServiceOfferingType: {query.ByField(query.InOperator, "id", serviceOfferingIDs.Value().([]string)...)},
		types.ServicePlanType:     {query.ByField(query.InOperator, "id", plansIDs.Value().([]string)...)},
		types.VisibilityType:      {query.ByField(query.InOperator, "service_plan_id", plansIDs.Value().([]string)...)},
	}
}
