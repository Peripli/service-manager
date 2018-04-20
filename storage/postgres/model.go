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

import (
	"time"

	"github.com/Peripli/service-manager/types"
)

const (
	// platformTable db table name for platforms
	platformTable = "platforms"

	// brokerTable db table name for brokers
	brokerTable = "brokers"

	// table db table name for credentials
	credentialsTable = "credentials"

	basicCredentialsType = 1
)

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
	ID            string    `db:"id"`
	Name          string    `db:"name"`
	Description   string    `db:"description"`
	CreatedAt     time.Time `db:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"`
	BrokerURL     string    `db:"broker_url"`
	CredentialsID int       `db:"credentials_id"`
}

// Convert converts to types.Broker
func (brokerDTO *Broker) Convert() *types.Broker {
	return &types.Broker{ID: brokerDTO.ID,
		Name:        brokerDTO.Name,
		Description: brokerDTO.Description,
		CreatedAt:   brokerDTO.CreatedAt,
		UpdatedAt:   brokerDTO.UpdatedAt,
		BrokerURL:   brokerDTO.BrokerURL,
	}
}

// Convert converts to types.Platform
func (platformDTO *Platform) Convert() *types.Platform {
	return &types.Platform{
		ID:          platformDTO.ID,
		Type:        platformDTO.Type,
		Name:        platformDTO.Name,
		Description: platformDTO.Description,
		CreatedAt:   platformDTO.CreatedAt,
		UpdatedAt:   platformDTO.UpdatedAt,
	}
}

func convertCredentialsToDTO(credentials *types.Credentials) *Credentials {
	return &Credentials{
		Type:     basicCredentialsType,
		Username: credentials.Basic.Username,
		Password: credentials.Basic.Password,
		Token:    "",
	}
}

func convertPlatformToDTO(platform *types.Platform) *Platform {
	return &Platform{
		ID:          platform.ID,
		Type:        platform.Type,
		Name:        platform.Name,
		Description: platform.Description,
		CreatedAt:   platform.CreatedAt,
		UpdatedAt:   platform.UpdatedAt,
	}
}

func convertBrokerToDTO(broker *types.Broker) *Broker {
	return &Broker{
		ID:          broker.ID,
		Name:        broker.Name,
		Description: broker.Description,
		BrokerURL:   broker.BrokerURL,
		CreatedAt:   broker.CreatedAt,
		UpdatedAt:   broker.UpdatedAt,
	}
}
