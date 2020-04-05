package service_plans

import (
	"context"
	"fmt"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
)

func ResolveSupportedPlatformIDsForPlans(ctx context.Context, plans []*types.ServicePlan, repository storage.Repository) ([]string, error) {
	var platformTypes map[string]bool
	platformNames := make(map[string]bool)
	allPlatformsSupported := false
	for _, plan := range plans {
		if plan.SupportsAllPlatforms() {
			// all platforms are supported by one of the plan, no need for further processing
			allPlatformsSupported = true
			break
		}

		planSupportedPlatformNames := plan.SupportedPlatformNames()
		if len(planSupportedPlatformNames) == 0 {
			// no explicit supported platform names defined - collect the supported platform types
			if platformTypes == nil {
				platformTypes = make(map[string]bool)
			}

			supportedPlatformTypes := plan.SupportedPlatformTypes()
			for _, t := range supportedPlatformTypes {
				platformTypes[t] = true
			}
		} else {
			// explicit platform names are defined for the plan
			for _, name := range planSupportedPlatformNames {
				platformNames[name] = true
			}
		}
	}

	platformIDs := make(map[string]bool)
	var criteria []query.Criterion

	if !allPlatformsSupported {
		if len(platformNames) != 0 {
			// fetch IDs of platform instances with the requested names
			supportedPlatformNames := make([]string, 0)
			for name := range platformNames {
				supportedPlatformNames = append(supportedPlatformNames, name)
			}
			err := addIDsOfSupportedPlatformNames(ctx, repository, supportedPlatformNames, platformIDs)
			if err != nil {
				return nil, err
			}
		}

		//add a criteria for the supported types
		supportedPlatformTypes := make([]string, 0)
		for platform := range platformTypes {
			supportedPlatformTypes = append(supportedPlatformTypes, platform)
		}

		if len(supportedPlatformTypes) != 0 {
			criteria = []query.Criterion{query.ByField(query.InOperator, "type", supportedPlatformTypes...)}
		}
	}

	if allPlatformsSupported || criteria != nil {
		// fetch IDs of platform instances of the supported types from DB
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

func addIDsOfSupportedPlatformNames(ctx context.Context, repository storage.Repository, supportedPlatformNames []string, platformIDs map[string]bool) error {
	var criteria []query.Criterion
	if len(supportedPlatformNames) != 0 {
		criteria = []query.Criterion{query.ByField(query.InOperator, "name", supportedPlatformNames...)}
	} else {
		return fmt.Errorf("supportedPlatformNames must be a non empty array")
	}

	objList, err := repository.List(ctx, types.PlatformType, criteria...)
	if err != nil {
		return err
	}

	for i := 0; i < objList.Len(); i++ {
		platformIDs[objList.ItemAt(i).GetID()] = true
	}
	return nil
}
