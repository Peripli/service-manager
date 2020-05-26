package cascade

import (
	"context"
	"encoding/json"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
)

type ChildrenCriterion = map[types.ObjectType][]query.Criterion

// key for configurable hierarchies
type ContainerKey struct{}

type Cascade interface {
	GetChildrenCriterion() ChildrenCriterion
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
		containerIDValue := ctx.Value(ContainerKey{})
		return &ServiceInstanceCascade{object.(*types.ServiceInstance), containerIDValue}, true
	}
	return nil, false
}
