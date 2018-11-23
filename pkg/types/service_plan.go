package types

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Peripli/service-manager/pkg/util"
)

type ServicePlan struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	CatalogID     string `json:"catalog_id"`
	CatalogName   string `json:"catalog_name"`
	Free          bool   `json:"free"`
	Bindable      bool   `json:"bindable"`
	PlanUpdatable bool   `json:"plan_updateable"`

	Metadata             json.RawMessage `json:"metadata,omitempty"`
	CreateInstanceSchema json.RawMessage `json:"create_instance_schema"`
	UpdateInstanceSchema json.RawMessage `json:"update_instance_schema"`
	CreateBindingSchema  json.RawMessage `json:"create_binding_schema"`

	ServiceOfferingID string `json:"service_offering_id"`
}

// MarshalJSON override json serialization for http response
func (sp *ServicePlan) MarshalJSON() ([]byte, error) {
	type SP ServicePlan
	toMarshal := struct {
		CreatedAt *string `json:"created_at,omitempty"`
		UpdatedAt *string `json:"updated_at,omitempty"`
		*SP
	}{
		SP: (*SP)(sp),
	}

	if !sp.CreatedAt.IsZero() {
		str := util.ToRFCFormat(sp.CreatedAt)
		toMarshal.CreatedAt = &str
	}
	if !sp.UpdatedAt.IsZero() {
		str := util.ToRFCFormat(sp.UpdatedAt)
		toMarshal.UpdatedAt = &str
	}
	return json.Marshal(toMarshal)
}

// Validate implements InputValidator and verifies all mandatory fields are populated
func (sp *ServicePlan) Validate() error {
	if util.HasRFC3986ReservedSymbols(sp.ID) {
		return fmt.Errorf("%s contains invalid character(s)", sp.ID)
	}
	if sp.Name == "" {
		return fmt.Errorf("service plan name missing")
	}
	if sp.CatalogID == "" {
		return fmt.Errorf("service plan catalog id missing")
	}
	if sp.CatalogName == "" {
		return fmt.Errorf("service plan catalog name missing")
	}
	if sp.ServiceOfferingID == "" {
		return fmt.Errorf("service plan service offering id missing")
	}
	return nil
}
