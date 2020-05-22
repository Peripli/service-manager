package cascade

import (
	"context"
	"encoding/json"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/storage"
)

type ChildrenCriterion = map[types.ObjectType][]query.Criterion

type ContainerKey struct{}

type Cascade interface {
	GetChildrenCriterion() ChildrenCriterion
}

type Validate interface {
	ValidateChildren() func(ctx context.Context, objectChildren []types.ObjectList, repository storage.Repository, labelKeys ...string) error
}

type CascadedOperations struct {
	AllOperationsCount   int
	FailedOperations     []*types.Operation
	InProgressOperations []*types.Operation
	SucceededOperations  []*types.Operation
	PendingOperations    []*types.Operation
}

type Error struct {
	ParentType   types.ObjectType `json:"parent_type"`
	ParentID     string           `json:"parent_id"`
	ResourceType types.ObjectType `json:"resource_type"`
	ResourceID   string           `json:"resource_id"`
	Message      json.RawMessage  `json:"message"`
}

type CascadeErrors struct {
	Errors []*Error `json:"cascade_errors"`
}

func (c *CascadeErrors) Add(e *Error) {
	c.Errors = append(c.Errors, e)
}

func GetCascadeObject(ctx context.Context, object types.Object) (Cascade, bool) {
	switch object.GetType() {
	case types.TenantType:
		return &TenantCascade{object.(*types.Tenant)}, true
	case types.PlatformType:
		return &PlatformCascade{object.(*types.Platform)}, true
	case types.ServiceBrokerType:
		return &ServiceBrokerCascade{object.(*types.ServiceBroker)}, true
	case types.ServiceInstanceType:
		containerId, _ := ctx.Value(ContainerKey{}).(string)
		return &ServiceInstanceCascade{object.(*types.ServiceInstance), containerId}, true
	}
	return nil, false
}
