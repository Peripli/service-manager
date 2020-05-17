package cascade

import (
	"github.com/Peripli/service-manager/pkg/query"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/tidwall/gjson"
)

type GetChildrenCriteria = map[types.ObjectType][]query.Criterion

func GetServiceInstanceChildrenCriteria(s *types.ServiceInstance) GetChildrenCriteria {
	return GetChildrenCriteria{
		types.ServiceBindingType: {query.ByField(query.EqualsOperator, "service_instance_id", s.GetID())},
	}
}

func GetPlatformChildrenCriteria(p *types.Platform) GetChildrenCriteria {
	return GetChildrenCriteria{
		types.ServiceInstanceType: {query.ByField(query.EqualsOperator, "platform_id", p.GetID())},
	}
}

func GetServiceBrokerChildrenCriteria(s *types.ServiceBroker) GetChildrenCriteria {
	// todo: check query is working
	plansIDs := gjson.GetBytes(s.Catalog, `services.#.plans.#.id`)
	return GetChildrenCriteria{
		types.VisibilityType:      {query.ByField(query.InOperator, "service_plan_id", plansIDs.Value().([]string)...)},
		types.ServicePlanType:     {query.ByField(query.InOperator, "id", plansIDs.Value().([]string)...)},
		types.ServiceOfferingType: {query.ByField(query.EqualsOperator, "broker_id", s.GetID())},
		types.ServiceInstanceType: {query.ByField(query.InOperator, "service_plan_id", plansIDs.Value().([]string)...)},
	}
}

func GetTenantChildrenCriteria(t *types.Tenant) GetChildrenCriteria {
	byTenantIdentifierLabel := query.ByLabel(query.EqualsOperator, t.TenantIdentifier, t.GetID())
	return GetChildrenCriteria{
		types.VisibilityType:      {byTenantIdentifierLabel},
		types.PlatformType:        {byTenantIdentifierLabel},
		types.ServiceBrokerType:   {byTenantIdentifierLabel},
		types.ServiceInstanceType: {byTenantIdentifierLabel, query.ByField(query.EqualsOperator, "platform_id", types.SMPlatform)},
	}
}
