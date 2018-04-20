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

package types

// Basic basic credentials
type Basic struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Credentials credentials
type Credentials struct {
	Basic *Basic `json:"basic,omitempty"`
}

// NewBasicCredentials returns new basic credentials object
func NewBasicCredentials(username string, password string) *Credentials {
	return &Credentials{
		Basic: &Basic{
			Username: username,
			Password: password,
		},
	}
}
