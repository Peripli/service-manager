package operations

import (
	"context"
	"encoding/json"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/pkg/types/cascade"
	"github.com/Peripli/service-manager/storage"
	"github.com/gofrs/uuid"
	"github.com/tidwall/gjson"
	"time"
)

type CascadeUtils struct {
	TenantIdentifier string
}

func (u *CascadeUtils) GetAllLevelsCascadeOperations(ctx context.Context, object types.Object, operation *types.Operation, storage storage.Repository) ([]*types.Operation, error) {
	var operations []*types.Operation
	objectChildren, err := getObjectChildren(ctx, object, storage)
	if err != nil {
		return nil, err
	}
	validate, hasChildrenValidator := object.(cascade.Validate)
	if hasChildrenValidator {
		err := validate.ValidateChildren()(ctx, objectChildren, storage, u.TenantIdentifier)
		if err != nil {
			return nil, err
		}
	}
	for _, children := range objectChildren {
		for i := 0; i < children.Len(); i++ {
			childOBJ := children.ItemAt(i)
			childOP, err := makeCascadeOPForChild(childOBJ, operation)
			if err != nil {
				return nil, err
			}
			operations = append(operations, childOP)
			childrenSubOPs, err := u.GetAllLevelsCascadeOperations(ctx, childOBJ, childOP, storage)
			if err != nil {
				return nil, err
			}
			operations = append(operations, childrenSubOPs...)
		}
	}
	return operations, nil
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

func GetSubOperations(ctx context.Context, operation *types.Operation, repository storage.Repository) (*cascade.CascadedOperations, error) {
	objs, err := repository.List(ctx, types.OperationType, query.ByField(query.EqualsOperator, "parent_id", operation.ID))
	subOperations := objs.(*types.Operations)
	cascadedOperations := cascade.CascadedOperations{}
	cascadedOperations.AllOperationsCount = len(subOperations.Operations)
	if err != nil {
		return nil, err
	}
	for i := 0; i < subOperations.Len(); i++ {
		subOperation := subOperations.ItemAt(i).(*types.Operation)
		switch subOperation.State {
		case types.SUCCEEDED:
			cascadedOperations.SucceededOperations = append(cascadedOperations.SucceededOperations, subOperation)
		case types.FAILED:
			cascadedOperations.FailedOperations = append(cascadedOperations.FailedOperations, subOperation)
		case types.IN_PROGRESS:
			cascadedOperations.InProgressOperations = append(cascadedOperations.InProgressOperations, subOperation)
		case types.PENDING:
			cascadedOperations.InProgressOperations = append(cascadedOperations.PendingOperations, subOperation)
		}
	}
	return &cascadedOperations, nil
}

func makeCascadeOPForChild(object types.Object, operation *types.Operation) (*types.Operation, error) {
	UUID, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	now := time.Now()
	return &types.Operation{
		Base: types.Base{
			ID:        UUID.String(),
			CreatedAt: now,
			UpdatedAt: now,
			Labels:    operation.Labels,
			Ready:     true,
		},
		Type:          types.DELETE,
		State:         types.PENDING,
		ResourceID:    object.GetID(),
		ResourceType:  object.GetType(),
		PlatformID:    operation.PlatformID,
		ParentID:      operation.ID,
		CorrelationID: operation.CorrelationID,
		CascadeRootID: operation.CascadeRootID,
	}, nil
}

func SameResourceIsAlreadyPolling(ctx context.Context, storage storage.Repository, resourceID string) (bool, error) {
	criteria := []query.Criterion{
		query.ByField(query.EqualsOperator, "resource_id", resourceID),
		query.ByField(query.EqualsOperator, "state", string(types.IN_PROGRESS)),
		query.ByField(query.EqualsOperator, "reschedule", "true"),
		query.ByField(query.EqualsOperator, "deletion_scheduled", ZeroTime),
	}
	cnt, err := storage.Count(ctx, types.OperationType, criteria...)
	if err != nil {
		return false, err
	}
	if cnt > 0 {
		return true, nil
	}
	return false, nil
}

func SameResourceInCurrentTreeHasFinished(ctx context.Context, storage storage.Repository, resourceID string, cascadeRootID string) (types.OperationState, error) {
	// checking if there is a completed suboperation in same cascade tree with the same resourceID
	completedCriteria := []query.Criterion{
		query.ByField(query.EqualsOperator, "resource_id", resourceID),
		query.ByField(query.InOperator, "state", string(types.SUCCEEDED), string(types.FAILED)),
		query.ByField(query.EqualsOperator, "cascade_root_id", cascadeRootID),
		query.OrderResultBy("paging_sequence", query.DescOrder),
		query.LimitResultBy(1),
	}
	completed, err := storage.List(ctx, types.OperationType, completedCriteria...)
	if err != nil {
		return "", err
	}
	if completed.Len() > 0 {
		return completed.ItemAt(0).(*types.Operation).State, nil
	}
	return "", nil
}

func PrepareAggregatedErrorsArray(failedSubOperations []*types.Operation, resourceID string, resourceType types.ObjectType) ([]byte, error) {
	cascadeErrors := cascade.CascadeErrors{}
	for _, failedOP := range failedSubOperations {
		childErrorsResult := gjson.GetBytes(failedOP.Errors, "cascade_errors")
		if childErrorsResult.Exists() {
			var childErrors []*cascade.Error
			err := json.Unmarshal([]byte(childErrorsResult.String()), &childErrors)
			if err == nil {
				cascadeErrors.Errors = append(cascadeErrors.Errors, childErrors...)
				continue
			}
		}
		// in case we are failing to convert it, save it as a regular error
		cascadeErrors.Add(&cascade.Error{
			ParentType:   resourceType,
			ParentID:     resourceID,
			ResourceType: failedOP.ResourceType,
			ResourceID:   failedOP.ResourceID,
			Message:      failedOP.Errors,
		})
	}
	return json.Marshal(cascadeErrors)
}
