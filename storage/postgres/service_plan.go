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

	"github.com/Peripli/service-manager/pkg/types"
)

type servicePlanStorage struct {
	db pgDB
}

func (sps *servicePlanStorage) Create(ctx context.Context, servicePlan *types.ServicePlan) (string, error) {
	plan := &ServicePlan{}
	plan.FromDTO(servicePlan)
	return create(ctx, sps.db, servicePlanTable, plan)
}

func (sps *servicePlanStorage) Get(ctx context.Context, id string) (*types.ServicePlan, error) {
	plan := &ServicePlan{}
	if err := get(ctx, sps.db, id, servicePlanTable, plan); err != nil {
		return nil, err
	}
	return plan.ToDTO(), nil
}

func (sps *servicePlanStorage) List(ctx context.Context) ([]*types.ServicePlan, error) {
	var plans []ServicePlan
	err := list(ctx, sps.db, servicePlanTable, map[string][]string{}, &plans)
	if err != nil || len(plans) == 0 {
		return []*types.ServicePlan{}, err
	}
	servicePlans := make([]*types.ServicePlan, 0, len(plans))
	for _, plan := range plans {
		servicePlans = append(servicePlans, plan.ToDTO())
	}
	return servicePlans, nil
}

func (sps *servicePlanStorage) ListByCatalogName(ctx context.Context, name string) ([]*types.ServicePlan, error) {
	var plans []ServicePlan
	err := list(ctx, sps.db, servicePlanTable, map[string][]string{"catalog_name": {name}}, &plans)
	if err != nil || len(plans) == 0 {
		return []*types.ServicePlan{}, err
	}
	servicePlans := make([]*types.ServicePlan, 0, len(plans))
	for _, plan := range plans {
		servicePlans = append(servicePlans, plan.ToDTO())
	}
	return servicePlans, nil
}

func (sps *servicePlanStorage) Delete(ctx context.Context, id string) error {
	return remove(ctx, sps.db, id, servicePlanTable)
}

func (sps *servicePlanStorage) Update(ctx context.Context, servicePlan *types.ServicePlan) error {
	plan := &ServicePlan{}
	plan.FromDTO(servicePlan)
	return update(ctx, sps.db, servicePlanTable, plan)
}
