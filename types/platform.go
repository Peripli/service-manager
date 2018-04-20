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
	return json.Marshal(&struct {
		CreatedAt string `json:"created_at,omitempty"`
		UpdatedAt string `json:"updated_at,omitempty"`
		*P
	}{
		P:         (*P)(p),
		CreatedAt: util.ToRFCFormat(p.CreatedAt),
		UpdatedAt: util.ToRFCFormat(p.UpdatedAt),
	})
}
