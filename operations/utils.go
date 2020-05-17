package operations

import (
	"context"
	"github.com/Peripli/service-manager/pkg/cascadetypes"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
	"github.com/gofrs/uuid"
	"time"
)

func GetAllLevelsCascadeOperations(ctx context.Context, parentObject types.Object, parentOperation *types.Operation, storage storage.Repository) ([]*types.Operation, error) {
	var operations []*types.Operation
	objectChildren, err := getObjectChildren(ctx, parentObject, storage)
	if err != nil {
		return nil, err
	}
	for _, children := range objectChildren {
		for i := 0; i < children.Len(); i++ {
			child := children.ItemAt(i)
			childOperation, err := makeCascadeOPForChild(child, parentOperation)
			if err != nil {
				return nil, err
			}
			operations = append(operations, childOperation)
			grandChildrenOperations, err := GetAllLevelsCascadeOperations(ctx, child, childOperation, storage)
			if err != nil {
				return nil, err
			}
			operations = append(operations, grandChildrenOperations...)
		}
	}
	return operations, nil
}

func getObjectChildren(ctx context.Context, object types.Object, storage storage.Repository) ([]types.ObjectList, error) {
	var children []types.ObjectList
	if childrenCriterions, isCascade := cascadetypes.GetCascadeObject(object); isCascade {
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
	Operation            types.Operation
	Operations           []*types.Operation
	FailedOperations     []*types.Operation
	InProgressOperations []*types.Operation
	SucceededOperations  []*types.Operation
}

func GetCascadedOperations(ctx context.Context, operation *types.Operation, repository storage.Repository) (*CascadedOperations, error) {
	objs, err := repository.List(ctx, types.OperationType, query.ByField(query.EqualsOperator, "parent", operation.ResourceID))
	suboperations := objs.(*types.Operations)
	cascadedOperations := &CascadedOperations{}
	cascadedOperations.Operations = suboperations.Operations
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
		}
		children, err := GetCascadedOperations(ctx, suboperation, repository)
		if err != nil {
			return nil, err
		}
		cascadedOperations.Operations = append(cascadedOperations.Operations, children.Operations...)
		cascadedOperations.SucceededOperations = append(cascadedOperations.Operations, children.SucceededOperations...)
		cascadedOperations.FailedOperations = append(cascadedOperations.Operations, children.FailedOperations...)
		cascadedOperations.InProgressOperations = append(cascadedOperations.Operations, children.InProgressOperations...)
	}
	return cascadedOperations, nil
}

func makeCascadeOPForChild(object types.Object, parentOperation *types.Operation) (*types.Operation, error) {
	UUID, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	now := time.Now()
	// todo:  ExternalID CorrelationID
	return &types.Operation{
		Base: types.Base{
			ID:        UUID.String(),
			CreatedAt: now,
			UpdatedAt: now,
			Labels:    parentOperation.Labels,
			Ready:     true,
		},
		Type:          types.DELETE,
		State:         types.IN_PROGRESS,
		ResourceID:    object.GetID(),
		ResourceType:  object.GetType(),
		PlatformID:    parentOperation.PlatformID,
		Cascade:       true,
		Parent:        parentOperation.ID,
		CorrelationID: parentOperation.CorrelationID,
	}, nil
}
