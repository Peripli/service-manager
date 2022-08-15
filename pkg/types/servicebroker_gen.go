// GENERATED. DO NOT MODIFY!

package types

import (
	"encoding/json"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/util"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web"
)

const ServiceBrokerType ObjectType = web.ServiceBrokersURL

type ServiceBrokers struct {
	ServiceBrokers []*ServiceBroker `json:"service_brokers"`
}

func (e *ServiceBrokers) Add(object Object) {
	e.ServiceBrokers = append(e.ServiceBrokers, object.(*ServiceBroker))
}

func (e *ServiceBrokers) ItemAt(index int) Object {
	return e.ServiceBrokers[index]
}

func (e *ServiceBrokers) Len() int {
	return len(e.ServiceBrokers)
}

func (e *ServiceBroker) GetType() ObjectType {
	return ServiceBrokerType
}

// MarshalJSON override json serialization for http response
func (e *ServiceBroker) MarshalJSON() ([]byte, error) {
	type E ServiceBroker
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
