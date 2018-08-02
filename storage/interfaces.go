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

// Package storage contains logic around the Service Manager persistent storage
package storage

import (
	"fmt"

	"github.com/Peripli/service-manager/pkg/types"
	"github.com/Peripli/service-manager/security"
)

// Settings type to be loaded from the environment
type Settings struct {
	URI string
}

// Validate validates the storage settings
func (s *Settings) Validate() error {
	if len(s.URI) == 0 {
		return fmt.Errorf("validate Settings: StorageURI missing")
	}
	return nil
}

// Storage interface provides entity-specific storages
//go:generate counterfeiter . Storage
type Storage interface {
	// Open initializes the storage, e.g. opens a connection to the underlying storage
	Open(uri string, encryptionKey []byte) error

	// Close clears resources associated with this storage, e.g. closes the connection the underlying storage
	Close() error

	// Ping verifies a connection to the database is still alive, establishing a connection if necessary.
	Ping() error

	// Broker provides access to service broker db operations
	Broker() Broker

	// Platform provides access to platform db operations
	Platform() Platform

	// Credentials provides access to credentials db operations
	Credentials() Credentials

	// Security provides access to encryption key management
	Security() Security
}

// Broker interface for Broker db operations
//go:generate counterfeiter . Broker
type Broker interface {
	// Create stores a broker in SM DB
	Create(broker *types.Broker) error

	// Get retrieves a broker using the provided id from SM DB
	Get(id string) (*types.Broker, error)

	// GetAll retrieves all brokers from SM DB
	GetAll() ([]*types.Broker, error)

	// Delete deletes a broker from SM DB
	Delete(id string) error

	// Update updates a broker from SM DB
	Update(broker *types.Broker) error
}

// Platform interface for Platform db operations
//go:generate counterfeiter . Platform
type Platform interface {
	// Create stores a platform in SM DB
	Create(platform *types.Platform) error

	// Get retrieves a platform using the provided id from SM DB
	Get(id string) (*types.Platform, error)

	// GetAll retrieves all platforms from SM DB
	GetAll() ([]*types.Platform, error)

	// Delete deletes a platform from SM DB
	Delete(id string) error

	// Update updates a platform from SM DB
	Update(platform *types.Platform) error
}

// Credentials interface for Credentials db operations
type Credentials interface {
	// Get retrieves credentials using the provided username from SM DB
	Get(username string) (*types.Credentials, error)
}

// Security interface for encryption key operations
type Security interface{
	// Fetcher provides means to obtain the encryption key
	Fetcher() security.KeyFetcher
	// Setter provides means to change the encryption key
	Setter() security.KeySetter
}