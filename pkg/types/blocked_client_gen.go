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
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
)

const BlockedClientsType ObjectType = web.BlockedClientsConfigURL

type BlockedClients struct {
	BlockedClients []*BlockedClient `json:"blocked_clients"`
}

func (e *BlockedClients) Add(object Object) {
	e.BlockedClients = append(e.BlockedClients, object.(*BlockedClient))
}

func (e *BlockedClients) ItemAt(index int) Object {
	return e.BlockedClients[index]
}

func (e *BlockedClients) Len() int {
	return len(e.BlockedClients)
}

func (e *BlockedClients) GetType() ObjectType {
	return BlockedClientsType
}

// MarshalJSON override json serialization for http response
func (e *BlockedClient) MarshalJSON() ([]byte, error) {
	type E BlockedClient
	toMarshal := struct {
		*E
		CreatedAt *string `json:"created_at,omitempty"`
		UpdatedAt *string `json:"updated_at,omitempty"`
		Labels    Labels  `json:"labels,omitempty"`
	}{
		E:      (*E)(e),
		Labels: e.Labels,
	}
	if !e.CreatedAt.IsZero() {
		str := util.ToRFCNanoFormat(e.CreatedAt)
		toMarshal.CreatedAt = &str
	}
	if !e.UpdatedAt.IsZero() {
		str := util.ToRFCNanoFormat(e.UpdatedAt)
		toMarshal.UpdatedAt = &str
	}
	hasNoLabels := true
	for key, values := range e.Labels {
		if key != "" && len(values) != 0 {
			hasNoLabels = false
			break
		}
	}
	if hasNoLabels {
		toMarshal.Labels = nil
	}
	return json.Marshal(toMarshal)
}
