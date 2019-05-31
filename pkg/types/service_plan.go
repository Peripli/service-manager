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

	"github.com/Peripli/service-manager/pkg/util"
)

//go:generate smgen api ServicePlan
// Service Plan struct
type ServicePlan struct {
	Base
	Name        string `json:"name"`
	Description string `json:"description"`

	CatalogID     string `json:"catalog_id"`
	CatalogName   string `json:"catalog_name"`
	Free          bool   `json:"free"`
	Bindable      bool   `json:"bindable"`
	PlanUpdatable bool   `json:"plan_updateable"`

	Metadata json.RawMessage `json:"metadata,omitempty"`
	Schemas  json.RawMessage `json:"schemas,omitempty"`

	ServiceOfferingID string `json:"service_offering_id"`
}

// Validate implements InputValidator and verifies all mandatory fields are populated
func (e *ServicePlan) Validate() error {
	if util.HasRFC3986ReservedSymbols(e.ID) {
		return fmt.Errorf("%s contains invalid character(s)", e.ID)
	}
	if e.Name == "" {
		return fmt.Errorf("service plan name missing")
	}
	if e.CatalogID == "" {
		return fmt.Errorf("service plan catalog id missing")
	}
	if e.CatalogName == "" {
		return fmt.Errorf("service plan catalog name missing")
	}
	if e.ServiceOfferingID == "" {
		return fmt.Errorf("service plan service offering id missing")
	}
	var obj map[string]interface{}
	if len(e.Schemas) != 0 {
		if err := json.Unmarshal(e.Schemas, &obj); err != nil {
			return fmt.Errorf("service plan schemas is invalid JSON")
		}
	}
	if len(e.Metadata) != 0 {
		if err := json.Unmarshal(e.Metadata, &obj); err != nil {
			return fmt.Errorf("service plan metadata is invalid JSON")
		}
	}

	return nil
}
