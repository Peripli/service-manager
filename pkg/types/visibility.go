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
	"errors"
	"fmt"
	"time"

	"github.com/Peripli/service-manager/pkg/util"
)

//go:generate smgen api Visibility labels
// Visibility struct
type Visibility struct {
	ID         string    `json:"id"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	PlatformID string    `json:"platform_id"`
	Labels     Labels    `json:"labels,omitempty"`

	ServicePlanID string `json:"service_plan_id"`
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

func (e *Visibility) SetLabels(labels Labels) {
	e.Labels = labels
	return
}

func (e *Visibility) GetLabels() Labels {
	return e.Labels
}

func (e *Visibility) SupportsLabels() bool {
	return true
}

func (e *Visibility) SetID(id string) {
	e.ID = id
}

func (e *Visibility) GetID() string {
	return e.ID
}

func (e *Visibility) SetCreatedAt(time time.Time) {
	e.CreatedAt = time
}

func (e *Visibility) GetCreatedAt() time.Time {
	return e.CreatedAt
}

func (e *Visibility) SetUpdatedAt(time time.Time) {
	e.UpdatedAt = time
}

func (e *Visibility) GetUpdatedAt() time.Time {
	return e.UpdatedAt
}
