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

	"github.com/Peripli/service-manager/pkg/query"

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

type Payload struct {
	New          *ObjectPayload     `json:"new,omitempty"`
	Old          *ObjectPayload     `json:"old,omitempty"`
	LabelChanges query.LabelChanges `json:"label_changes,omitempty"`
}

type ObjectPayload struct {
	Resource   Object         `json:"resource,omitempty"`
	Additional json.Marshaler `json:"additional,omitempty"`
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

func (p *Payload) Validate(op OperationType) error {
	switch op {
	case CREATED:
		if p.New == nil {
			return fmt.Errorf("new resource is required for CREATED notifications")
		}

		if err := p.New.Resource.Validate(); err != nil {
			return fmt.Errorf("invalid new resource in CREATED notification: %s", err)
		}
	case MODIFIED:
		if p.Old == nil {
			return fmt.Errorf("old resource is required for MODIFIED notifications")
		}

		if err := p.Old.Resource.Validate(); err != nil {
			return fmt.Errorf("invalid old resource in MODIFIED notification: %s", err)
		}

		if p.New == nil {
			return fmt.Errorf("new resource is required for MODIFIED notifications")
		}

		if err := p.New.Resource.Validate(); err != nil {
			return fmt.Errorf("invalid new resource in MODIFIED notification: %s", err)
		}
	case DELETED:
		if p.Old == nil {
			return fmt.Errorf("old resource is required for DELETED notifications")
		}

		if err := p.Old.Resource.Validate(); err != nil {
			return fmt.Errorf("invalid new resource in DELETED notification: %s", err)
		}
	}

	return nil
}
