// GENERATED. DO NOT MODIFY!

package types

import (
	"encoding/json"

	"github.com/Peripli/service-manager/pkg/util"
)

const NotificationType ObjectType = "types.Notification"

type Notifications struct {
	Notifications []*Notification `json:"notifications"`
}

func (e *Notifications) Add(object Object) {
	e.Notifications = append(e.Notifications, object.(*Notification))
}

func (e *Notifications) ItemAt(index int) Object {
	return e.Notifications[index]
}

func (e *Notifications) Len() int {
	return len(e.Notifications)
}

func (e *Notification) GetType() ObjectType {
	return NotificationType
}

// MarshalJSON override json serialization for http response
func (e *Notification) MarshalJSON() ([]byte, error) {
	type E Notification
	toMarshal := struct {
		*E
		CreatedAt *string `json:"created_at,omitempty"`
		UpdatedAt *string `json:"updated_at,omitempty"`
	}{
		E: (*E)(e),
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
