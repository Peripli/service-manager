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

// ServicePlans struct
type ServicePlans struct {
	ServicePlans []Object `json:"service_plans"`
}

func (sps *ServicePlans) Add(object Object) {
	sps.ServicePlans = append(sps.ServicePlans, object)
}

func (sps *ServicePlans) ItemAt(index int) Object {
	return sps.ServicePlans[index]
}

func (sps *ServicePlans) Len() int {
	return len(sps.ServicePlans)
}

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
	Schemas  json.RawMessage `json:"schemas,omitempty"`

	ServiceOfferingID string `json:"service_offering_id"`
}

func (sp *ServicePlan) GetUpdatedAt() time.Time {
	return sp.UpdatedAt
}

func (sp *ServicePlan) GetCreatedAt() time.Time {
	return sp.CreatedAt
}

func (sp *ServicePlan) SetID(id string) {
	sp.ID = id
}

func (sp *ServicePlan) GetID() string {
	return sp.ID
}

func (sp *ServicePlan) SetCreatedAt(time time.Time) {
	sp.CreatedAt = time
}

func (sp *ServicePlan) SetUpdatedAt(time time.Time) {
	sp.UpdatedAt = time
}

func (sp *ServicePlan) SetCredentials(credentials *Credentials) {
	return
}

func (sp *ServicePlan) SupportsLabels() bool {
	return false
}

func (sp *ServicePlan) GetType() ObjectType {
	return ServicePlanType
}

func (sp *ServicePlan) GetLabels() Labels {
	return Labels{}
}

func (sp *ServicePlan) EmptyList() ObjectList {
	return &ServicePlans{ServicePlans: make([]Object, 0)}
}

func (sp *ServicePlan) SetLabels(labels Labels) {
	return
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
