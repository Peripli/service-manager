package operations

import (
	"context"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
	"github.com/gofrs/uuid"
	"github.com/tidwall/gjson"
	"time"
)

type OperationUtils struct {
	TenantIdentifier string
}

// Get all cascade operations
func (o *OperationUtils) GetCascadeOperations(ctx context.Context, operation *types.Operation, storage storage.Repository) ([]*types.Operation, error) {
	cascadeOperations, err := o.GetChildrenCascadeOperations(ctx, operation, storage)
	if err != nil {
		return nil, err
	}
	for _, ch := range cascadeOperations {
		childrenCascadeOperations, err := o.GetCascadeOperations(ctx, ch, storage)
		if err != nil {
			return nil, err
		}
		cascadeOperations = append(cascadeOperations, childrenCascadeOperations...)
	}
	return cascadeOperations, nil
}

func (o *OperationUtils) GetChildrenCascadeOperations(ctx context.Context, operation *types.Operation, storage storage.Repository) ([]*types.Operation, error) {
	switch operation.ResourceType {
	case types.Tenant:
		return o.getTenantChildrenOperations(ctx, operation, storage)
	case types.PlatformType:
		return o.getPlatformChildrenOperations(ctx, operation, storage)
	case types.ServiceBrokerType:
		return o.getServiceBrokerChildrenOperations(ctx, operation, storage)
	case types.ServiceInstanceType:
		return o.getInstanceChildrenOperations(ctx, operation, storage)
	default:
		return []*types.Operation{}, nil
	}
}

func (o *OperationUtils) getTenantChildrenOperations(ctx context.Context, operation *types.Operation, repository storage.Repository) ([]*types.Operation, error) {
	var tenantChildren []*types.Operation

	criterions := []query.Criterion{query.ByLabel(query.EqualsOperator, o.TenantIdentifier, operation.ResourceID)}
	tenantVisibilityChildren, err := getChildren(ctx, repository, operation, types.VisibilityType, criterions...)
	if err != nil {
		return nil, err
	}
	tenantChildren = append(tenantChildren, tenantVisibilityChildren...)

	criterions = []query.Criterion{query.ByLabel(query.EqualsOperator, o.TenantIdentifier, operation.ResourceID)}
	tenantPlatformChildren, err := getChildren(ctx, repository, operation, types.PlatformType, criterions...)
	if err != nil {
		return nil, err
	}
	tenantChildren = append(tenantChildren, tenantPlatformChildren...)

	criterions = []query.Criterion{query.ByLabel(query.EqualsOperator, o.TenantIdentifier, operation.ResourceID)}
	tenantBrokerChildren, err := getChildren(ctx, repository, operation, types.ServiceBrokerType, criterions...)
	if err != nil {
		return nil, err
	}
	tenantChildren = append(tenantChildren, tenantBrokerChildren...)

	criterions = []query.Criterion{query.ByLabel(query.EqualsOperator, o.TenantIdentifier, operation.ResourceID), query.ByField(query.EqualsOperator, "platform_id", types.SMPlatform)}
	tenantInstancesChildren, err := getChildren(ctx, repository, operation, types.ServiceInstanceType, criterions...)
	if err != nil {
		return nil, err
	}
	tenantChildren = append(tenantChildren, tenantInstancesChildren...)
	return tenantChildren, nil
}

func (o *OperationUtils) getPlatformChildrenOperations(ctx context.Context, operation *types.Operation, repository storage.Repository) ([]*types.Operation, error) {
	criterion := query.ByField(query.EqualsOperator, "platform_id", operation.ResourceID)
	return getChildren(ctx, repository, operation, types.ServiceInstanceType, criterion)
}

func (o *OperationUtils) getInstanceChildrenOperations(ctx context.Context, operation *types.Operation, repository storage.Repository) ([]*types.Operation, error) {
	criterion := query.ByField(query.EqualsOperator, "service_instance_id", operation.ResourceID)
	return getChildren(ctx, repository, operation, types.ServiceBindingType, criterion)
}

func (o *OperationUtils) getServiceBrokerChildrenOperations(ctx context.Context, operation *types.Operation, repository storage.Repository) ([]*types.Operation, error) {
	broker, err := repository.Get(ctx, types.ServiceBrokerType, query.ByField(query.EqualsOperator, "id", operation.ResourceID))
	if err != nil {
		return nil, err
	}
	// todo: check query works
	plansIDs := gjson.GetBytes(broker.(*types.ServiceBroker).Catalog, `services.#.plans.#.id`)
	criterion := query.ByField(query.InOperator, "platform_id", plansIDs.Value().([]string)...)
	return getChildren(ctx, repository, operation, types.ServiceInstanceType, criterion)
}

func getChildren(ctx context.Context, repository storage.Repository, operation *types.Operation, childrenType types.ObjectType, criterions ...query.Criterion) ([]*types.Operation, error) {
	children, err := repository.List(ctx, childrenType, criterions...)
	if err != nil {
		return nil, err
	}

	var operations []*types.Operation
	for i := 0; i < children.Len(); i++ {
		child := children.ItemAt(i)
		operation, err := createOperation(child.GetID(), child.GetType(), operation.PlatformID)
		if err != nil {
			return nil, err
		}
		operations = append(operations, operation)
	}
	return operations, nil
}

func createOperation(resourceID string, resourceType types.ObjectType, platformID string) (*types.Operation, error) {
	UUID, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	// todo:  ExternalID CorrelationID
	return &types.Operation{
		Base: types.Base{
			ID:        UUID.String(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Labels:    make(map[string][]string),
			Ready:     true,
		},
		Type:         types.DELETE,
		State:        types.IN_PROGRESS,
		ResourceID:   resourceID,
		ResourceType: resourceType,
		PlatformID:   platformID,
	}, nil
}
