/*
 * Copyright 2018 The Service Manager Authors
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */

package types

import (
	"encoding/json"
	"fmt"
	"time"

	"errors"

	"github.com/Peripli/service-manager/pkg/util"
)

// Visibilities struct
type Visibilities struct {
	Visibilities []*Visibility `json:"visibilities"`
}

func (v *Visibilities) Add(object Object) {
	v.Visibilities = append(v.Visibilities, object.(*Visibility))
}

func (v *Visibilities) ItemAt(index int) Object {
	return v.Visibilities[index]
}

func (v *Visibilities) Len() int {
	return len(v.Visibilities)
}

// Visibility struct
type Visibility struct {
	ID            string    `json:"id"`
	PlatformID    string    `json:"platform_id"`
	ServicePlanID string    `json:"service_plan_id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	Labels        Labels    `json:"labels,omitempty"`
}

func (v *Visibility) SetID(id string) {
	panic("implement me")
}

func (v *Visibility) GetID() string {
	panic("implement me")
}

func (v *Visibility) SetCreatedAt(time time.Time) {
	panic("implement me")
}

func (v *Visibility) GetCreatedAt() time.Time {
	panic("implement me")
}

func (v *Visibility) SetUpdatedAt(time time.Time) {
	panic("implement me")
}

func (v *Visibility) GetUpdatedAt() time.Time {
	panic("implement me")
}

func (v *Visibility) SetCredentials(credentials *Credentials) {

}

func (v *Visibility) SupportsLabels() bool {
	return true
}

func (v *Visibility) GetType() ObjectType {
	return VisibilityType
}

func (v *Visibility) GetLabels() Labels {
	return v.Labels
}

func (v *Visibility) EmptyList() ObjectList {
	return &Visibilities{Visibilities: make([]*Visibility, 0)}
}

func (v *Visibility) SetLabels(labels Labels) {
	v.Labels = labels
	return
}

// Validate implements InputValidator and verifies all mandatory fields are populated
func (v *Visibility) Validate() error {
	if v.ServicePlanID == "" {
		return errors.New("missing visibility service plan id")
	}
	if util.HasRFC3986ReservedSymbols(v.ID) {
		return fmt.Errorf("%s contains invalid character(s)", v.ID)
	}
	if err := v.Labels.Validate(); err != nil {
		return err
	}
	return nil
}

// MarshalJSON override json serialization for http response
func (v *Visibility) MarshalJSON() ([]byte, error) {
	type V Visibility
	toMarshal := struct {
		*V
		CreatedAt *string `json:"created_at,omitempty"`
		UpdatedAt *string `json:"updated_at,omitempty"`
	}{
		V: (*V)(v),
	}
	if !v.CreatedAt.IsZero() {
		str := util.ToRFCFormat(v.CreatedAt)
		toMarshal.CreatedAt = &str
	}
	if !v.UpdatedAt.IsZero() {
		str := util.ToRFCFormat(v.UpdatedAt)
		toMarshal.UpdatedAt = &str
	}

	hasNoLabels := true
	for key, values := range v.Labels {
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
