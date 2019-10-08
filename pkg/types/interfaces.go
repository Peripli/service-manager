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
	"time"

	"github.com/Peripli/service-manager/pkg/util"
)

// ObjectType is the type of the object in the Service Manager
type ObjectType string

// Secured interface indicates that an object requires credentials to access it
type Secured interface {
	SetCredentials(credentials *Credentials)
	GetCredentials() *Credentials
}

// Object is the common interface that all resources in the Service Manager must implement
type Object interface {
	util.InputValidator

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
}

// ObjectList is the interface that lists of objects must implement
type ObjectList interface {
	Add(object Object)
	ItemAt(index int) Object
	Len() int
}

// ObjectPage is the DTO for a given page of resources when listing
type ObjectPage struct {
	//Token represents the base64 encoded paging_sequence of the last entity in items list
	Token      string   `json:"token,omitempty"`
	ItemsCount int      `json:"num_items"`
	Items      []Object `json:"items"`
}
