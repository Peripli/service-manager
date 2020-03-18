package common

import (
	"context"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
	"github.com/Peripli/service-manager/storage/interceptors"
)

func ResolveSupportedPlatformIDsForPlan(ctx context.Context, plan *types.ServicePlan, repository storage.Repository) ([]string, error) {
	resolvedPlatformIDs, err := interceptors.ResolveSupportedPlatformIDsForPlans(ctx, []*types.ServicePlan{plan}, repository)
	if err != nil {
		return nil, err
	}

	if resolvedPlatformIDs != nil {
		return resolvedPlatformIDs, nil
	}

	// all platforms supported - load them from DB
	criteria := []query.Criterion{
		query.ByField(query.NotEqualsOperator, "type", types.SMPlatform),
	}

	objList, err := repository.List(ctx, types.PlatformType, criteria...)
	if err != nil {
		return nil, err
	}

	platformIDs := make(map[string]bool)
	for i := 0; i < objList.Len(); i++ {
		platformIDs[objList.ItemAt(i).GetID()] = true
	}

	supportedPlatformIDs := make([]string, 0)
	for id := range platformIDs {
		supportedPlatformIDs = append(supportedPlatformIDs, id)
	}

	return supportedPlatformIDs, nil
}
