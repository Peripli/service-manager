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
	"time"

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
	// PENDING represents the state of an operation that waiting to be handled
	PENDING OperationState = "pending"

	// SUCCEEDED represents the state of an operation after successful execution
	SUCCEEDED OperationState = "succeeded"

	// IN_PROGRESS represents the state of an operation after execution has started but has not yet finished
	IN_PROGRESS OperationState = "in progress"

	// FAILED represents the state of an operation after unsuccessful execution
	FAILED OperationState = "failed"
)

type RelatedType struct {
	ID            string            `json:"id,omitempty"`
	Criteria      interface{}       `json:"criteria,omitempty"`
	Type          ObjectType        `json:"type"`
	OperationType OperationCategory `json:"operation_type"`
}

//go:generate smgen api Operation
// Operation struct
type Operation struct {
	Base
	Description         string            `json:"description,omitempty"`
	Type                OperationCategory `json:"type"`
	State               OperationState    `json:"state"`
	ResourceID          string            `json:"resource_id"`
	TransitiveResources []*RelatedType    `json:"transitive_resources,omitempty"`
	ResourceType        ObjectType        `json:"resource_type"`
	Errors              json.RawMessage   `json:"errors,omitempty"`
	PlatformID          string            `json:"platform_id"`
	CorrelationID       string            `json:"correlation_id"`
	ExternalID          string            `json:"-"`
	ParentID            string            `json:"parent_id,omitempty"`
	CascadeRootID       string            `json:"cascade_root_id,omitempty"`

	// Reschedule specifies that the operation has reached a state after which it can be retried (checkpoint)
	Reschedule bool `json:"reschedule"`
	// RescheduleTimestamp is the time when an operation became reschedulable=true for the first time
	RescheduleTimestamp time.Time `json:"reschedule_timestamp,omitempty"`
	// DeletionScheduled specifies the time when an operation was marked for deletion
	DeletionScheduled time.Time `json:"deletion_scheduled,omitempty"`
}

func (e *Operation) Equals(obj Object) bool {
	if !Equals(e, obj) {
		return false
	}

	operation := obj.(*Operation)
	if e.Description != operation.Description ||
		e.ResourceID != operation.ResourceID ||
		e.ResourceType != operation.ResourceType ||
		e.CorrelationID != operation.CorrelationID ||
		e.ExternalID != operation.ExternalID ||
		e.State != operation.State ||
		e.Type != operation.Type ||
		e.PlatformID != operation.PlatformID ||
		e.CascadeRootID != operation.CascadeRootID ||
		e.ParentID != operation.ParentID ||
		!reflect.DeepEqual(e.Errors, operation.Errors) ||
		!reflect.DeepEqual(e.TransitiveResources, operation.TransitiveResources) {
		return false
	}

	return true
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
		return fmt.Errorf("missing resource id")
	}

	if o.ResourceType == "" {
		return fmt.Errorf("missing resource type")
	}

	if o.State == PENDING && o.CascadeRootID == "" {
		return fmt.Errorf("PENDING state only allowed for cascade operations")
	}

	if len(o.CascadeRootID) > 0 && len(o.ParentID) == 0 && o.CascadeRootID != o.ID {
		return fmt.Errorf("root operation should have the same CascadeRootID and ID")
	}

	return nil
}

func (o *Operation) InOrphanMitigationState() bool {
	return !o.DeletionScheduled.IsZero()
}

func (o *Operation) IsForceDeleteCascadeOperation() bool {
	forceCascade, found := o.Labels["force"]
	hasForceLabel := found && len(forceCascade) > 0 && forceCascade[0] == "true"

	return o.Type == DELETE && o.CascadeRootID != "" && hasForceLabel
}
