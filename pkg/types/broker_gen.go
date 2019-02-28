// GENERATED. DO NOT MODIFY!

package types

import (
	"encoding/json"
	"github.com/Peripli/service-manager/pkg/util"
)

type Brokers struct {
	Brokers []*Broker `json:"brokers"`
}

func (e *Brokers) Add(object Object) {
	e.Brokers = append(e.Brokers, object.(*Broker))
}

func (e *Brokers) ItemAt(index int) Object {
	return e.Brokers[index]
}

func (e *Brokers) Len() int {
	return len(e.Brokers)
}

func (e *Broker) SupportsLabels() bool {
	return true
}

func (e *Broker) EmptyList() ObjectList {
	return &Brokers{Brokers: make([]*Broker, 0)}
}

func (e *Broker) WithLabels(labels Labels) Object {
    e.Labels = labels
	return e
}

func (e *Broker) GetType() ObjectType {
	return BrokerType
}

func (e *Broker) GetLabels() Labels {
    return e.Labels
}

// MarshalJSON override json serialization for http response
func (e *Broker) MarshalJSON() ([]byte, error) {
	type B Broker
	toMarshal := struct {
		*B
		CreatedAt *string `json:"created_at,omitempty"`
		UpdatedAt *string `json:"updated_at,omitempty"`
	}{
		B: (*B)(e),
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

