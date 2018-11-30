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
	"time"

	"github.com/Peripli/service-manager/pkg/util"
)

// ServiceOfferings struct
type ServiceOfferings struct {
	ServiceOfferings []*ServiceOffering `json:"services"`
}

// Service Offering struct
type ServiceOffering struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

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

// MarshalJSON override json serialization for http response
func (so *ServiceOffering) MarshalJSON() ([]byte, error) {
	type SO ServiceOffering
	toMarshal := struct {
		CreatedAt *string `json:"created_at,omitempty"`
		UpdatedAt *string `json:"updated_at,omitempty"`
		*SO
	}{
		SO: (*SO)(so),
	}

	if !so.CreatedAt.IsZero() {
		str := util.ToRFCFormat(so.CreatedAt)
		toMarshal.CreatedAt = &str
	}
	if !so.UpdatedAt.IsZero() {
		str := util.ToRFCFormat(so.UpdatedAt)
		toMarshal.UpdatedAt = &str
	}
	return json.Marshal(toMarshal)
}

// Validate implements InputValidator and verifies all mandatory fields are populated
func (so *ServiceOffering) Validate() error {
	if util.HasRFC3986ReservedSymbols(so.ID) {
		return fmt.Errorf("%s contains invalid character(s)", so.ID)
	}
	if so.Name == "" {
		return errors.New("service offering catalog name missing")
	}
	if so.CatalogID == "" {
		return errors.New("service offering catalog id missing")
	}
	if so.CatalogName == "" {
		return errors.New("service offering catalog name missing")
	}
	if so.BrokerID == "" {
		return errors.New("service offering broker id missing")
	}
	return nil
}
