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
	"errors"
)

//go:generate smgen api broker labels
// Broker broker struct
type Broker struct {
	Secured
	BaseLabelled
	Name        string             `json:"name"`
	Description string             `json:"description"`
	BrokerURL   string             `json:"broker_url"`
	Credentials *Credentials       `json:"credentials,omitempty" structs:"-"`
	Services    []*ServiceOffering `json:"services,omitempty" structs:"-"`
}

func (e *Broker) SetCredentials(credentials *Credentials) {
	e.Credentials = credentials
}

func (e *Broker) GetCredentials() *Credentials {
	return e.Credentials
}

// Validate implements InputValidator and verifies all mandatory fields are populated
func (b *Broker) Validate() error {
	if b.Name == "" {
		return errors.New("missing broker name")
	}
	if b.BrokerURL == "" {
		return errors.New("missing broker url")
	}

	if err := b.Labels.Validate(); err != nil {
		return err
	}

	if b.Credentials == nil {
		return errors.New("missing credentials")
	}
	return b.Credentials.Validate()
}
