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
	"database/sql"
	"encoding/json"
	"time"

	"github.com/Peripli/service-manager/pkg/types"
	sqlxtypes "github.com/jmoiron/sqlx/types"
)

const (
	// platformTable db table name for platforms
	platformTable = "platforms"

	// brokerTable db table name for brokers
	brokerTable = "brokers"
)

// Safe is a representation of how a secret is stored
type Safe struct {
	Secret    []byte    `db:"secret"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// Platform dto
type Platform struct {
	ID          string         `db:"id"`
	Type        string         `db:"type"`
	Name        string         `db:"name"`
	Description sql.NullString `db:"description"`
	CreatedAt   time.Time      `db:"created_at"`
	UpdatedAt   time.Time      `db:"updated_at"`
	Username    string         `db:"username"`
	Password    string         `db:"password"`
}

// Broker dto
type Broker struct {
	ID          string             `db:"id"`
	Name        string             `db:"name"`
	Description sql.NullString     `db:"description"`
	CreatedAt   time.Time          `db:"created_at"`
	UpdatedAt   time.Time          `db:"updated_at"`
	BrokerURL   string             `db:"broker_url"`
	Username    string             `db:"username"`
	Password    string             `db:"password"`
	Catalog     sqlxtypes.JSONText `db:"catalog"`
}

// Convert converts to types.Broker
func (brokerDTO *Broker) Convert() *types.Broker {
	broker := &types.Broker{ID: brokerDTO.ID,
		Name:        brokerDTO.Name,
		Description: brokerDTO.Description.String,
		CreatedAt:   brokerDTO.CreatedAt,
		UpdatedAt:   brokerDTO.UpdatedAt,
		BrokerURL:   brokerDTO.BrokerURL,
		Catalog:     json.RawMessage(brokerDTO.Catalog),
	}
	if brokerDTO.Username != "" {
		broker.Credentials = types.NewBasicCredentials(brokerDTO.Username, brokerDTO.Password)
	}
	return broker
}

// Convert converts to types.Platform
func (platformDTO *Platform) Convert() *types.Platform {
	platform := &types.Platform{
		ID:          platformDTO.ID,
		Type:        platformDTO.Type,
		Name:        platformDTO.Name,
		Description: platformDTO.Description.String,
		CreatedAt:   platformDTO.CreatedAt,
		UpdatedAt:   platformDTO.UpdatedAt,
	}
	if platformDTO.Username != "" {
		platform.Credentials = types.NewBasicCredentials(platformDTO.Username, platformDTO.Password)
	}
	return platform
}

func convertPlatformToDTO(platform *types.Platform) *Platform {
	result := &Platform{
		ID:        platform.ID,
		Type:      platform.Type,
		Name:      platform.Name,
		CreatedAt: platform.CreatedAt,
		UpdatedAt: platform.UpdatedAt,
	}
	if platform.Description != "" {
		result.Description = sql.NullString{String: platform.Description, Valid: true}
	}
	if platform.Credentials != nil {
		result.Username = platform.Credentials.Basic.Username
		result.Password = platform.Credentials.Basic.Password
	}
	return result
}

func convertBrokerToDTO(broker *types.Broker) *Broker {
	result := &Broker{
		ID:        broker.ID,
		Name:      broker.Name,
		BrokerURL: broker.BrokerURL,
		CreatedAt: broker.CreatedAt,
		UpdatedAt: broker.UpdatedAt,
		Catalog:   sqlxtypes.JSONText(broker.Catalog),
	}
	if broker.Description != "" {
		result.Description = sql.NullString{String: broker.Description, Valid: true}
	}
	if broker.Credentials != nil {
		result.Username = broker.Credentials.Basic.Username
		result.Password = broker.Credentials.Basic.Password
	}
	return result
}
