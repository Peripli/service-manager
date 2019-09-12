/*
 * Copyright 2018 The Service Manager Authors
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

package web

// AuthenticationType specifies the authentication type that is stored in the user context
type AuthenticationType string

const (
	Basic  AuthenticationType = "Basic"
	Bearer AuthenticationType = "Bearer"
)

// UserContext holds the information for the current user
type UserContext struct {
	// Data unmarshals the additional user context details into the specified struct
	Data func(data interface{}) error
	// AuthenticationType is the authentication type for this user context
	AuthenticationType AuthenticationType
	// Name is the name of the authenticated user
	Name string
}
