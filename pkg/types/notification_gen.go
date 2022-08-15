// GENERATED. DO NOT MODIFY!

package types

import (
	"encoding/json"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/util"
	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/web"
)

const NotificationType ObjectType = web.NotificationsURL

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
