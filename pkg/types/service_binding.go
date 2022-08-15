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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/util"
)

//go:generate smgen api ServiceBinding
// ServiceBinding struct
type ServiceBinding struct {
	Base
	Secured           `json:"-"`
	Name              string                 `json:"name"`
	ServiceInstanceID string                 `json:"service_instance_id"`
	SyslogDrainURL    string                 `json:"syslog_drain_url,omitempty"`
	RouteServiceURL   string                 `json:"route_service_url,omitempty"`
	VolumeMounts      json.RawMessage        `json:"volume_mounts,omitempty"`
	Endpoints         json.RawMessage        `json:"endpoints,omitempty"`
	Context           json.RawMessage        `json:"context,omitempty"`
	BindResource      json.RawMessage        `json:"bind_resource,omitempty"`
	Credentials       json.RawMessage        `json:"credentials,omitempty"`
	Parameters        map[string]interface{} `json:"parameters,omitempty"`

	Integrity []byte `json:"-"`
}

func (e *ServiceBinding) Encrypt(ctx context.Context, encryptionFunc func(context.Context, []byte) ([]byte, error)) error {
	return e.transform(ctx, encryptionFunc)
}

func (e *ServiceBinding) Decrypt(ctx context.Context, decryptionFunc func(context.Context, []byte) ([]byte, error)) error {
	return e.transform(ctx, decryptionFunc)
}

func (e *ServiceBinding) IntegralData() []byte {
	return e.Credentials
}

func (e *ServiceBinding) SetIntegrity(integrity []byte) {
	e.Integrity = integrity
}

func (e *ServiceBinding) GetIntegrity() []byte {
	return e.Integrity
}

func (e *ServiceBinding) transform(ctx context.Context, transformationFunc func(context.Context, []byte) ([]byte, error)) error {
	if len(e.Credentials) == 0 {
		return nil
	}
	transformedCredentials, err := transformationFunc(ctx, e.Credentials)
	if err != nil {
		return err
	}
	e.Credentials = transformedCredentials
	return nil
}

func (e *ServiceBinding) Equals(obj Object) bool {
	if !Equals(e, obj) {
		return false
	}

	binding := obj.(*ServiceBinding)
	if e.Name != binding.Name ||
		e.ServiceInstanceID != binding.ServiceInstanceID ||
		e.SyslogDrainURL != binding.SyslogDrainURL ||
		e.RouteServiceURL != binding.RouteServiceURL ||
		!reflect.DeepEqual(e.VolumeMounts, binding.VolumeMounts) ||
		!reflect.DeepEqual(e.Endpoints, binding.Endpoints) ||
		!reflect.DeepEqual(e.Context, binding.Context) ||
		!reflect.DeepEqual(e.BindResource, binding.BindResource) ||
		!reflect.DeepEqual(e.Credentials, binding.Credentials) {
		return false
	}

	return true
}

// Validate implements InputValidator and verifies all mandatory fields are populated
func (e *ServiceBinding) Validate() error {
	if util.HasRFC3986ReservedSymbols(e.ID) {
		return fmt.Errorf("%s contains invalid character(s)", e.ID)
	}
	if e.Name == "" {
		return errors.New("missing service binding name")
	}
	if e.ServiceInstanceID == "" {
		return errors.New("missing service binding service instance ID")
	}
	if err := e.Labels.Validate(); err != nil {
		return err
	}

	return nil
}
