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
	"fmt"

	"github.com/Peripli/service-manager/pkg/util"
)

// OperationType is the notification type
type OperationType string

const (
	// CREATED represents a notification type for creating a resource
	CREATED OperationType = "CREATED"

	// MODIFIED represents a notification type for modifying a resource
	MODIFIED OperationType = "MODIFIED"

	// DELETED represents a notification type for deleting a resource
	DELETED OperationType = "DELETED"

	// InvalidRevision revision with invalid value
	InvalidRevision int64 = -1
)

//go:generate smgen api Notification
// Notification struct
type Notification struct {
	Base
	Resource   ObjectType      `json:"resource"`
	Type       OperationType   `json:"type"`
	PlatformID string          `json:"platform_id,omitempty"`
	Revision   int64           `json:"revision"`
	Payload    json.RawMessage `json:"payload"`
}

// Validate implements InputValidator and verifies all mandatory fields are populated
func (n *Notification) Validate() error {
	if util.HasRFC3986ReservedSymbols(n.ID) {
		return fmt.Errorf("%s contains invalid character(s)", n.ID)
	}
	if n.Resource == "" {
		return fmt.Errorf("notification resource missing")
	}
	if n.Type == "" {
		return fmt.Errorf("notification type missing")
	}

	return nil
}
