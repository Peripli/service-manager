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

	"encoding/json"

	"github.com/Peripli/service-manager/types"
	sqlxtypes "github.com/jmoiron/sqlx/types"
)

const (
	// platformTable db table name for platforms
	platformTable = "platforms"

	// brokerTable db table name for brokers
	brokerTable = "brokers"
)

// Platform dto
type Platform struct {
	ID          string    `db:"id"`
	Type        string    `db:"type"`
	Name        string    `db:"name"`
	Description string    `db:"description"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
	Username    string    `db:"username"`
	Password    string    `db:"password"`
}

// Broker dto
type Broker struct {
	ID          string             `db:"id"`
	Name        string             `db:"name"`
	Description string             `db:"description"`
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
		Description: brokerDTO.Description,
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
		Description: platformDTO.Description,
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
		ID:          platform.ID,
		Type:        platform.Type,
		Name:        platform.Name,
		Description: platform.Description,
		CreatedAt:   platform.CreatedAt,
		UpdatedAt:   platform.UpdatedAt,
	}
	if platform.Credentials != nil {
		result.Username = platform.Credentials.Basic.Username
		result.Password = platform.Credentials.Basic.Password
	}
	return result
}

func convertBrokerToDTO(broker *types.Broker) *Broker {
	result := &Broker{
		ID:          broker.ID,
		Name:        broker.Name,
		Description: broker.Description,
		BrokerURL:   broker.BrokerURL,
		CreatedAt:   broker.CreatedAt,
		UpdatedAt:   broker.UpdatedAt,
		Catalog:     sqlxtypes.JSONText(broker.Catalog),
	}
	if broker.Credentials != nil {
		result.Username = broker.Credentials.Basic.Username
		result.Password = broker.Credentials.Basic.Password
	}
	return result
}
