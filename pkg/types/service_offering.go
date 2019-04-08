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
	"errors"
	"fmt"

	"github.com/Peripli/service-manager/pkg/util"
)

//go:generate smgen api ServiceOffering
// Service Offering struct
type ServiceOffering struct {
	Base
	Name        string `json:"name"`
	Description string `json:"description"`

	Bindable             bool   `json:"bindable"`
	InstancesRetrievable bool   `json:"instances_retrievable"`
	BindingsRetrievable  bool   `json:"bindings_retrievable"`
	PlanUpdatable        bool   `json:"plan_updateable"`
	CatalogID            string `json:"catalog_id"`
	CatalogName          string `json:"catalog_name"`

	Tags     json.RawMessage `json:"tags,omitempty"`
	Requires json.RawMessage `json:"requires,omitempty"`
	Metadata json.RawMessage `json:"metadata,omitempty"`

	BrokerID string         `json:"broker_id"`
	Plans    []*ServicePlan `json:"plans"`
}

// Validate implements InputValidator and verifies all mandatory fields are populated
func (e *ServiceOffering) Validate() error {
	if util.HasRFC3986ReservedSymbols(e.ID) {
		return fmt.Errorf("%s contains invalid character(s)", e.ID)
	}
	if e.Name == "" {
		return errors.New("service offering catalog name missing")
	}
	if e.CatalogID == "" {
		return errors.New("service offering catalog id missing")
	}
	if e.CatalogName == "" {
		return errors.New("service offering catalog name missing")
	}
	if e.BrokerID == "" {
		return errors.New("service offering broker id missing")
	}
	return nil
}
