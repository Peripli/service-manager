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

// OperationCategory is the type of an operation
type OperationCategory string

const (
	// CREATE represents an operation type for creating a resource
	CREATE OperationCategory = "create"

	// UPDATE represents an operation type for updating a resource
	UPDATE OperationCategory = "update"

	// DELETE represents an operation type for deleting a resource
	DELETE OperationCategory = "delete"
)

// OperationState is the state of an operation
type OperationState string

const (
	// SUCCEEDED represents the state of an operation after successful execution
	SUCCEEDED OperationState = "succeeded"

	// IN_PROGRESS represents the state of an operation after execution has started but has not yet finished
	IN_PROGRESS OperationState = "in progress"

	// FAILED represents the state of an operation after unsuccessful execution
	FAILED OperationState = "failed"
)

//go:generate smgen api Operation
// Operation struct
type Operation struct {
	Base
	Description   string            `json:"description"`
	Type          OperationCategory `json:"type"`
	State         OperationState    `json:"state"`
	ResourceID    string            `json:"resource_id"`
	ResourceType  string            `json:"resource_type"`
	Errors        json.RawMessage   `json:"errors"`
	CorrelationID string            `json:"correlation_id"`
	ExternalID    string            `json:"-"`
}

// Validate implements InputValidator and verifies all mandatory fields are populated
func (o *Operation) Validate() error {
	if util.HasRFC3986ReservedSymbols(o.ID) {
		return fmt.Errorf("%s contains invalid character(s)", o.ID)
	}

	if o.Type == "" {
		return fmt.Errorf("missing operation type")
	}

	if o.State == "" {
		return fmt.Errorf("missing operation state")
	}

	if o.ResourceID == "" {
		return fmt.Errorf("missing resource_id")
	}

	if o.ResourceType == "" {
		return fmt.Errorf("missing resource_type")
	}

	return nil
}
