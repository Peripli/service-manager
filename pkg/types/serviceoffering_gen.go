// GENERATED. DO NOT MODIFY!

package types

import (
	"encoding/json"

	"github.com/Peripli/service-manager/pkg/util"
)

const ServiceOfferingType ObjectType = "types.ServiceOffering"

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
		Labels    Labels  `json:"labels,omitempty"`
	}{
		E:      (*E)(e),
		Labels: e.Labels,
	}
	if !e.CreatedAt.IsZero() {
		str := util.ToRFCNanoFormat(e.CreatedAt)
		toMarshal.CreatedAt = &str
	}
	if !e.UpdatedAt.IsZero() {
		str := util.ToRFCNanoFormat(e.UpdatedAt)
		toMarshal.UpdatedAt = &str
	}
	hasNoLabels := true
	for key, values := range e.Labels {
		if key != "" && len(values) != 0 {
			hasNoLabels = false
			break
		}
	}
	if hasNoLabels {
		toMarshal.Labels = nil
	}
	return json.Marshal(toMarshal)
}
