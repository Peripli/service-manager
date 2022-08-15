package service_plans

import (
	"context"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/query"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/util/slice"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/storage"
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

func ResolveSupportedPlatformsForTenant(ctx context.Context, plans []*types.ServicePlan, repository storage.Repository, tenantKey string, tenantValue string) (map[string]*types.Platform, error) {
	isGlobal := func(platform *types.Platform) bool {
		return platform.Labels == nil || len(platform.Labels[tenantKey]) == 0
	}

	isTenantScoped := func(platform *types.Platform) bool {
		return platform.Labels != nil && len(platform.Labels[tenantKey]) > 0 && platform.Labels[tenantKey][0] == tenantValue
	}

	platforms, err := ResolveSupportedPlatformsForPlans(ctx, plans, repository)
	if err != nil {
		return nil, err
	}

	platformsForTenant := make(map[string]*types.Platform)
	for _, platform := range platforms {
		if isGlobal(platform) || isTenantScoped(platform) {
			platformsForTenant[platform.ID] = platform
		}
	}
	return platformsForTenant, nil
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
				if slice.StringsAnyEquals(types.GetSMSupportedPlatformTypes(), platformType) {
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
