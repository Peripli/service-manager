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

package postgres

import "time"

import "github.com/Peripli/service-manager/rest"

// Credentials dto
type Credentials struct {
	ID       int    `db:"id"`
	Type     int    `db:"type"`
	Username string `db:"username"`
	Password string `db:"password"`
	Token    string `db:"token"`
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
	Description   string `db:"description"`
	CreatedAt     string `db:"created_at"`
	UpdatedAt     string `db:"updated_at"`
	BrokerURL     string `db:"broker_url"`
	CredentialsID int    `db:"credentials_id"`
}

func (brokerDTO *Broker) ConvertToRestModel() *rest.Broker {
	return &rest.Broker{ID: brokerDTO.ID,
		Name:        brokerDTO.Name,
		Description: brokerDTO.Description,
		CreatedAt:   brokerDTO.CreatedAt,
		UpdatedAt:   brokerDTO.UpdatedAt,
		BrokerURL:   brokerDTO.BrokerURL}
}

func ConvertCredentialsToDTO(credentials *rest.Credentials) *Credentials {
	return &Credentials{
		Type:     1,
		Username: credentials.Basic.Username,
		Password: credentials.Basic.Password,
		Token:    "",
	}
}

func ConvertBrokerToDTO(broker *rest.Broker) *Broker {
	return &Broker{
		ID:          broker.ID,
		Name:        broker.Name,
		Description: broker.Description,
		BrokerURL:   broker.BrokerURL,
		CreatedAt:   broker.CreatedAt,
		UpdatedAt:   broker.UpdatedAt,
	}
}
