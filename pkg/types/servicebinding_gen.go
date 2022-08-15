// GENERATED. DO NOT MODIFY!

package types

import (
	"encoding/json"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/util"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web"
)

const ServiceBindingType ObjectType = web.ServiceBindingsURL

type ServiceBindings struct {
	ServiceBindings []*ServiceBinding `json:"service_bindings"`
}

func (e *ServiceBindings) Add(object Object) {
	e.ServiceBindings = append(e.ServiceBindings, object.(*ServiceBinding))
}

func (e *ServiceBindings) ItemAt(index int) Object {
	return e.ServiceBindings[index]
}

func (e *ServiceBindings) Len() int {
	return len(e.ServiceBindings)
}

func (e *ServiceBinding) GetType() ObjectType {
	return ServiceBindingType
}

// MarshalJSON override json serialization for http response
func (e *ServiceBinding) MarshalJSON() ([]byte, error) {
	type E ServiceBinding
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
