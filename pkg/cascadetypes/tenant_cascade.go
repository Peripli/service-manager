package cascadetypes

import (
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
)

type TenantCascade struct {
	*types.Tenant
}

func (t *TenantCascade) GetChildrenCriteria() map[types.ObjectType][]query.Criterion {
	return map[types.ObjectType][]query.Criterion{
		types.VisibilityType:      {query.ByLabel(query.EqualsOperator, t.TenantIdentifier, t.ID)},
		types.PlatformType:        {query.ByLabel(query.EqualsOperator, t.TenantIdentifier, t.ID)},
		types.ServiceBrokerType:   {query.ByLabel(query.EqualsOperator, t.TenantIdentifier, t.ID)},
		types.ServiceInstanceType: {query.ByLabel(query.EqualsOperator, t.TenantIdentifier, t.ID), query.ByField(query.EqualsOperator, "platform_id", types.SMPlatform)},
	}
}
