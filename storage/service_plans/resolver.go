package service_plans

import (
	"context"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
)

func ResolveSupportedPlatformIDsForPlans(ctx context.Context, plans []*types.ServicePlan, repository storage.Repository) ([]string, error) {
	platforms, err := ResolveSupportedPlatformsForPlans(ctx, plans, repository)
	if err != nil {
		return nil, err
	}

	platformIDs := make([]string, 0)
	for id := range platforms {
		platformIDs = append(platformIDs, id)
	}
	return platformIDs, nil
}


func ResolveSupportedPlatformsForPlans(ctx context.Context, plans []*types.ServicePlan, repository storage.Repository) (map[string]*types.Platform, error) {
	platformsMap := make(map[string]*types.Platform)

	for _, plan := range plans {
		allPlatformsSupported := false
		criteria := make([]query.Criterion, 0)
		if excludedPlatformNames := plan.ExcludedPlatformNames(); len(excludedPlatformNames) > 0 {
			// plan explicitly defined excluded platforms, all other platforms are supported
			criteria = append(criteria, query.ByField(query.NotInOperator, "name", excludedPlatformNames...))
		} else if planSupportedPlatformNames := plan.SupportedPlatformNames(); len(planSupportedPlatformNames) > 0 {
			// plan explicitly defined supported platform names
			criteria = append(criteria, query.ByField(query.InOperator, "name", planSupportedPlatformNames...))
		} else if planSupportedPlatformTypes := plan.SupportedPlatformTypes(); len(planSupportedPlatformTypes) > 0 {
			// plan explicitly defined supported platform types
			supportedPlatformTypes := make([]string, 0)
			for _, platformType := range planSupportedPlatformTypes {
				if platformType == types.GetSMSupportedPlatformType() {
					platformType = types.SMPlatform
				}
				supportedPlatformTypes = append(supportedPlatformTypes, platformType)
			}
			criteria = append(criteria, query.ByField(query.InOperator, "type", supportedPlatformTypes...))
		} else {
			allPlatformsSupported = true
		}

		// fetch IDs of platform instances of the supported types from DB
		objList, err := repository.List(ctx, types.PlatformType, criteria...)
		if err != nil {
			return nil, err
		}

		for i := 0; i < objList.Len(); i++ {
			platformsMap[objList.ItemAt(i).GetID()] = objList.ItemAt(i).(*types.Platform)
		}

		if allPlatformsSupported {
			// all platform IDs already included, no need to process additional plans
			break
		}
	}

	return platformsMap, nil
}
