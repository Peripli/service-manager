package cascade

import (
	"context"
	"encoding/json"
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
)

type ChildrenCriterion = map[types.ObjectType][]query.Criterion
type CascadeChildren = map[types.ObjectType]types.ObjectList

// key for configurable hierarchies
type ParentInstanceLabelKeys struct{}

type Cascade interface {
	GetChildrenCriterion() ChildrenCriterion
}

type DuplicatesCleaner interface {
	CleanDuplicates(children CascadeChildren)
}

type CascadedOperations struct {
	AllOperationsCount         int
	FailedOperations           []*types.Operation
	InProgressOperations       []*types.Operation
	SucceededOperations        []*types.Operation
	OrphanMitigationOperations []*types.Operation
	PendingOperations          []*types.Operation
}

type Error struct {
	ParentType   types.ObjectType `json:"parent_type,omitempty"`
	ParentID     string           `json:"parent_id,omitempty"`
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
		parentInstanceLabelKeysInterface := ctx.Value(ParentInstanceLabelKeys{})
		var parentInstanceKeys []string
		if keys, ok := parentInstanceLabelKeysInterface.([]string); ok {
			parentInstanceKeys = keys
		}
		return &ServiceInstanceCascade{object.(*types.ServiceInstance), parentInstanceKeys}, true
	}
	return nil, false
}
