package cascade

import (
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
)

type TenantCascade struct {
	*types.Tenant
}

func (t *TenantCascade) Clean(children Children) {
	subacountBrokerPlanIDs := make(map[string]bool)
	subaccountPlatformIDs := make(map[string]bool)

	brokers, found := children[types.ServiceBrokerType]
	if found {
		for i := 0; i < brokers.Len(); i++ {
			broker := brokers.ItemAt(i).(*types.ServiceBroker)
			for _, service := range broker.Services {
				for _, plan := range service.Plans {
					subacountBrokerPlanIDs[plan.ID] = true
				}
			}
		}
	}

	platforms, found := children[types.PlatformType]
	if found {
		for i := 0; i < platforms.Len(); i++ {
			subaccountPlatformIDs[platforms.ItemAt(i).GetID()] = true
		}
	}

	newServiceInstancesList := types.ServiceInstances{}
	instances, found := children[types.ServiceInstanceType]
	if found {
		for i := 0; i < instances.Len(); i++ {
			instance := instances.ItemAt(i).(*types.ServiceInstance)
			_, isSubaccountBrokerInstance := subacountBrokerPlanIDs[instance.ServicePlanID]
			_, isSubaccountPlatformInstance := subaccountPlatformIDs[instance.PlatformID]

			if !isSubaccountBrokerInstance && !isSubaccountPlatformInstance {
				newServiceInstancesList.Add(instance)
			}
		}
	}

	children[types.ServiceInstanceType] = &newServiceInstancesList
}

func (t *TenantCascade) GetChildrenCriterion() ChildrenCriterion {
	return ChildrenCriterion{
		types.VisibilityType:      {query.ByLabel(query.EqualsOperator, t.TenantIdentifier, t.ID)},
		types.PlatformType:        {query.ByLabel(query.EqualsOperator, t.TenantIdentifier, t.ID)},
		types.ServiceBrokerType:   {query.ByLabel(query.EqualsOperator, t.TenantIdentifier, t.ID)},
		types.ServiceInstanceType: {query.ByLabel(query.EqualsOperator, t.TenantIdentifier, t.ID)},
	}
}
