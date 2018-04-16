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

// Package storage provides generic interfaces around the Service Manager storage and provides logic
// for registration and usage of storages
package storage

import (
	"errors"

	"github.com/Peripli/service-manager/rest"
)

// Storage interface provides entity-specific storages.
//go:generate counterfeiter . Storage
type Storage interface {
	// Open initializes the storage, e.g. opens a connection to the underlying storage
	Open(uri string) error

	// Close clears resources associated with this storage, e.g. closes the connection the underlying storage
	Close() error

	// Broker provides access to service broker db operations
	Broker() Broker

	Platform() Platform
}

// ErrNotFound error returned from storage when entity is not found
var ErrNotFound = errors.New("Not found")

// ErrUniqueViolation error returned from storage when entity has conflicting fields
var ErrUniqueViolation = errors.New("Unique constraint violation")

// Broker interface for Broker db operations
type Broker interface {
	Create(broker *rest.Broker) error
	Get(id string) (*rest.Broker, error)
	GetAll() ([]rest.Broker, error)
	Delete(id string) error
	Update(broker *rest.Broker) error
}

// Platform interface for Platform db operations
type Platform interface {
	Create(platform *rest.Platform) error
	Get(id string) (*rest.Platform, error)
	GetAll() ([]rest.Platform, error)
	Delete(id string) error
	Update(platform *rest.Platform) error
}
