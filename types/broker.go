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

package types

import (
	"encoding/json"
	"time"

	"github.com/Peripli/service-manager/util"
)

// Broker broker struct
type Broker struct {
	ID          string          `json:"id,omitempty"`
	Name        string          `json:"name,omitempty"`
	Description string          `json:"description,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	BrokerURL   string          `json:"broker_url,omitempty"`
	Credentials *Credentials    `json:"credentials,omitempty" structs:"-"`
	Catalog     json.RawMessage `json:"catalog,omitempty"`
}

// MarshalJSON override json serialization for http response
func (b *Broker) MarshalJSON() ([]byte, error) {
	type B Broker
	toMarshal := struct {
		*B
		CreatedAt *string `json:"created_at,omitempty"`
		UpdatedAt *string `json:"updated_at,omitempty"`
	}{
		B: (*B)(b),
	}
	if !b.CreatedAt.IsZero() {
		str := util.ToRFCFormat(b.CreatedAt)
		toMarshal.CreatedAt = &str
	}
	if !b.UpdatedAt.IsZero() {
		str := util.ToRFCFormat(b.UpdatedAt)
		toMarshal.UpdatedAt = &str
	}

	return json.Marshal(toMarshal)
}
