// GENERATED. DO NOT MODIFY!

package types

import (
	"encoding/json"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/util"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web"
)

const ServicePlanType ObjectType = web.ServicePlansURL

type ServicePlans struct {
	ServicePlans []*ServicePlan `json:"service_plans"`
}

func (e *ServicePlans) Add(object Object) {
	e.ServicePlans = append(e.ServicePlans, object.(*ServicePlan))
}

func (e *ServicePlans) ItemAt(index int) Object {
	return e.ServicePlans[index]
}

func (e *ServicePlans) Len() int {
	return len(e.ServicePlans)
}

func (e *ServicePlan) GetType() ObjectType {
	return ServicePlanType
}

// MarshalJSON override json serialization for http response
func (e *ServicePlan) MarshalJSON() ([]byte, error) {
	type E ServicePlan
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
