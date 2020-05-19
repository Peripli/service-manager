package operations

import (
	"context"
	"fmt"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/types/cascade"
	"github.com/Peripli/service-manager/storage"
	"github.com/gofrs/uuid"
	"time"
)

func GetAllLevelsCascadeOperations(ctx context.Context, parentObject types.Object, parentOperation *types.Operation, rootId string, storage storage.Repository) ([]*types.Operation, error) {
	var operations []*types.Operation
	objectChildren, err := getObjectChildren(ctx, parentObject, storage)
	if err != nil {
		return nil, err
	}
	err = validateNoGlobalPlatformInstances(ctx, parentObject, objectChildren, storage)
	if err != nil {
		return nil, err
	}

	for _, children := range objectChildren {
		for i := 0; i < children.Len(); i++ {
			child := children.ItemAt(i)
			childOperation, err := makeCascadeOPForChild(child, parentOperation, rootId)
			if err != nil {
				return nil, err
			}
			operations = append(operations, childOperation)
			grandChildrenOperations, err := GetAllLevelsCascadeOperations(ctx, child, childOperation, rootId, storage)
			if err != nil {
				return nil, err
			}
			operations = append(operations, grandChildrenOperations...)
		}
	}
	return operations, nil
}

func validateNoGlobalPlatformInstances(ctx context.Context, parent types.Object, objectChildren []types.ObjectList, repository storage.Repository) error {
	if parent.GetType() != types.ServiceBrokerType {
		return nil
	}

	platformIdsMap := make(map[string]bool)
	for _, children := range objectChildren {
		for i := 0; i < children.Len(); i++ {
			instance, ok := children.ItemAt(i).(*types.ServiceInstance)
			if !ok {
				return fmt.Errorf("broker %s has children not of type %s", parent.GetID(), types.ServiceInstanceType)
			}
			if _, ok := platformIdsMap[instance.PlatformID]; !ok {
				platformIdsMap[instance.PlatformID] = true
			}
		}
	}

	platformIds := make([]string, 0, len(platformIdsMap))
	for id := range platformIdsMap {
		platformIds = append(platformIds, id)
	}

	platforms, err := repository.List(ctx, types.PlatformType, query.ByField(query.InOperator, "id", platformIds...))
	if err != nil {
		return err
	}

	for i := 0; i < platforms.Len(); i++ {
		platform := platforms.ItemAt(i)
		if len(platform.GetLabels()) == 0 {
			return fmt.Errorf("broker %s has instances from global platform", parent.GetID())
		}
	}
	return nil
}

func getObjectChildren(ctx context.Context, object types.Object, storage storage.Repository) ([]types.ObjectList, error) {
	var children []types.ObjectList
	if childrenCriterions, isCascade := cascade.GetCascadeObject(ctx, object); isCascade {
		for childType, childCriteria := range childrenCriterions.GetChildrenCriterion() {
			list, err := storage.List(ctx, childType, childCriteria...)
			if err != nil {
				return nil, err
			}
			children = append(children, list)
		}
	}
	return children, nil
}

type CascadedOperations struct {
	AllOperationsCount           int
	FailedOperations     []*types.Operation
	InProgressOperations []*types.Operation
	SucceededOperations  []*types.Operation
	NotStartedOperations  []*types.Operation
}

func GetSubOperations(ctx context.Context, operation *types.Operation, repository storage.Repository) (*CascadedOperations, error) {
	objs, err := repository.List(ctx, types.OperationType, query.ByField(query.EqualsOperator, "parent", operation.ID))
	suboperations := objs.(*types.Operations)
	cascadedOperations := &CascadedOperations{}
	cascadedOperations.AllOperationsCount = len(suboperations.Operations)
	if err != nil {
		return nil, err
	}
	for i := 0; i < suboperations.Len(); i++ {
		suboperation := suboperations.ItemAt(i).(*types.Operation)
		switch suboperation.State {
		case types.SUCCEEDED:
			cascadedOperations.SucceededOperations = append(cascadedOperations.SucceededOperations, suboperation)
		case types.FAILED:
			cascadedOperations.FailedOperations = append(cascadedOperations.FailedOperations, suboperation)
		case types.IN_PROGRESS:
			cascadedOperations.InProgressOperations = append(cascadedOperations.InProgressOperations, suboperation)
		case types.PENDING:
			cascadedOperations.InProgressOperations = append(cascadedOperations.NotStartedOperations, suboperation)
		}
	}
	return cascadedOperations, nil
}

func makeCascadeOPForChild(object types.Object, parentOperation *types.Operation, rootId string) (*types.Operation, error) {
	UUID, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	now := time.Now()
	// todo:  ExternalID
	return &types.Operation{
		Base: types.Base{
			ID:        UUID.String(),
			CreatedAt: now,
			UpdatedAt: now,
			Labels:    parentOperation.Labels,
			Ready:     true,
		},
		Type:          types.DELETE,
		State:         types.PENDING,
		ResourceID:    object.GetID(),
		ResourceType:  object.GetType(),
		PlatformID:    parentOperation.PlatformID,
		Cascade:       true,
		Parent:        parentOperation.ID,
		CorrelationID: parentOperation.CorrelationID,
		Root:          rootId,
	}, nil
}
