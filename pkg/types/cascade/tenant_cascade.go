package cascade

import (
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
)

type TenantCascade struct {
	*types.Tenant
}

func (t *TenantCascade) GetChildrenCriterion() ChildrenCriterion {
	return ChildrenCriterion{
		types.VisibilityType:      {query.ByLabel(query.EqualsOperator, t.TenantIdentifier, t.ID)},
		types.PlatformType:        {query.ByLabel(query.EqualsOperator, t.TenantIdentifier, t.ID)},
		types.ServiceBrokerType:   {query.ByLabel(query.EqualsOperator, t.TenantIdentifier, t.ID)},
		types.ServiceInstanceType: {query.ByLabel(query.EqualsOperator, t.TenantIdentifier, t.ID), query.ByField(query.EqualsOperator, "platform_id", types.SMPlatform)},
	}
}
