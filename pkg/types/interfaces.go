/*
 * Copyright 2018 The Service Manager Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package types

import (
	"context"
	"strings"
	"time"

	"github.wdf.sap.corp/SvcManager/sm-sap/peripli/service-manager/pkg/util"
)

const prefix = "types."

// ObjectType is the type of the object in the Service Manager
type ObjectType string

func (ot ObjectType) String() string {
	return strings.TrimPrefix(string(ot), prefix)
}

// Strip interface indicates that an object needs to be sanitized before it is returned to the client
type Strip interface {
	Sanitize(context.Context)
}

// Secured interface indicates that an object needs to be processed before stored/retrieved to/from storage
type Secured interface {
	Encrypt(context.Context, func(context.Context, []byte) ([]byte, error)) error
	Decrypt(context.Context, func(context.Context, []byte) ([]byte, error)) error
}

// Object is the common interface that all resources in the Service Manager must implement
type Object interface {
	util.InputValidator

	Equals(object Object) bool
	SetID(id string)
	GetID() string
	GetType() ObjectType
	GetLabels() Labels
	SetLabels(labels Labels)
	SetCreatedAt(time time.Time)
	GetCreatedAt() time.Time
	SetUpdatedAt(time time.Time)
	GetUpdatedAt() time.Time
	GetPagingSequence() int64
	SetReady(bool)
	GetReady() bool
	SetLastOperation(*Operation)
	GetLastOperation() *Operation
}

func Equals(obj, other Object) bool {
	if obj.GetID() != other.GetID() ||
		obj.GetType() != other.GetType() ||
		!obj.GetCreatedAt().Equal(other.GetCreatedAt()) {
		return false
	}
	return true
}

// ObjectList is the interface that lists of objects must implement
type ObjectList interface {
	Add(object Object)
	ItemAt(index int) Object
	Len() int
}

func ObjectListIDsToStringArray(objectList ObjectList) []string {
	var array []string
	for index := 0; index < objectList.Len(); index++ {
		array = append(array, objectList.ItemAt(index).GetID())
	}
	return array
}

// ObjectPage is the DTO for a given page of resources when listing
type ObjectPage struct {
	//Token represents the base64 encoded paging_sequence of the last entity in items list
	Token      string   `json:"token,omitempty"`
	ItemsCount int      `json:"num_items"`
	Items      []Object `json:"items"`
}

// ObjectArray is an ObjectList backed by a slice of Object's
type ObjectArray struct {
	Objects []Object
}

func NewObjectArray(objects ...Object) *ObjectArray {
	return &ObjectArray{objects}
}

func (a *ObjectArray) Add(object Object) {
	a.Objects = append(a.Objects, object)
}

func (a *ObjectArray) ItemAt(index int) Object {
	return a.Objects[index]
}

func (a *ObjectArray) Len() int {
	return len(a.Objects)
}
