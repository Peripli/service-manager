package cascade

import (
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/query"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/types"
)

type TenantCascade struct {
	*types.Tenant
}

func (tc *TenantCascade) CleanDuplicates(children CascadeChildren) {
	// cleaning instances that will be under platforms or brokers
	instances, found := children[types.ServiceInstanceType]
	if !found || instances.Len() == 0 {
		return
	}

	tenantScopedBrokerPlanIDs := make(map[string]bool)
	brokers, found := children[types.ServiceBrokerType]
	if found {
		for i := 0; i < brokers.Len(); i++ {
			broker := brokers.ItemAt(i).(*types.ServiceBroker)
			for _, service := range broker.Services {
				for _, plan := range service.Plans {
					tenantScopedBrokerPlanIDs[plan.ID] = true
				}
			}
		}
	}

	tenantScopedPlatformIDs := make(map[string]bool)
	platforms, found := children[types.PlatformType]
	if found {
		for i := 0; i < platforms.Len(); i++ {
			tenantScopedPlatformIDs[platforms.ItemAt(i).GetID()] = true
		}
	}

	filteredServiceInstances := types.ServiceInstances{}
	for i := 0; i < instances.Len(); i++ {
		instance := instances.ItemAt(i).(*types.ServiceInstance)
		_, isTenantScopedBrokerInstance := tenantScopedBrokerPlanIDs[instance.ServicePlanID]
		_, isTenantScopedPlatformInstance := tenantScopedPlatformIDs[instance.PlatformID]

		if !isTenantScopedBrokerInstance && !isTenantScopedPlatformInstance {
			filteredServiceInstances.Add(instance)
		}
	}
	children[types.ServiceInstanceType] = &filteredServiceInstances
}

func (tc *TenantCascade) GetChildrenCriterion() ChildrenCriterion {
	return ChildrenCriterion{
		types.PlatformType:        {{query.ByLabel(query.EqualsOperator, tc.TenantIdentifier, tc.ID)}},
		types.ServiceBrokerType:   {{query.ByLabel(query.EqualsOperator, tc.TenantIdentifier, tc.ID)}},
		types.ServiceInstanceType: {{query.ByLabel(query.EqualsOperator, tc.TenantIdentifier, tc.ID)}},
	}
}
