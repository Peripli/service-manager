package types

import (
	"errors"
)

const TenantType ObjectType = "Tenant"

type Tenant struct {
	Base
	TenantIdentifier string
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
