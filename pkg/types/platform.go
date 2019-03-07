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
	"fmt"
	"time"

	"errors"

	"github.com/Peripli/service-manager/pkg/util"
)

//go:generate ./generate_type.sh Platform
// Platform platform struct
type Platform struct {
	ID          string       `json:"id"`
	Type        string       `json:"type" valid_string:"regex:[a-z]|len[1,255]"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
	Credentials *Credentials `json:"credentials,omitempty"`
}

func (e *Platform) SetCredentials(credentials *Credentials) {
	panic("implement me")
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
