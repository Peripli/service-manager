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
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
	"strings"
)

type ServiceInstanceCascade struct {
	*types.ServiceInstance

	// some implementations may implement instance nesting using parent references based on label
	parentInstanceLabelKeys []string
}

func (si *ServiceInstanceCascade) GetChildrenCriterion() ChildrenCriterion {
	criterion := ChildrenCriterion{
		types.ServiceBindingType: {query.ByField(query.EqualsOperator, "service_instance_id", si.ID)},
	}
	if len(si.parentInstanceLabelKeys) > 0 {
		params := storage.SubQueryParams{
			"PARENT_ID":   si.ID,
			"PARENT_KEYS": "'" + strings.Join(si.parentInstanceLabelKeys, "','") + "'",
		}
		subQuery, _ := storage.GetSubQueryWithParams(storage.QueryForInstanceChildrenByLabel, params)
		criterion[types.ServiceInstanceType] = []query.Criterion{query.ByExists(subQuery)}
	}
	return criterion
}
