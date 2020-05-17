package parent

import (
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/tidwall/gjson"
)

type ChildrenCriteria = map[types.ObjectType][]query.Criterion

func GetChildrenCriteria(object types.Object) ChildrenCriteria {
	switch object.GetType() {
	case types.TenantType:
		return getTenantChildrenCriteria(object.(*types.Tenant))
	case types.PlatformType:
		return getPlatformChildrenCriteria(object.(*types.Platform))
	case types.ServiceBrokerType:
		return getServiceBrokerChildrenCriteria(object.(*types.ServiceBroker))
	case types.ServiceInstanceType:
		return getServiceInstanceChildrenCriteria(object.(*types.ServiceInstance))
	}
	return make(ChildrenCriteria)
}

func getServiceInstanceChildrenCriteria(instance *types.ServiceInstance) ChildrenCriteria {
	return ChildrenCriteria{
		types.ServiceBindingType: {query.ByField(query.EqualsOperator, "service_instance_id", instance.GetID())},
	}
}

func getPlatformChildrenCriteria(p *types.Platform) ChildrenCriteria {
	return ChildrenCriteria{
		types.ServiceInstanceType: {query.ByField(query.EqualsOperator, "platform_id", p.GetID())},
	}
}

func getServiceBrokerChildrenCriteria(s *types.ServiceBroker) ChildrenCriteria {
	// todo: check query is working
	plansIDs := gjson.GetBytes(s.Catalog, `services.#.plans.#.id`)
	return ChildrenCriteria{
		types.VisibilityType:      {query.ByField(query.InOperator, "service_plan_id", plansIDs.Value().([]string)...)},
		types.ServicePlanType:     {query.ByField(query.InOperator, "id", plansIDs.Value().([]string)...)},
		types.ServiceOfferingType: {query.ByField(query.EqualsOperator, "broker_id", s.GetID())},
		types.ServiceInstanceType: {query.ByField(query.InOperator, "service_plan_id", plansIDs.Value().([]string)...)},
	}
}

func getTenantChildrenCriteria(t *types.Tenant) ChildrenCriteria {
	byTenantIdentifierLabel := query.ByLabel(query.EqualsOperator, t.TenantIdentifier, t.GetID())
	return ChildrenCriteria{
		types.VisibilityType:      {byTenantIdentifierLabel},
		types.PlatformType:        {byTenantIdentifierLabel},
		types.ServiceBrokerType:   {byTenantIdentifierLabel},
		types.ServiceInstanceType: {byTenantIdentifierLabel, query.ByField(query.EqualsOperator, "platform_id", "basic-auth-default-test-platform")},
	}
}
