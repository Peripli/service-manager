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

package dto

import "time"

// Credentials dto
type Credentials struct {
	Type     int    `db:"type"`
	Username string `db:"username"`
	Password string `db:"password"`
}

// Platform dto
type Platform struct {
	ID            string    `db:"id"`
	Type          string    `db:"type"`
	Name          string    `db:"name"`
	Description   string    `db:"description"`
	CreatedAt     time.Time `db:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"`
	CredentialsID int       `db:"credentials_id"`
}

// Broker dto
type Broker struct {
	ID            string `db:"id"`
	Name          string `db:"name"`
	URL           string `db:"broker_url"`
	CreatedAt     string `db:"created_at"`
	UpdatedAt     string `db:"updated_at"`
	CredentialsID int    `db:"credentials_id"`
}
