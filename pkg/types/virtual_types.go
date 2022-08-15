package types

import (
	"errors"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web"
)

const TenantType ObjectType = web.TenantURL

type VirtualType struct {
	Base
}

func (v VirtualType) Validate() error {
	if v.GetID() == "" {
		return errors.New("validate Settings:  ID is missing")
	}
	return nil
}

func (v VirtualType) Equals(object Object) bool {
	return object.GetID() == v.GetID()
}

type Tenant struct {
	VirtualType
	TenantIdentifier string
}

func (e *Tenant) GetType() ObjectType {
	return TenantType
}

func NewTenant(id string, tenantIdentifier string) *Tenant {
	return &Tenant{
		VirtualType: VirtualType{
			Base: Base{
				ID: id,
			},
		},
		TenantIdentifier: tenantIdentifier,
	}
}

func IsVirtualType(objectType ObjectType) bool {
	return objectType == TenantType
}
