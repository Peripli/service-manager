package common

import (
	"context"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
)

func ResolveSupportedPlatformIDsForPlan(ctx context.Context, plan *types.ServicePlan, repository storage.Repository) ([]string, error) {
	var platformTypes map[string]bool
	platformIDs := make(map[string]bool)
	planSupportedPlatformIDs := plan.SupportedPlatformIDs()
	if len(planSupportedPlatformIDs) == 0 {
		// no explicit supported platform IDs defined - collect the supported platform types

		if platformTypes == nil {
			//only initialize this map if any plan not specifying explicit platform IDs is found
			platformTypes = make(map[string]bool)
		}

		types := plan.SupportedPlatformTypes()
		for _, t := range types {
			platformTypes[t] = true
		}
	} else {
		// explicit platform IDs are defined for the plan
		for _, id := range planSupportedPlatformIDs {
			platformIDs[id] = true
		}
	}

	if platformTypes != nil {
		if len(platformTypes) == 0 && len(platformIDs) == 0 {
			// plan available on all platforms
			return nil, nil
		}

		// fetch IDs of platform instances of the supported types from DB
		supportedPlatforms := make([]string, 0)
		for platform := range platformTypes {
			supportedPlatforms = append(supportedPlatforms, platform)
		}

		criteria := []query.Criterion{
			query.ByField(query.NotEqualsOperator, "type", types.SMPlatform),
		}

		if len(supportedPlatforms) != 0 {
			criteria = append(criteria, query.ByField(query.InOperator, "type", supportedPlatforms...))
		}

		objList, err := repository.List(ctx, types.PlatformType, criteria...)
		if err != nil {
			return nil, err
		}

		for i := 0; i < objList.Len(); i++ {
			platformIDs[objList.ItemAt(i).GetID()] = true
		}
	}

	supportedPlatformIDs := make([]string, 0)
	for id := range platformIDs {
		supportedPlatformIDs = append(supportedPlatformIDs, id)
	}

	return supportedPlatformIDs, nil
}
