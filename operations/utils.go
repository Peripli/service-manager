package operations

import (
	"context"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/types/parent"
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
	childrenCriterions := parent.GetChildrenCriteria(object)
	for childType, childCriteria := range childrenCriterions {
		list, err := storage.List(ctx, childType, childCriteria...)
		if err != nil {
			return nil, err
		}
		children = append(children, list)
	}
	return children, nil
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
