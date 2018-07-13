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
	"errors"

	"github.com/Peripli/service-manager/types"
)

var (
	// ErrNotFound error returned from storage when entity is not found
	ErrNotFound = errors.New("not found")

	// ErrUniqueViolation error returned from storage when entity has conflicting fields
	ErrUniqueViolation = errors.New("unique constraint violation")
)

// Settings type to be loaded from the environment
type Settings struct {
	URI string
}

// Storage interface provides entity-specific storages
//go:generate counterfeiter . Storage
type Storage interface {
	// Open initializes the storage, e.g. opens a connection to the underlying storage
	Open(uri string) error

	// Close clears resources associated with this storage, e.g. closes the connection the underlying storage
	Close() error

	// Ping verifies a connection to the database is still alive, establishing a connection if necessary.
	Ping() error

	// Broker provides access to service broker db operations
	Broker() Broker

	// Platform provides access to platform db operations
	Platform() Platform
}

// Broker interface for Broker db operations
//go:generate counterfeiter . Broker
type Broker interface {
	// Create stores a broker in SM DB
	Create(broker *types.Broker) error

	// Get retrieves a broker using the provided id from SM DB
	Get(id string) (*types.Broker, error)

	// GetAll retrieves all brokers from SM DB
	GetAll() ([]types.Broker, error)

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
	GetAll() ([]types.Platform, error)

	// Delete deletes a platform from SM DB
	Delete(id string) error

	// Update updates a platform from SM DB
	Update(platform *types.Platform) error
}
