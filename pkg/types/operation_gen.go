// GENERATED. DO NOT MODIFY!

package types

import (
	"encoding/json"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/util"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web"
)

const OperationType ObjectType = web.OperationsURL

type Operations struct {
	Operations []*Operation `json:"operations"`
}

func (e *Operations) Add(object Object) {
	e.Operations = append(e.Operations, object.(*Operation))
}

func (e *Operations) ItemAt(index int) Object {
	return e.Operations[index]
}

func (e *Operations) Len() int {
	return len(e.Operations)
}

func (e *Operation) GetType() ObjectType {
	return OperationType
}

// MarshalJSON override json serialization for http response
func (e *Operation) MarshalJSON() ([]byte, error) {
	type E Operation
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
