// GENERATED. DO NOT MODIFY!

package types

import (
	"encoding/json"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/util"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web"
)

const ServiceInstanceType ObjectType = web.ServiceInstancesURL

type ServiceInstances struct {
	ServiceInstances []*ServiceInstance `json:"service_instances"`
}

func (e *ServiceInstances) Add(object Object) {
	e.ServiceInstances = append(e.ServiceInstances, object.(*ServiceInstance))
}

func (e *ServiceInstances) ItemAt(index int) Object {
	return e.ServiceInstances[index]
}

func (e *ServiceInstances) Len() int {
	return len(e.ServiceInstances)
}

func (e *ServiceInstance) GetType() ObjectType {
	return ServiceInstanceType
}

// MarshalJSON override json serialization for http response
func (e *ServiceInstance) MarshalJSON() ([]byte, error) {
	type E ServiceInstance
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
