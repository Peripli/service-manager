package instance_sharing

import (
	"context"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
)

func GetObjectByField(ctx context.Context, repository storage.Repository, objectType types.ObjectType, byKey, byValue string) (types.Object, error) {
	byID := query.ByField(query.EqualsOperator, byKey, byValue)
	dbObject, err := repository.Get(ctx, objectType, byID)
	if err != nil {
		return nil, err
	}
	return dbObject, nil
}

func IsReferencePlan(ctx context.Context, repository storage.TransactionalRepository, byKey, servicePlanID string) (bool, error) {
	dbPlanObject, err := GetObjectByField(ctx, repository, types.ServicePlanType, byKey, servicePlanID)
	if err != nil {
		return false, err
	}
	plan := dbPlanObject.(*types.ServicePlan)
	return plan.Name == ReferencePlanName, nil
}
