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
	"fmt"
	"time"

	"errors"

	"github.com/Peripli/service-manager/pkg/util"
)

// Platform platform struct
type Platform struct {
	ID          string       `json:"id"`
	Type        string       `json:"type"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
	Credentials *Credentials `json:"credentials,omitempty"`
}

// MarshalJSON override json serialization for http response
func (p *Platform) MarshalJSON() ([]byte, error) {
	type P Platform
	toMarshal := struct {
		CreatedAt *string `json:"created_at,omitempty"`
		UpdatedAt *string `json:"updated_at,omitempty"`
		*P
	}{
		P: (*P)(p),
	}

	if !p.CreatedAt.IsZero() {
		str := util.ToRFCFormat(p.CreatedAt)
		toMarshal.CreatedAt = &str
	}
	if !p.UpdatedAt.IsZero() {
		str := util.ToRFCFormat(p.UpdatedAt)
		toMarshal.UpdatedAt = &str
	}
	return json.Marshal(toMarshal)
}

// Validate implements InputValidator and verifies all mandatory fields are populated
func (p *Platform) Validate() error {
	if p.Type == "" {
		return errors.New("missing platform type")
	}
	if p.Name == "" {
		return errors.New("missing platform name")
	}
	if util.HasRFC3986ReservedSymbols(p.ID) {
		return fmt.Errorf("%s contains invalid character(s)", p.ID)
	}
	return nil
}
