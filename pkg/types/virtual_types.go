package types

import "errors"

const TenantType ObjectType = "Tenant"

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
	if object.GetID() != v.GetID() {
		return false
	}
	return true
}

type Tenant struct {
	VirtualType
	TenantIdentifier string
}

func (e *Tenant) GetType() ObjectType {
	return TenantType
}
