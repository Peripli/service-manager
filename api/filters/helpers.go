package filters

import (
	"context"

	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
)

func servicesCriteria(ctx context.Context, repository storage.Repository, planQuery *query.Criterion) (*query.Criterion, error) {
	objectList, err := repository.List(ctx, types.ServicePlanType, *planQuery)
	if err != nil {
		return nil, err
	}
	plans := objectList.(*types.ServicePlans)
	if plans.Len() < 1 {
		return nil, nil
	}
	serviceIDs := make([]string, 0, plans.Len())
	for _, p := range plans.ServicePlans {
		serviceIDs = append(serviceIDs, p.ServiceOfferingID)
	}
	c := query.ByField(query.InOperator, "id", serviceIDs...)
	return &c, nil
}

func plansCriteria(ctx context.Context, repository storage.Repository, platformID string) (*query.Criterion, error) {
	objectList, err := repository.List(ctx, types.VisibilityType, query.ByField(query.EqualsOrNilOperator, "platform_id", platformID))
	if err != nil {
		return nil, err
	}
	visibilityList := objectList.(*types.Visibilities)
	if visibilityList.Len() < 1 {
		return nil, nil
	}
	planIDs := make([]string, 0, visibilityList.Len())
	for _, vis := range visibilityList.Visibilities {
		planIDs = append(planIDs, vis.ServicePlanID)
	}
	c := query.ByField(query.InOperator, "id", planIDs...)
	return &c, nil
}
