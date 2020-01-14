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

//go:generate smgen api ServiceBinding
// ServiceBinding struct
type ServiceBinding struct {
	Base
	CredentialsObject `json:"-"`
	Name              string          `json:"name"`
	ServiceInstanceID string          `json:"service_instance_id"`
	SyslogDrainURL    string          `json:"syslog_drain_url,omitempty"`
	RouteServiceURL   string          `json:"route_service_url,omitempty"`
	VolumeMounts      json.RawMessage `json:"volume_mounts,omitempty"`
	Endpoints         json.RawMessage `json:"endpoints,omitempty"`
	Context           json.RawMessage `json:"-"`
	BindResource      json.RawMessage `json:"-"`
	Credentials       json.RawMessage `json:"credentials"`
}

func (sb *ServiceBinding) SetCredentials(credentials json.RawMessage) {
	sb.Credentials = credentials
}

func (sb *ServiceBinding) GetCredentials() json.RawMessage {
	return sb.Credentials
}

func (sb *ServiceBinding) Equals(obj Object) bool {
	if !Equals(sb, obj) {
		return false
	}

	binding := obj.(*ServiceBinding)
	if sb.Name != binding.Name ||
		sb.ServiceInstanceID != binding.ServiceInstanceID ||
		sb.SyslogDrainURL != binding.SyslogDrainURL ||
		sb.RouteServiceURL != binding.RouteServiceURL ||
		!reflect.DeepEqual(sb.VolumeMounts, binding.VolumeMounts) ||
		!reflect.DeepEqual(sb.Endpoints, binding.Endpoints) ||
		!reflect.DeepEqual(sb.Context, binding.Context) ||
		!reflect.DeepEqual(sb.BindResource, binding.BindResource) ||
		!reflect.DeepEqual(sb.Credentials, binding.Credentials) {
		return false
	}

	return true
}

// Validate implements InputValidator and verifies all mandatory fields are populated
func (sb *ServiceBinding) Validate() error {
	if util.HasRFC3986ReservedSymbols(sb.ID) {
		return fmt.Errorf("%s contains invalid character(s)", sb.ID)
	}
	if sb.Name == "" {
		return errors.New("missing service binding name")
	}
	if err := sb.Labels.Validate(); err != nil {
		return err
	}

	return nil
}
