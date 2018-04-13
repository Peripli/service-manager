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

// Package types defines the entity types used in the Service Manager
package types

// Broker Just to showcase how to use
type Broker struct {
	ID        string `db:"id"`
	Name      string `db:"name"`
	URL       string `db:"broker_url"`
	CreatedAt string `db:"created_at"`
	UpdatedAt string `db:"updated_at"`
	User      string `db:"user"`
	Password  string `db:"password"`
}
