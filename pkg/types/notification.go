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
	"reflect"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/util"
)

// NotificationOperation is the notification type
type NotificationOperation string

const (
	// CREATED represents a notification type for creating a resource
	CREATED NotificationOperation = "CREATED"

	// MODIFIED represents a notification type for modifying a resource
	MODIFIED NotificationOperation = "MODIFIED"

	// DELETED represents a notification type for deleting a resource
	DELETED NotificationOperation = "DELETED"

	// InvalidRevision revision with invalid value
	InvalidRevision int64 = -1
)

//go:generate smgen api Notification
// Notification struct
type Notification struct {
	Base
	Resource      ObjectType            `json:"resource"`
	Type          NotificationOperation `json:"type"`
	PlatformID    string                `json:"platform_id,omitempty"`
	Revision      int64                 `json:"revision"`
	Payload       json.RawMessage       `json:"payload"`
	CorrelationID string                `json:"correlation_id"`
}

func (e *Notification) Equals(obj Object) bool {
	if !Equals(e, obj) {
		return false
	}

	notification := obj.(*Notification)
	if e.PlatformID != notification.PlatformID ||
		e.Type != notification.Type ||
		e.Resource != notification.Resource ||
		e.Revision != notification.Revision ||
		e.CorrelationID != notification.CorrelationID ||
		!reflect.DeepEqual(e.Payload, notification.Payload) {
		return false
	}

	return true
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
