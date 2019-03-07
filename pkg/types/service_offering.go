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

func (sos *ServiceOfferings) Add(object Object) {
	sos.ServiceOfferings = append(sos.ServiceOfferings, object.(*ServiceOffering))
}

func (sos *ServiceOfferings) ItemAt(index int) Object {
	return sos.ServiceOfferings[index]
}

func (sos *ServiceOfferings) Len() int {
	return len(sos.ServiceOfferings)
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

func (so *ServiceOffering) GetCreatedAt() time.Time {
	return so.CreatedAt
}

func (so *ServiceOffering) SetID(id string) {
	so.ID = id
}

func (so *ServiceOffering) GetID() string {
	return so.ID
}

func (so *ServiceOffering) SetCreatedAt(time time.Time) {
	so.CreatedAt = time
}

func (so *ServiceOffering) SetUpdatedAt(time time.Time) {
	so.UpdatedAt = time
}

func (so *ServiceOffering) SetCredentials(credentials *Credentials) {
	return
}

func (so *ServiceOffering) SupportsLabels() bool {
	return false
}

func (so *ServiceOffering) GetType() ObjectType {
	return ServiceOfferingType
}

func (so *ServiceOffering) GetLabels() Labels {
	return Labels{}
}

func (so *ServiceOffering) EmptyList() ObjectList {
	return &ServiceOfferings{ServiceOfferings: make([]*ServiceOffering, 0)}
}

func (so *ServiceOffering) SetLabels(labels Labels) {
	return
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
