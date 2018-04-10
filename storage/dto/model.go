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

import "github.com/Peripli/service-manager/rest"

// Credentials dto
type Credentials struct {
	ID       int    `db:"id"`
	Username string `db:"username"`
	Password string `db:"password"`
}

// Platform dto
type Platform struct {
	ID            string `db:"id"`
	Name          string `db:"name"`
	Type          string `db:"type"`
	Description   string `db:"description"`
	CreatedAt     string `db:"created_at"`
	UpdatedAt     string `db:"updated_at"`
	CredentialsID int    `db:"credentials_id"`
}

// Broker dto
type Broker struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
	BrokerURL     string `json:"broker_url"`
	CredentialsID int    `json:"credentials_id"`
}

func (brokerDTO *Broker) ConvertToRestModel() *rest.Broker {
	return &rest.Broker{ID: brokerDTO.ID,
		Name:        brokerDTO.Name,
		Description: brokerDTO.Description,
		CreatedAt:   brokerDTO.CreatedAt,
		UpdatedAt:   brokerDTO.UpdatedAt,
		BrokerURL:   brokerDTO.BrokerURL}
}
