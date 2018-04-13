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
	"github.com/Peripli/service-manager/types"
	"context"
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
}

// Broker interface for Broker db operations
type Broker interface {
	// Just to showcase
	Create(ctx context.Context, broker *types.Broker) error

	// Just to showcase
	Find(ctx context.Context, id string) (*types.Broker, error)

	// Just to showcase
	FindAll(ctx context.Context) ([]*types.Broker, error)

	// Just to showcase
	Delete(ctx context.Context, id string) error

	// Just to showcase
	Update(ctx context.Context, broker *types.Broker) error
}
