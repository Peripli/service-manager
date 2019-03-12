// GENERATED. DO NOT MODIFY!

package types

import (
	"encoding/json"
	
	"github.com/Peripli/service-manager/pkg/util"
)

const ServiceOfferingType ObjectType = "ServiceOffering"

type ServiceOfferings struct {
	ServiceOfferings []*ServiceOffering `json:"service_offerings"`
}

func (e *ServiceOfferings) Add(object Object) {
	e.ServiceOfferings = append(e.ServiceOfferings, object.(*ServiceOffering))
}

func (e *ServiceOfferings) ItemAt(index int) Object {
	return e.ServiceOfferings[index]
}

func (e *ServiceOfferings) Len() int {
	return len(e.ServiceOfferings)
}

func (e *ServiceOffering) EmptyList() ObjectList {
	return &ServiceOfferings{ ServiceOfferings: make([]*ServiceOffering, 0) }
}

func (e *ServiceOffering) GetType() ObjectType {
	return ServiceOfferingType
}

// MarshalJSON override json serialization for http response
func (e *ServiceOffering) MarshalJSON() ([]byte, error) {
	type E ServiceOffering
	toMarshal := struct {
		*E
		CreatedAt *string `json:"created_at,omitempty"`
		UpdatedAt *string `json:"updated_at,omitempty"`
	}{
		E: (*E)(e),
	}
    if !e.CreatedAt.IsZero() {
        str := util.ToRFCFormat(e.CreatedAt)
        toMarshal.CreatedAt = &str
    }
    if !e.UpdatedAt.IsZero() {
        str := util.ToRFCFormat(e.UpdatedAt)
        toMarshal.UpdatedAt = &str
    }
    
	return json.Marshal(toMarshal)
}
