package postgres

import (
	"context"
	"fmt"

	"github.com/Peripli/service-manager/pkg/log"

	"github.com/Peripli/service-manager/pkg/types"
)

type servicePlanStorage struct {
	db pgDB
}

func (sps *servicePlanStorage) Create(ctx context.Context, servicePlan *types.ServicePlan) error {
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
	err := list(ctx, sps.db, servicePlanTable, map[string]string{}, &plans)
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
	err := list(ctx, sps.db, servicePlanTable, map[string]string{"catalog_name": name}, &plans)
	if err != nil || len(plans) == 0 {
		return []*types.ServicePlan{}, err
	}
	servicePlans := make([]*types.ServicePlan, 0, len(plans))
	for _, plan := range plans {
		servicePlans = append(servicePlans, plan.ToDTO())
	}
	return servicePlans, nil
}

func (sps *servicePlanStorage) ListByBrokerID(ctx context.Context, brokerID string) ([]*types.ServicePlan, error) {
	var plans []ServicePlan
	query := fmt.Sprintf(`
	SELECT * 
	FROM %[1]s JOIN %[2]s on %[1]s.service_id=%[2]s.id 
	WHERE %[2]s.broker_id=$1`,
		servicePlanTable, serviceOfferingTable)

	log.C(ctx).Debugf("Executing query %s", query)

	err := sps.db.SelectContext(ctx, &plans, query, brokerID)
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
	return delete(ctx, sps.db, id, servicePlanTable)
}

func (sps *servicePlanStorage) Update(ctx context.Context, servicePlan *types.ServicePlan) error {
	plan := &ServicePlan{}
	plan.FromDTO(servicePlan)
	return update(ctx, sps.db, servicePlanTable, plan)
}
