package cascade

import (
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
)

type ChildrenCriterion = map[types.ObjectType][]query.Criterion

type Cascade interface {
	GetChildrenCriterion() ChildrenCriterion
}

func GetCascadeObject(object types.Object) (Cascade, bool) {
	switch object.GetType() {
	case types.TenantType:
		return &TenantCascade{object.(*types.Tenant)}, true
	case types.PlatformType:
		return &PlatformCascade{object.(*types.Platform)}, true
	case types.ServiceBrokerType:
		return &ServiceBrokerCascade{object.(*types.ServiceBroker)}, true
	case types.ServiceInstanceType:
		return &ServiceInstanceCascade{object.(*types.ServiceInstance)}, true
	}
	return nil, false
}
