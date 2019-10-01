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
	"errors"
	"fmt"
	"time"

	"github.com/Peripli/service-manager/pkg/util"
)

//go:generate smgen api Platform
// Platform platform struct
type Platform struct {
	Base
	Secured     `json:"-"`
	Type        string       `json:"type"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Credentials *Credentials `json:"credentials,omitempty"`
	Active      bool         `json:"-"`
	LastActive  time.Time    `json:"-"`
}

func (e *Platform) SetCredentials(credentials *Credentials) {
	e.Credentials = credentials
}

func (e *Platform) GetCredentials() *Credentials {
	return e.Credentials
}

// Validate implements InputValidator and verifies all mandatory fields are populated
func (e *Platform) Validate() error {
	if e.Type == "" {
		return errors.New("missing platform type")
	}
	if e.Name == "" {
		return errors.New("missing platform name")
	}
	if util.HasRFC3986ReservedSymbols(e.ID) {
		return fmt.Errorf("%s contains invalid character(s)", e.ID)
	}
	return nil
}
