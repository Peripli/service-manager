package service_plans

import (
	"context"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
)

func ResolveSupportedPlatformIDsForPlans(ctx context.Context, plans []*types.ServicePlan, repository storage.Repository) ([]string, error) {
	platformIDsSet := make(map[string]bool)

	for _, plan := range plans {
		if err := addSupportedPlatformIDsForPlan(ctx, plan, repository, platformIDsSet); err != nil {
			return nil, err
		}
	}

	platformIDs := make([]string, 0)
	for id := range platformIDsSet {
		platformIDs = append(platformIDs, id)
	}
	return platformIDs, nil
}

func addSupportedPlatformIDsForPlan(ctx context.Context, plan *types.ServicePlan, repository storage.Repository, platformIDs map[string]bool) error {
	criterions := make([]query.Criterion, 0)
	if excludedPlatformNames := plan.ExcludedPlatformNames(); len(excludedPlatformNames) > 0 {
		// plan explicitly defined excluded platforms, all other platforms are supported
		criterions = append(criterions, query.ByField(query.NotInOperator, "name", excludedPlatformNames...))
	} else if planSupportedPlatformNames := plan.SupportedPlatformNames(); len(planSupportedPlatformNames) > 0 {
		// plan explicitly defined supported platform names
		criterions = append(criterions, query.ByField(query.InOperator, "name", planSupportedPlatformNames...))
	} else if planSupportedPlatformTypes := plan.SupportedPlatformTypes(); len(planSupportedPlatformTypes) > 0 {
		// plan explicitly defined supported platform types
		criterions = append(criterions, query.ByField(query.InOperator, "type", planSupportedPlatformTypes...))
	}

	// fetch IDs of platform instances of the supported types from DB
	objList, err := repository.List(ctx, types.PlatformType, criterions...)
	if err != nil {
		return err
	}

	for i := 0; i < objList.Len(); i++ {
		platformIDs[objList.ItemAt(i).GetID()] = true
	}

	return nil
}
