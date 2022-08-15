// GENERATED. DO NOT MODIFY!

package types

import (
	"encoding/json"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/util"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web"
)

const ServiceOfferingType ObjectType = web.ServiceOfferingsURL

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
