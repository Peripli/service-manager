// GENERATED. DO NOT MODIFY!

package types

import (
	"encoding/json"

	"github.com/Peripli/service-manager/pkg/util"
)

const BrokerVisibilityType ObjectType = "types.BrokerVisibility"

type BrokerVisibilities struct {
	BrokerVisibilities []*BrokerVisibility `json:"broker_visibilities"`
}

func (e *BrokerVisibilities) Add(object Object) {
	e.BrokerVisibilities = append(e.BrokerVisibilities, object.(*BrokerVisibility))
}

func (e *BrokerVisibilities) ItemAt(index int) Object {
	return e.BrokerVisibilities[index]
}

func (e *BrokerVisibilities) Len() int {
	return len(e.BrokerVisibilities)
}

func (e *BrokerVisibility) GetType() ObjectType {
	return BrokerVisibilityType
}

// MarshalJSON override json serialization for http response
func (e *BrokerVisibility) MarshalJSON() ([]byte, error) {
	type E BrokerVisibility
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
		str := util.ToRFCFormat(e.CreatedAt)
		toMarshal.CreatedAt = &str
	}
	if !e.UpdatedAt.IsZero() {
		str := util.ToRFCFormat(e.UpdatedAt)
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
