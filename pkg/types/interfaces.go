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

import "fmt"

type ObjectType string

const (
	BrokerType          ObjectType = "broker"
	PlatformType        ObjectType = "platform"
	ServiceOfferingType ObjectType = "service_offering"
	ServicePlanType     ObjectType = "service_plan"
	VisibilityType      ObjectType = "visibility"
)

var (
	knownTypes []ObjectType
)

func RegisterType(objectType ObjectType) error {
	for _, existing := range knownTypes {
		if existing == objectType {
			return fmt.Errorf("type %s is already registered", objectType)
		}
	}
	knownTypes = append(knownTypes, objectType)
	return nil
}

type Object interface {
	GetType() ObjectType
	SupportsLabels() bool
	GetLabels() Labels
	EmptyList() ObjectList
	WithLabels(labels Labels) Object
}

type ObjectList interface {
	Add(object Object)
	ItemAt(index int) Object
	Len() int
}
