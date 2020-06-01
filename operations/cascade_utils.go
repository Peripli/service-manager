package operations

import (
	"context"
	"encoding/json"
	"fmt"
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
	objectChildren, err := u.GetObjectChildren(ctx, object, storage)
	if err != nil {
		return nil, err
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

func (u *CascadeUtils) GetObjectChildren(ctx context.Context, object types.Object, storage storage.Repository) ([]types.ObjectList, error) {
	var children []types.ObjectList
	isBroker := object.GetType() == types.ServiceBrokerType
	if isBroker {
		if err := enrichBrokersOfferings(ctx, object, storage); err != nil {
			return nil, err
		}
	}

	if cascadeObject, isCascade := cascade.GetCascadeObject(ctx, object); isCascade {
		for childType, childCriteria := range cascadeObject.GetChildrenCriterion() {
			list, err := storage.List(ctx, childType, childCriteria...)
			if err != nil {
				return nil, err
			}
			children = append(children, list)
		}
	}
	if isBroker {
		err := u.validateNoGlobalInstances(ctx, object, children, storage)
		if err != nil {
			return nil, err
		}
	}
	return children, nil
}

func (u *CascadeUtils) validateNoGlobalInstances(ctx context.Context, broker types.Object, brokerChildren []types.ObjectList, repository storage.Repository) error {
	platformIdsMap := make(map[string]bool)
	for _, children := range brokerChildren {
		for i := 0; i < children.Len(); i++ {
			instance, ok := children.ItemAt(i).(*types.ServiceInstance)
			if !ok {
				return fmt.Errorf("broker %s has children not of type %s", broker.GetID(), types.ServiceInstanceType)
			}
			if _, ok := platformIdsMap[instance.PlatformID]; !ok {
				platformIdsMap[instance.PlatformID] = true
			}
		}
	}
	delete(platformIdsMap, types.SMPlatform)
	if len(platformIdsMap) == 0 {
		return nil
	}

	platformIds := make([]string, len(platformIdsMap))
	index := 0
	for id := range platformIdsMap {
		platformIds[index] = id
		index++
	}

	if len(platformIds) == 0 {
		return nil
	}
	platforms, err := repository.List(ctx, types.PlatformType, query.ByField(query.InOperator, "id", platformIds...))
	if err != nil {
		return err
	}
	for i := 0; i < platforms.Len(); i++ {
		platform := platforms.ItemAt(i)
		labels := platform.GetLabels()
		if _, found := labels[u.TenantIdentifier]; !found {
			return fmt.Errorf("broker %s has instances from global platform", broker.GetID())
		}
	}
	return nil
}

func enrichBrokersOfferings(ctx context.Context, brokerObj types.Object, storage storage.Repository) error {
	broker := brokerObj.(*types.ServiceBroker)
	serviceOfferings, err := storage.List(ctx, types.ServiceOfferingType, query.ByField(query.EqualsOperator, "broker_id", broker.GetID()))
	if err != nil {
		return err
	}
	for j := 0; j < serviceOfferings.Len(); j++ {
		serviceOffering := serviceOfferings.ItemAt(j).(*types.ServiceOffering)
		servicePlans, err := storage.List(ctx, types.ServicePlanType, query.ByField(query.EqualsOperator, "service_offering_id", serviceOffering.GetID()))
		if err != nil {
			return err
		}
		for g := 0; g < serviceOfferings.Len(); g++ {
			serviceOffering.Plans = append(serviceOffering.Plans, servicePlans.ItemAt(g).(*types.ServicePlan))
		}
		broker.Services = append(broker.Services, serviceOffering)
	}
	return nil
}

func GetSubOperations(ctx context.Context, operation *types.Operation, repository storage.Repository) (*cascade.CascadedOperations, error) {
	objs, err := repository.List(ctx, types.OperationType, query.ByField(query.EqualsOperator, "parent_id", operation.ID))
	if err != nil {
		return nil, err
	}
	subOperations := objs.(*types.Operations)
	cascadedOperations := cascade.CascadedOperations{}
	cascadedOperations.AllOperationsCount = len(subOperations.Operations)
	for i := 0; i < subOperations.Len(); i++ {
		subOperation := subOperations.ItemAt(i).(*types.Operation)
		if subOperation.InOrphanMitigationState() {
			cascadedOperations.OrphanMitigationOperations = append(cascadedOperations.OrphanMitigationOperations, subOperation)
		} else {
			switch subOperation.State {
			case types.SUCCEEDED:
				cascadedOperations.SucceededOperations = append(cascadedOperations.SucceededOperations, subOperation)
			case types.FAILED:
				cascadedOperations.FailedOperations = append(cascadedOperations.FailedOperations, subOperation)
			case types.IN_PROGRESS:
				cascadedOperations.InProgressOperations = append(cascadedOperations.InProgressOperations, subOperation)
			case types.PENDING:
				cascadedOperations.PendingOperations = append(cascadedOperations.PendingOperations, subOperation)
			}
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
/**
	returns 3 parameters:
	OperationState in case there is a duplicate operation that finished(SUCCESS/FAILURE)
	bool skip in case there is duplicate operation in_progress or in OM in the same tree
	error
 */
func handleDuplicateOperations(ctx context.Context, storage storage.Repository, operation *types.Operation) (types.OperationState, bool, error) {
	criteria := []query.Criterion{
		query.ByField(query.EqualsOperator, "resource_id", operation.ResourceID),
		query.ByField(query.EqualsOperator, "type", string(types.DELETE)),
	}
	sameResourceOperations, err := storage.List(ctx, types.OperationType, criteria...)
	if err != nil {
		return "", false, err
	}

	for i := 0; i < sameResourceOperations.Len(); i++ {
		same := sameResourceOperations.ItemAt(i).(*types.Operation)
		if same.CascadeRootID == operation.CascadeRootID && same.InOrphanMitigationState() {
			return "", true, nil
		}
		switch same.State {
		case types.IN_PROGRESS:
			// other operation with the same resourceID is already in progress, skipping this operation
			return "", true, nil
		case types.SUCCEEDED:
			fallthrough
		case types.FAILED:
			if same.CascadeRootID == operation.CascadeRootID {
				return same.State, false, nil
			}
			return "", false, nil
		}
	}
	// same operations are pending, proceeding the flow
	return "", false, nil
}

func SameResourceInCurrentTreeHasFinished(ctx context.Context, storage storage.Repository, resourceID string, cascadeRootID string) (types.OperationState, error) {
	// checking if there is a completed suboperation in same cascade tree with the same resourceID
	completedCriteria := []query.Criterion{
		query.ByField(query.EqualsOperator, "resource_id", resourceID),
		query.ByField(query.InOperator, "state", string(types.SUCCEEDED), string(types.FAILED)),
		query.ByField(query.EqualsOperator, "cascade_root_id", cascadeRootID),
		query.OrderResultBy("paging_sequence", query.DescOrder),
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
	cascadeErrors := cascade.CascadeErrors{Errors: []*cascade.Error{}}
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

		if len(failedOP.Errors) > 0 {
			// in case we are failing to convert it, save it as a regular error
			cascadeErrors.Add(&cascade.Error{
				ParentType:   resourceType,
				ParentID:     resourceID,
				ResourceType: failedOP.ResourceType,
				ResourceID:   failedOP.ResourceID,
				Message:      failedOP.Errors,
			})
		}
	}
	return json.Marshal(cascadeErrors)
}
