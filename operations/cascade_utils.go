package operations

import (
	"context"
	"encoding/json"
	"github.com/gofrs/uuid"
	"github.com/tidwall/gjson"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/query"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types/cascade"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/util"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/storage"
	"time"
)

func FindCascadeOperationForResource(ctx context.Context, storage storage.Repository, resourceID string) (*types.Operation, error) {
	// if same resource exists in other cascade tree the existing tree/subtree root will be returned
	// in case of service instance deletion failure errors might be lost but will be available in the original tree
	criteria := []query.Criterion{
		query.ByField(query.EqualsOperator, "resource_id", resourceID),
		query.ByField(query.InOperator, "state", string(types.IN_PROGRESS), string(types.PENDING)),
		query.ByField(query.NotEqualsOperator, "cascade_root_id", ""),
		query.OrderResultBy("updated_at", query.DescOrder),
	}
	op, err := storage.Get(ctx, types.OperationType, criteria...)
	if err != nil {
		if err == util.ErrNotFoundInStorage {
			return nil, nil
		}
		return nil, err
	}
	return op.(*types.Operation), nil
}

func GetAllLevelsCascadeOperations(ctx context.Context, object types.Object, operation *types.Operation, storage storage.Repository) ([]*types.Operation, error) {
	// if the root is broker we have to enrich his service offerings and plans
	if object.GetType() == types.ServiceBrokerType {
		if err := EnrichBrokersOfferings(ctx, object, storage); err != nil {
			return nil, err
		}
	}
	return recursiveGetAllLevelsCascadeOperations(ctx, object, operation, storage)
}

func recursiveGetAllLevelsCascadeOperations(ctx context.Context, object types.Object, operation *types.Operation, storage storage.Repository) ([]*types.Operation, error) {
	var operations []*types.Operation
	mapOfChildren, err := GetObjectChildren(ctx, object, storage)
	if err != nil {
		return nil, err
	}
	for _, list := range mapOfChildren {
		for i := 0; i < list.Len(); i++ {
			child := list.ItemAt(i)
			childOperation, err := makeCascadeOperationForChild(child, operation)
			if err != nil {
				return nil, err
			}
			operations = append(operations, childOperation)
			childrenSubOperations, err := recursiveGetAllLevelsCascadeOperations(ctx, child, childOperation, storage)
			if err != nil {
				return nil, err
			}
			operations = append(operations, childrenSubOperations...)
		}
	}
	return operations, nil
}

func ListCascadeChildren(ctx context.Context, childrenCriterion cascade.ChildrenCriterion, storage storage.Repository) (cascade.CascadeChildren, error) {
	cascadeChildren := make(cascade.CascadeChildren)
	for childType, criterionDisjunction := range childrenCriterion {
		for _, criterionConjunction := range criterionDisjunction {
			childrenList, err := storage.List(ctx, childType, criterionConjunction...)
			if err != nil {
				return nil, err
			}
			if cascadeChildren[childType] == nil {
				cascadeChildren[childType] = childrenList
			} else {
				for i := 0; i < childrenList.Len(); i++ {
					cascadeChildren[childType].Add(childrenList.ItemAt(i))
				}
			}
		}
	}
	return cascadeChildren, nil
}

func GetObjectChildren(ctx context.Context, object types.Object, storage storage.Repository) (cascade.CascadeChildren, error) {
	cascadeObject, isCascade := cascade.GetCascadeObject(ctx, object)
	if isCascade {
		children, err := ListCascadeChildren(ctx, cascadeObject.GetChildrenCriterion(), storage)
		if err != nil {
			return nil, err
		}
		if brokers, found := children[types.ServiceBrokerType]; found {
			for i := 0; i < brokers.Len(); i++ {
				if err := EnrichBrokersOfferings(ctx, brokers.ItemAt(i), storage); err != nil {
					return nil, err
				}
			}
		}
		removeDuplicateSubOperations(cascadeObject, children)
		return children, nil
	} else {
		return make(cascade.CascadeChildren), nil
	}
}

func removeDuplicateSubOperations(cascadeObject cascade.Cascade, children cascade.CascadeChildren) {
	cleaner, hasDuplicatesCleaner := cascadeObject.(cascade.DuplicatesCleaner)
	if hasDuplicatesCleaner {
		cleaner.CleanDuplicates(children)
	}
}

func EnrichBrokersOfferings(ctx context.Context, brokerObj types.Object, storage storage.Repository) error {
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
		for g := 0; g < servicePlans.Len(); g++ {
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
			case types.IN_PROGRESS:
				cascadedOperations.InProgressOperations = append(cascadedOperations.InProgressOperations, subOperation)
			case types.PENDING:
				cascadedOperations.PendingOperations = append(cascadedOperations.PendingOperations, subOperation)
			case types.FAILED:
				cascadedOperations.FailedOperations = append(cascadedOperations.FailedOperations, subOperation)
			}
		}
	}
	return &cascadedOperations, nil
}

func makeCascadeOperationForChild(object types.Object, operation *types.Operation) (*types.Operation, error) {
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
		query.ByField(query.EqualsOperator, "resource_type", string(operation.ResourceType)),
		query.ByField(query.EqualsOperator, "type", string(operation.Type)),
		query.ByField(query.EqualsOperator, "cascade_root_id", operation.CascadeRootID),
		query.OrderResultBy("updated_at", query.DescOrder),
	}
	sameResourceOperations, err := storage.List(ctx, types.OperationType, criteria...)
	if err != nil {
		return "", false, err
	}
	if sameResourceOperations.Len() > 0 {
		same := sameResourceOperations.ItemAt(0).(*types.Operation)
		if same.InOrphanMitigationState() {
			return "", true, nil
		}
		switch same.State {
		case types.IN_PROGRESS:
			// other operation with the same resourceID is already in progress, skipping this operation
			return "", true, nil
		case types.SUCCEEDED:
			fallthrough
		case types.FAILED:
			return same.State, false, nil
		}
	}
	// same operations are pending, proceeding the flow
	return "", false, nil
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
