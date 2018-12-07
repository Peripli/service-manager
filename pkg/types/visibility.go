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

// Visibility struct
type Visibility struct {
	ID            string    `json:"id"`
	PlatformID    string    `json:"platform_id"`
	ServicePlanID string    `json:"service_plan_id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Validate implements InputValidator and verifies all mandatory fields are populated
func (v *Visibility) Validate() error {
	if v.ServicePlanID == "" {
		return errors.New("missing visibility service plan id")
	}
	if util.HasRFC3986ReservedSymbols(v.ID) {
		return fmt.Errorf("%s contains invalid character(s)", v.ID)
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

	return json.Marshal(toMarshal)
}
