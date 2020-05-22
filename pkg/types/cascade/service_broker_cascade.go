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
	"context"
	"fmt"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
	"github.com/tidwall/gjson"
)

type ServiceBrokerCascade struct {
	*types.ServiceBroker
}

func (sb *ServiceBrokerCascade) ValidateChildren() func(ctx context.Context, objectChildren []types.ObjectList, repository storage.Repository, labelKeys ...string) error {
	return func(ctx context.Context, objectChildren []types.ObjectList, repository storage.Repository, labelKeys ...string) error {
		platformIdsMap := make(map[string]bool)
		for _, children := range objectChildren {
			for i := 0; i < children.Len(); i++ {
				instance, ok := children.ItemAt(i).(*types.ServiceInstance)
				if !ok {
					return fmt.Errorf("broker %s has children not of type %s", sb.GetID(), types.ServiceInstanceType)
				}
				if _, ok := platformIdsMap[instance.PlatformID]; !ok {
					platformIdsMap[instance.PlatformID] = true
				}
			}
		}
		delete(platformIdsMap, types.SMPlatform)

		platformIds := make([]string, len(platformIdsMap))
		index := 0
		for id, _ := range platformIdsMap {
			platformIds[index] = id
			index++
		}

		platforms, err := repository.List(ctx, types.PlatformType, query.ByField(query.InOperator, "id", platformIds...))
		if err != nil {
			return err
		}
		for i := 0; i < platforms.Len(); i++ {
			platform := platforms.ItemAt(i)
			labels := platform.GetLabels()
			if _, found := labels[labelKeys[0]]; !found {
				return fmt.Errorf("broker %s has instances from global platform", sb.GetID())
			}
		}
		return nil
	}
}

func (sb *ServiceBrokerCascade) GetChildrenCriterion() ChildrenCriterion {
	plansIDs := gjson.GetBytes(sb.Catalog, `services.#.plans.#.id`)
	return ChildrenCriterion{
		types.ServiceInstanceType: {query.ByField(query.InOperator, "service_plan_id", plansIDs.Value().([]string)...)},
	}
}
