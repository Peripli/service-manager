package types

import (
	"errors"
	"github.com/Peripli/service-manager/pkg/query"
)

const TenantType ObjectType = "Tenant"

type Tenant struct {
	Base
	TenantIdentifier string
}

func (t *Tenant) GetChildren() []query.Criterion {
	return nil
}

func (e *Tenant) Equals(obj Object) bool {
	if !Equals(e, obj) {
		return false
	}
	return true
}

// Validate implements InputValidator and verifies all mandatory fields are populated
func (e *Tenant) Validate() error {
	if e.ID == "" {
		return errors.New("missing tenant id")
	}
	return nil
}

func (t *Tenant) GetType() ObjectType {
	return TenantType
}

func (t *Tenant) GetChildrenCriteria() map[ObjectType][]query.Criterion {
	return map[ObjectType][]query.Criterion{
		VisibilityType:      {query.ByLabel(query.EqualsOperator, t.TenantIdentifier, t.ID)},
		PlatformType:        {query.ByLabel(query.EqualsOperator, t.TenantIdentifier, t.ID)},
		ServiceBrokerType:   {query.ByLabel(query.EqualsOperator, t.TenantIdentifier, t.ID)},
		ServiceInstanceType: {query.ByLabel(query.EqualsOperator, t.TenantIdentifier, t.ID), query.ByField(query.EqualsOperator, "platform_id", SMPlatform)},
	}
}
