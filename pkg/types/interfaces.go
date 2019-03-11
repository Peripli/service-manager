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

import "time"

//TODO after applying the other TODOs i think this becomes unused
type ObjectType string

const (
	PlatformType        ObjectType = "platform"
	ServiceOfferingType ObjectType = "service_offering"
	ServicePlanType     ObjectType = "service_plan"
	VisibilityType      ObjectType = "visibility"
)

type Object interface {
	SetID(id string)
	GetID() string
	//TODO return object so that e don't need the map, rename it to EmptyObject or ObjectInstance
	GetType() ObjectType
	//TODO well probably since we are generating everything now we can move the Labels to a common.Object that will be embeded in all api types and say everything supports labels
	SupportsLabels() bool
	EmptyList() ObjectList //TODO (just naming) if you pick ObjectInstance above, this should be ListInstance and the interface should be just List
	//TODO
	GetLabels() Labels
	SetLabels(labels Labels)
	//TODO only two objects have that so it should probably not be here but i am not certain how to handle removing the credentials on get apis then
	SetCredentials(credentials *Credentials)
	SetCreatedAt(time time.Time)
	GetCreatedAt() time.Time
	SetUpdatedAt(time time.Time)
	GetUpdatedAt() time.Time
}

type ObjectList interface {
	Add(object Object)
	ItemAt(index int) Object
	Len() int
}
