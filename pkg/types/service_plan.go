/*
 *    Copyright 2018 The Service Manager Authors
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

	"github.com/Peripli/service-manager/pkg/util"
)

// Service Plan struct
type ServicePlan struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	CatalogID     string `json:"catalog_id"`
	CatalogName   string `json:"catalog_name"`
	Free          bool   `json:"free"`
	Bindable      bool   `json:"bindable"`
	PlanUpdatable bool   `json:"plan_updateable"`

	Metadata json.RawMessage `json:"metadata,omitempty"`
	Schemas  json.RawMessage `json:"schemas"`

	ServiceOfferingID string `json:"service_offering_id"`
}

// MarshalJSON override json serialization for http response
func (sp *ServicePlan) MarshalJSON() ([]byte, error) {
	type SP ServicePlan
	toMarshal := struct {
		CreatedAt *string `json:"created_at,omitempty"`
		UpdatedAt *string `json:"updated_at,omitempty"`
		*SP
	}{
		SP: (*SP)(sp),
	}

	if !sp.CreatedAt.IsZero() {
		str := util.ToRFCFormat(sp.CreatedAt)
		toMarshal.CreatedAt = &str
	}
	if !sp.UpdatedAt.IsZero() {
		str := util.ToRFCFormat(sp.UpdatedAt)
		toMarshal.UpdatedAt = &str
	}
	return json.Marshal(toMarshal)
}

// Validate implements InputValidator and verifies all mandatory fields are populated
func (sp *ServicePlan) Validate() error {
	if util.HasRFC3986ReservedSymbols(sp.ID) {
		return fmt.Errorf("%s contains invalid character(s)", sp.ID)
	}
	if sp.Name == "" {
		return fmt.Errorf("service plan name missing")
	}
	if sp.CatalogID == "" {
		return fmt.Errorf("service plan catalog id missing")
	}
	if sp.CatalogName == "" {
		return fmt.Errorf("service plan catalog name missing")
	}
	if sp.ServiceOfferingID == "" {
		return fmt.Errorf("service plan service offering id missing")
	}
	return nil
}
