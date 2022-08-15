// GENERATED. DO NOT MODIFY!

package types

import (
	"encoding/json"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/util"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web"
)

const VisibilityType ObjectType = web.VisibilitiesURL

type Visibilities struct {
	Visibilities []*Visibility `json:"visibilities"`
}

func (e *Visibilities) Add(object Object) {
	e.Visibilities = append(e.Visibilities, object.(*Visibility))
}

func (e *Visibilities) ItemAt(index int) Object {
	return e.Visibilities[index]
}

func (e *Visibilities) Len() int {
	return len(e.Visibilities)
}

func (e *Visibility) GetType() ObjectType {
	return VisibilityType
}

// MarshalJSON override json serialization for http response
func (e *Visibility) MarshalJSON() ([]byte, error) {
	type E Visibility
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
