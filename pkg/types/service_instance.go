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

// Package types contains the Service Manager web entities
package types

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/Peripli/service-manager/pkg/util"
)

//go:generate smgen api ServiceInstance
// ServiceInstance struct
type ServiceInstance struct {
	Base
	Name            string          `json:"name"`
	ServicePlanID   string          `json:"service_plan_id"`
	PlatformID      string          `json:"platform_id"`
	DashboardURL    string          `json:"-"`
	MaintenanceInfo json.RawMessage `json:"maintenance_info,omitempty"`
	Context         json.RawMessage `json:"-"`
	PreviousValues  json.RawMessage `json:"-"`
	Ready           bool            `json:"ready"`
	Usable          bool            `json:"usable"`
}

func (e *ServiceInstance) Equals(obj Object) bool {
	if !Equals(e, obj) {
		return false
	}

	instance := obj.(*ServiceInstance)
	if e.Name != instance.Name ||
		e.PlatformID != instance.PlatformID ||
		e.ServicePlanID != instance.ServicePlanID ||
		e.DashboardURL != instance.DashboardURL ||
		!reflect.DeepEqual(e.PreviousValues, instance.PreviousValues) ||
		!reflect.DeepEqual(e.Context, instance.Context) ||
		!reflect.DeepEqual(e.MaintenanceInfo, instance.MaintenanceInfo) {
		return false
	}

	return true
}

// Validate implements InputValidator and verifies all mandatory fields are populated
func (e *ServiceInstance) Validate() error {
	if util.HasRFC3986ReservedSymbols(e.ID) {
		return fmt.Errorf("%s contains invalid character(s)", e.ID)
	}
	if e.Name == "" {
		return errors.New("missing service instance name")
	}
	if e.ServicePlanID == "" {
		return errors.New("missing service plan id")
	}
	if e.PlatformID == "" {
		return errors.New("missing platform id")
	}
	if err := e.Labels.Validate(); err != nil {
		return err
	}

	return nil
}
